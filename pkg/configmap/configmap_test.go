package configmap

import (
	"errors"
	"fmt"
	"testing"

	builders "github.com/openshift/gcp-project-operator/pkg/util/mocks/structs"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestGetConfigMap(t *testing.T) {
	tests := []struct {
		name               string
		localObjects       []runtime.Object
		ConfigMap          string
		ConfigMapNamespace string
		expectedConfigMap  *corev1.ConfigMap
		expectedErr        bool
		validateResult     func(*testing.T, *corev1.ConfigMap, *corev1.ConfigMap)
	}{
		{
			name:               "Existing ConfigMap",
			ConfigMap:          "testName",
			ConfigMapNamespace: "testNamespace",
			localObjects: []runtime.Object{
				builders.NewTestConfigMapBuilder("testName", "testNamespace", "foo", "111111").GetConfigMap(),
			},
			expectedConfigMap: builders.NewTestConfigMapBuilder("testName", "testNamespace", "foo", "111111").GetConfigMap(),
			expectedErr:       false,
			validateResult: func(t *testing.T, expected, result *corev1.ConfigMap) {
				assert.Equal(t, expected, result)
			},
		},
		{
			name:               "ConfigMap does not exist",
			ConfigMap:          "badName",
			ConfigMapNamespace: "testNamespace",
			localObjects: []runtime.Object{
				builders.NewTestConfigMapBuilder("testName", "testNamespace", "foo", "111111").GetConfigMap(),
			},
			expectedConfigMap: &corev1.ConfigMap{},
			expectedErr:       true,
			validateResult: func(t *testing.T, expected, result *corev1.ConfigMap) {
				assert.Equal(t, expected, result)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mocks := builders.SetupDefaultMocks(t, test.localObjects)

			configmap := GetGcpProjectOperatorConfigMap(mocks.FakeKubeClient, test.ConfigMap, test.ConfigMapNamespace)
			result, err := configmap.getConfigMap()

			if test.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if test.validateResult != nil {
				test.validateResult(t, test.expectedConfigMap, result)
			}
		})
	}

}

func TestGetParentFolder(t *testing.T) {
	tests := []struct {
		name                      string
		ConfigMap                 string
		localObjects              []runtime.Object
		ConfigMapNamespace        string
		expectedorgParentFolderID string
		expectedErr               error
		validateResult            func(*testing.T, string, string)
		validateErr               func(*testing.T, error, error)
	}{
		{
			name:      "Correct orgParentFolderID",
			ConfigMap: "test",
			localObjects: []runtime.Object{
				builders.NewTestConfigMapBuilder("test", "testNamespace", "foo", "1234567").GetConfigMap(),
			},
			ConfigMapNamespace:        "testNamespace",
			expectedorgParentFolderID: "1234567",
			validateResult: func(t *testing.T, expected, result string) {
				assert.Equal(t, expected, result)
			},
		},
		{
			name:                      "orgParentFolderID not found",
			ConfigMap:                 "test",
			localObjects:              []runtime.Object{},
			ConfigMapNamespace:        "testNamespace",
			expectedorgParentFolderID: "",
			expectedErr:               errors.New("error"),
			validateResult: func(t *testing.T, expected, result string) {
				assert.Equal(t, expected, result)
			},
		},
		{
			name:      "Bad data in ConfigMap",
			ConfigMap: "test",
			localObjects: func() []runtime.Object {
				sec := &corev1.ConfigMap{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ConfigMap",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "testNamespace",
					},
				}
				return []runtime.Object{sec}
			}(),
			ConfigMapNamespace:        "testNamespace",
			expectedorgParentFolderID: "",
			expectedErr:               fmt.Errorf("GCP configmap test did not contain key orgParentFolderID"),
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

			configmap := GetGcpProjectOperatorConfigMap(mocks.FakeKubeClient, test.ConfigMap, test.ConfigMapNamespace)
			result, err := configmap.GetParentFolder()

			if test.expectedErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if test.validateResult != nil {
				test.validateResult(t, test.expectedorgParentFolderID, result)
			}

			if test.validateErr != nil {
				test.validateErr(t, test.expectedErr, err)
			}

		})
	}
}

func TestGetBillingAccount(t *testing.T) {
	tests := []struct {
		name                      string
		ConfigMap                 string
		localObjects              []runtime.Object
		ConfigMapNamespace        string
		expectedGetBillingAccount string
		expectedErr               error
		validateResult            func(*testing.T, string, string)
		validateErr               func(*testing.T, error, error)
	}{
		{
			name:      "Correct billingaccount",
			ConfigMap: "test",
			localObjects: []runtime.Object{
				builders.NewTestConfigMapBuilder("test", "testNamespace", "foo", "1234567").GetConfigMap(),
			},
			ConfigMapNamespace:        "testNamespace",
			expectedGetBillingAccount: "foo",
			validateResult: func(t *testing.T, expected, result string) {
				assert.Equal(t, expected, result)
			},
		},
		{
			name:                      "billingaccount not found",
			ConfigMap:                 "test",
			localObjects:              []runtime.Object{},
			ConfigMapNamespace:        "testNamespace",
			expectedGetBillingAccount: "",
			expectedErr:               errors.New("error"),
			validateResult: func(t *testing.T, expected, result string) {
				assert.Equal(t, expected, result)
			},
		},
		{
			name:      "Bad data in ConfigMap",
			ConfigMap: "test",
			localObjects: func() []runtime.Object {
				sec := &corev1.ConfigMap{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ConfigMap",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "testNamespace",
					},
				}
				return []runtime.Object{sec}
			}(),
			ConfigMapNamespace:        "testNamespace",
			expectedGetBillingAccount: "",
			expectedErr:               fmt.Errorf("GCP configmap test did not contain key billingaccount"),
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

			configmap := GetGcpProjectOperatorConfigMap(mocks.FakeKubeClient, test.ConfigMap, test.ConfigMapNamespace)
			result, err := configmap.GetBillingAccount()

			if test.expectedErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if test.validateResult != nil {
				test.validateResult(t, test.expectedGetBillingAccount, result)
			}

			if test.validateErr != nil {
				test.validateErr(t, test.expectedErr, err)
			}

		})
	}
}
