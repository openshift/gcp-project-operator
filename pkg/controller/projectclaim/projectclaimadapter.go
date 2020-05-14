package projectclaim

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/openshift/cluster-api/pkg/util"
	gcpv1alpha1 "github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1"
	condition "github.com/openshift/gcp-project-operator/pkg/condition"
	operrors "github.com/openshift/gcp-project-operator/pkg/util/errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ProjectClaimAdapter struct {
	projectClaim     *gcpv1alpha1.ProjectClaim
	logger           logr.Logger
	client           client.Client
	projectReference *gcpv1alpha1.ProjectReference
	conditionManager condition.Conditions
}

type ObjectState bool

const (
	ObjectModified  ObjectState = true
	ObjectUnchanged ObjectState = false
)

const ProjectClaimFinalizer string = "finalizer.gcp.managed.openshift.io"

// Regions supported in the gcp-project-operator
var supportedRegions = map[string]bool{
	"asia-east1":      true,
	"asia-northeast1": true,
	"asia-southeast1": true,
	"europe-west1":    true,
	"europe-west4":    true,
	"us-central1":     true,
	"us-east1":        true,
	"us-east4":        true,
	"us-west1":        true,

	// Regions below don't have enough quota configured by default, but our org has sufficient quota
	"asia-east2":   true,
	"asia-south1":  true,
	"europe-west2": true,
	"us-west2":     true,

	// Regions below have enough quota, but the region name will produce too long hostnames.
	// This is an issue the openshift installer needs to fix.
	// "australia-southeast1":    true,
	// "northamerica-northeast1": true,
	// "southamerica-east1":      true,

	// Regions below are disabled do not have enough quota configured (CPU < 28 or SSD storage < 896)
	// "europe-west3":            true,
	// "europe-west6":            true,
	// "europe-north1":           true,
	// "asia-northeast2":         true,
}

func NewProjectClaimAdapter(projectClaim *gcpv1alpha1.ProjectClaim, logger logr.Logger, client client.Client, manager condition.Conditions) *ProjectClaimAdapter {
	projectReference := newMatchingProjectReference(projectClaim)
	return &ProjectClaimAdapter{projectClaim, logger, client, projectReference, manager}
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

func (c *ProjectClaimAdapter) ProjectReferenceExists() (bool, error) {
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

func (c *ProjectClaimAdapter) IsProjectClaimDeletion() bool {
	return c.projectClaim.DeletionTimestamp != nil
}

func (c *ProjectClaimAdapter) IsProjectReferenceDeletion() bool {
	return c.projectReference.DeletionTimestamp != nil
}

func (c *ProjectClaimAdapter) EnsureFinalizerDeleted() error {
	c.logger.Info("Deleting Finalizer")
	finalizers := c.projectClaim.GetFinalizers()
	if util.Contains(finalizers, ProjectClaimFinalizer) {
		c.projectClaim.SetFinalizers(util.Filter(finalizers, ProjectClaimFinalizer))
		return c.client.Update(context.TODO(), c.projectClaim)
	}
	return nil
}

func (c *ProjectClaimAdapter) FinalizeProjectClaim() (ObjectState, error) {
	projectReferenceExists, err := c.ProjectReferenceExists()
	if err != nil {
		return ObjectUnchanged, err
	}

	projectReferenceDeletionRequested := c.IsProjectReferenceDeletion()
	if projectReferenceExists && !projectReferenceDeletionRequested {
		err := c.client.Delete(context.TODO(), c.projectReference)
		if err != nil {
			return ObjectUnchanged, err
		}
	}

	// Assure the finalizer is not deleted as long as ProjectReference exists
	if !projectReferenceExists {
		err := c.EnsureFinalizerDeleted()
		if err != nil {
			return ObjectUnchanged, err
		}
		return ObjectModified, nil
	}

	return ObjectUnchanged, nil
}

func (c *ProjectClaimAdapter) EnsureProjectClaimInitialized() (ObjectState, error) {
	if c.projectClaim.Status.Conditions == nil {
		c.projectClaim.Status.Conditions = []gcpv1alpha1.Condition{}
		err := c.client.Status().Update(context.TODO(), c.projectClaim)
		if err != nil {
			c.logger.Error(err, "Failed to initalize ProjectClaim")
			return ObjectUnchanged, err
		}
		return ObjectModified, nil
	}
	return ObjectUnchanged, nil
}

func (c *ProjectClaimAdapter) EnsureProjectReferenceLink() (ObjectState, error) {
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

func (c *ProjectClaimAdapter) EnsureFinalizer() (ObjectState, error) {
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

func (c *ProjectClaimAdapter) EnsureProjectReferenceExists() error {
	projectReferenceExists, err := c.ProjectReferenceExists()
	if err != nil {
		return err
	}

	if !projectReferenceExists {
		return c.client.Create(context.TODO(), c.projectReference)
	}
	return nil
}

func (c *ProjectClaimAdapter) EnsureProjectClaimState(state gcpv1alpha1.ClaimStatus) error {
	if c.projectClaim.Status.State == state {
		return nil
	}

	if state == gcpv1alpha1.ClaimStatusPending && c.projectClaim.Status.State != gcpv1alpha1.ClaimStatusError {
		if c.projectClaim.Status.State != "" {
			return nil
		}
	}

	if state == gcpv1alpha1.ClaimStatusPendingProject {
		if c.projectClaim.Status.State != gcpv1alpha1.ClaimStatusPending {
			return nil
		}
	}

	c.projectClaim.Status.State = state
	return c.StatusUpdate()
}

// SetProjectClaimCondition calls SetCondition() with project claim conditions
func (c *ProjectClaimAdapter) SetProjectClaimCondition(reason string, err error) error {
	conditions := &c.projectClaim.Status.Conditions
	conditionType := gcpv1alpha1.ConditionError
	if err != nil {
		c.conditionManager.SetCondition(conditions, conditionType, corev1.ConditionTrue, reason, err.Error())
	} else {
		if len(*conditions) != 0 {
			reason = reason + "Resolved"
			c.conditionManager.SetCondition(conditions, conditionType, corev1.ConditionFalse, reason, "")
		} else {
			return nil
		}
	}

	return c.StatusUpdate()
}

// IsRegionSupported checks if current region is supported.
// It returns an error message if a region is not supported.
func (c *ProjectClaimAdapter) IsRegionSupported() error {
	if _, ok := supportedRegions[c.projectClaim.Spec.Region]; !ok {
		return operrors.ErrRegionNotSupported
	}
	return nil
}

// EnsureRegionSupported modifies projectClaim.Status.State with result from IsRegionSupported.
// If a region is not supported it returns an error and sets projectClaim.Status.State to ClaimStatusError.
func (c *ProjectClaimAdapter) EnsureRegionSupported() error {
	if err := c.IsRegionSupported(); err != nil {
		c.projectClaim.Status.State = gcpv1alpha1.ClaimStatusError
		c.StatusUpdate()
		return fmt.Errorf("EnsureRegionSupported: %v", err)
	}
	return nil
}

// StatusUpdate updates the project claim status
func (c *ProjectClaimAdapter) StatusUpdate() error {
	err := c.client.Status().Update(context.TODO(), c.projectClaim)
	if err != nil {
		c.logger.Error(err, fmt.Sprintf("failed to update ProjectClaim state for %s", c.projectClaim.Name))
		return err
	}

	return nil
}
