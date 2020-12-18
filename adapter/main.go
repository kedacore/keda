package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog"
	"k8s.io/klog/klogr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	basecmd "github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/cmd"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/provider"

	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
	prommetrics "github.com/kedacore/keda/v2/pkg/metrics"
	kedaprovider "github.com/kedacore/keda/v2/pkg/provider"
	"github.com/kedacore/keda/v2/pkg/scaling"
	"github.com/kedacore/keda/v2/version"
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

func (a *Adapter) makeProvider(globalHTTPTimeout time.Duration) (provider.MetricsProvider, error) {
	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		logger.Error(err, "failed to get the config")
		return nil, fmt.Errorf("failed to get the config (%s)", err)
	}

	scheme := scheme.Scheme
	if err := appsv1.SchemeBuilder.AddToScheme(scheme); err != nil {
		logger.Error(err, "failed to add apps/v1 scheme to runtime scheme")
		return nil, fmt.Errorf("failed to add apps/v1 scheme to runtime scheme (%s)", err)
	}
	if err := kedav1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		logger.Error(err, "failed to add keda scheme to runtime scheme")
		return nil, fmt.Errorf("failed to add keda scheme to runtime scheme (%s)", err)
	}

	kubeclient, err := client.New(cfg, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		logger.Error(err, "unable to construct new client")
		return nil, fmt.Errorf("unable to construct new client (%s)", err)
	}

	handler := scaling.NewScaleHandler(kubeclient, nil, scheme, globalHTTPTimeout)

	namespace, err := getWatchNamespace()
	if err != nil {
		logger.Error(err, "failed to get watch namespace")
		return nil, fmt.Errorf("failed to get watch namespace (%s)", err)
	}

	prometheusServer := &prommetrics.PrometheusMetricServer{}
	go func() { prometheusServer.NewServer(fmt.Sprintf(":%v", prometheusMetricsPort), prometheusMetricsPath) }()

	return kedaprovider.NewProvider(logger, handler, kubeclient, namespace), nil
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
	var err error
	defer func() {
		if err != nil {
			logger.Error(err, "unable to run external metrics adapter")
		}
	}()

	defer klog.Flush()

	printVersion()

	cmd := &Adapter{}
	cmd.Flags().StringVar(&cmd.Message, "msg", "starting adapter...", "startup message")
	cmd.Flags().AddGoFlagSet(flag.CommandLine) // make sure we get the klog flags
	cmd.Flags().IntVar(&prometheusMetricsPort, "metrics-port", 9022, "Set the port to expose prometheus metrics")
	cmd.Flags().StringVar(&prometheusMetricsPath, "metrics-path", "/metrics", "Set the path for the prometheus metrics endpoint")
	if err := cmd.Flags().Parse(os.Args); err != nil {
		return
	}

	globalHTTPTimeoutStr := os.Getenv("KEDA_HTTP_DEFAULT_TIMEOUT")
	if globalHTTPTimeoutStr == "" {
		// default to 3 seconds if they don't pass the env var
		globalHTTPTimeoutStr = "3000"
	}

	globalHTTPTimeoutMS, err := strconv.Atoi(globalHTTPTimeoutStr)
	if err != nil {
		logger.Error(err, "Invalid KEDA_HTTP_DEFAULT_TIMEOUT")
		return
	}

	kedaProvider, err := cmd.makeProvider(time.Duration(globalHTTPTimeoutMS) * time.Millisecond)
	if err != nil {
		logger.Error(err, "making provider")
		return
	}
	cmd.WithExternalMetrics(kedaProvider)

	logger.Info(cmd.Message)
	if err = cmd.Run(wait.NeverStop); err != nil {
		return
	}
}
