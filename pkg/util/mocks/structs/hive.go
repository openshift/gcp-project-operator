package structs

import (
	//"errors"
	//"github.com/stretchr/testify/assert"

	hivev1alpha1 "github.com/openshift/hive/pkg/apis/hive/v1alpha1"
	hivev1gcp "github.com/openshift/hive/pkg/apis/hive/v1alpha1/gcp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type testClusterDeploymentBuilder struct {
	cd hivev1alpha1.ClusterDeployment
}

func (t *testClusterDeploymentBuilder) GetClusterDeployment() *hivev1alpha1.ClusterDeployment {
	return &t.cd
}

func NewTestClusterDeploymentBuilder() *testClusterDeploymentBuilder {
	return &testClusterDeploymentBuilder{
		cd: hivev1alpha1.ClusterDeployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testClusterName,
				Namespace: testNamespace,
				UID:       testUID,
				Labels: map[string]string{
					clusterPlatformLabel:          clusterPlatformGCP,
					clusterDeploymentManagedLabel: "true",
				},
			},
			Spec: hivev1alpha1.ClusterDeploymentSpec{
				Installed:   false,
				BaseDomain:  testBaseDomain,
				ClusterName: testClusterName,
				Platform: hivev1alpha1.Platform{
					GCP: &hivev1gcp.Platform{
						ProjectID: testProject,
						Region:    testRegion,
					},
				},
			},
		},
	}
}

func (t *testClusterDeploymentBuilder) WithClusterPlatformLabel(value string) *testClusterDeploymentBuilder {
	t.cd.ObjectMeta.Labels[clusterPlatformLabel] = value
	return t
}

func (t *testClusterDeploymentBuilder) WithOutClusterPlatformLabel() *testClusterDeploymentBuilder {
	delete(t.cd.ObjectMeta.Labels, clusterPlatformLabel)
	return t
}

func (t *testClusterDeploymentBuilder) WithClusterDeploymentManagedLabel(value string) *testClusterDeploymentBuilder {
	t.cd.ObjectMeta.Labels[clusterDeploymentManagedLabel] = value
	return t
}

func (t *testClusterDeploymentBuilder) WithOutClusterDeploymentManagedLabel() *testClusterDeploymentBuilder {
	delete(t.cd.ObjectMeta.Labels, clusterDeploymentManagedLabel)
	return t
}

func (t *testClusterDeploymentBuilder) Installed() *testClusterDeploymentBuilder {
	t.cd.Spec.Installed = true
	return t
}

func (t *testClusterDeploymentBuilder) WithRegion(region string) *testClusterDeploymentBuilder {
	t.cd.Spec.GCP.Region = region
	return t
}

func (t *testClusterDeploymentBuilder) WithOutRegion() *testClusterDeploymentBuilder {
	t.cd.Spec.GCP.Region = ""
	return t
}

func (t *testClusterDeploymentBuilder) WithOutProjectID() *testClusterDeploymentBuilder {
	t.cd.Spec.GCP.ProjectID = ""
	return t
}
