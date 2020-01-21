package structs

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type testSecretBuilder struct {
	s corev1.Secret
}

func (t *testSecretBuilder) GetTestSecret() *corev1.Secret {
	return &t.s
}

func NewTestSecretBuilder(secretName, namespace, creds string) *testSecretBuilder {
	return &testSecretBuilder{
		s: corev1.Secret{
			Type: "Opaque",
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: namespace,
			},
			Data: map[string][]byte{
				"osServiceAccount.json": []byte(creds),
				"billingaccount":        []byte("billingaccount"),
			},
		},
	}
}

func (t *testSecretBuilder) WihtoutKey(key string) *testSecretBuilder {
	delete(t.s.Data, key)
	return t
}
