package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ProjectClaimSpec defines the desired state of ProjectClaim
// +k8s:openapi-gen=true
type ProjectClaimSpec struct {
	LegalEntity         LegalEntity    `json:"legalEntity"`
	GCPCredentialSecret NamespacedName `json:"gcpCredentialSecret"`
	Region              string         `json:"region"`
	GCPProjectName      string         `json:"gcpProjectName,omitempty"`
	ProjectCRLink       NamespacedName `json:"projectCRLink,omitempty"`
}

// ProjectClaimStatus defines the observed state of ProjectClaim
// +k8s:openapi-gen=true
type ProjectClaimStatus struct {
	Conditions []ProjectClaimCondition `json:"conditions"`
	State      ClaimStatus             `json:"state"`
}

// ProjectClaimCondition contains details for the current condition of a gcp Project claim
type ProjectClaimCondition struct {
	// Type is the type of the condition.
	Type ProjectClaimConditionType `json:"type"`
	// Status is the status of the condition.
	Status corev1.ConditionStatus `json:"status"`
	// LastProbeTime is the last time we probed the condition.
	// +optional
	LastProbeTime metav1.Time `json:"lastProbeTime,omitempty"`
	// LastTransitionTime is the last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// Reason is a unique, one-word, CamelCase reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`
	// Message is a human-readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty"`
}

// ProjectClaimConditionType is a valid value for ProjectClaimCondition.Type
type ProjectClaimConditionType string

const (
	// ClaimConditionReady is set when a Project claim state changes Ready state
	ClaimConditionReady ProjectClaimConditionType = "Ready"
	// ClaimConditionPending is set when a project claim state changes to Pending
	ClaimConditionPending ProjectClaimConditionType = "Pending"
	// ClaimConditionVerification is set when a project claim state changes to Verification state
	ClaimConditionVerification ProjectClaimConditionType = "Verification"
	// ClaimConditionError is set when a project claim state changes to Error
	ClaimConditionError ProjectClaimConditionType = "Error"
)

// ClaimStatus is a valid value from ProjectClaim.Status
type ClaimStatus string

const (
	// ClaimStatusPending pending status for a claim
	ClaimStatusPending ClaimStatus = "Pending"
	// ClaimStatusReady ready status for a claim
	ClaimStatusReady ClaimStatus = "Ready"
	// ClaimStatusError error status for a claim
	ClaimStatusError ClaimStatus = "Error"
	// ClaimStatusVerification pending verification status for a claim
	ClaimStatusVerification ClaimStatus = "Verification"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProjectClaim is the Schema for the projectclaims API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state",description="Status of the project claim"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Age since the project claim was created"
type ProjectClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectClaimSpec   `json:"spec,omitempty"`
	Status ProjectClaimStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProjectClaimList contains a list of ProjectClaim
type ProjectClaimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProjectClaim `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ProjectClaim{}, &ProjectClaimList{})
}
