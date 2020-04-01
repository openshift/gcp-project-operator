package clusterdeployment

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/openshift/gcp-project-operator/pkg/gcpclient"
	mockGCP "github.com/openshift/gcp-project-operator/pkg/util/mocks/gcpclient"
	builders "github.com/openshift/gcp-project-operator/pkg/util/mocks/structs"
	hiveapis "github.com/openshift/hive/pkg/apis"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	testClusterName   = "clusterName"
	testNamespace     = "namespace"
	testConfigMapName = "gcp-project-operator"
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
			localObjects: []runtime.Object{
				builders.NewTestConfigMapBuilder(testConfigMapName, operatorNamespace, "foo", "111111").GetConfigMap(),
			},
		},
		{
			name:        "CD check fail ErrMissingRegion",
			expectedErr: fmt.Errorf("MissingRegion"),
			localObjects: []runtime.Object{
				builders.NewTestClusterDeploymentBuilder().WithOutRegion().GetClusterDeployment(),
				builders.NewTestConfigMapBuilder(testConfigMapName, operatorNamespace, "foo", "111111").GetConfigMap(),
			},
			setupGCPMock: func(r *mockGCP.MockClientMockRecorder) { gomock.Any() },
		},
		{
			name:        "CD check fail ErrClusterInstalled",
			expectedErr: nil,
			localObjects: []runtime.Object{
				builders.NewTestClusterDeploymentBuilder().Installed().GetClusterDeployment(),
				builders.NewTestConfigMapBuilder(testConfigMapName, operatorNamespace, "foo", "111111").GetConfigMap(),
			},
			setupGCPMock: func(r *mockGCP.MockClientMockRecorder) { gomock.Any() },
		},
		{
			name:        "failed to get ORG creds",
			expectedErr: fmt.Errorf("clusterdeployment.getGCPCredentialsFromSecret.Get secrets \"gcp-project-operator-credentials\" not found"),
			localObjects: []runtime.Object{
				builders.NewTestConfigMapBuilder(testConfigMapName, operatorNamespace, "foo", "111111").GetConfigMap(),
				builders.NewTestClusterDeploymentBuilder().GetClusterDeployment(),
			},
			setupGCPMock: func(r *mockGCP.MockClientMockRecorder) { gomock.Any() },
		},
		{
			name:        "final secret exists",
			expectedErr: nil,
			localObjects: []runtime.Object{
				builders.NewTestClusterDeploymentBuilder().GetClusterDeployment(),
				builders.NewTestConfigMapBuilder(testConfigMapName, operatorNamespace, "foo", "111111").GetConfigMap(),
				builders.NewTestSecretBuilder(orgGcpSecretName, operatorNamespace, "testCreds").GetTestSecret(),
				builders.NewTestSecretBuilder(gcpSecretName, testNamespace, "testCreds").GetTestSecret(),
			},
			setupGCPMock: func(r *mockGCP.MockClientMockRecorder) { gomock.Any() },
		},
		{
			name:        "no ParentFolderID in configmap",
			expectedErr: fmt.Errorf("missing configmap key: ParentFolderID"),
			localObjects: []runtime.Object{
				builders.NewTestClusterDeploymentBuilder().GetClusterDeployment(),
				builders.NewTestSecretBuilder(orgGcpSecretName, operatorNamespace, "testCreds").GetTestSecret(),
				builders.NewTestConfigMapBuilder(testConfigMapName, operatorNamespace, "foo", "111111").WithoutKey("parentFolderID").GetConfigMap(),
			},
			setupGCPMock: func(r *mockGCP.MockClientMockRecorder) { gomock.Any() },
		},
		{
			name:        "no billing account in configmap",
			expectedErr: fmt.Errorf("missing configmap key: BillingAccount"),
			localObjects: []runtime.Object{
				builders.NewTestClusterDeploymentBuilder().GetClusterDeployment(),
				builders.NewTestSecretBuilder(orgGcpSecretName, operatorNamespace, "testCreds").GetTestSecret(),
				builders.NewTestConfigMapBuilder(testConfigMapName, operatorNamespace, "foo", "111111").WithoutKey("billingAccount").GetConfigMap(),
			},
			setupGCPMock: func(r *mockGCP.MockClientMockRecorder) { gomock.Any() },
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Arrage
			mocks := builders.SetupDefaultMocks(t, test.localObjects)
			test.setupGCPMock(mocks.MockGCPClient.EXPECT())

			gcpBuilder := func(projectName string, authJSON []byte) (gcpclient.Client, error) {
				return mocks.MockGCPClient, nil
			}

			// This is necessary for the mocks to report failures like methods not being called an expected number of times.
			// after mocks is defined
			defer mocks.MockCtrl.Finish()

			rcd := &ReconcileClusterDeployment{
				mocks.FakeKubeClient,
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
