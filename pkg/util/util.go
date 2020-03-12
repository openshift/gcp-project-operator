package util

import (
	"context"
	"fmt"
	"reflect"

	"google.golang.org/api/cloudresourcemanager/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubetypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	kubeclientpkg "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// secret information
	gcpSecretName = "gcp"
)

// SecretExists returns a boolean to the caller based on the secretName and namespace args.
func SecretExists(kubeClient client.Client, secretName, namespace string) bool {
	s := &corev1.Secret{}

	err := kubeClient.Get(context.TODO(), kubetypes.NamespacedName{Name: secretName, Namespace: namespace}, s)
	return err == nil
}

// getSecret returns a secret based on a secretName and namespace.
func getSecret(kubeClient client.Client, secretName, namespace string) (*corev1.Secret, error) {
	s := &corev1.Secret{}

	err := kubeClient.Get(context.TODO(), kubetypes.NamespacedName{Name: secretName, Namespace: namespace}, s)

	if err != nil {
		return &corev1.Secret{}, err
	}
	return s, nil
}

// NewGCPSecretCR returns a Secret CR formatted for GCP
// To be removed along with ClusterDeployment controller
func NewGCPSecretCR(namespace, creds string) *corev1.Secret {
	return &corev1.Secret{
		Type: "Opaque",
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      gcpSecretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"osServiceAccount.json": []byte(creds),
		},
	}
}

// NewGCPSecretCRV2 returns a Secret CR formatted for GCP for use in projectreference controller.
func NewGCPSecretCRV2(creds string, namespacedNamed kubetypes.NamespacedName) *corev1.Secret {
	return &corev1.Secret{
		Type: "Opaque",
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespacedNamed.Name,
			Namespace: namespacedNamed.Namespace,
		},
		Data: map[string][]byte{
			"osServiceAccount.json": []byte(creds),
		},
	}
}

func GetGCPCredentialsFromSecret(kubeClient kubeclientpkg.Client, namespace, name string) ([]byte, error) {
	secret := &corev1.Secret{}
	err := kubeClient.Get(context.TODO(),
		kubetypes.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
		secret)
	if err != nil {
		return []byte{}, fmt.Errorf("clusterdeployment.getGCPCredentialsFromSecret.Get %v", err)
	}
	var osServiceAccountJson []byte
	var ok bool
	osServiceAccountJson, ok = secret.Data["osServiceAccount.json"]
	if !ok {
		osServiceAccountJson, ok = secret.Data["key.json"]
	}
	if !ok {
		return []byte{}, fmt.Errorf("GCP credentials secret %v did not contain key %v",
			name, "{osServiceAccount,key}.json")
	}

	return osServiceAccountJson, nil
}

// AddOrUpdateBinding checks if a binding from a map of bindings whose keys are the binding.Role exists in a list and if so it appends any new members to that binding.
// If the required binding does not exist it creates a new binding for the role
// it returns a []*cloudresourcemanager.Binding that contains all the previous bindings and the new ones if no new bindings are required it returns false
// TODO(MJ): add tests
func AddOrUpdateBinding(existingBindings []*cloudresourcemanager.Binding, requiredBindings []string, serviceAccount string) ([]*cloudresourcemanager.Binding, bool) {
	Modified := false
	// get map of required rolebindings
	requiredBindingMap := rolebindingMap(requiredBindings, serviceAccount)
	var result []*cloudresourcemanager.Binding

	for i, eBinding := range existingBindings {
		if rBinding, ok := requiredBindingMap[eBinding.Role]; ok {
			result = append(result, &cloudresourcemanager.Binding{
				Members: eBinding.Members,
				Role:    eBinding.Role,
			})
			// check if members list contains from existing contains members from required
			for _, rMember := range rBinding.Members {
				exist, _ := InArray(rMember, eBinding.Members)
				if !exist {
					Modified = true
					// If required member is not in existing member list add it
					result[i].Members = append(result[i].Members, rMember)
				}
			}
			// delete processed key from requiredBindings
			delete(requiredBindingMap, eBinding.Role)
		}
	}

	if len(requiredBindingMap) > 0 {
		Modified = true
		for _, binding := range requiredBindingMap {
			result = append(result, &cloudresourcemanager.Binding{
				Members: binding.Members,
				Role:    binding.Role,
			})
		}
	}
	return result, Modified
}

// roleBindingMap returns a map of requiredBindings role bindings for the added members
func rolebindingMap(roles []string, member string) map[string]cloudresourcemanager.Binding {
	requiredBindings := make(map[string]cloudresourcemanager.Binding)
	for _, role := range roles {
		requiredBindings[role] = cloudresourcemanager.Binding{
			Members: []string{"serviceAccount:" + member},
			Role:    role,
		}
	}
	return requiredBindings
}

func InArray(needle interface{}, haystack interface{}) (exists bool, index int) {
	exists = false
	index = -1

	switch reflect.TypeOf(haystack).Kind() {
	case reflect.Slice:
		s := reflect.ValueOf(haystack)

		for i := 0; i < s.Len(); i++ {
			if reflect.DeepEqual(needle, s.Index(i).Interface()) {
				index = i
				exists = true
				return
			}
		}
	}

	return
}
