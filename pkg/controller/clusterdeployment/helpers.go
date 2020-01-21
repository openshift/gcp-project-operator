package clusterdeployment

import (
	"github.com/openshift/gcp-project-operator/pkg/util/errors"
	hivev1alpha1 "github.com/openshift/hive/pkg/apis/hive/v1alpha1"
)

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
