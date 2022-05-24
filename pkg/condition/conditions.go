package condition

import (
	gcpv1alpha1 "github.com/openshift/gcp-project-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Conditions is a wrapper object for actual Condition functions to allow for easier mocking/testing.
//go:generate mockgen -destination=../util/mocks/$GOPACKAGE/conditions.go -package=$GOPACKAGE -source conditions.go
type Conditions interface {
	SetCondition(conditions *[]gcpv1alpha1.Condition, conditionType gcpv1alpha1.ConditionType, status corev1.ConditionStatus, reason string, message string)
	FindCondition(conditions *[]gcpv1alpha1.Condition, conditionType gcpv1alpha1.ConditionType) (*gcpv1alpha1.Condition, bool)
	HasCondition(conditions *[]gcpv1alpha1.Condition, conditionType gcpv1alpha1.ConditionType) bool
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
	condition, _ := c.FindCondition(conditions, conditionType)
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
// the second return code is true if the condition already existed before
func (c *ConditionManager) FindCondition(conditions *[]gcpv1alpha1.Condition, conditionType gcpv1alpha1.ConditionType) (*gcpv1alpha1.Condition, bool) {
	for i, condition := range *conditions {
		if condition.Type == conditionType {
			return &(*conditions)[i], true
		}
	}

	*conditions = append(
		*conditions,
		gcpv1alpha1.Condition{
			Type: conditionType,
		},
	)

	return &(*conditions)[len(*conditions)-1], false
}

// HasCondition checks for the existance of a given Condition type
func (c *ConditionManager) HasCondition(conditions *[]gcpv1alpha1.Condition, conditionType gcpv1alpha1.ConditionType) bool {
	for _, condition := range *conditions {
		if condition.Type == conditionType {
			return true
		}
	}
	return false
}
