package localmetrics

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	MetricTotalProjectClaims = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: prefix + "_total_projectclaims",
		Help: "Report how many gcp ProjectClaim objects exist in the cluster",
	}, []string{"name"})
)

const prefix = "gcp_project_operator"

var (
	// MetricsList is  the list of metrics imported to prometheus
	MetricsList = []prometheus.Collector{
		MetricTotalProjectClaims,
	}
)

type MetricsConfig struct {
	c   client.Client
	log logr.Logger
}

// NewMetricsConfig returns a new instance for configuring the metrics
func NewMetricsConfig(c client.Client, log logr.Logger) *MetricsConfig {
	return &MetricsConfig{c: c, log: log}
}

func (m MetricsConfig) PublishMetrics() {
	m.TotalProjectClaims()
	go func() {
		for {
			select {
			case <-time.After(3 * time.Second):
				err := m.TotalProjectClaims()
				if err != nil {
					m.log.Error(err, "Cannot Expose metrics to prometheus")
				}
			}
		}
	}()
}

func (m MetricsConfig) TotalProjectClaims() error {
	r := &v1alpha1.ProjectClaimList{}
	if err := m.c.List(context.TODO(), &client.ListOptions{}, r); err != nil {
		return fmt.Errorf("Cannot list `ProjectClaim`, error is  %v", err)
	}
	items := len(r.Items)
	MetricTotalProjectClaims.With(prometheus.Labels{"name": "gcp-project-operator"}).Set(float64(items))
	return nil
}
