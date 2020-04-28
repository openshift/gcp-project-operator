package condition

import (
	gcpv1alpha1 "github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Conditions is a wrapper object for actual Condition functions to allow for easier mocking/testing.
type Conditions interface {
	SetCondition(conditions *[]gcpv1alpha1.Condition, conditionType gcpv1alpha1.ConditionType, status corev1.ConditionStatus, reason string, message string)
}

type ConditionManager struct {
}

// NewConditionManager returns a ConditionManager object
func NewConditionManager() Conditions {
	return &ConditionManager{}
}

// SetCondition sets a condition on a custom resource's status
func (c *ConditionManager) SetCondition(conditions *[]gcpv1alpha1.Condition, conditionType gcpv1alpha1.ConditionType, status corev1.ConditionStatus, reason string, message string) {
	now := metav1.Now()
	condition := c.FindCondition(conditions, conditionType)
	if message != condition.Message ||
		status != condition.Status ||
		reason != condition.Reason ||
		conditionType != condition.Type {

		condition.LastTransitionTime = now
	}
	if message != "" {
		condition.Message = message
	}
	condition.LastProbeTime = now
	condition.Reason = reason
	condition.Status = status
}

// FindCondition finds the suitable Condition object
// by looking for adapter's condition list.
// If none exists, it appends one.
func (c *ConditionManager) FindCondition(conditions *[]gcpv1alpha1.Condition, conditionType gcpv1alpha1.ConditionType) *gcpv1alpha1.Condition {
	for i, condition := range *conditions {
		if condition.Type == conditionType {
			return &(*conditions)[i]
		}
	}

	*conditions = append(
		*conditions,
		gcpv1alpha1.Condition{
			Type: conditionType,
		},
	)

	return &(*conditions)[len(*conditions)-1]
}
