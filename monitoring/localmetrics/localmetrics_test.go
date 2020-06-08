package localmetrics_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/openshift/gcp-project-operator/monitoring/localmetrics"

	"github.com/go-logr/logr/testing"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Localmetrics", func() {
	Describe("TotalProjectClaims", func() {
		It("Should return 0 when no claims exist", func() {
			c := fake.NewFakeClient()
			l := testing.NullLogger{}
			m := NewMetricsConfig(c, l)
			m.TotalProjectClaims()
			metricsSize := testutil.ToFloat64(MetricTotalProjectClaims)

			Expect(metricsSize).To(Equal(float64(0)))
		})
		// It("Should return 1 with one project present", func() {
		// 	expected := &v1alpha1.ProjectClaim{}
		// 	c := fake.NewFakeClient(expected)
		// 	l := testing.NullLogger{}
		// 	m := NewMetricsConfig(c, l)
		// 	m.TotalProjectClaims()
		// 	metricsSize := testutil.ToFloat64(MetricTotalProjectClaims)

		// 	Expect(metricsSize).To(Equal(float64(1)))
		// })
	})
})
