/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProjectClaimSpec defines the desired state of ProjectClaim
// +k8s:openapi-gen=true
type ProjectClaimSpec struct {
	LegalEntity            LegalEntity    `json:"legalEntity"`
	GCPCredentialSecret    NamespacedName `json:"gcpCredentialSecret"`
	Region                 string         `json:"region"`
	GCPProjectID           string         `json:"gcpProjectID,omitempty"`
	ProjectReferenceCRLink NamespacedName `json:"projectReferenceCRLink,omitempty"`
	// +listType=atomic
	AvailabilityZones []string       `json:"availabilityZones,omitempty"`
	CCS               bool           `json:"ccs,omitempty"`
	CCSSecretRef      NamespacedName `json:"ccsSecretRef,omitempty"`
	CCSProjectID      string         `json:"ccsProjectID,omitempty"`
}

// ProjectClaimStatus defines the observed state of ProjectClaim
// +k8s:openapi-gen=true
type ProjectClaimStatus struct {
	// +listType=atomic
	Conditions []Condition `json:"conditions"`
	State      ClaimStatus `json:"state"`
}

// ClaimStatus is a valid value from ProjectClaim.Status
type ClaimStatus string

const (
	// ClaimStatusPending pending status for a claim
	ClaimStatusPending ClaimStatus = "Pending"
	// ClaimStatusPendingProject pending project status for a claim
	ClaimStatusPendingProject ClaimStatus = "PendingProject"
	// ClaimStatusReady ready status for a claim
	ClaimStatusReady ClaimStatus = "Ready"
	// ClaimStatusError error status for a claim
	ClaimStatusError ClaimStatus = "Error"
	// ClaimStatusVerification pending verification status for a claim
	ClaimStatusVerification ClaimStatus = "Verification"
)

// ProjectClaim is the Schema for the projectclaims API
// +kubebuilder:object:root=true
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state",description="Status of the project claim"
// +kubebuilder:printcolumn:name="GCPProjectID",type="string",JSONPath=".spec.gcpProjectID",description="ID of the GCP Project that has been created"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Age since the project claim was created"
type ProjectClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectClaimSpec   `json:"spec,omitempty"`
	Status ProjectClaimStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true
// ProjectClaimList contains a list of ProjectClaim
type ProjectClaimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProjectClaim `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ProjectClaim{}, &ProjectClaimList{})
}
