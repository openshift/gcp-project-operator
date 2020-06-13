package main

import (
	"net/http"
	"os"

	"github.com/openshift/gcp-project-operator/monitoring/localmetrics"
	"github.com/openshift/gcp-project-operator/pkg/apis"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	controllers "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	kuberneteslog "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	log          = kuberneteslog.Log.WithName("cmd")
	promRegistry = prometheus.NewRegistry()
)

func init() {
	for _, metric := range localmetrics.MetricsList {
		//prometheus.Register(metric)
		promRegistry.MustRegister(metric)
	}
}

func main() {

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{Namespace: ""})
	if err != nil {
		log.Error(err, "unable to start manager")
		os.Exit(1)
	}
	stop := controllers.SetupSignalHandler()
	// start cache and wait for sync
	log.Info("Initialize and start cache")
	cache := mgr.GetCache()
	go cache.Start(stop)
	cache.WaitForCacheSync(stop)

	log.Info("Starting the Cmd.")
	apis.AddToSchemes.AddToScheme(mgr.GetScheme())
	kubeClient := mgr.GetClient()

	m := localmetrics.NewMetricsConfig(kubeClient, log)
	m.PublishMetrics(stop)
	//http.Handle("/metrics", promhttp.Handler())
	http.Handle("/metrics", promhttp.HandlerFor(promRegistry, promhttp.HandlerOpts{}))
	http.ListenAndServe(":2112", nil)
	mgr.Start(stop)
}
