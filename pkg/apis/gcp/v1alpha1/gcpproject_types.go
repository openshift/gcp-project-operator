package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// GCPProjectSpec defines the desired state of Project
// +k8s:openapi-gen=true
type GCPProjectSpec struct {
	GCPProjectID       string         `json:"gcpProjectID,omitempty"`
	ProjectClaimCRLink NamespacedName `json:"projectClaimCRLink"`
	LegalEntity        LegalEntity    `json:"legalEntity"`
}

// GCPProjectStatus defines the observed state of Project
// +k8s:openapi-gen=true
type GCPProjectStatus struct {
	Conditions []GCPProjectCondition `json:"conditions,omitempty"`
	State      GCPProjectState       `json:"state,omitempty"`
}

// GCPProjectCondition contains details for the current condition of a GCPProject CR
type GCPProjectCondition struct {
	// Type is the type of the condition.
	Type GCPProjectConditionType `json:"type"`
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

// GCPProjectConditionType is a valid value for GCPProjectCondition.Type
type GCPProjectConditionType string

// GCPProjectState is a valid value from GCPProject.Status
type GCPProjectState string

const (
	// GCPProjectStatusCreating creating status for a GCPProject CR
	GCPProjectStatusCreating GCPProjectState = "Creating"
	// GCPProjectStatusReady ready status for a GCPProject CR
	GCPProjectStatusReady GCPProjectState = "Ready"
	// GCPProjectStatusError error status for a GCPProject CR
	GCPProjectStatusError GCPProjectState = "Error"
	// GCPProjectStatusVerification pending verification status for a GCPProject CR
	GCPProjectStatusVerification GCPProjectState = "Verification"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GCPProject is the Schema for the GCPprojects API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state",description="Status of the GCPproject"
// +kubebuilder:printcolumn:name="ClaimName",type="string",JSONPath=".spec.gcpprojectClaimCRLink.name",description="Name of corresponding project claim CR"
// +kubebuilder:printcolumn:name="ClaimNameSpace",type="string",JSONPath=".spec.gcpprojectClaimCRLink.namespace",description="Namesspace of corresponding project claim CR"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Age since the GCPproject was created"
type GCPProject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GCPProjectSpec   `json:"spec,omitempty"`
	Status GCPProjectStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GCPProjectList contains a list of GCPProject
type GCPProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GCPProject `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GCPProject{}, &GCPProjectList{})
}
