// Package projectclaim contains the logic to reconcile the change of a ProjectClaim CR
// On the initial creation of a ProjectClaim, the main objective is to create a ProjectReference
// for the case that the Region is supported. After the attempt of reconciling the ProjectRefrence the
// ProjectClaim is updated with the result of the procedure.
package projectclaim

import (
	"context"
	"time"

	gcpv1alpha1 "github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1"
	condition "github.com/openshift/gcp-project-operator/pkg/condition"
	"github.com/openshift/gcp-project-operator/pkg/util"
	gcputil "github.com/openshift/gcp-project-operator/pkg/util"
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

var log = logf.Log.WithName("controller_projectclaim")

//go:generate mockgen -destination=../../util/mocks/$GOPACKAGE/customeresourceadapter.go -package=$GOPACKAGE github.com/openshift/gcp-project-operator/pkg/controller/projectclaim CustomResourceAdapter
type CustomResourceAdapter interface {
	EnsureProjectClaimFakeProcessed() (gcputil.OperationResult, error)
	EnsureProjectClaimDeletionProcessed() (gcputil.OperationResult, error)
	ProjectReferenceExists() (bool, error)
	EnsureProjectClaimInitialized() (gcputil.OperationResult, error)
	EnsureProjectClaimStatePending() (gcputil.OperationResult, error)
	EnsureProjectClaimStatePendingProject() (gcputil.OperationResult, error)
	EnsureRegionSupported() (gcputil.OperationResult, error)
	EnsureProjectReferenceExists() (gcputil.OperationResult, error)
	EnsureProjectReferenceLink() (gcputil.OperationResult, error)
	EnsureFinalizer() (gcputil.OperationResult, error)
	EnsureCCSSecretFinalizer() (gcputil.OperationResult, error)
	FinalizeProjectClaim() (ObjectState, error)
	SetProjectClaimCondition(gcpv1alpha1.ConditionType, string, error) (gcputil.OperationResult, error)
}

// Add creates a new ProjectClaim Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileProjectClaim{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("projectclaim-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource ProjectClaim
	err = c.Watch(&source.Kind{Type: &gcpv1alpha1.ProjectClaim{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileProjectClaim implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileProjectClaim{}

// ReconcileProjectClaim reconciles a ProjectClaim object
type ReconcileProjectClaim struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

func NewReconcileProjectClaim(client client.Client, scheme *runtime.Scheme) *ReconcileProjectClaim {
	return &ReconcileProjectClaim{client, scheme}
}

// Reconcile calls ReconcileHandler and updates the CRD if any err occurs
func (r *ReconcileProjectClaim) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)

	// Fetch the ProjectClaim instance
	instance := &gcpv1alpha1.ProjectClaim{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			return r.doNotRequeue()
		}
		return r.requeueOnErr(err)
	}

	conditionManager := condition.NewConditionManager()
	adapter := NewProjectClaimAdapter(instance, reqLogger, r.client, conditionManager)
	result, err := r.ReconcileHandler(adapter)
	reason := "ReconcileError"
	_, _ = adapter.SetProjectClaimCondition(gcpv1alpha1.ConditionError, reason, err)

	return result, err
}

type ReconcileOperation func() (util.OperationResult, error)

// ReconcileHandler reads that state of the cluster for a ProjectClaim object and makes changes based on the state read
// and what is in the ProjectClaim.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileProjectClaim) ReconcileHandler(adapter CustomResourceAdapter) (reconcile.Result, error) {
	operations := []ReconcileOperation{
		adapter.EnsureProjectClaimFakeProcessed,
		adapter.EnsureProjectClaimDeletionProcessed,
		adapter.EnsureProjectClaimInitialized,
		adapter.EnsureRegionSupported,
		adapter.EnsureProjectClaimStatePending,
		adapter.EnsureProjectReferenceExists,
		adapter.EnsureProjectReferenceLink,
		adapter.EnsureFinalizer,
		adapter.EnsureCCSSecretFinalizer,
		adapter.EnsureProjectClaimStatePendingProject,
	}
	for _, operation := range operations {
		result, err := operation()
		if err != nil || result.RequeueRequest {
			return r.requeueAfter(result.RequeueDelay, err)
		}
		if result.CancelRequest {
			return r.doNotRequeue()
		}
	}
	return r.doNotRequeue()
}

func (r *ReconcileProjectClaim) doNotRequeue() (reconcile.Result, error) {
	return reconcile.Result{}, nil
}

func (r *ReconcileProjectClaim) requeueOnErr(err error) (reconcile.Result, error) {
	return reconcile.Result{}, err
}

func (r *ReconcileProjectClaim) requeueAfter(duration time.Duration, err error) (reconcile.Result, error) {
	return reconcile.Result{RequeueAfter: duration}, err
}
