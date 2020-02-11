package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ProjectSpec defines the desired state of Project
// +k8s:openapi-gen=true
type ProjectSpec struct {
	GCPProjectName     string         `json:"gcpProjectName,omitempty"`
	ProjectClaimCRLink NamespacedName `json:"projectClaimCRLink"`
	LegalEntity        LegalEntity    `json:"legalEntity"`
}

// ProjectStatus defines the observed state of Project
// +k8s:openapi-gen=true
type ProjectStatus struct {
	Conditions []ProjectCondition `json:"conditions,omitempty"`
	State      ProjectState       `json:"state,omitempty"`
}

// ProjectCondition contains details for the current condition of a gcp Project CR
type ProjectCondition struct {
	// Type is the type of the condition.
	Type ProjectConditionType `json:"type"`
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

// ProjectConditionType is a valid value for ProjectCondition.Type
type ProjectConditionType string

// ProjectState is a valid value from Project.Status
type ProjectState string

const (
	// ProjectStatusCreating creating status for a Project CR
	ProjectStatusCreating ProjectState = "Creating"
	// ProjectStatusReady ready status for a Project CR
	ProjectStatusReady ProjectState = "Ready"
	// ProjectStatusError error status for a Project CR
	ProjectStatusError ProjectState = "Error"
	// ProjectStatusVerification pending verification status for a Project CR
	ProjectStatusVerification ProjectState = "Verification"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Project is the Schema for the projects API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state",description="Status of the project"
// +kubebuilder:printcolumn:name="ClaimName",type="string",JSONPath=".spec.projectClaimCRLink.name",description="Name of corresponding project claim CR"
// +kubebuilder:printcolumn:name="ClaimNameSpace",type="string",JSONPath=".spec.projectClaimCRLink.namespace",description="Namesspace of corresponding project CR"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Age since the project was created"
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectSpec   `json:"spec,omitempty"`
	Status ProjectStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProjectList contains a list of Project
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Project `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Project{}, &ProjectList{})
}
