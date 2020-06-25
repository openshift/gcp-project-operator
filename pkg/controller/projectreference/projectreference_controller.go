package projectreference

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	gcpv1alpha1 "github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1"
	condition "github.com/openshift/gcp-project-operator/pkg/condition"
	"github.com/openshift/gcp-project-operator/pkg/gcpclient"
	"github.com/openshift/gcp-project-operator/pkg/util"
	operrors "github.com/openshift/gcp-project-operator/pkg/util/errors"
	logtypes "github.com/openshift/gcp-project-operator/pkg/util/types"
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

	gcpClient, err := r.getGcpClient(projectReference.Spec.GCPProjectID, reqLogger)
	if err != nil {
		return r.requeueOnErr(err)
	}

	conditionManager := condition.NewConditionManager()
	adapter, err := NewReferenceAdapter(projectReference, reqLogger, r.client, gcpClient, conditionManager)
	if err != nil {
		err = operrors.Wrap(err, "could not create ReferenceAdapter")
		return r.requeueOnErr(err)
	}

	result, err := r.ReconcileHandler(adapter, reqLogger)
	reason := "ReconcileError"
	_ = adapter.SetProjectReferenceCondition(reason, err)

	reqLogger.V(int(logtypes.ProjectReference)).Info(fmt.Sprintf("Finished Reconcile. Error occured: %t, Requeing: %t, Delay: %d", err != nil, result.Requeue, result.RequeueAfter))
	return result, err
}

// ReconcileHandler reads that state of the cluster for a ProjectReference object and makes changes based on the state read
// and what is in the ProjectReference.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.

type OperationResult struct {
	RequeueDelay   time.Duration
	RequeueRequest bool
	CancelRequest  bool
}

func StopProcessing() (result OperationResult, err error) {
	result = OperationResult{
		RequeueDelay:   0,
		RequeueRequest: false,
		CancelRequest:  true,
	}
	return
}

func RequeueWithError(errIn error) (result OperationResult, err error) {
	result = OperationResult{
		RequeueDelay:   0,
		RequeueRequest: true,
		CancelRequest:  false,
	}
	err = errIn
	return
}

func RequeueOnErrorOrStop(errIn error) (result OperationResult, err error) {
	result = OperationResult{
		RequeueDelay:   0,
		RequeueRequest: false,
		CancelRequest:  true,
	}
	err = errIn
	return
}

func RequeueAfter(delay time.Duration, errIn error) (result OperationResult, err error) {
	result = OperationResult{
		RequeueDelay:   delay,
		RequeueRequest: true,
		CancelRequest:  false,
	}
	err = errIn
	return
}

func ContinueProcessing() (result OperationResult, err error) {
	result = OperationResult{
		RequeueDelay:   0,
		RequeueRequest: false,
		CancelRequest:  false,
	}
	return
}

type ReconcileOperation func(*ReferenceAdapter) (OperationResult, error)

func (r *ReconcileProjectReference) ReconcileHandler(adapter *ReferenceAdapter, reqLogger logr.Logger) (reconcile.Result, error) {
	operations := []ReconcileOperation{
		EnsureProjectReferenceInitialized, //Set conditions
		EnsureDeletionProcessed,           // Cleanup
		EnsureProjectClaimReady,           // Make projectReference  be processed based on state of ProjectClaim and Project Reference
		VerifyProjectClaimPending,         //only make changes to ProjectReference if ProjectClaim is pending
		EnsureProjectReferenceStatusCreating,
		EnsureProjectID,
		EnsureFinalizerAdded,
		EnsureProjectConfigured,
		EnsureStateReady,
	}

	for _, operation := range operations {
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

func (r *ReconcileProjectReference) getGcpClient(projectID string, logger logr.Logger) (gcpclient.Client, error) {
	// Get org creds from secret
	creds, err := util.GetGCPCredentialsFromSecret(r.client, operatorNamespace, orgGcpSecretName)
	if err != nil {
		err = operrors.Wrap(err, fmt.Sprintf("could not get org Creds from secret: %s, for namespace %s", orgGcpSecretName, operatorNamespace))
		return nil, err
	}

	// Get gcpclient with creds
	gcpClient, err := r.gcpClientBuilder(projectID, creds)
	if err != nil {
		return nil, operrors.Wrap(err, fmt.Sprintf("could not get gcp client with secret: %s, for namespace %s", orgGcpSecretName, operatorNamespace))
	}

	return gcpClient, nil
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
