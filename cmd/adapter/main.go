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

	"github.com/prometheus/client_golang/prometheus/collectors"
	_ "go.uber.org/automaxprocs"
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

var logger = klogr.New().WithName("keda_metrics_adapter")

var (
	adapterClientRequestQPS             float32
	adapterClientRequestBurst           int
	metricsAPIServerPort                int
	disableCompression                  bool
	metricsServiceAddr                  string
	profilingAddr                       string
	insecureMetricsServiceSkipTLSVerify bool
)

func (a *Adapter) makeProvider(ctx context.Context) (provider.ExternalMetricsProvider, <-chan struct{}, error) {
	scheme := scheme.Scheme
	if err := appsv1.SchemeBuilder.AddToScheme(scheme); err != nil {
		logger.Error(err, "failed to add apps/v1 scheme to runtime scheme")
		return nil, nil, fmt.Errorf("failed to add apps/v1 scheme to runtime scheme (%s)", err)
	}
	if err := kedav1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		logger.Error(err, "failed to add keda scheme to runtime scheme")
		return nil, nil, fmt.Errorf("failed to add keda scheme to runtime scheme (%s)", err)
	}
	namespaces, err := kedautil.GetWatchNamespaces()
	if err != nil {
		logger.Error(err, "failed to get watch namespace")
		return nil, nil, fmt.Errorf("failed to get watch namespace (%s)", err)
	}

	leaseDuration, err := kedautil.ResolveOsEnvDuration("KEDA_METRICS_LEADER_ELECTION_LEASE_DURATION")
	if err != nil {
		logger.Error(err, "invalid KEDA_METRICS_LEADER_ELECTION_LEASE_DURATION")
		return nil, nil, fmt.Errorf("invalid KEDA_METRICS_LEADER_ELECTION_LEASE_DURATION (%s)", err)
	}

	renewDeadline, err := kedautil.ResolveOsEnvDuration("KEDA_METRICS_LEADER_ELECTION_RENEW_DEADLINE")
	if err != nil {
		logger.Error(err, "Invalid KEDA_METRICS_LEADER_ELECTION_RENEW_DEADLINE")
		return nil, nil, fmt.Errorf("invalid KEDA_METRICS_LEADER_ELECTION_RENEW_DEADLINE (%s)", err)
	}

	retryPeriod, err := kedautil.ResolveOsEnvDuration("KEDA_METRICS_LEADER_ELECTION_RETRY_PERIOD")
	if err != nil {
		logger.Error(err, "Invalid KEDA_METRICS_LEADER_ELECTION_RETRY_PERIOD")
		return nil, nil, fmt.Errorf("invalid KEDA_METRICS_LEADER_ELECTION_RETRY_PERIOD (%s)", err)
	}

	// Get a config to talk to the apiserver
	cfg := ctrl.GetConfigOrDie()
	cfg.QPS = adapterClientRequestQPS
	cfg.Burst = adapterClientRequestBurst
	cfg.DisableCompression = disableCompression

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Metrics: server.Options{
			BindAddress: "0", // disabled since we use our own server to serve metrics
		},
		Scheme: scheme,
		Cache: ctrlcache.Options{
			DefaultNamespaces: namespaces,
		},
		PprofBindAddress: profilingAddr,
		LeaseDuration:    leaseDuration,
		RenewDeadline:    renewDeadline,
		RetryPeriod:      retryPeriod,
	})
	if err != nil {
		logger.Error(err, "failed to setup manager")
		return nil, nil, err
	}

	logger.Info("Connecting Metrics Service gRPC client to the server", "address", metricsServiceAddr)
	grpcClient, err := metricsservice.NewGrpcClient(metricsServiceAddr, a.SecureServing.ServerCert.CertDirectory, insecureMetricsServiceSkipTLSVerify)
	if err != nil {
		logger.Error(err, "error connecting Metrics Service gRPC client to the server", "address", metricsServiceAddr)
		return nil, nil, err
	}
	stopCh := make(chan struct{})
	go func() {
		if err := mgr.Start(ctx); err != nil {
			logger.Error(err, "controller-runtime encountered an error")
			stopCh <- struct{}{}
			close(stopCh)
		}
	}()
	return kedaprovider.NewProvider(ctx, logger, mgr.GetClient(), *grpcClient), stopCh, nil
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

// RunMetricsServer runs a http listener and handles the /metrics endpoint
// this is needed to consolidate apiserver and controller-runtime metrics
// we have to use a separate http server & can't rely on the controller-runtime implementation
// because apiserver doesn't provide a way to register metrics to other prometheus registries
func RunMetricsServer(ctx context.Context, stopCh <-chan struct{}) {
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
		<-stopCh
		logger.Info("Shutting down the /metrics server gracefully...")

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
	cmd.Flags().StringVar(&metricsServiceAddr, "metrics-service-address", generateDefaultMetricsServiceAddr(), "The address of the gRPRC Metrics Service Server.")
	cmd.Flags().StringVar(&profilingAddr, "profiling-bind-address", "", "The address the profiling would be exposed on.")
	cmd.Flags().Float32Var(&adapterClientRequestQPS, "kube-api-qps", 20.0, "Set the QPS rate for throttling requests sent to the apiserver")
	cmd.Flags().IntVar(&adapterClientRequestBurst, "kube-api-burst", 30, "Set the burst for throttling requests sent to the apiserver")
	cmd.Flags().BoolVar(&disableCompression, "disable-compression", true, "Disable response compression for k8s restAPI in client-go. ")
	cmd.Flags().BoolVar(&insecureMetricsServiceSkipTLSVerify, "insecure-metrics-service-skip-tls-verify", false, "Skip TLS verification on the GRPC connection to the metrics service")

	if err := cmd.Flags().Parse(os.Args); err != nil {
		return
	}

	ctrl.SetLogger(logger)

	err = printWelcomeMsg(cmd)
	if err != nil {
		return
	}

	kedaProvider, stopCh, err := cmd.makeProvider(ctx)
	if err != nil {
		logger.Error(err, "making provider")
		return
	}
	cmd.WithExternalMetrics(kedaProvider)

	logger.Info(cmd.Message)

	RunMetricsServer(ctx, stopCh)

	if err = cmd.Run(stopCh); err != nil {
		return
	}
}
