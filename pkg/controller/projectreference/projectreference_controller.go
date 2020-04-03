package projectreference

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	gcpv1alpha1 "github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1"
	"github.com/openshift/gcp-project-operator/pkg/gcpclient"
	"github.com/openshift/gcp-project-operator/pkg/util"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_projectreference")

const (
	// Operator config
	operatorNamespace = "gcp-project-operator"

	// secret information
	orgGcpSecretName = "gcp-project-operator-credentials"

	// Configmap related configs
	orgGcpConfigMap = "gcp-project-operator"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new ProjectReference Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileProjectReference{client: mgr.GetClient(), scheme: mgr.GetScheme(), gcpClientBuilder: gcpclient.NewClient}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("projectreference-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource ProjectReference
	err = c.Watch(&source.Kind{Type: &gcpv1alpha1.ProjectReference{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileProjectReference implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileProjectReference{}

// ReconcileProjectReference reconciles a ProjectReference object
type ReconcileProjectReference struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client           client.Client
	scheme           *runtime.Scheme
	gcpClientBuilder func(projectName string, authJSON []byte) (gcpclient.Client, error)
}

// Reconcile reads that state of the cluster for a ProjectReference object and makes changes based on the state read
// and what is in the ProjectReference.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileProjectReference) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling ProjectReference")

	projectReference := &gcpv1alpha1.ProjectReference{}
	err := r.client.Get(context.TODO(), request.NamespacedName, projectReference)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return r.doNotRequeue()
		}
		return r.requeueOnErr(err)
	}

	gcpClient, err := r.getGcpClient(projectReference.Spec.GCPProjectID, reqLogger)
	if err != nil {
		return r.requeueOnErr(err)
	}

	adapter, err := newReferenceAdapter(projectReference, reqLogger, r.client, gcpClient)
	if err != nil {
		reqLogger.Error(err, "could not create ReferenceAdapter")
		return r.requeueOnErr(err)
	}

	// Cleanup
	if adapter.IsDeletionRequested() {
		err := adapter.EnsureProjectCleanedUp()
		if err != nil {
			return r.requeueAfter(5*time.Second, err)
		}
		return r.doNotRequeue()
	}

	// If ProjectReference is in error state exit and do nothing
	if projectReference.Status.State == gcpv1alpha1.ProjectReferenceStatusError {
		reqLogger.Info("ProjectReference CR is in an Error state")
		return r.doNotRequeue()
	}

	// Make projectReference  be processed based on state of ProjectClaim and Project Reference
	claimStatus, err := adapter.EnsureProjectClaimReady()
	if claimStatus == gcpv1alpha1.ClaimStatusReady || err != nil {
		return r.requeueOnErr(err)
	}

	//only make changes to ProjectReference if ProjelctClaim is pending
	if adapter.projectClaim.Status.State != gcpv1alpha1.ClaimStatusPendingProject {
		return r.requeueAfter(5*time.Second, nil)
	}

	// make sure we meet mimimum requirements to process request and set its state to creating or error if its not supported
	if projectReference.Status.State == "" {
		reqLogger.Info("Checking Requirements")
		err := adapter.checkRequirements()
		if err != nil {
			// TODO: add condition here SupportedRegion = false to give more information on the error state
			reqLogger.Error(err, "Region not supported")
			projectReference.Status.State = gcpv1alpha1.ProjectReferenceStatusError
			err := r.client.Status().Update(context.TODO(), projectReference)
			if err != nil {
				reqLogger.Error(err, "Error updating ProjectReference Status")
				return r.requeueOnErr(err)
			}
			return r.doNotRequeue()
		}

		reqLogger.Info(fmt.Sprintf("Setting ProjectReferenceStatus %s", gcpv1alpha1.ProjectReferenceStatusCreating))
		// passed requirementes check set to creating
		projectReference.Status.State = gcpv1alpha1.ProjectReferenceStatusCreating
		err = r.client.Status().Update(context.TODO(), projectReference)
		if err != nil {
			reqLogger.Error(err, "Error updating ProjectReference Status")
			return r.requeueOnErr(err)
		}
	}

	if projectReference.Spec.GCPProjectID == "" {
		reqLogger.Info("Creating ProjectID in ProjectReference CR")
		err := adapter.updateProjectID()
		if err != nil {
			reqLogger.Error(err, "Could not update ProjectID in Project Reference CR")
			return r.requeueOnErr(err)
		}
		return r.requeue()
	}

	reqLogger.Info("Adding a Finalizer")
	err = adapter.EnsureFinalizerAdded()
	if err != nil {
		reqLogger.Error(err, "Error adding the finalizer")
		return r.requeueOnErr(err)
	}

	reqLogger.Info("Configuring Project")
	err = adapter.EnsureProjectConfigured()
	if err != nil {
		return r.requeueAfter(5*time.Second, err)
	}

	err = adapter.EnsureStateReady()
	return r.requeueOnErr(err)
}

func (r *ReconcileProjectReference) getGcpClient(projectID string, logger logr.Logger) (gcpclient.Client, error) {
	// Get org creds from secret
	creds, err := util.GetGCPCredentialsFromSecret(r.client, operatorNamespace, orgGcpSecretName)
	if err != nil {
		logger.Error(err, "could not get org Creds from secret", "Secret Name", orgGcpSecretName, "Operator Namespace", operatorNamespace)
		return nil, err
	}

	// Get gcpclient with creds
	gcpClient, err := r.gcpClientBuilder(projectID, creds)

	if err != nil {
		logger.Error(err, "could not get gcp client with secret creds", "Secret Name", orgGcpSecretName, "Operator Namespace", operatorNamespace)
	}
	return gcpClient, err
}

func (r *ReconcileProjectReference) doNotRequeue() (reconcile.Result, error) {
	return reconcile.Result{}, nil
}

func (r *ReconcileProjectReference) requeueOnErr(err error) (reconcile.Result, error) {
	return reconcile.Result{}, err
}

func (r *ReconcileProjectReference) requeue() (reconcile.Result, error) {
	return reconcile.Result{Requeue: true}, nil
}

func (r *ReconcileProjectReference) requeueAfter(duration time.Duration, err error) (reconcile.Result, error) {
	return reconcile.Result{RequeueAfter: duration}, err
}
