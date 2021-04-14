package projectreference

import (
	"context"
	"fmt"
	"strings"
	"time"

	"reflect"
	goruntime "runtime"

	"github.com/go-logr/logr"
	gcpv1alpha1 "github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1"
	condition "github.com/openshift/gcp-project-operator/pkg/condition"
	"github.com/openshift/gcp-project-operator/pkg/configmap"
	"github.com/openshift/gcp-project-operator/pkg/gcpclient"
	"github.com/openshift/gcp-project-operator/pkg/util"
	operrors "github.com/openshift/gcp-project-operator/pkg/util/errors"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_projectreference")

const (
	// Operator config
	operatorNamespace = "gcp-project-operator"

	// secret information
	orgGcpSecretName = "gcp-project-operator-credentials"
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

// Reconcile wraps ReconcileHandler() and updates the conditions if any error occurs
func (r *ReconcileProjectReference) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)

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

	gcpClient, err := r.getGcpClient(projectReference, reqLogger)
	if err != nil {
		return r.requeueOnErr(err)
	}

	cm, err := r.getConfigMap()
	if err != nil {
		return r.requeueOnErr(err)
	}

	conditionManager := condition.NewConditionManager()
	adapter, err := NewReferenceAdapter(projectReference, reqLogger, r.client, gcpClient, conditionManager, cm)
	if err != nil {
		err = operrors.Wrap(err, "could not create ReferenceAdapter")
		return r.requeueOnErr(err)
	}

	result, err := r.ReconcileHandler(adapter, reqLogger)
	reason := "ReconcileError"
	_ = adapter.SetProjectReferenceCondition(reason, err)

	reqLogger.V(1).Info(fmt.Sprintf("Finished Reconcile. Error occured: %t, Requeing: %t, Delay: %d", err != nil, result.Requeue, result.RequeueAfter))
	return result, err
}

type ReconcileOperation func(*ReferenceAdapter) (util.OperationResult, error)

// ReconcileHandler reads that state of the cluster for a ProjectReference object and makes changes based on the state read
// and what is in the ProjectReference.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileProjectReference) ReconcileHandler(adapter *ReferenceAdapter, reqLogger logr.Logger) (reconcile.Result, error) {
	operations := []ReconcileOperation{
		EnsureServiceAccountNameMigration,
		EnsureProjectReferenceInitialized, //Set conditions
		EnsureDeletionProcessed,           // Cleanup
		EnsureProjectClaimReady,           // Make projectReference  be processed based on state of ProjectClaim and Project Reference
		VerifyProjectClaimPending,         //only make changes to ProjectReference if ProjectClaim is pending
		EnsureProjectReferenceStatusCreating,
		EnsureProjectID,
		EnsureServiceAccountName,
		EnsureFinalizerAdded,
		EnsureProjectCreated,
		EnsureProjectConfigured,
		EnsureStateReady,
	}
	for _, operation := range operations {
		if log.V(3).Enabled() {
			log.V(3).Info("func", strings.Split(goruntime.FuncForPC(reflect.ValueOf(operation).Pointer()).Name(), ".")[2])
		}
		result, err := operation(adapter)
		if err != nil || result.RequeueRequest {
			return r.requeueAfter(result.RequeueDelay, err)
		}
		if result.CancelRequest {
			return r.doNotRequeue()
		}
	}
	return r.doNotRequeue()
}

func (r *ReconcileProjectReference) getGcpClient(projectReference *gcpv1alpha1.ProjectReference, logger logr.Logger) (gcpclient.Client, error) {
	credSecretNamespace := operatorNamespace
	credSecretName := orgGcpSecretName
	if projectReference.Spec.CCS {
		credSecretNamespace = projectReference.Spec.CCSSecretRef.Namespace
		credSecretName = projectReference.Spec.CCSSecretRef.Name
	}
	// Get org creds from secret
	creds, err := util.GetGCPCredentialsFromSecret(r.client, credSecretNamespace, credSecretName)
	if err != nil {
		err = operrors.Wrap(err, fmt.Sprintf("could not get org Creds from secret: %s, for namespace %s", orgGcpSecretName, operatorNamespace))
		return nil, err
	}

	// Get gcpclient with creds
	gcpClient, err := r.gcpClientBuilder(projectReference.Spec.GCPProjectID, creds)
	if err != nil {
		return nil, operrors.Wrap(err, fmt.Sprintf("could not get gcp client with secret: %s, for namespace %s", orgGcpSecretName, operatorNamespace))
	}

	return gcpClient, nil
}

func (r *ReconcileProjectReference) getConfigMap() (configmap.OperatorConfigMap, error) {
	operatorConfigMap, err := configmap.GetOperatorConfigMap(r.client)
	if err != nil {
		return operatorConfigMap, operrors.Wrap(err, "could not find the OperatorConfigMap")
	}

	if err := configmap.ValidateOperatorConfigMap(operatorConfigMap); err != nil {
		return operatorConfigMap, operrors.Wrap(err, "configmap didn't get filled properly")
	}

	return operatorConfigMap, err
}

func (r *ReconcileProjectReference) doNotRequeue() (reconcile.Result, error) {
	return reconcile.Result{}, nil
}

func (r *ReconcileProjectReference) requeueOnErr(err error) (reconcile.Result, error) {
	return reconcile.Result{}, err
}

func (r *ReconcileProjectReference) requeueAfter(duration time.Duration, err error) (reconcile.Result, error) {
	return reconcile.Result{Requeue: true, RequeueAfter: duration}, err
}
