package configmap

import (
	"errors"
	"testing"

	builders "github.com/openshift/gcp-project-operator/pkg/util/mocks/structs"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestCheckValueNotExist(t *testing.T) {
	sut := OperatorConfigMap{
		BillingAccount: "somethingdefined",
		ParentFolderID: "",
	}

	err := CheckValueNotExist(sut)
	if err != nil {
		assert.Error(t, err)
	}

	sut.ParentFolderID = "1234567"
	if err = CheckValueNotExist(sut); err != nil {
		t.Errorf("no err expected since OperatorConfigMap filled properly")
	}
}

func TestGetOperatorConfigMap(t *testing.T) {
	tests := []struct {
		name                   string
		localObjects           []runtime.Object
		expectedParentFolderID string
		expectedBillingAccount string
		expectedErr            error
		validateResult         func(*testing.T, string, string)
		validateErr            func(*testing.T, error, error)
	}{
		{
			name: "Correct parentFolderID and billingAccount exist in configmap",
			localObjects: []runtime.Object{
				builders.NewTestConfigMapBuilder("gcp-project-operator", "gcp-project-operator", "billing123", "1234567").GetConfigMap(),
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
				}
				return []runtime.Object{sec}
			}(),
			expectedParentFolderID: "",
			expectedErr:            nil,
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

			if test.validateErr != nil {
				test.validateErr(t, test.expectedErr, err)
			}

		})
	}
}
