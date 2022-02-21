package main

import (
	"flag"
	"fmt"
	"runtime"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)

	"k8s.io/apimachinery/pkg/api/meta"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"

	"github.com/openshift/gcp-project-operator/pkg/apis"
	"github.com/openshift/gcp-project-operator/pkg/controller"
	"github.com/openshift/gcp-project-operator/version"

	"github.com/spf13/pflag"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

// Change below variables to serve metrics on different host or port.
var (
	metricsHost       = "0.0.0.0"
	metricsPort int32 = 8383
)
var log = logf.Log.WithName("cmd")

func printVersion() {
	log.V(1).Info(fmt.Sprintf("Operator Version: %s", version.Version))
	log.V(1).Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.V(1).Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	// Add flags registered by imported packages (e.g. glog and
	// controller-runtime)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	pflag.Parse()

	// Use a zap logr.Logger implementation. If none of the zap
	// flags are configured (or if the zap flag set is not being
	// used), this defaults to a production zap logger.
	//
	// The logger instantiated here can be changed to any logger
	// implementing the logr.Logger interface. This logger will
	// be propagated through the whole operator, generating
	// uniform and structured logs.
	loggerOptions := &zap.Options{}
	loggerOptions.BindFlags(flag.CommandLine)
	logger := zap.New(zap.UseFlagOptions(loggerOptions))
	logf.SetLogger(logger)

	stopCh := signals.SetupSignalHandler()

	printVersion()

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "")
		return err
	}

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{
		Namespace:                  "", //watch all namespaces
		LeaderElection:             true,
		LeaderElectionResourceLock: "gcp-project-operator-lock",
		MapperProvider:             func(cfg *rest.Config) (meta.RESTMapper, error) { return apiutil.NewDynamicRESTMapper(cfg) },
		MetricsBindAddress:         fmt.Sprintf("%s:%d", metricsHost, metricsPort),
	})
	if err != nil {
		log.Error(err, "")
		return err
	}

	log.V(2).Info("Add api scheme to Manager")
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		return err
	}

	log.V(2).Info("Add controllers to Manager")
	if err := controller.AddToManager(mgr); err != nil {
		log.Error(err, "")
		return err
	}

	log.Info("Starting the 'gcp-project-operator' Reconcile loop")
	return mgr.Start(stopCh)
}
