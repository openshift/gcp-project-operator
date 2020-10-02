package structs

import (
	"fmt"

	api "github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

type SecretMatcher struct {
	ActualSecret *corev1.Secret
	FailReason   string
}

func NewSecretMatcher() *SecretMatcher {
	return &SecretMatcher{&corev1.Secret{}, ""}
}

func (m *SecretMatcher) Matches(x interface{}) bool {
	ref, isCorrectType := x.(*corev1.Secret)
	if !isCorrectType {
		m.FailReason = fmt.Sprintf("Unexpected type passed: want '%T', got '%T'", api.ProjectClaim{}, x)
		return false
	}
	m.ActualSecret = ref
	return true
}

func (m *SecretMatcher) String() string {
	return "Fail reason: " + m.FailReason
}
