package localmetrics_test

import (
	"errors"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/openshift/gcp-project-operator/monitoring/localmetrics"

	"github.com/openshift/gcp-project-operator/pkg/apis"
	"github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/go-logr/logr/testing"
	"github.com/openshift/gcp-project-operator/pkg/util/mocks"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// AddCustomResourcesToScheme adds the gcp-project-operator crd to the client to allow
// injestion in the api commands
func AddCustomResourcesToScheme() {
	apis.AddToSchemes.AddToScheme(scheme.Scheme)
}

var _ = Describe("Localmetrics", func() {
	var (
		logger = testing.NullLogger{}
		c      client.Client
	)
	Describe("TotalProjectClaims", func() {
		BeforeEach(func() {
			AddCustomResourcesToScheme()
		})
		It("Should return 0 results when no claims exist", func() {
			c = fake.NewFakeClient()
			m := NewMetricsConfig(c, logger)
			err := m.TotalProjectClaims()
			metricsSize := testutil.ToFloat64(MetricTotalProjectClaims)

			Expect(metricsSize).To(Equal(float64(0)))
			Expect(err).NotTo(HaveOccurred())
		})
		It("Should return 1 with one project present", func() {
			expected := &v1alpha1.ProjectClaim{}
			c = fake.NewFakeClient(expected)
			m := NewMetricsConfig(c, logger)
			err := m.TotalProjectClaims()
			metricsSize := testutil.ToFloat64(MetricTotalProjectClaims)

			Expect(metricsSize).To(Equal(float64(1)))
			Expect(err).NotTo(HaveOccurred())

		})
		It("Should return error and 0 with no project present", func() {

			mockCtrl := gomock.NewController(GinkgoT())
			emptyClaimList := v1alpha1.ProjectClaimList{}
			expected := errors.New("abbab")
			cMock := mocks.NewMockClient(mockCtrl)

			m := NewMetricsConfig(cMock, logger)

			cMock.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, emptyClaimList).Return(expected)

			// Act
			err := m.TotalProjectClaims()

			// Assert
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Cannot list `ProjectClaim`, error is")) //TODO: look why this fails
		})
	})
})
