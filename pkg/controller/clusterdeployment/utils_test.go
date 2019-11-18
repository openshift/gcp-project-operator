package clusterdeployment

import (
	"errors"
	"fmt"
	"testing"

	hivev1alpha1 "github.com/openshift/hive/pkg/apis/hive/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// type mocks struct {
// 	fakeKubeClient client.Client
// 	mockCtrl       *gomock.Controller
// 	mockGCPClient  *mockGCP.MockClient
// }

// // setupDefaultMocks is an easy way to setup all of the default mocks
// func setupDefaultMocks(t *testing.T, localObjects []runtime.Object) *mocks {
// 	mocks := &mocks{
// 		fakeKubeClient: fakekubeclient.NewFakeClient(localObjects...),
// 		mockCtrl:       gomock.NewController(t),
// 	}

// 	mocks.mockGCPClient = mockGCP.NewMockClient(mocks.mockCtrl)
// 	return mocks
// }

func TestStringInSlice(t *testing.T) {
	tests := []struct {
		name           string
		stringSlice    []string
		searchString   string
		expectedReturn bool
	}{
		{
			name:           "String contained in slice",
			stringSlice:    []string{"one", "two", "three"},
			searchString:   "one",
			expectedReturn: true,
		},
		{
			name:           "String not contained in slice",
			stringSlice:    []string{"one", "two", "three"},
			searchString:   "four",
			expectedReturn: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := stringInSlice(test.searchString, test.stringSlice)
			assert.Equal(t, test.expectedReturn, result)
		})
	}
}

func TestRemove(t *testing.T) {
	tests := []struct {
		name           string
		stringSlice    []string
		indexToRemove  int
		expectedReturn []string
	}{
		{
			name:           "Remove index 0",
			stringSlice:    []string{"zero", "one", "two"},
			indexToRemove:  0,
			expectedReturn: []string{"one", "two"},
		},
		{
			name:           "Remove index 2",
			stringSlice:    []string{"zero", "one", "two"},
			indexToRemove:  2,
			expectedReturn: []string{"zero", "one"},
		},
		{
			name:           "Remove index 0 return order change",
			stringSlice:    []string{"zero", "one", "two"},
			indexToRemove:  0,
			expectedReturn: []string{"two", "one"},
		},
		{
			name:           "Remove index 2 return order change",
			stringSlice:    []string{"zero", "one", "two"},
			indexToRemove:  2,
			expectedReturn: []string{"one", "zero"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := remove(test.stringSlice, test.indexToRemove)
			assert.ElementsMatch(t, test.expectedReturn, result)
		})
	}
}

func TestFindMemberIndex(t *testing.T) {
	tests := []struct {
		name           string
		searchMember   string
		members        []string
		expectedReturn int
	}{
		{
			name:           "String is in slice index 2",
			searchMember:   "test",
			members:        []string{"apple", "orange", "test"},
			expectedReturn: 2,
		},
		{
			name:           "String is in slice index 0",
			searchMember:   "test",
			members:        []string{"test", "apple", "orange"},
			expectedReturn: 0,
		},
		{
			name:           "String is not in slice",
			searchMember:   "test",
			members:        []string{"apple", "orange"},
			expectedReturn: -1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := findMemberIndex(test.searchMember, test.members)
			assert.Equal(t, test.expectedReturn, result)
		})
	}
}

func TestSecretExists(t *testing.T) {
	tests := []struct {
		name            string
		localObjects    []runtime.Object
		secretName      string
		secretNamespace string
		expectedResult  bool
	}{
		{
			name:            "Secret Exists",
			expectedResult:  true,
			secretName:      "testName",
			secretNamespace: "testNamespace",
			localObjects: []runtime.Object{
				testSecret("testName", "testNamespace", "testCreds"),
			},
		},
		{
			name:            "Secret does not exist",
			expectedResult:  false,
			secretName:      "badName",
			secretNamespace: "testNamespace",
			localObjects: []runtime.Object{
				testSecret("testName", "testNamespace", "testCreds"),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mocks := setupDefaultMocks(t, test.localObjects)

			result := secretExists(mocks.fakeKubeClient, test.secretName, test.secretNamespace)
			assert.Equal(t, test.expectedResult, result)
		})
	}

}

