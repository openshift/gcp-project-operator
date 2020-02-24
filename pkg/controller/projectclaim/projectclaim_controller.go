package projectclaim

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/openshift/cluster-api/pkg/util"
	gcpv1alpha1 "github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_projectclaim")

const projectClaimFinalizer string = "finalizer.gcp.managed.openshift.io"

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

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

// Reconcile reads that state of the cluster for a ProjectClaim object and makes changes based on the state read
// and what is in the ProjectClaim.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
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

	projectReference := newMatchingProjectReference(instance)

	if r.isProjectClaimDeletion(instance) {
		err = r.finalizeProjectClaim(instance, projectReference)
		if err != nil {
			return r.requeueOnErr(err)
		}
		return r.doNotRequeue()
	}

	err = r.ensureProjectReferenceExists(projectReference)
	if err != nil {
		return r.requeueOnErr(err)
	}

	crChanged, err := r.ensureProjectReferenceLink(instance, projectReference)
	if crChanged || err != nil {
		return r.requeueOnErr(err)
	}

	crChanged, err = r.ensureFinalizer(reqLogger, instance)
	if crChanged || err != nil {
		return r.requeueOnErr(err)
	}

	return r.doNotRequeue()
}

func newMatchingProjectReference(projectClaim *gcpv1alpha1.ProjectClaim) *gcpv1alpha1.ProjectReference {

	return &gcpv1alpha1.ProjectReference{
		ObjectMeta: metav1.ObjectMeta{
			Name:      projectClaim.GetNamespace() + "-" + projectClaim.GetName(),
			Namespace: gcpv1alpha1.ProjectReferenceNamespace,
		},
		Spec: gcpv1alpha1.ProjectReferenceSpec{
			GCPProjectID: "",
			ProjectClaimCRLink: gcpv1alpha1.NamespacedName{
				Name:      projectClaim.GetName(),
				Namespace: projectClaim.GetNamespace(),
			},
			LegalEntity: *projectClaim.Spec.LegalEntity.DeepCopy(),
		},
	}
}

func (r *ReconcileProjectClaim) projectReferenceExists(projectReference *gcpv1alpha1.ProjectReference) (bool, error) {
	found := &gcpv1alpha1.ProjectReference{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: projectReference.Name, Namespace: projectReference.Namespace}, found)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *ReconcileProjectClaim) isProjectClaimDeletion(projectClaim *gcpv1alpha1.ProjectClaim) bool {
	return projectClaim.DeletionTimestamp != nil
}

func (r *ReconcileProjectClaim) finalizeProjectClaim(projectClaim *gcpv1alpha1.ProjectClaim, projectReference *gcpv1alpha1.ProjectReference) error {
	projectReferenceExists, err := r.projectReferenceExists(projectReference)
	if err != nil {
		return err
	}

	if projectReferenceExists {
		err := r.client.Delete(context.TODO(), projectReference)
		if err != nil {
			return err
		}
	}
	finalizers := projectClaim.GetFinalizers()
	if util.Contains(finalizers, projectClaimFinalizer) {
		projectClaim.SetFinalizers(util.Filter(finalizers, projectClaimFinalizer))
		return r.client.Update(context.TODO(), projectClaim)
	}
	return nil
}

func (r *ReconcileProjectClaim) ensureProjectReferenceLink(projectClaim *gcpv1alpha1.ProjectClaim, projectReference *gcpv1alpha1.ProjectReference) (bool, error) {
	expectedLink := gcpv1alpha1.NamespacedName{
		Name:      projectReference.GetName(),
		Namespace: projectReference.GetNamespace(),
	}
	if projectClaim.Spec.ProjectReferenceCRLink == expectedLink {
		return false, nil
	}
	projectClaim.Spec.ProjectReferenceCRLink = expectedLink
	err := r.client.Update(context.TODO(), projectClaim)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *ReconcileProjectClaim) ensureFinalizer(reqLogger logr.Logger, projectClaim *gcpv1alpha1.ProjectClaim) (bool, error) {
	if !util.Contains(projectClaim.GetFinalizers(), projectClaimFinalizer) {
		reqLogger.Info("Adding Finalizer to the ProjectClaim")
		projectClaim.SetFinalizers(append(projectClaim.GetFinalizers(), projectClaimFinalizer))

		err := r.client.Update(context.TODO(), projectClaim)
		if err != nil {
			reqLogger.Error(err, "Failed to update ProjectClaim with finalizer")
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (r *ReconcileProjectClaim) ensureProjectReferenceExists(projectReference *gcpv1alpha1.ProjectReference) error {
	projectReferenceExists, err := r.projectReferenceExists(projectReference)
	if err != nil {
		return err
	}

	if !projectReferenceExists {
		return r.client.Create(context.TODO(), projectReference)
	}
	return nil
}

func (r *ReconcileProjectClaim) doNotRequeue() (reconcile.Result, error) {
	return reconcile.Result{}, nil
}

func (r *ReconcileProjectClaim) requeueOnErr(err error) (reconcile.Result, error) {
	return reconcile.Result{}, err
}
