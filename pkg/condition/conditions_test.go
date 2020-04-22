package condition

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestSetCondition(t *testing.T) {
	conditionManager := NewConditionManager()
	sut := conditionManager.GetConditions()
	status := corev1.ConditionTrue
	reason := "dummy reconcile"
	message := "fake error"

	conditionManager.SetCondition(status, reason, message)

	if len(*sut) != 1 {
		t.Errorf("item count should be 1, got item: %v", len(*sut))
	}

	obj := (*sut)[0]
	if obj.Status != status {
		t.Errorf("expected status: %v, got %v", status, obj.Status)
	}
	if obj.Reason != reason {
		t.Errorf("expected reason: %v, got %v", reason, obj.Reason)
	}
	if obj.Message != message {
		t.Errorf("expected message: %v, got %v", message, obj.Message)
	}

	probe := obj.LastProbeTime
	transition := obj.LastTransitionTime

	conditionManager.SetCondition(status, reason, message)
	// get new updated obj
	obj = (*sut)[0]

	if obj.LastProbeTime == probe {
		t.Errorf("expected %v should not equal to %v", probe, obj.LastProbeTime)
	}

	if obj.LastTransitionTime != transition {
		t.Errorf("transition time shouldn't be changed, expected %v, got %v", transition, obj.LastTransitionTime)
	}
}
