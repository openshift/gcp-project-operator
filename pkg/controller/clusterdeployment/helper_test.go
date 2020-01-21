package clusterdeployment

import (
	"testing"

	"github.com/openshift/gcp-project-operator/pkg/util/errors"
	builders "github.com/openshift/gcp-project-operator/pkg/util/mocks/structs"
	hivev1alpha1 "github.com/openshift/hive/pkg/apis/hive/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestCheckDeploymentConfigRequirements(t *testing.T) {
	tests := []struct {
		name              string
		clusterDeployment *hivev1alpha1.ClusterDeployment
		expectedErr       error
		validateErr       func(*testing.T, error, error)
	}{
		{
			name:              "All requirements Pass",
			clusterDeployment: builders.NewTestClusterDeploymentBuilder().GetClusterDeployment(),
		},
		{
			name:              "No clusterPlatformLabel",
			clusterDeployment: builders.NewTestClusterDeploymentBuilder().WithOutClusterPlatformLabel().GetClusterDeployment(),
			expectedErr:       errors.ErrNotGCPCluster,
			validateErr: func(t *testing.T, expected, result error) {
				assert.Equal(t, expected, result)
			},
		},
		{
			name:              "Wrong clusterPlatformLabel",
			clusterDeployment: builders.NewTestClusterDeploymentBuilder().WithClusterPlatformLabel("AWS").GetClusterDeployment(),
			expectedErr:       errors.ErrNotGCPCluster,
			validateErr: func(t *testing.T, expected, result error) {
				assert.Equal(t, expected, result)
			},
		},
		{
			name:              "No clusterDeploymentManagedLabel",
			clusterDeployment: builders.NewTestClusterDeploymentBuilder().WithOutClusterDeploymentManagedLabel().GetClusterDeployment(),
			expectedErr:       errors.ErrNotManagedCluster,
			validateErr: func(t *testing.T, expected, result error) {
				assert.Equal(t, expected, result)
			},
		},
		{
			name:              "Wrong clusterDeploymentManagedLabel",
			clusterDeployment: builders.NewTestClusterDeploymentBuilder().WithClusterDeploymentManagedLabel("false").GetClusterDeployment(),
			expectedErr:       errors.ErrNotManagedCluster,
			validateErr: func(t *testing.T, expected, result error) {
				assert.Equal(t, expected, result)
			},
		},
		{
			name:              "Cluster installed",
			clusterDeployment: builders.NewTestClusterDeploymentBuilder().Installed().GetClusterDeployment(),
			expectedErr:       errors.ErrClusterInstalled,
			validateErr: func(t *testing.T, expected, result error) {
				assert.Equal(t, expected, result)
			},
		},
		{
			name:              "No region",
			clusterDeployment: builders.NewTestClusterDeploymentBuilder().WithOutRegion().GetClusterDeployment(),
			expectedErr:       errors.ErrMissingRegion,
			validateErr: func(t *testing.T, expected, result error) {
				assert.Equal(t, expected, result)
			},
		},
		{
			name:              "Not supported region",
			clusterDeployment: builders.NewTestClusterDeploymentBuilder().WithRegion("not supported").GetClusterDeployment(),
			expectedErr:       errors.ErrRegionNotSupported,
			validateErr: func(t *testing.T, expected, result error) {
				assert.Equal(t, expected, result)
			},
		},
		{
			name:              "No projectID",
			clusterDeployment: builders.NewTestClusterDeploymentBuilder().WithOutProjectID().GetClusterDeployment(),
			expectedErr:       errors.ErrMissingProjectID,
			validateErr: func(t *testing.T, expected, result error) {
				assert.Equal(t, expected, result)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := checkDeploymentConfigRequirements(test.clusterDeployment)

			if test.expectedErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if test.validateErr != nil {
				test.validateErr(t, test.expectedErr, err)
			}

		})
	}
}
