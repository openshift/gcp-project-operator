package errors

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("wrapper.go", func() {
	Context("when Wrap() called with proper error and message", func() {
		BeforeEach(func() {
		})
		It("should return a formatted error", func() {
			sut := Wrap(errors.New("dummy context"), "testing")
			Expect(sut.Error()).NotTo(BeNil())
			Expect(sut.Error()).Should(ContainSubstring("gcp-project-operator/pkg/util/errors/wrapper_test.go"))
			Expect(sut.Error()).Should(ContainSubstring("Line:"))
			Expect(sut.Error()).Should(ContainSubstring("testing"))
			Expect(sut.Error()).Should(ContainSubstring("dummy context"))
		})
	})
})
