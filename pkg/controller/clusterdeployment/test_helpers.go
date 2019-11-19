package clusterdeployment

import (
	//"errors"
	//"github.com/stretchr/testify/assert"
	"testing"

	"github.com/golang/mock/gomock"
	mockGCP "github.com/openshift/gcp-project-operator/pkg/gcpclient/mock"
	hivev1alpha1 "github.com/openshift/hive/pkg/apis/hive/v1alpha1"
	hivev1gcp "github.com/openshift/hive/pkg/apis/hive/v1alpha1/gcp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakekubeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testClusterName          = "clusterName"
	testUID                  = types.UID("1234")
	testNamespace            = "namespace"
	testBaseDomain           = "testing.example.com"
	testGCPCredentialsSecret = "GCPCredentialsSecret"
	testProject              = "project"
	testRegion               = "us-east1"
)

type mocks struct {
	fakeKubeClient client.Client
	mockCtrl       *gomock.Controller
	mockGCPClient  *mockGCP.MockClient
}

// setupDefaultMocks is an easy way to setup all of the default mocks
func setupDefaultMocks(t *testing.T, localObjects []runtime.Object) *mocks {
	mockKubeClient := fakekubeclient.NewFakeClient(localObjects...)
	mockCtrl := gomock.NewController(t)
	mockGCPClient := mockGCP.NewMockClient(mockCtrl)

	return &mocks{
		fakeKubeClient: mockKubeClient,
		mockCtrl:       mockCtrl,
		mockGCPClient:  mockGCPClient,
	}
}

type testSecretBuilder struct {
	s corev1.Secret
}

func (t *testSecretBuilder) getTestSecret() *corev1.Secret {
	return &t.s
}

func newtestSecretBuilder(secretName, namespace, creds string) *testSecretBuilder {
	return &testSecretBuilder{
		s: corev1.Secret{
			Type: "Opaque",
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: namespace,
			},
			Data: map[string][]byte{
				"osServiceAccount.json": []byte(creds),
				"billingaccount":        []byte("billingaccount"),
			},
		},
	}
}

func (t *testSecretBuilder) wihtoutKey(key string) *testSecretBuilder {
	delete(t.s.Data, key)
	return t
}

type testClusterDeploymentBuilder struct {
	cd hivev1alpha1.ClusterDeployment
}

func (t *testClusterDeploymentBuilder) getClusterDeployment() *hivev1alpha1.ClusterDeployment {
	return &t.cd
}

func newtestClusterDeploymentBuilder() *testClusterDeploymentBuilder {
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

func (t *testClusterDeploymentBuilder) withClusterPlatformLabel(value string) *testClusterDeploymentBuilder {
	t.cd.ObjectMeta.Labels[clusterPlatformLabel] = value
	return t
}

func (t *testClusterDeploymentBuilder) withOutClusterPlatformLabel() *testClusterDeploymentBuilder {
	delete(t.cd.ObjectMeta.Labels, clusterPlatformLabel)
	return t
}

func (t *testClusterDeploymentBuilder) withClusterDeploymentManagedLabel(value string) *testClusterDeploymentBuilder {
	t.cd.ObjectMeta.Labels[clusterDeploymentManagedLabel] = value
	return t
}

func (t *testClusterDeploymentBuilder) withOutClusterDeploymentManagedLabel() *testClusterDeploymentBuilder {
	delete(t.cd.ObjectMeta.Labels, clusterDeploymentManagedLabel)
	return t
}

func (t *testClusterDeploymentBuilder) installed() *testClusterDeploymentBuilder {
	t.cd.Spec.Installed = true
	return t
}

func (t *testClusterDeploymentBuilder) withRegion(region string) *testClusterDeploymentBuilder {
	t.cd.Spec.GCP.Region = region
	return t
}

func (t *testClusterDeploymentBuilder) withOutRegion() *testClusterDeploymentBuilder {
	t.cd.Spec.GCP.Region = ""
	return t
}

func (t *testClusterDeploymentBuilder) withOutProjectID() *testClusterDeploymentBuilder {
	t.cd.Spec.GCP.ProjectID = ""
	return t
}
