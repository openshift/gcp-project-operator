package util

import (
	"fmt"
	"testing"

	"github.com/openshift/gcp-project-operator/pkg/util/errors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/cloudresourcemanager/v1"
	"k8s.io/apimachinery/pkg/runtime"

	builders "github.com/openshift/gcp-project-operator/pkg/util/mocks/structs"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
				builders.NewTestSecretBuilder("testName", "testNamespace", "testCreds").GetTestSecret(),
			},
		},
		{
			name:            "Secret does not exist",
			expectedResult:  false,
			secretName:      "badName",
			secretNamespace: "testNamespace",
			localObjects: []runtime.Object{
				builders.NewTestSecretBuilder("testName", "testNamespace", "testCreds").GetTestSecret(),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mocks := builders.SetupDefaultMocks(t, test.localObjects)

			result := SecretExists(mocks.FakeKubeClient, test.secretName, test.secretNamespace)
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
				builders.NewTestSecretBuilder("testName", "testNamespace", "testCreds").GetTestSecret(),
			},
			expectedSecret: builders.NewTestSecretBuilder("testName", "testNamespace", "testCreds").GetTestSecret(),
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
				builders.NewTestSecretBuilder("testName", "testNamespace", "testCreds").GetTestSecret(),
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
			mocks := builders.SetupDefaultMocks(t, test.localObjects)

			result, err := GetSecret(mocks.FakeKubeClient, test.secretName, test.secretNamespace)

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

func TestGetGCPCredentialsFromSecret(t *testing.T) {
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
				builders.NewTestSecretBuilder("testCreds", "testNamespace", "testCredsContent").GetTestSecret(),
			},
			secretNamespace: "testNamespace",
			expectedCreds:   []byte("testCredsContent"),
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
						Name:      "badName",
						Namespace: "testNamespace",
					},
				}
				return []runtime.Object{sec}
			}(),
			secretNamespace: "testNamespace",
			expectedCreds:   []byte{},
			expectedErr:     fmt.Errorf("GetGCPCredentialsFromSecret.Get secrets \"%v\" not found", "testCreds"),
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
			mocks := builders.SetupDefaultMocks(t, test.localObjects)

			result, err := GetGCPCredentialsFromSecret(mocks.FakeKubeClient, test.secretNamespace, "testCreds")

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

func TestRemoveOrUpdateBinding(t *testing.T) {
	tests := []struct {
		name               string
		UserToDelete       string
		UserType           IamMemberType
		inputBindings      []*cloudresourcemanager.Binding
		expectedBindings   []*cloudresourcemanager.Binding
		expectModification bool
	}{
		{
			name:         "User Group Binding Exists",
			UserToDelete: "CCSConsoleAccess",
			UserType:     GoogleGroup,
			inputBindings: []*cloudresourcemanager.Binding{
				{Role: "role/admin", Members: []string{"group:CCSConsoleAccess", "serviceAccount:customerAcc"}},
				{Role: "role/viewer", Members: []string{"group:CCSReadOnlyConsoleAccess", "serviceAccount:customerAcc"}},
				{Role: "role/dev", Members: []string{"group:customerGroup"}},
			},
			expectedBindings: []*cloudresourcemanager.Binding{
				{Role: "role/admin", Members: []string{"serviceAccount:customerAcc"}},
				{Role: "role/viewer", Members: []string{"group:CCSReadOnlyConsoleAccess", "serviceAccount:customerAcc"}},
				{Role: "role/dev", Members: []string{"group:customerGroup"}},
			},
			expectModification: true,
		},
		{
			name:         "User Serviceaccount Binding Exists",
			UserToDelete: "CCSReadOnlyConsoleAccess",
			UserType:     ServiceAccount,
			inputBindings: []*cloudresourcemanager.Binding{
				{Role: "role/admin", Members: []string{"group:CCSConsoleAccess", "serviceAccount:customerAcc"}},
				{Role: "role/viewer", Members: []string{"serviceAccount:CCSReadOnlyConsoleAccess", "serviceAccount:customerAcc"}},
				{Role: "role/dev", Members: []string{"serviceAccount:customerGroup"}},
			},
			expectedBindings: []*cloudresourcemanager.Binding{
				{Role: "role/admin", Members: []string{"group:CCSConsoleAccess", "serviceAccount:customerAcc"}},
				{Role: "role/viewer", Members: []string{"serviceAccount:customerAcc"}},
				{Role: "role/dev", Members: []string{"serviceAccount:customerGroup"}},
			},
			expectModification: true,
		},
		{
			name:         "User Binding doesn't Exists",
			UserToDelete: "CCSConsoleAccess",
			UserType:     GoogleGroup,
			inputBindings: []*cloudresourcemanager.Binding{
				{Role: "role/admin", Members: []string{"serviceAccount:customerAcc"}},
				{Role: "role/dev", Members: []string{"group:customerGroup"}},
			},
			expectedBindings: []*cloudresourcemanager.Binding{
				{Role: "role/admin", Members: []string{"serviceAccount:customerAcc"}},
				{Role: "role/dev", Members: []string{"group:customerGroup"}},
			},
			expectModification: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, modified := RemoveOrUpdateBinding(test.inputBindings, test.UserToDelete, test.UserType)
			assert.Equal(t, modified, test.expectModification)
			assert.Equal(t, result, test.expectedBindings)
		})
	}

}