func TestGetSecret(t *testing.T) {
	tests := []struct {
		name            string
		localObjects    []runtime.Object
		secretName      string
		secretNamespace string
		expectedSecret  *corev1.Secret
		expectedErr     bool
		validateResult  func(*testing.T, *corev1.Secret, *corev1.Secret)
	}{
		{
			name:            "Existing Secret",
			secretName:      "testName",
			secretNamespace: "testNamespace",
			localObjects: []runtime.Object{
				testSecret("testName", "testNamespace", "testCreds"),
			},
			expectedSecret: testSecret("testName", "testNamespace", "testCreds"),
			expectedErr:    false,
			validateResult: func(t *testing.T, expected, result *corev1.Secret) {
				assert.Equal(t, expected, result)
			},
		},
		{
			name:            "Secret does not exist",
			secretName:      "badName",
			secretNamespace: "testNamespace",
			localObjects: []runtime.Object{
				testSecret("testName", "testNamespace", "testCreds"),
			},
			expectedSecret: &corev1.Secret{},
			expectedErr:    true,
			validateResult: func(t *testing.T, expected, result *corev1.Secret) {
				assert.Equal(t, expected, result)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mocks := setupDefaultMocks(t, test.localObjects)

			result, err := getSecret(mocks.fakeKubeClient, test.secretName, test.secretNamespace)

			if test.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if test.validateResult != nil {
				test.validateResult(t, test.expectedSecret, result)
			}
		})
	}

}

func TestNewGCPSecretCR(t *testing.T) {
	tests := []struct {
		name            string
		secretName      string
		secretNamespace string
		secretCreds     string
		expectedSecret  *corev1.Secret
		validateResult  func(*testing.T, *corev1.Secret, *corev1.Secret)
	}{
		{
			name:            "Correct GCP Secert",
			secretName:      gcpSecretName,
			secretNamespace: "testNamespace",
			secretCreds:     "testCreds",
			expectedSecret:  testSecret(gcpSecretName, "testNamespace", "testCreds"),
			validateResult: func(t *testing.T, expected, result *corev1.Secret) {
				assert.Equal(t, expected, result)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			result := newGCPSecretCR(test.secretNamespace, test.secretCreds)

			if test.validateResult != nil {
				test.validateResult(t, test.expectedSecret, result)
			}
		})
	}
}

func TestGetOrgGCPCreds(t *testing.T) {
	tests := []struct {
		name            string
		localObjects    []runtime.Object
		secretNamespace string
		expectedCreds   []byte
		expectedErr     error
		validateResult  func(*testing.T, []byte, []byte)
		validateErr     func(*testing.T, error, error)
	}{
		{
			name: "Correct ORG GCP Secert",
			localObjects: []runtime.Object{
				testSecret(orgGcpSecretName, "testNamespace", "testCreds"),
			},
			secretNamespace: "testNamespace",
			expectedCreds:   []byte("testCreds"),
			//ExpectedErr:     nil,
			validateResult: func(t *testing.T, expected, result []byte) {
				assert.Equal(t, expected, result)
			},
		},
		{
			name:            "ORG GCP Secert not found",
			localObjects:    []runtime.Object{},
			secretNamespace: "testNamespace",
			expectedCreds:   []byte{},
			expectedErr:     errors.New("error"),
			validateResult: func(t *testing.T, expected, result []byte) {
				assert.Equal(t, expected, result)
			},
		},
		{
			name: "Bad data in ORG GCP Secert",
			localObjects: func() []runtime.Object {
				sec := &corev1.Secret{
					Type: "Opaque",
					TypeMeta: metav1.TypeMeta{
						Kind:       "Secret",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      orgGcpSecretName,
						Namespace: "testNamespace",
					},
				}
				return []runtime.Object{sec}
			}(),
			secretNamespace: "testNamespace",
			expectedCreds:   []byte{},
			expectedErr:     fmt.Errorf("GCP credentials secret %v did not contain key {osServiceAccount,key}.json", orgGcpSecretName),
			validateResult: func(t *testing.T, expected, result []byte) {
				assert.Equal(t, expected, result)
			},
			validateErr: func(t *testing.T, expected, result error) {
				assert.Equal(t, expected, result)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mocks := setupDefaultMocks(t, test.localObjects)

			result, err := getGCPCredentialsFromSecret(mocks.fakeKubeClient, test.secretNamespace, "testCreds")

			if test.expectedErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if test.validateResult != nil {
				test.validateResult(t, test.expectedCreds, result)
			}

			if test.validateErr != nil {
				test.validateErr(t, test.expectedErr, err)
			}

		})
	}
}

