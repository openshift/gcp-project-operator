package clusterdeployment

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/openshift/gcp-project-operator/pkg/gcpclient"
	mockGCP "github.com/openshift/gcp-project-operator/pkg/gcpclient/mock"
	hiveapis "github.com/openshift/hive/pkg/apis"
	cloudresourcemanager "google.golang.org/api/cloudresourcemanager/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcile(t *testing.T) {
	hiveapis.AddToScheme(scheme.Scheme)

	tests := []struct {
		name         string
		expectedErr  error
		localObjects []runtime.Object
		setupGCPMock func(r *mockGCP.MockClientMockRecorder)
	}{
		{
			name:         "cluster deployment not found",
			expectedErr:  nil,
			setupGCPMock: func(r *mockGCP.MockClientMockRecorder) { gomock.Any() },
		},
		{
			name:        "CD check fail ErrMissingRegion",
			expectedErr: fmt.Errorf("MissingRegion"),
			localObjects: []runtime.Object{
				newtestClusterDeploymentBuilder().withOutRegion().getClusterDeployment(),
			},
			setupGCPMock: func(r *mockGCP.MockClientMockRecorder) { gomock.Any() },
		},
		{
			name:        "CD check fail ErrClusterInstalled",
			expectedErr: nil,
			localObjects: []runtime.Object{
				newtestClusterDeploymentBuilder().installed().getClusterDeployment(),
			},
			setupGCPMock: func(r *mockGCP.MockClientMockRecorder) { gomock.Any() },
		},
		{
			name:        "failed to get ORG creds",
			expectedErr: fmt.Errorf("clusterdeployment.getGCPCredentialsFromSecret.Get secrets \"gcp-project-operator\" not found"),
			localObjects: []runtime.Object{
				newtestClusterDeploymentBuilder().getClusterDeployment(),
			},
			setupGCPMock: func(r *mockGCP.MockClientMockRecorder) { gomock.Any() },
		},
		{
			name:        "final secret exists",
			expectedErr: nil,
			localObjects: []runtime.Object{
				newtestClusterDeploymentBuilder().getClusterDeployment(),
				newtestSecretBuilder(orgGcpSecretName, operatorNamespace, "testCreds").getTestSecret(),
				newtestSecretBuilder(gcpSecretName, testNamespace, "testCreds").getTestSecret(),
			},
			setupGCPMock: func(r *mockGCP.MockClientMockRecorder) { gomock.Any() },
		},
		{
			name:        "no billing key in secret",
			expectedErr: fmt.Errorf("GCP credentials secret gcp-project-operator did not contain key billingaccount"),
			localObjects: []runtime.Object{
				newtestClusterDeploymentBuilder().getClusterDeployment(),
				newtestSecretBuilder(orgGcpSecretName, operatorNamespace, "testCreds").wihtoutKey("billingaccount").getTestSecret(),
			},
			setupGCPMock: func(r *mockGCP.MockClientMockRecorder) {
				gomock.InOrder(
					r.CreateProject(gomock.Any()).Return(
						&cloudresourcemanager.Operation{}, nil).Times(1))
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
			if !reflect.DeepEqual(err, test.expectedErr) {
				t.Errorf("%s: expected error: %v, got error: %v", test.name, test.expectedErr, err)
			}

		})
	}
}
