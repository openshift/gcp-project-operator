package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProjectReferenceSpec defines the desired state of Project
// +k8s:openapi-gen=true
type ProjectReferenceSpec struct {
	GCPProjectID       string         `json:"gcpProjectID,omitempty"`
	ProjectClaimCRLink NamespacedName `json:"projectClaimCRLink"`
	LegalEntity        LegalEntity    `json:"legalEntity"`
}

// ProjectReferenceStatus defines the observed state of Project
// +k8s:openapi-gen=true
type ProjectReferenceStatus struct {
	Conditions []ProjectReferenceCondition `json:"conditions"`
	State      ProjectReferenceState       `json:"state,omitempty"`
}

// ProjectReferenceCondition contains details for the current condition of a ProjectReference CR
type ProjectReferenceCondition struct {
	// Type is the type of the condition.
	Type ProjectReferenceConditionType `json:"type"`
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

// ProjectReferenceConditionType is a valid value for ProjectReferenceCondition.Type
type ProjectReferenceConditionType string

const (
	// ProjectReferenceConditionError is set when a project Reference state changes to Error
	ProjectReferenceConditionError ProjectReferenceConditionType = "Error"
)

// ProjectReferenceState is a valid value from ProjectReference.Status
type ProjectReferenceState string

// ProjectReferenceNamespace namespace, where ProjectReference CRs will be created
const (
	ProjectReferenceNamespace string = "gcp-project-operator"
)

const (
	// ProjectReferenceStatusCreating creating status for a ProjectReference CR
	ProjectReferenceStatusCreating ProjectReferenceState = "Creating"
	// ProjectReferenceStatusReady ready status for a ProjectReference CR
	ProjectReferenceStatusReady ProjectReferenceState = "Ready"
	// ProjectReferenceStatusError error status for a ProjectReference CR
	ProjectReferenceStatusError ProjectReferenceState = "Error"
	// ProjectReferenceStatusVerification pending verification status for a ProjectReference CR
	ProjectReferenceStatusVerification ProjectReferenceState = "Verification"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProjectReference is the Schema for the ProjectReferences API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state",description="Status of the ProjectReference"
// +kubebuilder:printcolumn:name="ClaimName",type="string",JSONPath=".spec.projectClaimCRLink.name",description="Name of corresponding project claim CR"
// +kubebuilder:printcolumn:name="ClaimNameSpace",type="string",JSONPath=".spec.projectClaimCRLink.namespace",description="Namesspace of corresponding project claim CR"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Age since the ProjectReference was created"
type ProjectReference struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectReferenceSpec   `json:"spec,omitempty"`
	Status ProjectReferenceStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProjectReferenceList contains a list of ProjectReference
type ProjectReferenceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProjectReference `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ProjectReference{}, &ProjectReferenceList{})
}
