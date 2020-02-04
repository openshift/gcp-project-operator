package clusterdeployment

import (
	"net/http"
	"time"

	"github.com/openshift/gcp-project-operator/pkg/gcpclient"
	"github.com/openshift/gcp-project-operator/pkg/util"
	"github.com/openshift/gcp-project-operator/pkg/util/errors"
	hivev1alpha1 "github.com/openshift/hive/pkg/apis/hive/v1alpha1"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/googleapi"
)

// AddorUpdateBindingResponse contines the data that is returned by the AddOrUpdarteBindings function
type AddorUpdateBindingResponse struct {
	modified bool
	policy   *cloudresourcemanager.Policy
	bindings []*cloudresourcemanager.Binding
}

// AddOrUpdateBindings gets the policy and checks if the bindings match the required roles
func AddOrUpdateBindings(gcpclient gcpclient.Client, projectID, serviceAccountEmail string) (AddorUpdateBindingResponse, error) {
	policy, err := gcpclient.GetIamPolicy(projectID)
	if err != nil {
		return AddorUpdateBindingResponse{}, err
	}

	//Checking if policy is modified
	newBindings, modified := util.AddOrUpdateBinding(policy.Bindings, OSDRequiredRoles, serviceAccountEmail)

	// add new bindings to policy
	policy.Bindings = newBindings
	return AddorUpdateBindingResponse{
		modified: modified,
		policy:   policy,
	}, nil
}

// SetIAMPolicy attempts to update policy if the policy needs to be modified
func SetIAMPolicy(gcpclient gcpclient.Client, projectID, serviceAccountEmail string) error {
	// Checking if policy needs to be updated
	var retry int
	for {
		retry++
		time.Sleep(time.Second)
		addorUpdateResponse, err := AddOrUpdateBindings(gcpclient, projectID, serviceAccountEmail)
		if err != nil {
			return err
		}

		// If existing bindings have been modified update the policy
		if addorUpdateResponse.modified {
			setIamPolicyRequest := &cloudresourcemanager.SetIamPolicyRequest{
				Policy: addorUpdateResponse.policy,
			}

			_, err = gcpclient.SetIamPolicy(setIamPolicyRequest)
			if err != nil {
				ae, ok := err.(*googleapi.Error)
				// retry rules below:

				if ok && ae.Code == http.StatusConflict && retry <= 3 {
					continue
				}
				return err
			}
			return nil
		}
		return nil
	}
}

// checkDeploymentConfigRequirements checks that parameters required exist and that they are set correctly. If not it returns an error
func checkDeploymentConfigRequirements(cd *hivev1alpha1.ClusterDeployment) error {
	// Do not make do anything if the cluster is not a GCP cluster.
	val, ok := cd.Labels[clusterPlatformLabel]
	if !ok || val != clusterPlatformGCP {
		return errors.ErrNotGCPCluster
	}

	// Do not do anything if the cluster is not a Red Hat managed cluster.
	val, ok = cd.Labels[clusterDeploymentManagedLabel]
	if !ok || val != "true" {
		return errors.ErrNotManagedCluster
	}

	//Do not reconcile if cluster is installed or remove cleanup and remove project
	if cd.Spec.Installed {
		return errors.ErrClusterInstalled
	}

	if cd.Spec.Platform.GCP.Region == "" {
		return errors.ErrMissingRegion
	}

	if cd.Spec.Platform.GCP.ProjectID == "" {
		return errors.ErrMissingProjectID
	}

	if _, ok := supportedRegions[cd.Spec.Platform.GCP.Region]; !ok {
		return errors.ErrRegionNotSupported
	}

	return nil
}
