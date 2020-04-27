package condition

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	gcpv1alpha1 "github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("ConditionManager", func() {
	conditionManager := NewConditionManager()
	sut := &[]gcpv1alpha1.Condition{}
	conditionType := gcpv1alpha1.ConditionError
	reason := "dummyReconcile"
	message := "fake error"

	Context("when SetCondition() called with status condition true", func() {
		status := corev1.ConditionTrue
		BeforeEach(func() {
			*sut = []gcpv1alpha1.Condition{}
		})
		It("should update the condition list", func() {
			conditionManager.SetCondition(sut, conditionType, status, reason, message)

			Expect(len(*sut)).To(Equal(1))
			obj := getFirst(*sut)
			Expect(obj.Status).To(Equal(status))
			Expect(obj.Message).To(Equal(message))
			Expect(obj.Reason).To(Equal(reason))
			Expect(obj.Status).To(Equal(status))
		})
		It("should update the fields with given parameters except for LastTransitionTime if there's one existing condition", func() {
			// Set Existing condition
			conditionManager.SetCondition(sut, conditionType, status, reason, message)
			// Get current values
			obj := getFirst(*sut)
			probe := obj.LastProbeTime
			transition := obj.LastTransitionTime

			conditionManager.SetCondition(sut, conditionType, status, reason, message)
			obj = getFirst(*sut)

			Expect(len(*sut)).To(Equal(1))
			Expect(obj.LastProbeTime).NotTo(Equal(probe))
			Expect(obj.LastTransitionTime).To(Equal(transition))
			Expect(obj.Message).To(Equal(message))
			Expect(obj.Reason).To(Equal(reason))
		})
	})

	Context("when SetCondition() called with status condition false", func() {
		status := corev1.ConditionFalse
		now := metav1.Now()
		BeforeEach(func() {
			*sut = []gcpv1alpha1.Condition{}
		})
		It("should mark the existing condition as resolved", func() {
			// Set existing condition
			*sut = append(*sut, gcpv1alpha1.Condition{
				Message:            "DummyError",
				Status:             corev1.ConditionTrue,
				LastTransitionTime: now,
				LastProbeTime:      now,
				Reason:             "Dummy",
				Type:               gcpv1alpha1.ConditionError,
			})

			conditionManager.SetCondition(sut, conditionType, status, "DummyResolved", "DummyError")
			obj := getFirst(*sut)
			Expect(obj.Message).To(Equal("DummyError"))
			Expect(obj.Reason).To(Equal("DummyResolved"))
			Expect(obj.Status).To(Equal(status))
			Expect(obj.LastProbeTime).NotTo(Equal(now))
			Expect(obj.LastTransitionTime).NotTo(Equal(now))
		})
	})

	Context("when SetCondition() called with status condition true and a new err message", func() {
		status := corev1.ConditionTrue
		now := metav1.Now()
		BeforeEach(func() {
			*sut = []gcpv1alpha1.Condition{}
		})
		It("should set a new error condition", func() {
			// Set existing condition
			*sut = append(*sut, gcpv1alpha1.Condition{
				Message:            "DummyError",
				Status:             corev1.ConditionFalse,
				LastTransitionTime: now,
				LastProbeTime:      now,
				Reason:             "DummyResolved",
				Type:               gcpv1alpha1.ConditionError,
			})
			old := getFirst(*sut)
			conditionManager.SetCondition(sut, conditionType, status, "SecondFakeReconcileError", "SecondFakeReconcileMessage")
			obj := getFirst(*sut)
			Expect(obj.Message).To(Equal("SecondFakeReconcileMessage"))
			Expect(obj.Reason).To(Equal("SecondFakeReconcileError"))
			Expect(obj.Status).To(Equal(status))
			Expect(obj.LastTransitionTime).NotTo(Equal(old.LastTransitionTime))
			Expect(obj.LastProbeTime).NotTo(Equal(old.LastProbeTime))
		})
	})
})

func getFirst(list []gcpv1alpha1.Condition) gcpv1alpha1.Condition {
	return list[0]
}
