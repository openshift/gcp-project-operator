package projectreference_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/openshift/gcp-project-operator/pkg/controller/projectreference"
)

var _ = Describe("ProjectreferenceAdapter", func() {
	Context("generated project names", func() {
		It("are shorter than 30 characters", func() {
			projectID, err := GenerateProjectID()
			Expect(err).NotTo(HaveOccurred())
			Expect(len(projectID)).To(BeNumerically("<=", 30))
		})

		It("are longer than 6 characters", func() {
			projectID, err := GenerateProjectID()
			Expect(err).NotTo(HaveOccurred())
			Expect(len(projectID)).To(BeNumerically(">=", 6))
		})

		It("start with a lowercase letter", func() {
			projectID, err := GenerateProjectID()
			Expect(err).NotTo(HaveOccurred())
			Expect("abcdefghijklmnopqrstuvwxyz").To(ContainSubstring(string(projectID[0])))
		})

		It("contains only lowercase letters, numbers or hyphens", func() {
			projectID, err := GenerateProjectID()
			Expect(err).NotTo(HaveOccurred())
			for _, char := range projectID {
				Expect("abcdefghijklmnopqrstuvwxyz1234567890-").To(ContainSubstring(string(char)))
			}
		})
	})
})
