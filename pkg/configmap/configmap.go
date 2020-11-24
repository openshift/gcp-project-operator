package configmap

import (
	"context"
	"fmt"

	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	kubetypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// OperatorConfigMapName holds the name of configmap
const (
	OperatorConfigMapName      = "gcp-project-operator"
	OperatorConfigMapNamespace = "gcp-project-operator"
	OperatorConfigMapKey       = "config.yaml"
)

// OperatorConfigMap store data for the specified configmap
type OperatorConfigMap struct {
	BillingAccount           string   `yaml:"billingAccount"`
	ParentFolderID           string   `yaml:"parentFolderID"`
	CCSConsoleAccess         []string `yaml:"ccsConsoleAccess,omitempty"`
	CCSReadOnlyConsoleAccess []string `yaml:"ccsReadOnlyConsoleAccess,omitempty"`
	DisabledRegions          []string `yaml:"disabledRegions,omitempty"`
}

// ValidateOperatorConfigMap checks if OperatorConfigMap filled properly
func ValidateOperatorConfigMap(configmap OperatorConfigMap) error {
	if configmap.BillingAccount == "" {
		return fmt.Errorf("missing configmap key: billingAccount")
	}

	if configmap.ParentFolderID == "" {
		return fmt.Errorf("missing configmap key: parentFolderID")
	}

	return nil
}

// GetOperatorConfigMap returns a configmap defined in requested namespace and name
func GetOperatorConfigMap(kubeClient client.Client) (OperatorConfigMap, error) {
	var operatorConfigMap OperatorConfigMap
	configmap := &corev1.ConfigMap{}
	if err := kubeClient.Get(context.TODO(), kubetypes.NamespacedName{Name: OperatorConfigMapName, Namespace: OperatorConfigMapNamespace}, configmap); err != nil {
		return operatorConfigMap, fmt.Errorf("unable to get configmap: %v", err)
	}

	if data, ok := configmap.Data[OperatorConfigMapKey]; !ok {
		return operatorConfigMap, fmt.Errorf("unable to get config from key %s", OperatorConfigMapKey)
	} else {
		if err := yaml.Unmarshal([]byte(data), &operatorConfigMap); err != nil {
			return operatorConfigMap, err
		}
	}

	return operatorConfigMap, nil
}
