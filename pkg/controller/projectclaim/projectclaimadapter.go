package projectclaim

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/openshift/cluster-api/pkg/util"

	gcpv1alpha1 "github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1"
	condition "github.com/openshift/gcp-project-operator/pkg/condition"
	"github.com/openshift/gcp-project-operator/pkg/configmap"
	gcputil "github.com/openshift/gcp-project-operator/pkg/util"
	operrors "github.com/openshift/gcp-project-operator/pkg/util/errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
const CCSSecretFinalizer string = "finalizer.gcp.managed.openshift.io/ccs"
const RegionCheckFailed string = "RegionCheckFailed"
const FakeProjectClaim string = "managed.openshift.com/fake"

func NewProjectClaimAdapter(projectClaim *gcpv1alpha1.ProjectClaim, logger logr.Logger, client client.Client, manager condition.Conditions) *ProjectClaimAdapter {
	projectReference := newMatchingProjectReference(projectClaim)
	return &ProjectClaimAdapter{projectClaim, logger, client, projectReference, manager}
}

func newMatchingProjectReference(projectClaim *gcpv1alpha1.ProjectClaim) *gcpv1alpha1.ProjectReference {
	gcpProjectID := ""
	if projectClaim.Spec.CCS {
		gcpProjectID = projectClaim.Spec.CCSProjectID
	}

	return &gcpv1alpha1.ProjectReference{
		ObjectMeta: metav1.ObjectMeta{
			Name:      projectClaim.GetNamespace() + "-" + projectClaim.GetName(),
			Namespace: gcpv1alpha1.ProjectReferenceNamespace,
		},
		Spec: gcpv1alpha1.ProjectReferenceSpec{
			GCPProjectID: gcpProjectID,
			ProjectClaimCRLink: gcpv1alpha1.NamespacedName{
				Name:      projectClaim.GetName(),
				Namespace: projectClaim.GetNamespace(),
			},
			LegalEntity:  *projectClaim.Spec.LegalEntity.DeepCopy(),
			CCS:          projectClaim.Spec.CCS,
			CCSSecretRef: *projectClaim.Spec.CCSSecretRef.DeepCopy(),
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

func (adapter *ProjectClaimAdapter) EnsureProjectClaimDeletionProcessed() (gcputil.OperationResult, error) {
	if adapter.IsProjectClaimDeletion() {
		crState, err := adapter.FinalizeProjectClaim()
		if crState == ObjectUnchanged || err != nil {
			return gcputil.RequeueAfter(5*time.Second, err)
		}
		return gcputil.StopProcessing()
	}
	return gcputil.ContinueProcessing()
}

func (c *ProjectClaimAdapter) IsProjectClaimDeletion() bool {
	return c.projectClaim.DeletionTimestamp != nil
}

func (c *ProjectClaimAdapter) IsProjectReferenceDeletion() bool {
	return c.projectReference.DeletionTimestamp != nil
}

func (c *ProjectClaimAdapter) EnsureProjectClaimFinalizerDeleted() error {
	c.logger.Info("Deleting ProjectClaim Finalizer")
	return c.deleteFinalizer(c.projectClaim, ProjectClaimFinalizer)
}

func (c *ProjectClaimAdapter) EnsureCCSSecretFinalizerDeleted() error {
	if c.projectClaim.Spec.CCS {
		secret, err := c.getCCSSecret()
		if err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
			return err
		}
		c.logger.Info("Deleting CCS Secret Finalizer")
		err = c.deleteFinalizer(secret, CCSSecretFinalizer)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *ProjectClaimAdapter) deleteFinalizer(object runtime.Object, finalizer string) error {
	metadata, err := meta.Accessor(object)
	if err != nil {
		return operrors.Wrap(err, "Failed to delete finalizer "+finalizer)
	}
	finalizers := metadata.GetFinalizers()
	if util.Contains(finalizers, finalizer) {
		metadata.SetFinalizers(util.Filter(finalizers, finalizer))
		return c.client.Update(context.TODO(), object)
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
		err := c.EnsureCCSSecretFinalizerDeleted()
		if err != nil {
			return ObjectUnchanged, err
		}

		err = c.EnsureProjectClaimFinalizerDeleted()
		if err != nil {
			return ObjectUnchanged, err
		}
		return ObjectModified, nil
	}

	return ObjectUnchanged, nil
}

func (c *ProjectClaimAdapter) IsProjectClaimFake() (gcputil.OperationResult, error) {
	if c.projectClaim.Annotations[FakeProjectClaim] == "true" {
		if _, err := c.EnsureFinalizer(); err != nil {
			return gcputil.RequeueWithError(operrors.Wrap(err, fmt.Sprintf("Could not update project claim state for %s", c.projectClaim.Name)))
		}

		// If project claim is marked for deletion, remove also fake secret
		if c.projectClaim.DeletionTimestamp != nil && gcputil.SecretExists(c.client, c.projectClaim.Spec.GCPCredentialSecret.Name, c.projectClaim.Spec.GCPCredentialSecret.Namespace) {
			secret := &corev1.Secret{}
			c.client.Get(context.TODO(), types.NamespacedName{
				Name:      c.projectClaim.Spec.GCPCredentialSecret.Name,
				Namespace: c.projectClaim.Spec.GCPCredentialSecret.Namespace},
				secret,
			)
			err := c.client.Delete(context.TODO(), secret)
			if err != nil {
				return gcputil.RequeueWithError(operrors.Wrap(err, fmt.Sprintf("Could not delete fake secret %s", c.projectClaim.Spec.GCPCredentialSecret.Name)))
			}

			// Remove finalizer to unlock deletion of the project claim
			err = c.EnsureProjectClaimFinalizerDeleted()
			if err != nil {
				return gcputil.RequeueWithError(operrors.Wrap(err, fmt.Sprintf("Could not delete project claim finalizer for %s", c.projectClaim.Name)))
			}
			return gcputil.StopProcessing()
		}

		// Create fake secret if not existing
		if !gcputil.SecretExists(c.client, c.projectClaim.Spec.GCPCredentialSecret.Name, c.projectClaim.Spec.GCPCredentialSecret.Namespace) {
			privateKeyString, err := base64.StdEncoding.DecodeString("SS1hbS1mYWtlLXBhc3M=")
			if err != nil {
				return gcputil.RequeueWithError(operrors.Wrap(err, "Could not decode hardcoded secret"))
			}
			if err := c.client.Create(context.TODO(), gcputil.NewGCPSecretCR(string(privateKeyString), types.NamespacedName{Namespace: c.projectClaim.Spec.GCPCredentialSecret.Namespace, Name: c.projectClaim.Spec.GCPCredentialSecret.Name})); err != nil {
				return gcputil.RequeueWithError(operrors.Wrap(err, fmt.Sprintf("Could not create fake secret %s", c.projectClaim.Spec.GCPCredentialSecret.Name)))
			}
		}

		// Update project claim with fake specs
		if c.projectClaim.Status.State != gcpv1alpha1.ClaimStatusReady {
			fakeProjectClaim := &gcpv1alpha1.ProjectClaim{
				Spec: gcpv1alpha1.ProjectClaimSpec{
					GCPProjectID: "fakeProjectClaim",
					GCPCredentialSecret: gcpv1alpha1.NamespacedName{
						Name:      c.projectClaim.GetName(),
						Namespace: c.projectClaim.GetNamespace(),
					},
					LegalEntity: gcpv1alpha1.LegalEntity{
						Name: "fakeLegalEntityName",
						ID:   "fakeLegalEntityID",
					},
					Region: "fakeRegion",
					AvailabilityZones: []string{
						"fake-az-a",
						"fake-az-b",
						"fake-az-c",
					},
				},
				Status: gcpv1alpha1.ProjectClaimStatus{
					State: "Ready",
				},
			}
			c.projectClaim.Spec = fakeProjectClaim.Spec
			c.projectClaim.Status = fakeProjectClaim.Status
			if err := c.client.Update(context.TODO(), c.projectClaim); err != nil {
				return gcputil.RequeueWithError(operrors.Wrap(err, fmt.Sprintf("Could not update project claim for %s", c.projectClaim.Name)))
			}
		}
		return gcputil.StopProcessing()
	}
	return gcputil.ContinueProcessing()
}

func (c *ProjectClaimAdapter) EnsureProjectClaimInitialized() (gcputil.OperationResult, error) {

	if c.projectClaim.Status.Conditions == nil {
		c.projectClaim.Status.Conditions = []gcpv1alpha1.Condition{}
		err := c.client.Status().Update(context.TODO(), c.projectClaim)
		if err != nil {
			return gcputil.RequeueWithError(operrors.Wrap(err, "failed to initalize projectclaim"))
		}
		return gcputil.StopProcessing()
	}
	return gcputil.ContinueProcessing()
}

func (c *ProjectClaimAdapter) EnsureProjectReferenceLink() (gcputil.OperationResult, error) {
	expectedLink := gcpv1alpha1.NamespacedName{
		Name:      c.projectReference.GetName(),
		Namespace: c.projectReference.GetNamespace(),
	}
	if c.projectClaim.Spec.ProjectReferenceCRLink == expectedLink {
		return gcputil.ContinueProcessing()
	}
	c.projectClaim.Spec.ProjectReferenceCRLink = expectedLink
	err := c.client.Update(context.TODO(), c.projectClaim)
	if err != nil {
		return gcputil.RequeueWithError(err)
	}
	return gcputil.StopProcessing()
}

func (c *ProjectClaimAdapter) EnsureFinalizer() (gcputil.OperationResult, error) {
	if !util.Contains(c.projectClaim.GetFinalizers(), ProjectClaimFinalizer) {
		c.logger.Info("Adding Finalizer to the ProjectClaim")
		err := c.addFinalizer(c.projectClaim, ProjectClaimFinalizer)
		return gcputil.RequeueOnErrorOrStop(err)
	}
	return gcputil.ContinueProcessing()
}

func (c *ProjectClaimAdapter) EnsureCCSSecretFinalizer() (gcputil.OperationResult, error) {
	if !c.projectClaim.Spec.CCS {
		return gcputil.ContinueProcessing()
	}

	secret, err := c.getCCSSecret()
	if err != nil {
		return gcputil.RequeueWithError(operrors.Wrap(err, "failed to set ccs secret finalizer"))
	}
	err = c.addFinalizer(secret, CCSSecretFinalizer)
	return gcputil.RequeueOnErrorOrContinue(err)
}

func (c *ProjectClaimAdapter) addFinalizer(object runtime.Object, finalizer string) error {
	metadata, err := meta.Accessor(object)
	if err != nil {
		return operrors.Wrap(err, "Failed to add finalizer "+finalizer)
	}
	finalizers := metadata.GetFinalizers()
	if !util.Contains(finalizers, finalizer) {
		metadata.SetFinalizers(append(finalizers, finalizer))
		return c.client.Update(context.TODO(), object)
	}
	return nil
}

func (c *ProjectClaimAdapter) getCCSSecret() (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	secretName := types.NamespacedName{
		Namespace: c.projectClaim.Spec.CCSSecretRef.Namespace,
		Name:      c.projectClaim.Spec.CCSSecretRef.Name,
	}
	err := c.client.Get(context.TODO(), secretName, secret)
	if err != nil {
		return nil, err
	}
	return secret, nil
}

func (c *ProjectClaimAdapter) EnsureProjectReferenceExists() (gcputil.OperationResult, error) {
	projectReferenceExists, err := c.ProjectReferenceExists()
	if err != nil {
		return gcputil.RequeueWithError(err)
	}

	if !projectReferenceExists {
		return gcputil.RequeueOnErrorOrContinue(c.client.Create(context.TODO(), c.projectReference))
	}
	return gcputil.ContinueProcessing()
}

func (c *ProjectClaimAdapter) EnsureProjectClaimStatePending() (gcputil.OperationResult, error) {
	return c.EnsureProjectClaimState(gcpv1alpha1.ClaimStatusPending)
}

func (c *ProjectClaimAdapter) EnsureProjectClaimStatePendingProject() (gcputil.OperationResult, error) {
	return c.EnsureProjectClaimState(gcpv1alpha1.ClaimStatusPendingProject)
}

func (c *ProjectClaimAdapter) EnsureProjectClaimState(state gcpv1alpha1.ClaimStatus) (gcputil.OperationResult, error) {
	if c.projectClaim.Status.State == state {
		return gcputil.ContinueProcessing()
	}

	if state == gcpv1alpha1.ClaimStatusPending && c.projectClaim.Status.State != gcpv1alpha1.ClaimStatusError {
		if c.projectClaim.Status.State != "" {
			return gcputil.ContinueProcessing()
		}
	}

	if state == gcpv1alpha1.ClaimStatusPendingProject {
		if c.projectClaim.Status.State != gcpv1alpha1.ClaimStatusPending {
			return gcputil.ContinueProcessing()
		}
	}

	c.projectClaim.Status.State = state
	return gcputil.RequeueOnErrorOrStop(c.StatusUpdate())
}

// SetProjectClaimCondition calls SetCondition() with project claim conditions
func (c *ProjectClaimAdapter) SetProjectClaimCondition(conditionType gcpv1alpha1.ConditionType, reason string, err error) (gcputil.OperationResult, error) {
	conditions := &c.projectClaim.Status.Conditions
	if err != nil {
		c.conditionManager.SetCondition(conditions, conditionType, corev1.ConditionTrue, reason, err.Error())
		return gcputil.RequeueOnErrorOrStop(c.StatusUpdate())
	}

	if !c.conditionManager.HasCondition(conditions, conditionType) {
		return gcputil.ContinueProcessing()
	}
	reason = reason + "Resolved"
	if condition, _ := c.conditionManager.FindCondition(conditions, conditionType); condition.Reason == reason {
		return gcputil.ContinueProcessing()
	}

	c.conditionManager.SetCondition(conditions, conditionType, corev1.ConditionFalse, reason, "")
	return gcputil.RequeueOnErrorOrStop(c.StatusUpdate())
}

// IsRegionSupported checks if current region is supported.
// It returns an error message if a region is not supported.
func (c *ProjectClaimAdapter) IsRegionSupported() (bool, error) {
	if c.projectClaim.Spec.CCS {
		return true, nil
	}

	operatorConfigMap, err := configmap.GetOperatorConfigMap(c.client)
	if err != nil {
		return true, operrors.Wrap(err, "could not find the OperatorConfigMap")
	}
	if util.Contains(operatorConfigMap.DisabledRegions, c.projectClaim.Spec.Region) {
		return false, nil
	}
	return true, nil
}

// EnsureRegionSupported modifies projectClaim.Status.State with result from IsRegionSupported.
// If a region is not supported it returns an error and sets projectClaim.Status.State to ClaimStatusError.
func (c *ProjectClaimAdapter) EnsureRegionSupported() (gcputil.OperationResult, error) {
	supported, err := c.IsRegionSupported()
	if err != nil {
		return gcputil.RequeueWithError(err)
	}

	if !supported {
		c.projectClaim.Status.State = gcpv1alpha1.ClaimStatusError
		err = operrors.ErrRegionNotSupported
	}

	if supported && c.projectClaim.Status.State == gcpv1alpha1.ClaimStatusError {
		c.projectClaim.Status.State = gcpv1alpha1.ClaimStatusPending
	}
	return c.SetProjectClaimCondition(gcpv1alpha1.ConditionInvalid, RegionCheckFailed, err)
}

// StatusUpdate updates the project claim status
func (c *ProjectClaimAdapter) StatusUpdate() error {
	if err := c.client.Status().Update(context.TODO(), c.projectClaim); err != nil {
		return operrors.Wrap(err, fmt.Sprintf("failed to update ProjectClaim state for %s", c.projectClaim.Name))
	}

	return nil
}
