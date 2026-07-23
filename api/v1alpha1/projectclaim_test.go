package v1alpha1

import (
	"errors"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestProjectClaimValidate(t *testing.T) {
	const claimNamespace = "tenant-ns"

	tests := []struct {
		name        string
		claim       ProjectClaim
		expectedErr error
	}{
		{
			name: "valid non-CCS claim with matching GCPCredentialSecret namespace",
			claim: ProjectClaim{
				ObjectMeta: metav1.ObjectMeta{Namespace: claimNamespace},
				Spec: ProjectClaimSpec{
					GCPCredentialSecret: NamespacedName{Namespace: claimNamespace, Name: "creds"},
				},
			},
			expectedErr: nil,
		},
		{
			name: "valid non-CCS claim with empty GCPCredentialSecret namespace",
			claim: ProjectClaim{
				ObjectMeta: metav1.ObjectMeta{Namespace: claimNamespace},
				Spec: ProjectClaimSpec{
					GCPCredentialSecret: NamespacedName{Name: "creds"},
				},
			},
			expectedErr: nil,
		},
		{
			name: "invalid non-CCS claim with cross-namespace GCPCredentialSecret",
			claim: ProjectClaim{
				ObjectMeta: metav1.ObjectMeta{Namespace: claimNamespace},
				Spec: ProjectClaimSpec{
					GCPCredentialSecret: NamespacedName{Namespace: "gcp-project-operator", Name: "creds"},
				},
			},
			expectedErr: ErrGCPCredentialSecretNamespaceMismatch,
		},
		{
			name: "valid CCS claim with matching namespaces",
			claim: ProjectClaim{
				ObjectMeta: metav1.ObjectMeta{Namespace: claimNamespace},
				Spec: ProjectClaimSpec{
					CCS:                 true,
					CCSSecretRef:        NamespacedName{Namespace: claimNamespace, Name: "ccs-secret"},
					GCPCredentialSecret: NamespacedName{Namespace: claimNamespace, Name: "creds"},
				},
			},
			expectedErr: nil,
		},
		{
			name: "valid CCS claim with empty CCSSecretRef namespace",
			claim: ProjectClaim{
				ObjectMeta: metav1.ObjectMeta{Namespace: claimNamespace},
				Spec: ProjectClaimSpec{
					CCS:                 true,
					CCSSecretRef:        NamespacedName{Name: "ccs-secret"},
					GCPCredentialSecret: NamespacedName{Namespace: claimNamespace, Name: "creds"},
				},
			},
			expectedErr: nil,
		},
		{
			name: "invalid CCS claim with cross-namespace CCSSecretRef",
			claim: ProjectClaim{
				ObjectMeta: metav1.ObjectMeta{Namespace: claimNamespace},
				Spec: ProjectClaimSpec{
					CCS:                 true,
					CCSSecretRef:        NamespacedName{Namespace: "gcp-project-operator", Name: "gcp-project-operator-credentials"},
					GCPCredentialSecret: NamespacedName{Namespace: claimNamespace, Name: "creds"},
				},
			},
			expectedErr: ErrCCSSecretRefNamespaceMismatch,
		},
		{
			name: "invalid CCS claim with cross-namespace GCPCredentialSecret",
			claim: ProjectClaim{
				ObjectMeta: metav1.ObjectMeta{Namespace: claimNamespace},
				Spec: ProjectClaimSpec{
					CCS:                 true,
					CCSSecretRef:        NamespacedName{Namespace: claimNamespace, Name: "ccs-secret"},
					GCPCredentialSecret: NamespacedName{Namespace: "other-namespace", Name: "creds"},
				},
			},
			expectedErr: ErrGCPCredentialSecretNamespaceMismatch,
		},
		{
			name: "CCSSecretRef ignored for non-CCS claims",
			claim: ProjectClaim{
				ObjectMeta: metav1.ObjectMeta{Namespace: claimNamespace},
				Spec: ProjectClaimSpec{
					CCS:                 false,
					CCSSecretRef:        NamespacedName{Namespace: "other-namespace", Name: "ccs-secret"},
					GCPCredentialSecret: NamespacedName{Namespace: claimNamespace, Name: "creds"},
				},
			},
			expectedErr: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.claim.Validate()
			if !errors.Is(err, test.expectedErr) {
				t.Errorf("got %v, wanted %v", err, test.expectedErr)
			}
		})
	}
}
