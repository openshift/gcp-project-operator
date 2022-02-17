package projectclaim

// This file contains the functions for handling a fake ProjectClaim, that doesn't actually allocate resources on GCP

import (
	"context"
	"encoding/base64"
	"fmt"

	gcpv1alpha1 "github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1"
	gcputil "github.com/openshift/gcp-project-operator/pkg/util"
	operrors "github.com/openshift/gcp-project-operator/pkg/util/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (c *ProjectClaimAdapter) CreateFakeSecret() error {
	if !gcputil.SecretExists(c.client, c.projectClaim.Spec.GCPCredentialSecret.Name, c.projectClaim.Spec.GCPCredentialSecret.Namespace) {
		privateKeyString, err := base64.StdEncoding.DecodeString("SS1hbS1mYWtlLXBhc3M=")
		if err != nil {
			return err
		}
		if err := c.client.Create(context.TODO(), gcputil.NewGCPSecretCR(string(privateKeyString), types.NamespacedName{Namespace: c.projectClaim.Spec.GCPCredentialSecret.Namespace, Name: c.projectClaim.Spec.GCPCredentialSecret.Name})); err != nil {
			return err
		}
	}
	return nil
}

func (c *ProjectClaimAdapter) DeleteFakeSecret() error {
	secret := &corev1.Secret{}
	err := c.client.Get(context.TODO(), types.NamespacedName{
		Name:      c.projectClaim.Spec.GCPCredentialSecret.Name,
		Namespace: c.projectClaim.Spec.GCPCredentialSecret.Namespace},
		secret,
	)
	if err != nil {
		return err
	}

	err = c.client.Delete(context.TODO(), secret)
	if err != nil {
		return err
	}
	return nil
}

func (c *ProjectClaimAdapter) UpdateFakeProjectClaimSpecs() (bool, error) {
	if c.projectClaim.Spec.GCPProjectID != "fakeProjectClaim" {
		c.projectClaim.Spec.GCPProjectID = "fakeProjectClaim"
		c.projectClaim.Spec.GCPCredentialSecret = gcpv1alpha1.NamespacedName{
			Name:      c.projectClaim.GetName(),
			Namespace: c.projectClaim.GetNamespace(),
		}
		c.projectClaim.Spec.Region = "fakeRegion"
		c.projectClaim.Spec.AvailabilityZones = []string{
			"fake-az-a",
			"fake-az-b",
			"fake-az-c",
		}
		err := c.client.Update(context.TODO(), c.projectClaim)
		if err != nil {
			return true, err
		}
		return false, nil
	}
	return true, nil
}

func (c *ProjectClaimAdapter) UpdateFakeProjectClaimState() (bool, error) {
	if c.projectClaim.Status.State != gcpv1alpha1.ClaimStatusReady {
		c.projectClaim.Status.Conditions = []gcpv1alpha1.Condition{}
		c.projectClaim.Status.State = gcpv1alpha1.ClaimStatusReady
		err := c.client.Status().Update(context.TODO(), c.projectClaim)
		if err != nil {
			return true, err
		}
		return false, nil
	}
	return true, nil
}

func (c *ProjectClaimAdapter) EnsureProjectClaimFakeProcessed() (gcputil.OperationResult, error) {
	if c.projectClaim.Annotations[FakeProjectClaim] != "true" {
		return gcputil.ContinueProcessing()
	}
	if _, err := c.EnsureFinalizer(); err != nil {
		return gcputil.RequeueWithError(operrors.Wrap(err, fmt.Sprintf("Failed to add finalizer for %s", c.projectClaim.Name)))
	}
	// If project claim is marked for deletion, remove fake secret and project claim
	if c.projectClaim.DeletionTimestamp != nil {
		if err := c.DeleteFakeSecret(); err != nil {
			return gcputil.RequeueWithError(operrors.Wrap(err, fmt.Sprintf("Could not delete fake secret %s", c.projectClaim.Spec.GCPCredentialSecret.Name)))
		}
		if _, err := c.EnsureProjectClaimDeletionProcessed(); err != nil {
			return gcputil.RequeueWithError(operrors.Wrap(err, fmt.Sprintf("Could not delete project claim %s", c.projectClaim.Name)))
		}
		return gcputil.StopProcessing()
	}
	if err := c.CreateFakeSecret(); err != nil {
		return gcputil.RequeueWithError(operrors.Wrap(err, fmt.Sprintf("Could not create fake secret %s", c.projectClaim.Spec.GCPCredentialSecret.Name)))
	}
	result, err := c.UpdateFakeProjectClaimSpecs()
	if err != nil {
		return gcputil.RequeueWithError(operrors.Wrap(err, fmt.Sprintf("Could not update project claim specs for %s", c.projectClaim.Name)))
	}
	if !result {
		return gcputil.StopProcessing()
	}
	result, err = c.UpdateFakeProjectClaimState()
	if err != nil {
		return gcputil.RequeueWithError(operrors.Wrap(err, fmt.Sprintf("Could not update project claim specs for %s", c.projectClaim.Name)))
	}
	if !result {
		return gcputil.StopProcessing()
	}
	return gcputil.StopProcessing()
}
