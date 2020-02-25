package projectclaim

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/openshift/cluster-api/pkg/util"
	gcpv1alpha1 "github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ProjectClaimReconciler struct {
	projectClaim     *gcpv1alpha1.ProjectClaim
	logger           logr.Logger
	client           client.Client
	projectReference *gcpv1alpha1.ProjectReference
}

func NewProjectClaimReconciler(projectClaim *gcpv1alpha1.ProjectClaim, logger logr.Logger, client client.Client) *ProjectClaimReconciler {
	projectReference := newMatchingProjectReference(projectClaim)
	return &ProjectClaimReconciler{projectClaim, logger, client, projectReference}
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

func (r *ProjectClaimReconciler) projectReferenceExists() (bool, error) {
	found := &gcpv1alpha1.ProjectReference{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: r.projectReference.Name, Namespace: r.projectReference.Namespace}, found)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *ProjectClaimReconciler) isProjectClaimDeletion() bool {
	return r.projectClaim.DeletionTimestamp != nil
}

func (r *ProjectClaimReconciler) finalizeProjectClaim() error {
	projectReferenceExists, err := r.projectReferenceExists()
	if err != nil {
		return err
	}

	if projectReferenceExists {
		err := r.client.Delete(context.TODO(), r.projectReference)
		if err != nil {
			return err
		}
	}
	finalizers := r.projectClaim.GetFinalizers()
	if util.Contains(finalizers, projectClaimFinalizer) {
		r.projectClaim.SetFinalizers(util.Filter(finalizers, projectClaimFinalizer))
		return r.client.Update(context.TODO(), r.projectClaim)
	}
	return nil
}

func (r *ProjectClaimReconciler) ensureProjectReferenceLink() (bool, error) {
	expectedLink := gcpv1alpha1.NamespacedName{
		Name:      r.projectReference.GetName(),
		Namespace: r.projectReference.GetNamespace(),
	}
	if r.projectClaim.Spec.ProjectReferenceCRLink == expectedLink {
		return false, nil
	}
	r.projectClaim.Spec.ProjectReferenceCRLink = expectedLink
	err := r.client.Update(context.TODO(), r.projectClaim)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *ProjectClaimReconciler) ensureFinalizer() (bool, error) {
	if !util.Contains(r.projectClaim.GetFinalizers(), projectClaimFinalizer) {
		r.logger.Info("Adding Finalizer to the ProjectClaim")
		r.projectClaim.SetFinalizers(append(r.projectClaim.GetFinalizers(), projectClaimFinalizer))

		err := r.client.Update(context.TODO(), r.projectClaim)
		if err != nil {
			r.logger.Error(err, "Failed to update ProjectClaim with finalizer")
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (r *ProjectClaimReconciler) ensureProjectReferenceExists() error {
	projectReferenceExists, err := r.projectReferenceExists()
	if err != nil {
		return err
	}

	if !projectReferenceExists {
		return r.client.Create(context.TODO(), r.projectReference)
	}
	return nil
}
