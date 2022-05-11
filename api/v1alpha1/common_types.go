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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LegalEntity contains Red Hat specific identifiers to the original creator the clusters
type LegalEntity struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

// NamespacedName contains the name of a object and its namespace
type NamespacedName struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// Condition contains details for the current condition of a custom resource
type Condition struct {
	// Type is the type of the condition.
	Type ConditionType `json:"type"`
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

// ConditionType is a valid value for Condition.Type
type ConditionType string

const (
	// ConditionReady is set when a Project custom resource state changes Ready state
	ConditionReady ConditionType = "Ready"
	// ConditionPending is set when a project custom resource state changes to Pending
	ConditionPending ConditionType = "Pending"
	// ConditionVerification is set when a project custom resource state changes to Verification state
	ConditionVerification ConditionType = "Verification"
	// ConditionError is set when a project custom resource state changes to Error
	ConditionError ConditionType = "Error"
	// ConditionInvalid is set when a project custom resource has an invalid or unsupported configuration
	ConditionInvalid ConditionType = "Invalid"
	// ConditionComputeApiReady is set when the compute API is not yet ready
	ConditionComputeApiReady ConditionType = "ComputeApiReady"
)
