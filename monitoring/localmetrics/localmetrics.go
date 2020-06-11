package localmetrics

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-logr/logr"
	"github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1"
	operrors "github.com/openshift/gcp-project-operator/pkg/util/errors"
	"github.com/prometheus/client_golang/prometheus"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	kubetypes "k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	MetricTotalProjectClaims = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: prefix + "_total_projectclaims",
		Help: "Report how many gcp ProjectClaim objects exist in the cluster",
	}, []string{"name"})
)

const (
	prefix                  = "gcp_project_operator"
	monitoringContainerName = "monitoring"
)

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

// GetCurrentMonitoringImage retrieves the image name for the current pod
// it is able to extract that by changing the deployment according to
// https://kubernetes.io/docs/tasks/inject-data-application/downward-api-volume-expose-pod-information/
//
func (m MetricsConfig) GetCurrentMonitoringImage() (string, error) {

	var (
		podName      string
		podNamespace string
		ok           bool
	)
	podName, ok = os.LookupEnv("POD_NAME")
	if !ok {
		panic(1)
	}
	podNamespace, ok = os.LookupEnv("POD_NAMESPACE")
	if !ok {
		panic(1)
	}
	return m.GetPodImage(kubetypes.NamespacedName{Name: podName, Namespace: podNamespace})
}

// GetPodImage retrieves the image name for a given pod
func (m MetricsConfig) GetPodImage(namespacedName kubetypes.NamespacedName) (string, error) {
	found := &corev1.Pod{}
	err := m.c.Get(context.TODO(), namespacedName, found)
	if err != nil {
		if errors.IsNotFound(err) {
			return "", operrors.Wrap(err, "cannot get current pod details, check if POD_NAME && POD_NAMESPACE environment variables are configured correctly")
		}
		return "", err
	}

	for _, cont := range found.Spec.Containers {
		if cont.Name == monitoringContainerName {
			return cont.Image, nil
		}
	}
	return "", errors.NewNotFound(corev1.Resource("pod"), fmt.Sprintf("Could not find container with name '%s' inside pod '%s' on namespace '%s'", monitoringContainerName, namespacedName.Name, namespacedName.Namespace))
}

// CreateAndApplyDeployment deploys the new metrics deployment to
func (m MetricsConfig) CreateAndApplyDeployment(portNumber int32, imageName string, namespacedName kubetypes.NamespacedName) error {
	dep := m.newMetricsDeploymentTemplate(portNumber, imageName, namespacedName)
	found := &appsv1.Deployment{}
	if err := m.c.Get(context.TODO(), namespacedName, found); err != nil {
		if errors.IsNotFound(err) {
			return m.c.Create(context.TODO(), dep)
		}
		return err
	}
	if err := m.c.Delete(context.TODO(), found); err != nil {
		return err
	}
	return m.c.Create(context.TODO(), dep)

}

// newMetricsDeploymentTemplate creates a deployment scaffold.
func (m MetricsConfig) newMetricsDeploymentTemplate(portNumber int32, imageName string, namespacedName kubetypes.NamespacedName) *appsv1.Deployment {
	labels := map[string]string{
		"app": "monitoring",
	}
	int32Ptr := func(i int32) *int32 { return &i }

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespacedName.Name,
			Namespace: namespacedName.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  monitoringContainerName,
							Image: imageName,
							Ports: []corev1.ContainerPort{
								{
									Name:          "prometheus",
									ContainerPort: portNumber,
								},
							},
						},
					},
				},
			},
		},
	}
}
