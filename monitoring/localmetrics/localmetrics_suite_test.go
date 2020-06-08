package localmetrics_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestLocalmetrics(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Localmetrics Suite")
}
