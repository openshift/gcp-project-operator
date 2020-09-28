package configmap

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	builders "github.com/openshift/gcp-project-operator/pkg/util/mocks/structs"
)

func TestValidateOperatorConfigMap(t *testing.T) {
	sut := OperatorConfigMap{
		BillingAccount: "somethingdefined",
		ParentFolderID: "",
	}

	err := ValidateOperatorConfigMap(sut)
	// err expected since configmap didn't get filled properly
	assert.Error(t, err)

	sut.ParentFolderID = "1234567"
	if err = ValidateOperatorConfigMap(sut); err != nil {
		t.Errorf("no err expected since OperatorConfigMap filled properly")
	}

}

func TestGetOperatorConfigMap(t *testing.T) {
	tests := []struct {
		name                     string
		localObjects             []runtime.Object
		expectedParentFolderID   string
		expectedBillingAccount   string
		expectedCCSConsoleAccess []string
		expectedErr              error
		validateResult           func(*testing.T, string, string)
		validateErr              func(*testing.T, error, error)
	}{
		{
			name: "Correct parentFolderID and billingAccount exist in configmap",
			localObjects: []runtime.Object{
				&corev1.ConfigMap{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ConfigMap",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "gcp-project-operator",
						Namespace: "gcp-project-operator",
					},
					Data: map[string]string{
						OperatorConfigMapKey: `{parentFolderID: 1234567, billingAccount: "billing123"}`,
					},
				},
			},
			expectedParentFolderID: "1234567",
			expectedBillingAccount: "billing123",
			validateResult: func(t *testing.T, expected, result string) {
				assert.Equal(t, expected, result)
			},
		},
		{
			name:                   "configmap not found",
			localObjects:           []runtime.Object{},
			expectedParentFolderID: "",
			expectedErr:            errors.New("error"),
			validateResult: func(t *testing.T, expected, result string) {
				assert.Equal(t, expected, result)
			},
		},
		{
			name: "configmap is exist but not contains the right key",
			localObjects: []runtime.Object{
				&corev1.ConfigMap{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ConfigMap",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "gcp-project-operator",
						Namespace: "gcp-project-operator",
					},
					// the correct key should be data
					Data: map[string]string{
						"foo": "bar",
					},
				},
			},
			expectedParentFolderID: "",
			expectedErr:            errors.New("error"),
			validateResult: func(t *testing.T, expected, result string) {
				assert.Equal(t, expected, result)
			},
		},
		{
			name: "configmap is exist but not contains parentFolderID",
			localObjects: func() []runtime.Object {
				sec := &corev1.ConfigMap{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ConfigMap",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "gcp-project-operator",
						Namespace: "gcp-project-operator",
					},
					Data: map[string]string{
						OperatorConfigMapKey: `{billingAccount: foo}`,
					},
				}
				return []runtime.Object{sec}
			}(),
			expectedParentFolderID: "",
			expectedBillingAccount: "foo",
			expectedErr:            nil,
			validateResult: func(t *testing.T, expected, result string) {
				assert.Equal(t, expected, result)
			},
			validateErr: func(t *testing.T, expected, result error) {
				assert.Equal(t, expected, result)
			},
		},
		{
			name: "ccsConsoleAccess configured",
			localObjects: func() []runtime.Object {
				sec := &corev1.ConfigMap{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ConfigMap",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "gcp-project-operator",
						Namespace: "gcp-project-operator",
					},
					Data: map[string]string{
						OperatorConfigMapKey: `{parentFolderID: 1234567,billingAccount: "billing123",ccsConsoleAccess: [foo, bar]}`,
					},
				}
				return []runtime.Object{sec}
			}(),
			expectedParentFolderID:   "1234567",
			expectedBillingAccount:   "billing123",
			expectedCCSConsoleAccess: []string{"foo", "bar"},
			expectedErr:              nil,
			validateResult: func(t *testing.T, expected, result string) {
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

			operatorConfigMap, err := GetOperatorConfigMap(mocks.FakeKubeClient)

			if test.expectedErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if test.validateResult != nil {
				test.validateResult(t, test.expectedParentFolderID, operatorConfigMap.ParentFolderID)
				test.validateResult(t, test.expectedBillingAccount, operatorConfigMap.BillingAccount)
			}

			assert.Equal(t, test.expectedCCSConsoleAccess, operatorConfigMap.CCSConsoleAccess)

			if test.validateErr != nil {
				test.validateErr(t, test.expectedErr, err)
			}

		})
	}
}
