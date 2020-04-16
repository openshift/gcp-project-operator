package util

import (
	"fmt"
	"testing"

	gcpv1alpha1 "github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1"
	"github.com/openshift/gcp-project-operator/pkg/util/errors"
	builders "github.com/openshift/gcp-project-operator/pkg/util/mocks/structs"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

func TestSetCondition(t *testing.T) {
	sut := &[]gcpv1alpha1.Condition{}
	status := corev1.ConditionTrue
	reason := "dummy reconcile"
	message := "fake error"

	err := SetCondition(sut, status, reason, message)
	if err != nil {
		t.Errorf("no expected err, got %v", err)
	}

	if len(*sut) != 1 {
		t.Errorf("item count should be 1, got item: %v", len(*sut))
	}

	obj := (*sut)[0]
	if obj.Status != status {
		t.Errorf("expected status: %v, got %v", status, obj.Status)
	}
	if obj.Reason != reason {
		t.Errorf("expected reason: %v, got %v", reason, obj.Reason)
	}
	if obj.Message != message {
		t.Errorf("expected message: %v, got %v", message, obj.Message)
	}

	probe := obj.LastProbeTime
	transition := obj.LastTransitionTime

	err = SetCondition(sut, status, reason, message)
	// get new updated obj
	obj = (*sut)[0]
	if err != nil {
		t.Errorf("no expected err, got %v", err)
	}

	if obj.LastProbeTime == probe {
		t.Errorf("expected %v should not equal to %v", probe, obj.LastProbeTime)
	}

	if obj.LastTransitionTime != transition {
		t.Errorf("transition time shouldn't be changed, expected %v, got %v", transition, obj.LastTransitionTime)
	}
}
