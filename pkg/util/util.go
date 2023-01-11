package util

import (
	"context"
	"fmt"
	"github.com/openshift/gcp-project-operator/pkg/util/errors"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubetypes "k8s.io/apimachinery/pkg/types"
)

// IamMemberType represents different type of IAM members.
type IamMemberType int

const (
	ServiceAccount IamMemberType = iota
	GoogleGroup
)

// SecretExists returns a boolean to the caller based on the secretName and namespace args.
func SecretExists(kubeClient client.Client, secretName, namespace string) bool {
	s := &corev1.Secret{}

	err := kubeClient.Get(context.TODO(), kubetypes.NamespacedName{Name: secretName, Namespace: namespace}, s)
	return err == nil
}

// GetSecret returns a secret based on a secretName and namespace.
func GetSecret(kubeClient client.Client, secretName, namespace string) (*corev1.Secret, error) {
	s := &corev1.Secret{}

	err := kubeClient.Get(context.TODO(), kubetypes.NamespacedName{Name: secretName, Namespace: namespace}, s)

	if err != nil {
		return &corev1.Secret{}, err
	}
	return s, nil
}

// NewGCPSecretCR returns a Secret CR formatted for GCP for use in projectreference controller.
func NewGCPSecretCR(creds string, namespacedNamed kubetypes.NamespacedName) *corev1.Secret {
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

// GetGCPCredentialsFromSecret extracts the gcp credentials from a secret. return value is a bytearray
func GetGCPCredentialsFromSecret(kubeClient client.Client, namespace, name string) ([]byte, error) {
	secret := &corev1.Secret{}
	err := kubeClient.Get(context.TODO(),
		kubetypes.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
		secret)
	if err != nil {
		return []byte{}, fmt.Errorf("GetGCPCredentialsFromSecret.Get %v", err)
	}
	var osServiceAccountJSON []byte
	var ok bool
	osServiceAccountJSON, ok = secret.Data["osServiceAccount.json"]
	if !ok {
		osServiceAccountJSON, ok = secret.Data["key.json"]
	}
	if !ok {
		return []byte{}, fmt.Errorf("GCP credentials secret %v did not contain key %v",
			name, "{osServiceAccount,key}.json")
	}

	return osServiceAccountJSON, nil
}

func RemoveOrUpdateBinding(existingBindings []*cloudresourcemanager.Binding, serviceAccountEmail string, memberType IamMemberType) ([]*cloudresourcemanager.Binding, bool) {
	prefix := "serviceAccount:"
	if memberType == GoogleGroup {
		prefix = "group:"
	}
	modified := false
	memberToRemove := prefix + serviceAccountEmail
	for i, binding := range existingBindings {
		for index, v := range binding.Members {
			if v == memberToRemove {
				// removing member from policy binding
				newMembers := append(binding.Members[:index], binding.Members[index+1:]...)
				existingBindings[i].Members = newMembers
				modified = true
				break
			}
		}
	}
	return existingBindings, modified
}

// AddOrUpdateBinding checks if a binding from a map of bindings whose keys are the binding.Role exists in a list and if so it appends any new members to that binding.
// If the required binding does not exist it creates a new binding for the role
// it returns a []*cloudresourcemanager.Binding that contains all the previous bindings and the new ones if no new bindings are required it returns false
// TODO(MJ): add tests
func AddOrUpdateBinding(existingBindings []*cloudresourcemanager.Binding, requiredBindings []string, serviceAccount string, memberType IamMemberType) ([]*cloudresourcemanager.Binding, bool) {
	Modified := false
	// get map of required rolebindings
	requiredBindingMap := rolebindingMap(requiredBindings, serviceAccount, memberType)
	var result []*cloudresourcemanager.Binding

	for i, eBinding := range existingBindings {
		result = append(result, &cloudresourcemanager.Binding{
			Members: eBinding.Members,
			Role:    eBinding.Role,
		})
		if rBinding, ok := requiredBindingMap[eBinding.Role]; ok {
			// check if members list contains from existing contains members from required
			for _, rMember := range rBinding.Members {
				if exist := Contains(eBinding.Members, rMember); !exist {
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
func rolebindingMap(roles []string, member string, memberType IamMemberType) map[string]cloudresourcemanager.Binding {
	prefix := "serviceAccount:"
	if memberType == GoogleGroup {
		prefix = "group:"
	}
	requiredBindings := make(map[string]cloudresourcemanager.Binding)
	for _, role := range roles {
		requiredBindings[role] = cloudresourcemanager.Binding{
			Members: []string{prefix + member},
			Role:    role,
		}
	}
	return requiredBindings
}

// ValidateServiceAccountKey performs validation that a service account's keys can list at least 1 project to mirror the
// check that the installer does:
// https://github.com/openshift/installer/blob/ce13271664f492088048a2e90f208b8b51ea88ca/pkg/asset/installconfig/gcp/validation.go#L234-L248
//
// This check is necessary because GCP IAM is eventually consistent and in unfortunate times, eventually consistent
// after a few hours.
func ValidateServiceAccountKey(ctx context.Context, key *iam.ServiceAccountKey) error {
	creds, err := google.CredentialsFromJSON(ctx, []byte(key.PrivateKeyData), "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return fmt.Errorf("could not parse service account credentials: %w", err)
	}

	client, err := cloudresourcemanager.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		return fmt.Errorf("could not create cloudresourcemanager client with service account credentials: %w", err)
	}

	resp, err := client.Projects.List().Do()
	if err != nil {
		return fmt.Errorf("failed to list projects with service account credentials: %w", err)
	}

	if len(resp.Projects) == 0 {
		return errors.New("can view 0 projects with these service account credentials")
	}

	return nil
}

// Contains returns true if a list contains a string.
func Contains(list []string, strToSearch string) bool {
	for _, item := range list {
		if item == strToSearch {
			return true
		}
	}
	return false
}

// Filter filters a list for a string.
func Filter(list []string, strToFilter string) (newList []string) {
	for _, item := range list {
		if item != strToFilter {
			newList = append(newList, item)
		}
	}
	return
}
