package configmap

import (
	"context"
	"fmt"
	"reflect"

	"github.com/mitchellh/mapstructure"
	corev1 "k8s.io/api/core/v1"
	kubetypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// OperatorConfigMapName holds the name of configmap
const (
	OperatorConfigMapName      = "gcp-project-operator"
	operatorConfigMapNamespace = "gcp-project-operator"
)

// OperatorConfigMap store data for the specified configmap
type OperatorConfigMap struct {
	BillingAccount string `mapstructure:"billingAccount"`
	ParentFolderID string `mapstructure:"parentFolderID"`
}

// ValidateOperatorConfigMap checks if OperatorConfigMap filled properly
func ValidateOperatorConfigMap(configmap OperatorConfigMap) error {
	v := reflect.ValueOf(configmap)
	typeOfS := v.Type()

	for i := 0; i < v.NumField(); i++ {
		optional, _ := typeOfS.Field(i).Tag.Lookup("optional")
		if v.Field(i).Interface() == "" && optional != "true" {
			return fmt.Errorf("missing configmap key: %s", typeOfS.Field(i).Name)
		}
	}

	return nil
}

// GetOperatorConfigMap returns a configmap defined in requested namespace and name
func GetOperatorConfigMap(kubeClient client.Client) (OperatorConfigMap, error) {
	var OperatorConfigMap OperatorConfigMap
	configmap := &corev1.ConfigMap{}
	if err := kubeClient.Get(context.TODO(), kubetypes.NamespacedName{Name: OperatorConfigMapName, Namespace: operatorConfigMapNamespace}, configmap); err != nil {
		return OperatorConfigMap, fmt.Errorf("unable to get configmap: %v", err)
	}

	if err := mapstructure.Decode(configmap.Data, &OperatorConfigMap); err != nil {
		return OperatorConfigMap, fmt.Errorf("unable to unmarshal configmap: %v", err)
	}

	return OperatorConfigMap, nil
}
