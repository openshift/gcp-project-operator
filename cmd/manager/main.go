package main

import (
	"flag"
	"fmt"
	"runtime"

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/openshift/gcp-project-operator/pkg/apis"
	"github.com/openshift/gcp-project-operator/pkg/controller"
	"github.com/openshift/gcp-project-operator/version"
)

// Change below variables to serve metrics on different host or port.
var (
	metricsHost       = "0.0.0.0"
	metricsPort int32 = 8383
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	// Add the zap command line options to the set of flags managed by the standard flag
	// library:
	rootLoggerOpts := &zap.Options{}
	rootLoggerOpts.BindFlags(flag.CommandLine)

	// Add the flags managed by the standard flag library to pflag, and then use pflag to parse
	// the command line. This way all the flags will be parsed in the same way, including the
	// flags added by zap and the Kubernetes client.
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	// Create the root logger and configure the controller the Kubernetes API and controller
	// runtime libraries to use it:
	rootLogger := zap.New(zap.UseFlagOptions(rootLoggerOpts))
	logf.SetLogger(rootLogger)
	klog.SetLogger(rootLogger)

	// Create the logger for this command:
	cmdLogger := rootLogger.WithName("cmd")

	// Print some version information:
	cmdLogger.Info(fmt.Sprintf("Operator Version: %s", version.Version))
	cmdLogger.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	cmdLogger.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))

	// Load the configuration used to talk to the API server:
	cfg, err := config.GetConfig()
	if err != nil {
		cmdLogger.Error(err, "")
		return err
	}

	// Create the controller manager:
	mgr, err := manager.New(cfg, manager.Options{
		LeaderElection:     true,
		LeaderElectionID:   "gcp-project-operator-lock",
		MapperProvider:     func(cfg *rest.Config) (meta.RESTMapper, error) { return apiutil.NewDynamicRESTMapper(cfg) },
		MetricsBindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
	})
	if err != nil {
		cmdLogger.Error(err, "")
		return err
	}

	cmdLogger.Info("Add api scheme to Manager")
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		cmdLogger.Error(err, "")
		return err
	}

	cmdLogger.Info("Add controllers to Manager")
	if err := controller.AddToManager(mgr); err != nil {
		cmdLogger.Error(err, "")
		return err
	}

	cmdLogger.Info("Starting the 'gcp-project-operator' Reconcile loop")
	return mgr.Start(signals.SetupSignalHandler())
}
