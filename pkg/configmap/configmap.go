package configmap

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	kubetypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GcpProjectOperatorConfigMap store informations required for operations
type GcpProjectOperatorConfigMap struct {
	name       string
	namespace  string
	kubeClient client.Client
}

// GetGcpProjectOperatorConfigMap returns a new GcpProjectOperatorConfigMap object
func GetGcpProjectOperatorConfigMap(kubeClient client.Client, name, namespace string) *GcpProjectOperatorConfigMap {
	config := GcpProjectOperatorConfigMap{
		name:       name,
		namespace:  namespace,
		kubeClient: kubeClient,
	}

	return &config
}

// getConfigMap returns a configmap
func (c *GcpProjectOperatorConfigMap) getConfigMap() (*corev1.ConfigMap, error) {
	cfg := &corev1.ConfigMap{}
	if err := c.kubeClient.Get(context.TODO(), kubetypes.NamespacedName{Name: c.name, Namespace: c.namespace}, cfg); err != nil {
		return &corev1.ConfigMap{}, err
	}

	return cfg, nil
}

// getValue returns value if the key exists in configmap
func (c *GcpProjectOperatorConfigMap) getValue(key string) (string, error) {
	configmap, err := c.getConfigMap()
	if err != nil {
		return "", fmt.Errorf("clusterdeployment.GetGCPParentFolderFromConfigMap.Get %v", err)
	}

	value, ok := configmap.Data[key]
	if !ok {
		return "", fmt.Errorf("GCP configmap %v did not contain key %v",
			c.name, key)
	}

	return value, nil
}

// GetParentFolder returns orgParentFolderID if the key exists in configmap
func (c *GcpProjectOperatorConfigMap) GetParentFolder() (string, error) {
	value, err := c.getValue("orgParentFolderID")
	return value, err
}

// GetBillingAccount returns billingaccount if the key exists in configmap
func (c *GcpProjectOperatorConfigMap) GetBillingAccount() (string, error) {
	value, err := c.getValue("billingaccount")
	return value, err
}
