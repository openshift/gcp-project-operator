package clusterdeployment

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/openshift/gcp-project-operator/pkg/gcpclient"
	mockGCP "github.com/openshift/gcp-project-operator/pkg/gcpclient/mock"
	hiveapis "github.com/openshift/hive/pkg/apis"
	"github.com/stretchr/testify/assert"
	cloudresourcemanager "google.golang.org/api/cloudresourcemanager/v1"
	iam "google.golang.org/api/iam/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcile(t *testing.T) {
	hiveapis.AddToScheme(scheme.Scheme)

	tests := []struct {
		name         string
		expectedErr  bool
		localObjects []runtime.Object
		setupGCPMock func(r *mockGCP.MockClientMockRecorder)
	}{
		{
			name:         "Cluster Deployment not found",
			expectedErr:  false,
			setupGCPMock: func(r *mockGCP.MockClientMockRecorder) { gomock.Any() },
		},
		{
			name:        "CD check fail ErrMissingRegion",
			expectedErr: true,
			localObjects: []runtime.Object{
				// test cluster deployment
				newtestClusterDeploymentBuilder().withOutRegion().getClusterDeployment(),
			},
			setupGCPMock: func(r *mockGCP.MockClientMockRecorder) { gomock.Any() },
		},
		{
			name:        "CD check fail ErrClusterInstalled",
			expectedErr: false,
			localObjects: []runtime.Object{
				// test cluster deployment
				newtestClusterDeploymentBuilder().installed().getClusterDeployment(),
			},
			setupGCPMock: func(r *mockGCP.MockClientMockRecorder) { gomock.Any() },
		},
		{
			name:        "Failed to get ORG Creds",
			expectedErr: true,
			localObjects: []runtime.Object{
				// test cluster deployment
				newtestClusterDeploymentBuilder().getClusterDeployment(),
			},
			setupGCPMock: func(r *mockGCP.MockClientMockRecorder) { gomock.Any() },
		},
		{
			name:        "Failed to get ORG Creds",
			expectedErr: true,
			localObjects: []runtime.Object{
				// test cluster deployment
				newtestClusterDeploymentBuilder().getClusterDeployment(),
			},
			setupGCPMock: func(r *mockGCP.MockClientMockRecorder) { gomock.Any() },
		},
		{
			name:        "Final Secret Exists",
			expectedErr: false,
			localObjects: []runtime.Object{
				// test cluster deployment
				newtestClusterDeploymentBuilder().getClusterDeployment(),
				// GCP org secret in operator namespace
				testSecret(orgGcpSecretName, operatorNamespace, "testCreds"),
				// GCP secret in cluster deployment namespace
				testSecret(gcpSecretName, testNamespace, "testCreds"),
			},
			setupGCPMock: func(r *mockGCP.MockClientMockRecorder) { gomock.Any() },
		},
		{
			name:        "GetServiceAccount & CreateServiceAccount Error",
			expectedErr: true,
			localObjects: []runtime.Object{
				// test cluster deployment
				newtestClusterDeploymentBuilder().getClusterDeployment(),
				// GCP org secret in operator namespace
				testSecret(orgGcpSecretName, operatorNamespace, "testCreds"),
			},
			setupGCPMock: func(r *mockGCP.MockClientMockRecorder) {
				gomock.InOrder(
					r.GetServiceAccount(gomock.Any()).Return(
						&iam.ServiceAccount{}, errors.New("GetServiceAccount Error")).Times(1),
					r.CreateServiceAccount(gomock.Any(), gomock.Any()).Return(
						&iam.ServiceAccount{}, errors.New("CreateServiceAccount Error")).Times(1))
			},
		},
		{
			name:        "GetIamPolicy Error",
			expectedErr: true,
			localObjects: []runtime.Object{
				// test cluster deployment
				newtestClusterDeploymentBuilder().getClusterDeployment(),
				// GCP org secret in operator namespace
				testSecret(orgGcpSecretName, operatorNamespace, "testCreds"),
			},
			setupGCPMock: func(r *mockGCP.MockClientMockRecorder) {
				gomock.InOrder(
					r.GetServiceAccount(gomock.Any()).Return(
						&iam.ServiceAccount{}, errors.New("GetServiceAccount Error")).Times(1),
					r.CreateServiceAccount(gomock.Any(), gomock.Any()).Return(
						&iam.ServiceAccount{}, nil).Times(1),
					r.GetIamPolicy().Return(&cloudresourcemanager.Policy{}, errors.New("GetIamPolicy Error")).Times(1))
			},
		},
		{
			name:        "SetIamPolicy Error",
			expectedErr: true,
			localObjects: []runtime.Object{
				// test cluster deployment
				newtestClusterDeploymentBuilder().getClusterDeployment(),
				// GCP org secret in operator namespace
				testSecret(orgGcpSecretName, operatorNamespace, "testCreds"),
			},
			setupGCPMock: func(r *mockGCP.MockClientMockRecorder) {
				gomock.InOrder(
					r.GetServiceAccount(gomock.Any()).Return(
						&iam.ServiceAccount{}, errors.New("GetServiceAccount Error")).Times(1),
					r.CreateServiceAccount(gomock.Any(), gomock.Any()).Return(
						&iam.ServiceAccount{}, nil).Times(1),
					r.GetIamPolicy().Return(
						&cloudresourcemanager.Policy{}, nil).Times(1),
					r.SetIamPolicy(gomock.Any()).Return(
						&cloudresourcemanager.Policy{}, errors.New("SetIamPolicy Error")).Times(1))
			},
		},
		{
			name:        "DeleteServiceAccountKeys Error",
			expectedErr: true,
			localObjects: []runtime.Object{
				// test cluster deployment
				newtestClusterDeploymentBuilder().getClusterDeployment(),
				// GCP org secret in operator namespace
				testSecret(orgGcpSecretName, operatorNamespace, "testCreds"),
			},
			setupGCPMock: func(r *mockGCP.MockClientMockRecorder) {
				gomock.InOrder(
					r.GetServiceAccount(gomock.Any()).Return(
						&iam.ServiceAccount{}, errors.New("GetServiceAccount Error")).Times(1),
					r.CreateServiceAccount(gomock.Any(), gomock.Any()).Return(
						&iam.ServiceAccount{}, nil).Times(1),
					r.GetIamPolicy().Return(
						&cloudresourcemanager.Policy{}, nil).Times(1),
					r.SetIamPolicy(gomock.Any()).Return(
						&cloudresourcemanager.Policy{}, nil).Times(1),
					r.DeleteServiceAccountKeys(gomock.Any()).Return(
						errors.New("DeleteServiceAccountKeys Error")).Times(1))
			},
		},
		{
			name:        "CreateServiceAccountKey Error",
			expectedErr: true,
			localObjects: []runtime.Object{
				// test cluster deployment
				newtestClusterDeploymentBuilder().getClusterDeployment(),
				// GCP org secret in operator namespace
				testSecret(orgGcpSecretName, operatorNamespace, "testCreds"),
			},
			setupGCPMock: func(r *mockGCP.MockClientMockRecorder) {
				gomock.InOrder(
					r.GetServiceAccount(gomock.Any()).Return(
						&iam.ServiceAccount{}, errors.New("GetServiceAccount Error")).Times(1),
					r.CreateServiceAccount(gomock.Any(), gomock.Any()).Return(
						&iam.ServiceAccount{}, nil).Times(1),
					r.GetIamPolicy().Return(
						&cloudresourcemanager.Policy{}, nil).Times(1),
					r.SetIamPolicy(gomock.Any()).Return(
						&cloudresourcemanager.Policy{}, nil).Times(1),
					r.DeleteServiceAccountKeys(gomock.Any()).Return(
						nil).Times(1),
					r.CreateServiceAccountKey(gomock.Any()).Return(
						&iam.ServiceAccountKey{}, errors.New("CreateServiceAccountKey Error")).Times(1))
			},
		},
		{
			name:        "Error decoding base64",
			expectedErr: true,
			localObjects: []runtime.Object{
				// test cluster deployment
				newtestClusterDeploymentBuilder().getClusterDeployment(),
				// GCP org secret in operator namespace
				testSecret(orgGcpSecretName, operatorNamespace, "testCreds"),
			},
			setupGCPMock: func(r *mockGCP.MockClientMockRecorder) {
				gomock.InOrder(
					r.GetServiceAccount(gomock.Any()).Return(
						&iam.ServiceAccount{}, errors.New("GetServiceAccount Error")).Times(1),
					r.CreateServiceAccount(gomock.Any(), gomock.Any()).Return(
						&iam.ServiceAccount{}, nil).Times(1),
					r.GetIamPolicy().Return(
						&cloudresourcemanager.Policy{}, nil).Times(1),
					r.SetIamPolicy(gomock.Any()).Return(
						&cloudresourcemanager.Policy{}, nil).Times(1),
					r.DeleteServiceAccountKeys(gomock.Any()).Return(
						nil).Times(1),
					r.CreateServiceAccountKey(gomock.Any()).Return(
						&iam.ServiceAccountKey{
							PrivateKeyData: "Fake private data",
						}, nil).Times(1))
			},
		},
		{
			name:        "No errors without final Secret",
			expectedErr: false,
			localObjects: []runtime.Object{
				// test cluster deployment
				newtestClusterDeploymentBuilder().getClusterDeployment(),
				// GCP org secret in operator namespace
				testSecret(orgGcpSecretName, operatorNamespace, "testCreds"),
			},
			setupGCPMock: func(r *mockGCP.MockClientMockRecorder) {
				gomock.InOrder(
					r.GetServiceAccount(gomock.Any()).Return(
						&iam.ServiceAccount{}, errors.New("GetServiceAccount Error")).Times(1),
					r.CreateServiceAccount(gomock.Any(), gomock.Any()).Return(
						&iam.ServiceAccount{}, nil).Times(1),
					r.GetIamPolicy().Return(
						&cloudresourcemanager.Policy{
							Bindings: []*cloudresourcemanager.Binding{
								{
									Members: []string{"serviceAccount:service1@google.com"},
									Role:    "roles/storage.admin",
								},
							}}, nil).Times(1),
					r.SetIamPolicy(gomock.Any()).Return(
						&cloudresourcemanager.Policy{}, nil).Times(1),
					r.DeleteServiceAccountKeys(gomock.Any()).Return(
						nil).Times(1),
					r.CreateServiceAccountKey(gomock.Any()).Return(
						&iam.ServiceAccountKey{
							PrivateKeyData: "IkZha2UgcHJpdmF0ZSBkYXRhIg==",
						}, nil).Times(1))
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Arrage
			mocks := setupDefaultMocks(t, test.localObjects)
			test.setupGCPMock(mocks.mockGCPClient.EXPECT())

			gcpBuilder := func(projectName string, authJSON []byte) (gcpclient.Client, error) {
				return mocks.mockGCPClient, nil
			}

			// This is necessary for the mocks to report failures like methods not being called an expected number of times.
			// after mocks is defined
			defer mocks.mockCtrl.Finish()

			rcd := &ReconcileClusterDeployment{
				mocks.fakeKubeClient,
				scheme.Scheme,
				gcpBuilder,
			}

			// Act
			_, err := rcd.Reconcile(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      testClusterName,
					Namespace: testNamespace,
				},
			})

			// Assert
			if test.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

		})
	}
}
