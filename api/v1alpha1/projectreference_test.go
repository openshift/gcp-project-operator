package v1alpha1

import (
	"errors"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestProjectReferenceValidate(t *testing.T) {
	const (
		refNamespace   = "gcp-project-operator"
		tenantNS       = "tenant-ns"
		foreignNS      = "foreign-ns"
	)

	tests := []struct {
		name        string
		ref         ProjectReference
		expectedErr error
	}{
		{
			name: "valid non-CCS ProjectReference",
			ref: ProjectReference{
				ObjectMeta: metav1.ObjectMeta{Namespace: refNamespace},
				Spec:       ProjectReferenceSpec{CCS: false},
			},
			expectedErr: nil,
		},
		{
			name: "valid CCS ProjectReference with CCSSecretRef matching claim namespace",
			ref: ProjectReference{
				ObjectMeta: metav1.ObjectMeta{Namespace: refNamespace},
				Spec: ProjectReferenceSpec{
					CCS:                true,
					ProjectClaimCRLink: NamespacedName{Namespace: tenantNS, Name: "claim"},
					CCSSecretRef:       NamespacedName{Namespace: tenantNS, Name: "ccs-secret"},
				},
			},
			expectedErr: nil,
		},
		{
			name: "valid CCS ProjectReference with empty CCSSecretRef namespace",
			ref: ProjectReference{
				ObjectMeta: metav1.ObjectMeta{Namespace: refNamespace},
				Spec: ProjectReferenceSpec{
					CCS:                true,
					ProjectClaimCRLink: NamespacedName{Namespace: tenantNS, Name: "claim"},
					CCSSecretRef:       NamespacedName{Name: "ccs-secret"},
				},
			},
			expectedErr: nil,
		},
		{
			name: "invalid CCS ProjectReference with CCSSecretRef pointing to operator namespace",
			ref: ProjectReference{
				ObjectMeta: metav1.ObjectMeta{Namespace: refNamespace},
				Spec: ProjectReferenceSpec{
					CCS:                true,
					ProjectClaimCRLink: NamespacedName{Namespace: tenantNS, Name: "claim"},
					CCSSecretRef:       NamespacedName{Namespace: refNamespace, Name: "ccs-secret"},
				},
			},
			expectedErr: ErrProjectRefCCSSecretRefNamespaceMismatch,
		},
		{
			name: "invalid CCS ProjectReference with CCSSecretRef pointing to foreign namespace",
			ref: ProjectReference{
				ObjectMeta: metav1.ObjectMeta{Namespace: refNamespace},
				Spec: ProjectReferenceSpec{
					CCS:                true,
					ProjectClaimCRLink: NamespacedName{Namespace: tenantNS, Name: "claim"},
					CCSSecretRef:       NamespacedName{Namespace: foreignNS, Name: "ccs-secret"},
				},
			},
			expectedErr: ErrProjectRefCCSSecretRefNamespaceMismatch,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.ref.Validate()
			if !errors.Is(err, test.expectedErr) {
				t.Errorf("got %v, wanted %v", err, test.expectedErr)
			}
		})
	}
}
