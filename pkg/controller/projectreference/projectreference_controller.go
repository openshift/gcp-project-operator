package projectreference

import (
	"context"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	gcpv1alpha1 "github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1"
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
	return &ReconcileProjectReference{client: mgr.GetClient(), scheme: mgr.GetScheme()}
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
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a ProjectReference object and makes changes based on the state read
// and what is in the ProjectReference.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileProjectReference) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling ProjectReference")

	instance, err := r.getProjectReference(request)
	if cmp.Equal(*instance, gcpv1alpha1.ProjectReference{}) || (err != nil) {
		return reconcile.Result{}, err
	}

	if instance.Spec.GCPProjectID == "" {
		err = r.updateProjectName(instance) //this will update the spec, resulting in another run in the loop, hence return
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, err
}

func (r *ReconcileProjectReference) getProjectReference(request reconcile.Request) (*gcpv1alpha1.ProjectReference, error) {
	instance := &gcpv1alpha1.ProjectReference{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return &gcpv1alpha1.ProjectReference{}, nil
		}
	}
	return instance, err
}

func (r *ReconcileProjectReference) updateProjectName(instance *gcpv1alpha1.ProjectReference) error {
	instance.Spec.GCPProjectID = uuid.New().String()
	return r.client.Update(context.TODO(), instance)
}
