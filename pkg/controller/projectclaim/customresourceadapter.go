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

type CustomResourceAdapter struct {
	projectClaim     *gcpv1alpha1.ProjectClaim
	logger           logr.Logger
	client           client.Client
	projectReference *gcpv1alpha1.ProjectReference
}

type ObjectState bool

const (
	ObjectModified  ObjectState = true
	ObjectUnchanged ObjectState = false
)

const ProjectClaimFinalizer string = "finalizer.gcp.managed.openshift.io"

func NewCustomResourceAdapter(projectClaim *gcpv1alpha1.ProjectClaim, logger logr.Logger, client client.Client) *CustomResourceAdapter {
	projectReference := newMatchingProjectReference(projectClaim)
	return &CustomResourceAdapter{projectClaim, logger, client, projectReference}
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

func (c *CustomResourceAdapter) ProjectReferenceExists() (bool, error) {
	found := &gcpv1alpha1.ProjectReference{}
	err := c.client.Get(context.TODO(), types.NamespacedName{Name: c.projectReference.Name, Namespace: c.projectReference.Namespace}, found)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (c *CustomResourceAdapter) IsProjectClaimDeletion() bool {
	return c.projectClaim.DeletionTimestamp != nil
}

func (c *CustomResourceAdapter) FinalizeProjectClaim() error {
	projectReferenceExists, err := c.ProjectReferenceExists()
	if err != nil {
		return err
	}

	if projectReferenceExists {
		err := c.client.Delete(context.TODO(), c.projectReference)
		if err != nil {
			return err
		}
	}
	finalizers := c.projectClaim.GetFinalizers()
	if util.Contains(finalizers, ProjectClaimFinalizer) {
		c.projectClaim.SetFinalizers(util.Filter(finalizers, ProjectClaimFinalizer))
		return c.client.Update(context.TODO(), c.projectClaim)
	}
	return nil
}

func (c *CustomResourceAdapter) EnsureProjectClaimInitialized() (ObjectState, error) {
	if c.projectClaim.Status.Conditions == nil {
		c.projectClaim.Status.Conditions = []gcpv1alpha1.ProjectClaimCondition{}
		err := c.client.Status().Update(context.TODO(), c.projectClaim)
		if err != nil {
			c.logger.Error(err, "Failed to initalize ProjectClaim")
			return ObjectUnchanged, err
		}
		return ObjectModified, nil
	}
	return ObjectUnchanged, nil
}

func (c *CustomResourceAdapter) EnsureProjectReferenceLink() (ObjectState, error) {
	expectedLink := gcpv1alpha1.NamespacedName{
		Name:      c.projectReference.GetName(),
		Namespace: c.projectReference.GetNamespace(),
	}
	if c.projectClaim.Spec.ProjectReferenceCRLink == expectedLink {
		return ObjectUnchanged, nil
	}
	c.projectClaim.Spec.ProjectReferenceCRLink = expectedLink
	err := c.client.Update(context.TODO(), c.projectClaim)
	if err != nil {
		return ObjectUnchanged, err
	}
	return ObjectModified, nil
}

func (c *CustomResourceAdapter) EnsureFinalizer() (ObjectState, error) {
	if !util.Contains(c.projectClaim.GetFinalizers(), ProjectClaimFinalizer) {
		c.logger.Info("Adding Finalizer to the ProjectClaim")
		c.projectClaim.SetFinalizers(append(c.projectClaim.GetFinalizers(), ProjectClaimFinalizer))

		err := c.client.Update(context.TODO(), c.projectClaim)
		if err != nil {
			c.logger.Error(err, "Failed to update ProjectClaim with finalizer")
			return ObjectUnchanged, err
		}
		return ObjectModified, nil
	}
	return ObjectUnchanged, nil
}

func (c *CustomResourceAdapter) EnsureProjectReferenceExists() error {
	projectReferenceExists, err := c.ProjectReferenceExists()
	if err != nil {
		return err
	}

	if !projectReferenceExists {
		return c.client.Create(context.TODO(), c.projectReference)
	}
	return nil
}