func TestCheckDeploymentConfigRequirements(t *testing.T) {
	tests := []struct {
		name              string
		clusterDeployment *hivev1alpha1.ClusterDeployment
		expectedErr       error
		validateErr       func(*testing.T, error, error)
	}{
		{
			name:              "All requirements Pass",
			clusterDeployment: newtestClusterDeploymentBuilder().getClusterDeployment(),
		},
		{
			name:              "No clusterPlatformLabel",
			clusterDeployment: newtestClusterDeploymentBuilder().withOutClusterPlatformLabel().getClusterDeployment(),
			expectedErr:       ErrNotGCPCluster,
			validateErr: func(t *testing.T, expected, result error) {
				assert.Equal(t, expected, result)
			},
		},
		{
			name:              "Wrong clusterPlatformLabel",
			clusterDeployment: newtestClusterDeploymentBuilder().withClusterPlatformLabel("AWS").getClusterDeployment(),
			expectedErr:       ErrNotGCPCluster,
			validateErr: func(t *testing.T, expected, result error) {
				assert.Equal(t, expected, result)
			},
		},
		{
			name:              "No clusterDeploymentManagedLabel",
			clusterDeployment: newtestClusterDeploymentBuilder().withOutClusterDeploymentManagedLabel().getClusterDeployment(),
			expectedErr:       ErrNotManagedCluster,
			validateErr: func(t *testing.T, expected, result error) {
				assert.Equal(t, expected, result)
			},
		},
		{
			name:              "Wrong clusterDeploymentManagedLabel",
			clusterDeployment: newtestClusterDeploymentBuilder().withClusterDeploymentManagedLabel("false").getClusterDeployment(),
			expectedErr:       ErrNotManagedCluster,
			validateErr: func(t *testing.T, expected, result error) {
				assert.Equal(t, expected, result)
			},
		},
		{
			name:              "Cluster installed",
			clusterDeployment: newtestClusterDeploymentBuilder().installed().getClusterDeployment(),
			expectedErr:       ErrClusterInstalled,
			validateErr: func(t *testing.T, expected, result error) {
				assert.Equal(t, expected, result)
			},
		},
		{
			name:              "No region",
			clusterDeployment: newtestClusterDeploymentBuilder().withOutRegion().getClusterDeployment(),
			expectedErr:       ErrMissingRegion,
			validateErr: func(t *testing.T, expected, result error) {
				assert.Equal(t, expected, result)
			},
		},
		{
			name:              "Not supported region",
			clusterDeployment: newtestClusterDeploymentBuilder().withRegion("not supported").getClusterDeployment(),
			expectedErr:       ErrRegionNotSupported,
			validateErr: func(t *testing.T, expected, result error) {
				assert.Equal(t, expected, result)
			},
		},
		{
			name:              "No projectID",
			clusterDeployment: newtestClusterDeploymentBuilder().withOutProjectID().getClusterDeployment(),
			expectedErr:       ErrMissingProjectID,
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
