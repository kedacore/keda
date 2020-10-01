package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog"
	"k8s.io/klog/klogr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	basecmd "github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/cmd"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/provider"

	kedav1alpha1 "github.com/kedacore/keda/api/v1alpha1"
	prommetrics "github.com/kedacore/keda/pkg/metrics"
	kedaprovider "github.com/kedacore/keda/pkg/provider"
	"github.com/kedacore/keda/pkg/scaling"
	"github.com/kedacore/keda/version"
)

// Adapter creates External Metrics Provider
type Adapter struct {
	basecmd.AdapterBase

	// Message is printed on successful startup
	Message string
}

var logger = klogr.New().WithName("keda_metrics_adapter")

var (
	prometheusMetricsPort int
	prometheusMetricsPath string
)

func (a *Adapter) makeProviderOrDie() provider.MetricsProvider {

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		logger.Error(err, "failed to get the config")
		os.Exit(1)
	}

	scheme := scheme.Scheme
	if err := appsv1.SchemeBuilder.AddToScheme(scheme); err != nil {
		logger.Error(err, "failed to add apps/v1 scheme to runtime scheme")
		os.Exit(1)
	}
	if err := kedav1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		logger.Error(err, "failed to add keda scheme to runtime scheme")
		os.Exit(1)
	}

	kubeclient, err := client.New(cfg, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		logger.Error(err, "unable to construct new client")
		os.Exit(1)
	}

	handler := scaling.NewScaleHandler(kubeclient, nil, scheme)

	namespace, err := getWatchNamespace()
	if err != nil {
		logger.Error(err, "failed to get watch namespace")
		os.Exit(1)
	}

	prometheusServer := &prommetrics.PrometheusMetricServer{}
	go func() { prometheusServer.NewServer(fmt.Sprintf(":%v", prometheusMetricsPort), prometheusMetricsPath) }()

	return kedaprovider.NewProvider(logger, handler, kubeclient, namespace)
}

func printVersion() {
	logger.Info(fmt.Sprintf("KEDA Version: %s", version.Version))
	logger.Info(fmt.Sprintf("KEDA Commit: %s", version.GitCommit))
	logger.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	logger.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
}

// getWatchNamespace returns the namespace the operator should be watching for changes
func getWatchNamespace() (string, error) {
	const WatchNamespaceEnvVar = "WATCH_NAMESPACE"
	ns, found := os.LookupEnv(WatchNamespaceEnvVar)
	if !found {
		return "", fmt.Errorf("%s must be set", WatchNamespaceEnvVar)
	}
	return ns, nil
}

func main() {
	defer klog.Flush()

	printVersion()

	cmd := &Adapter{}
	cmd.Flags().StringVar(&cmd.Message, "msg", "starting adapter...", "startup message")
	cmd.Flags().AddGoFlagSet(flag.CommandLine) // make sure we get the klog flags
	cmd.Flags().IntVar(&prometheusMetricsPort, "metrics-port", 9022, "Set the port to expose prometheus metrics")
	cmd.Flags().StringVar(&prometheusMetricsPath, "metrics-path", "/metrics", "Set the path for the prometheus metrics endpoint")
	cmd.Flags().Parse(os.Args)

	kedaProvider := cmd.makeProviderOrDie()
	cmd.WithExternalMetrics(kedaProvider)

	logger.Info(cmd.Message)
	if err := cmd.Run(wait.NeverStop); err != nil {
		logger.Error(err, "unable to run external metrics adapter")
		os.Exit(1)
	}
}
