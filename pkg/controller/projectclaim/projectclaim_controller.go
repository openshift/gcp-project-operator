package projectclaim

import (
	"context"
	"time"

	gcpv1alpha1 "github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1"
	condition "github.com/openshift/gcp-project-operator/pkg/condition"
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

var log = logf.Log.WithName("controller_projectclaim")

//go:generate mockgen -destination=../../util/mocks/$GOPACKAGE/customeresourceadapter.go -package=$GOPACKAGE github.com/openshift/gcp-project-operator/pkg/controller/projectclaim CustomResourceAdapter
type CustomResourceAdapter interface {
	IsProjectClaimDeletion() bool
	ProjectReferenceExists() (bool, error)
	EnsureProjectClaimInitialized() (ObjectState, error)
	EnsureProjectClaimState(gcpv1alpha1.ClaimStatus) error
	EnsureRegionSupported() error
	EnsureProjectReferenceExists() error
	EnsureProjectReferenceLink() (ObjectState, error)
	EnsureFinalizer() (ObjectState, error)
	FinalizeProjectClaim() (ObjectState, error)
	SetProjectClaimCondition(reason string, err error) error
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
func (r *ReconcileProjectClaim) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling ProjectClaim")

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
	_ = adapter.SetProjectClaimCondition(reason, err)

	return result, err
}

// ReconcileHandler reads that state of the cluster for a ProjectClaim object and makes changes based on the state read
// and what is in the ProjectClaim.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileProjectClaim) ReconcileHandler(adapter CustomResourceAdapter) (reconcile.Result, error) {
	if adapter.IsProjectClaimDeletion() {
		crState, err := adapter.FinalizeProjectClaim()
		if crState == ObjectUnchanged || err != nil {
			return r.requeueAfter(5*time.Second, err)
		}
		return r.doNotRequeue()
	}

	crState, err := adapter.EnsureProjectClaimInitialized()
	if crState == ObjectModified || err != nil {
		return r.requeueOnErr(err)
	}

	err = adapter.EnsureRegionSupported()
	if err != nil {
		return r.requeueOnErr(err)
	}

	err = adapter.EnsureProjectClaimState(gcpv1alpha1.ClaimStatusPending)
	if err != nil {
		return r.requeueOnErr(err)
	}

	err = adapter.EnsureProjectReferenceExists()
	if err != nil {
		return r.requeueOnErr(err)
	}

	crState, err = adapter.EnsureProjectReferenceLink()
	if crState == ObjectModified || err != nil {
		return r.requeueOnErr(err)
	}

	crState, err = adapter.EnsureFinalizer()
	if crState == ObjectModified || err != nil {
		return r.requeueOnErr(err)
	}

	err = adapter.EnsureProjectClaimState(gcpv1alpha1.ClaimStatusPendingProject)
	if err != nil {
		return r.requeueOnErr(err)
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
