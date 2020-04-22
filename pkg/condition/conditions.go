package condition

import (
	gcpv1alpha1 "github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Conditions is a wrapper object for actual Condition functions to allow for easier mocking/testing.
type Conditions interface {
	SetCondition(status corev1.ConditionStatus, reason string, message string)
	GetConditions() *[]gcpv1alpha1.Condition
}

type ConditionManager struct {
	conditions []gcpv1alpha1.Condition
}

// NewConditionManager returns a ConditionManager object which holds condition list
func NewConditionManager() Conditions {
	return &ConditionManager{}
}

func (c *ConditionManager) GetConditions() *[]gcpv1alpha1.Condition {
	return &c.conditions
}

// SetCondition sets a condition on a custom resource's status
func (c *ConditionManager) SetCondition(status corev1.ConditionStatus, reason string, message string) {
	conditionType := gcpv1alpha1.ConditionError
	now := metav1.Now()
	existingCondition := c.FindCondition()
	if existingCondition == nil {
		if status == corev1.ConditionTrue {
			c.conditions = append(
				c.conditions,
				gcpv1alpha1.Condition{
					Type:               conditionType,
					Status:             status,
					Reason:             reason,
					Message:            message,
					LastTransitionTime: now,
					LastProbeTime:      now,
				},
			)
		}
	} else {
		// If it does not exist, assign it as now. Otherwise, do not touch
		if existingCondition.Status != status {
			existingCondition.LastTransitionTime = now
		}
		existingCondition.Status = status
		existingCondition.Reason = reason
		existingCondition.Message = message
		existingCondition.LastProbeTime = now
	}
}

// FindCondition finds the suitable Condition object
// by looking for adapter's condition list.
// If none exists, then returns nil.
func (c *ConditionManager) FindCondition() *gcpv1alpha1.Condition {
	conditionType := gcpv1alpha1.ConditionError
	for i, condition := range c.conditions {
		if condition.Type == conditionType {
			return &c.conditions[i]
		}
	}

	return nil
}
