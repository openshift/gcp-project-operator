package structs

import (
	//"errors"
	//"github.com/stretchr/testify/assert"
	"testing"

	"github.com/golang/mock/gomock"
	mockGCP "github.com/openshift/gcp-project-operator/pkg/util/mocks/gcpclient"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakekubeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type mocks struct {
	FakeKubeClient client.Client
	MockCtrl       *gomock.Controller
	MockGCPClient  *mockGCP.MockClient
}

// setupDefaultMocks is an easy way to setup all of the default mocks
func SetupDefaultMocks(t *testing.T, localObjects []runtime.Object) *mocks {
	mockKubeClient := fakekubeclient.NewClientBuilder().WithRuntimeObjects(localObjects...).Build()
	mockCtrl := gomock.NewController(t)
	mockGCPClient := mockGCP.NewMockClient(mockCtrl)

	return &mocks{
		FakeKubeClient: mockKubeClient,
		MockCtrl:       mockCtrl,
		MockGCPClient:  mockGCPClient,
	}
}
