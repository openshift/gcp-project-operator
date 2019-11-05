package clusterdeployment

import (
	"context"
	"fmt"

	hivev1alpha1 "github.com/openshift/hive/pkg/apis/hive/v1alpha1"
	"google.golang.org/api/cloudresourcemanager/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubetypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	kubeclientpkg "sigs.k8s.io/controller-runtime/pkg/client"
)

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// remove removes a given index from a []string
func remove(slice []string, s int) []string {
	return append(slice[:s], slice[s+1:]...)
}

func findMemberIndex(searchMember string, members []string) int {
	for index, value := range members {
		if value == searchMember {
			return index
		}
	}
	return -1
}

// secretExists returns a boolean to the caller based on the secretName and namespace args.
func secretExists(kubeClient client.Client, secretName, namespace string) bool {
	s := &corev1.Secret{}

	err := kubeClient.Get(context.TODO(), kubetypes.NamespacedName{Name: secretName, Namespace: namespace}, s)
	if err != nil {
		return false
	}

	return true
}

// getSecret returns a secret based on a secretName and namespace.
func getSecret(kubeClient client.Client, secretName, namespace string) (*corev1.Secret, error) {
	s := &corev1.Secret{}

	err := kubeClient.Get(context.TODO(), kubetypes.NamespacedName{Name: secretName, Namespace: namespace}, s)

	if err != nil {
		return nil, err
	}
	return s, nil
}

// newGCPSecretCR returns a Secret CR formatted for GCP
func newGCPSecretCR(namespace, creds string) *corev1.Secret {
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

func getOrgGCPCreds(kubeClient kubeclientpkg.Client, namespace string) ([]byte, error) {
	secret := &corev1.Secret{}
	err := kubeClient.Get(context.TODO(),
		kubetypes.NamespacedName{
			Namespace: namespace,
			Name:      orgGcpSecretName,
		},
		secret)
	if err != nil {
		return []byte{}, fmt.Errorf("clusterdeployment.GetGCPClientFromSecret.Get %v", err)
	}

	osServiceAccountJson, ok := secret.Data[osServiceAccountKey]
	if !ok {
		return []byte{}, fmt.Errorf("GCP credentials secret %v did not contain key %v",
			orgGcpSecretName, osServiceAccountKey)
	}

	return osServiceAccountJson, nil
}

// checkDeploymentConfigRequirements checks that parameters required exist and that they are set correctly. If not it returns an error
func checkDeploymentConfigRequirements(cd *hivev1alpha1.ClusterDeployment) error {
	// Do not make do anything if the cluster is not a GCP cluster.
	val, ok := cd.Labels[clusterPlatformLabel]
	if !ok || val != clusterPlatformGCP {
		return ErrNotGCPCluster
	}

	// Do not do anything if the cluster is not a Red Hat managed cluster.
	val, ok = cd.Labels[clusterDeploymentManagedLabel]
	if !ok || val != "true" {
		return ErrNotManagedCluster
	}

	//Do not reconcile if cluster is installed or remove cleanup and remove project
	if cd.Spec.Installed {
		return ErrClusterInstalled
	}

	if cd.Spec.Platform.GCP.Region == "" {
		return ErrMissingRegion
	}

	if cd.Spec.Platform.GCP.ProjectID == "" {
		return ErrMissingProjectID
	}

	if _, ok := supportedRegions[cd.Spec.Platform.GCP.Region]; !ok {
		return ErrRegionNotSupported
	}

	return nil
}

// getOSDRequiredBindingMap returns a map of requiredBindings OSD role bindings for the added members
func getOSDRequiredBindingMap(roles []string, members string) map[string]cloudresourcemanager.Binding {
	requiredBindings := make(map[string]cloudresourcemanager.Binding)
	for _, role := range roles {
		requiredBindings[role] = cloudresourcemanager.Binding{
			Members: []string{"serviceAccount:" + members},
			Role:    role,
		}
	}
	return requiredBindings
}

// addOrUpdateBinding checks if a binding from a map of bindings whose keys are the binding.Role exists in a list and if so it appends any new members to that binding.
// If the required binding does not exist it creates a new binding for the role
// it returns a []*cloudresourcemanager.Binding that contains all the previous bindings and the new ones if no new bindings are required it returns false
func addOrUpdateBinding(existingBindings []*cloudresourcemanager.Binding, requiredBindings map[string]cloudresourcemanager.Binding) ([]*cloudresourcemanager.Binding, bool) {
	Modified := false

	for _, eBinding := range existingBindings {
		if rBinding, ok := requiredBindings[eBinding.Role]; ok {
			// check if members list contains from existing contains members from required
			for _, rMember := range rBinding.Members {
				if !stringInSlice(rMember, eBinding.Members) {
					Modified = true
					// If required member is not in existing member list add it
					eBinding.Members = append(eBinding.Members, rMember)
				}
			}
			// delete processed key from requiredBindings
			delete(requiredBindings, eBinding.Role)
		}

	}

	// take the remaining bindings from map of required bindings and append it to the list of existing bindings
	if len(requiredBindings) > 0 {
		Modified = true
		for _, binding := range requiredBindings {
			existingBindings = append(existingBindings, &binding)
		}
	}

	return existingBindings, Modified
}
