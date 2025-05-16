/*
Copyright 2021 The KEDA Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"

	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	appsv1 "k8s.io/api/apps/v1"
	apimetrics "k8s.io/apiserver/pkg/endpoints/metrics"
	"k8s.io/client-go/kubernetes/scheme"
	kubemetrics "k8s.io/component-base/metrics"
	"k8s.io/component-base/metrics/legacyregistry"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlcache "sigs.k8s.io/controller-runtime/pkg/cache"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	basecmd "sigs.k8s.io/custom-metrics-apiserver/pkg/cmd"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/metricsservice"
	kedaprovider "github.com/kedacore/keda/v2/pkg/provider"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

// Adapter creates External Metrics Provider
type Adapter struct {
	basecmd.AdapterBase

	// Message is printed on successful startup
	Message string
}

// https://github.com/kedacore/keda/issues/5732
//
//nolint:staticcheck // SA1019: klogr.New is deprecated.
var logger = klogr.New().WithName("keda_metrics_adapter")

var (
	adapterClientRequestQPS     float32
	adapterClientRequestBurst   int
	metricsAPIServerPort        int
	disableCompression          bool
	metricsServiceAddr          string
	profilingAddr               string
	metricsServiceGRPCAuthority string
)

func (a *Adapter) makeProvider(ctx context.Context) (provider.ExternalMetricsProvider, error) {
	scheme := scheme.Scheme
	if err := appsv1.SchemeBuilder.AddToScheme(scheme); err != nil {
		logger.Error(err, "failed to add apps/v1 scheme to runtime scheme")
		return nil, fmt.Errorf("failed to add apps/v1 scheme to runtime scheme (%s)", err)
	}
	if err := kedav1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		logger.Error(err, "failed to add keda scheme to runtime scheme")
		return nil, fmt.Errorf("failed to add keda scheme to runtime scheme (%s)", err)
	}
	namespaces, err := kedautil.GetWatchNamespaces()
	if err != nil {
		logger.Error(err, "failed to get watch namespace")
		return nil, fmt.Errorf("failed to get watch namespace (%s)", err)
	}

	// Get a config to talk to the apiserver
	cfg := ctrl.GetConfigOrDie()
	cfg.QPS = adapterClientRequestQPS
	cfg.Burst = adapterClientRequestBurst
	cfg.DisableCompression = disableCompression

	clientMetrics := getMetricInterceptor()

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Metrics: server.Options{
			BindAddress: "0", // disabled since we use our own server to serve metrics
		},
		Scheme: scheme,
		Cache: ctrlcache.Options{
			DefaultNamespaces: namespaces,
		},
		PprofBindAddress: profilingAddr,
	})
	if err != nil {
		logger.Error(err, "failed to setup manager")
		return nil, err
	}

	logger.Info("Connecting Metrics Service gRPC client to the server", "address", metricsServiceAddr)
	grpcClient, err := metricsservice.NewGrpcClient(ctx, metricsServiceAddr, a.SecureServing.ServerCert.CertDirectory, metricsServiceGRPCAuthority, clientMetrics)
	if err != nil {
		logger.Error(err, "error connecting Metrics Service gRPC client to the server", "address", metricsServiceAddr)
		return nil, err
	}
	go func() {
		if err := mgr.Start(ctx); err != nil {
			logger.Error(err, "controller-runtime encountered an error")
			os.Exit(1)
		}
	}()
	return kedaprovider.NewProvider(ctx, logger, mgr.GetClient(), *grpcClient), nil
}

// getMetricHandler returns a http handler that exposes metrics from controller-runtime and apiserver
func getMetricHandler() http.HandlerFunc {
	// Register apiserver metrics in legacy registry
	// this contains the apiserver_* metrics
	apimetrics.Register()

	// unregister duplicate collectors that are already handled by controller-runtime's registry
	legacyregistry.Registerer().Unregister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	legacyregistry.Registerer().Unregister(collectors.NewGoCollector(collectors.WithGoCollectorRuntimeMetrics(collectors.MetricsAll)))

	// Return handler that serves metrics from both legacy and controller-runtime registry
	return func(w http.ResponseWriter, req *http.Request) {
		legacyregistry.Handler().ServeHTTP(w, req)

		kubemetrics.HandlerFor(ctrlmetrics.Registry, kubemetrics.HandlerOpts{}).ServeHTTP(w, req)
	}
}

// getMetricInterceptor returns a metrics inceptor that records metrics between the adapter and opertaor
func getMetricInterceptor() *grpcprom.ClientMetrics {
	metricsNamespace := "keda_internal_metricsservice"

	counterNamespace := func(o *prometheus.CounterOpts) {
		o.Namespace = metricsNamespace
	}

	histogramNamespace := func(o *prometheus.HistogramOpts) {
		o.Namespace = metricsNamespace
	}

	clientMetrics := grpcprom.NewClientMetrics(
		grpcprom.WithClientHandlingTimeHistogram(
			grpcprom.WithHistogramBuckets([]float64{0.001, 0.01, 0.1, 0.3, 0.6, 1, 3, 6, 9, 20, 30, 60, 90, 120}),
			histogramNamespace,
		),
		grpcprom.WithClientCounterOptions(counterNamespace),
	)
	legacyregistry.Registerer().MustRegister(clientMetrics)

	return clientMetrics
}

// RunMetricsServer runs a http listener and handles the /metrics endpoint
// this is needed to consolidate apiserver and controller-runtime metrics
// we have to use a separate http server & can't rely on the controller-runtime implementation
// because apiserver doesn't provide a way to register metrics to other prometheus registries
func RunMetricsServer(ctx context.Context) {
	h := getMetricHandler()
	mux := http.NewServeMux()
	mux.Handle("/metrics", h)
	metricsBindAddress := fmt.Sprintf(":%v", metricsAPIServerPort)

	server := &http.Server{
		Addr:    metricsBindAddress,
		Handler: mux,
	}

	go func() {
		logger.Info("starting /metrics server endpoint")
		// nosemgrep: use-tls
		err := server.ListenAndServe()
		if err != http.ErrServerClosed {
			panic(err)
		}
	}()

	go func() {
		<-ctx.Done()
		if err := server.Shutdown(ctx); err != nil {
			logger.Error(err, "http server shutdown error")
		}
	}()
}

// generateDefaultMetricsServiceAddr generates default Metrics Service gRPC Server address based on the current Namespace.
// By default the Metrics Service gRPC Server runs in the same namespace on the keda-operator pod.
func generateDefaultMetricsServiceAddr() string {
	return fmt.Sprintf("keda-operator.%s.svc.cluster.local:9666", kedautil.GetPodNamespace())
}

// printWelcomeMsg prints welcome message during the start of the adater
func printWelcomeMsg(cmd *Adapter) error {
	clientset, err := cmd.DiscoveryClient()
	if err != nil {
		logger.Error(err, "not able to get Kubernetes version")
		return err
	}
	version, err := clientset.ServerVersion()
	if err != nil {
		logger.Error(err, "not able to get Kubernetes version")
		return err
	}
	kedautil.PrintWelcome(logger, kedautil.NewK8sVersion(version), "metrics server")

	return nil
}

func main() {
	ctx := ctrl.SetupSignalHandler()
	var err error
	defer func() {
		if err != nil {
			logger.Error(err, "unable to run external metrics adapter")
		}
	}()

	defer klog.Flush()
	klog.InitFlags(nil)

	cmd := &Adapter{}
	cmd.Name = "keda-adapter"

	cmd.Flags().StringVar(&cmd.Message, "msg", "starting adapter...", "startup message")
	cmd.Flags().AddGoFlagSet(flag.CommandLine) // make sure we get the klog flags
	cmd.Flags().IntVar(&metricsAPIServerPort, "port", 8080, "Set the port for the metrics API server")
	cmd.Flags().StringVar(&metricsServiceAddr, "metrics-service-address", generateDefaultMetricsServiceAddr(), "The address of the GRPC Metrics Service Server.")
	cmd.Flags().StringVar(&metricsServiceGRPCAuthority, "metrics-service-grpc-authority", "", "Host Authority override for the Metrics Service if the Host Authority is not the same as the address used for the GRPC Metrics Service Server.")
	cmd.Flags().StringVar(&profilingAddr, "profiling-bind-address", "", "The address the profiling would be exposed on.")
	cmd.Flags().Float32Var(&adapterClientRequestQPS, "kube-api-qps", 20.0, "Set the QPS rate for throttling requests sent to the apiserver")
	cmd.Flags().IntVar(&adapterClientRequestBurst, "kube-api-burst", 30, "Set the burst for throttling requests sent to the apiserver")
	cmd.Flags().BoolVar(&disableCompression, "disable-compression", true, "Disable response compression for k8s restAPI in client-go. ")

	if err := cmd.Flags().Parse(os.Args); err != nil {
		return
	}

	ctrl.SetLogger(logger)

	err = printWelcomeMsg(cmd)
	if err != nil {
		return
	}

	err = kedautil.ConfigureMaxProcs(logger)
	if err != nil {
		logger.Error(err, "failed to set max procs")
		return
	}

	kedaProvider, err := cmd.makeProvider(ctx)
	if err != nil {
		logger.Error(err, "making provider")
		return
	}
	cmd.WithExternalMetrics(kedaProvider)

	logger.Info(cmd.Message)

	RunMetricsServer(ctx)

	if err = cmd.Run(ctx); err != nil {
		return
	}
}
