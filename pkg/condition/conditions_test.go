package condition

import (
	"testing"

	gcpv1alpha1 "github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func TestSetCondition(t *testing.T) {
	conditionManager := NewConditionManager()
	sut := &[]gcpv1alpha1.Condition{}
	conditionType := gcpv1alpha1.ConditionError
	status := corev1.ConditionTrue
	reason := "dummy reconcile"
	message := "fake error"

	conditionManager.SetCondition(sut, conditionType, status, reason, message)

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

	conditionManager.SetCondition(sut, conditionType, status, reason, message)
	// get new updated obj
	obj = (*sut)[0]

	if obj.LastProbeTime == probe {
		t.Errorf("expected %v should not equal to %v", probe, obj.LastProbeTime)
	}

	if obj.LastTransitionTime != transition {
		t.Errorf("transition time shouldn't be changed, expected %v, got %v", transition, obj.LastTransitionTime)
	}

	// call setCondition() with conditionFalse and see if the status is marked as resolved
	status = corev1.ConditionFalse
	var expectedStatus corev1.ConditionStatus
	expectedStatus = "False"
	reason = reason + "changed"
	conditionManager.SetCondition(sut, conditionType, status, reason, message)
	obj = (*sut)[0]
	if obj.Status != expectedStatus {
		t.Errorf("SetCondition() called with conditionFalse, expected %v, got %v", expectedStatus, obj.Status)
	}

	if obj.Reason != reason {
		t.Errorf("reason should be updated, expected %v, got %v", reason, obj.Reason)
	}
}
