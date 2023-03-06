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
	"os"
	"sync"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	basecmd "sigs.k8s.io/custom-metrics-apiserver/pkg/cmd"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	kedacontrollers "github.com/kedacore/keda/v2/controllers/keda"
	"github.com/kedacore/keda/v2/pkg/metricsservice"
	prommetrics "github.com/kedacore/keda/v2/pkg/prommetrics/adapter"
	kedaprovider "github.com/kedacore/keda/v2/pkg/provider"
	"github.com/kedacore/keda/v2/pkg/scaling"
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
	prometheusMetricsPort     int
	prometheusMetricsPath     string
	adapterClientRequestQPS   float32
	adapterClientRequestBurst int
	metricsAPIServerPort      int
	disableCompression        bool
	metricsServiceAddr        string
)

func (a *Adapter) makeProvider(ctx context.Context, globalHTTPTimeout time.Duration, maxConcurrentReconciles int) (provider.MetricsProvider, <-chan struct{}, error) {
	scheme := scheme.Scheme
	if err := appsv1.SchemeBuilder.AddToScheme(scheme); err != nil {
		logger.Error(err, "failed to add apps/v1 scheme to runtime scheme")
		return nil, nil, fmt.Errorf("failed to add apps/v1 scheme to runtime scheme (%s)", err)
	}
	if err := kedav1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		logger.Error(err, "failed to add keda scheme to runtime scheme")
		return nil, nil, fmt.Errorf("failed to add keda scheme to runtime scheme (%s)", err)
	}
	namespace, err := getWatchNamespace()
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

	useMetricsServiceGrpc, err := kedautil.ResolveOsEnvBool("KEDA_USE_METRICS_SERVICE_GRPC", true)
	if err != nil {
		logger.Error(err, "Invalid KEDA_USE_METRICS_SERVICE_GRPC")
		return nil, nil, fmt.Errorf("invalid KEDA_USE_METRICS_SERVICE_GRPC (%s)", err)
	}

	// Get a config to talk to the apiserver
	cfg := ctrl.GetConfigOrDie()
	cfg.QPS = adapterClientRequestQPS
	cfg.Burst = adapterClientRequestBurst
	cfg.DisableCompression = disableCompression

	metricsBindAddress := fmt.Sprintf(":%v", metricsAPIServerPort)
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		MetricsBindAddress: metricsBindAddress,
		Scheme:             scheme,
		Namespace:          namespace,
		LeaseDuration:      leaseDuration,
		RenewDeadline:      renewDeadline,
		RetryPeriod:        retryPeriod,
	})
	if err != nil {
		logger.Error(err, "failed to setup manager")
		return nil, nil, err
	}

	broadcaster := record.NewBroadcaster()
	recorder := broadcaster.NewRecorder(scheme, corev1.EventSource{Component: "keda-metrics-adapter"})

	kubeClientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		logger.Error(err, "Unable to create kube clientset")
		return nil, nil, err
	}
	objectNamespace, err := kedautil.GetClusterObjectNamespace()
	if err != nil {
		logger.Error(err, "Unable to get cluster object namespace")
		return nil, nil, err
	}
	// the namespaced kubeInformerFactory is used to restrict secret informer to only list/watch secrets in KEDA cluster object namespace,
	// refer to https://github.com/kedacore/keda/issues/3668
	kubeInformerFactory := kubeinformers.NewSharedInformerFactoryWithOptions(kubeClientset, 1*time.Hour, kubeinformers.WithNamespace(objectNamespace))
	secretInformer := kubeInformerFactory.Core().V1().Secrets()

	handler := scaling.NewScaleHandler(mgr.GetClient(), nil, scheme, globalHTTPTimeout, recorder, secretInformer.Lister())
	kubeInformerFactory.Start(ctx.Done())

	externalMetricsInfo := &[]provider.ExternalMetricInfo{}
	externalMetricsInfoLock := &sync.RWMutex{}

	prometheusServer := &prommetrics.PrometheusMetricServer{}
	go func() { prometheusServer.NewServer(fmt.Sprintf(":%v", prometheusMetricsPort), prometheusMetricsPath) }()

	stopCh := make(chan struct{})
	if err := runScaledObjectController(ctx, mgr, handler, logger, externalMetricsInfo, externalMetricsInfoLock, maxConcurrentReconciles, stopCh, secretInformer.Informer().HasSynced); err != nil {
		return nil, nil, err
	}

	logger.Info("Connecting Metrics Service gRPC client to the server", "address", metricsServiceAddr)
	grpcClient, err := metricsservice.NewGrpcClient(metricsServiceAddr, a.SecureServing.ServerCert.CertDirectory)
	if err != nil {
		logger.Error(err, "error connecting Metrics Service gRPC client to the server", "address", metricsServiceAddr)
		return nil, nil, err
	}

	return kedaprovider.NewProvider(ctx, logger, handler, mgr.GetClient(), *grpcClient, useMetricsServiceGrpc, namespace, externalMetricsInfo, externalMetricsInfoLock), stopCh, nil
}

func runScaledObjectController(ctx context.Context, mgr manager.Manager, scaleHandler scaling.ScaleHandler, logger logr.Logger, externalMetricsInfo *[]provider.ExternalMetricInfo, externalMetricsInfoLock *sync.RWMutex, maxConcurrentReconciles int, stopCh chan<- struct{}, secretSynced cache.InformerSynced) error {
	if err := (&kedacontrollers.MetricsScaledObjectReconciler{
		Client:                  mgr.GetClient(),
		ScaleHandler:            scaleHandler,
		ExternalMetricsInfo:     externalMetricsInfo,
		ExternalMetricsInfoLock: externalMetricsInfoLock,
	}).SetupWithManager(mgr, controller.Options{MaxConcurrentReconciles: maxConcurrentReconciles}); err != nil {
		return err
	}

	go func() {
		if err := mgr.Start(ctx); err != nil {
			logger.Error(err, "controller-runtime encountered an error")
			stopCh <- struct{}{}
			close(stopCh)
		}
	}()

	if ok := cache.WaitForCacheSync(ctx.Done(), secretSynced); !ok {
		return fmt.Errorf("failed to wait Secrets cache synced")
	}
	return nil
}

// generateDefaultMetricsServiceAddr generates default Metrics Service gRPC Server address based on the current Namespace.
// By default the Metrics Service gRPC Server runs in the same namespace on the keda-operator pod.
func generateDefaultMetricsServiceAddr() string {
	return fmt.Sprintf("keda-operator.%s.svc.cluster.local:9666", kedautil.GetPodNamespace())
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
	cmd.Flags().IntVar(&prometheusMetricsPort, "metrics-port", 9022, "Set the port to expose prometheus metrics")
	cmd.Flags().StringVar(&prometheusMetricsPath, "metrics-path", "/metrics", "Set the path for the prometheus metrics endpoint")
	cmd.Flags().StringVar(&metricsServiceAddr, "metrics-service-address", generateDefaultMetricsServiceAddr(), "The address of the gRPRC Metrics Service Server.")
	cmd.Flags().Float32Var(&adapterClientRequestQPS, "kube-api-qps", 20.0, "Set the QPS rate for throttling requests sent to the apiserver")
	cmd.Flags().IntVar(&adapterClientRequestBurst, "kube-api-burst", 30, "Set the burst for throttling requests sent to the apiserver")
	cmd.Flags().BoolVar(&disableCompression, "disable-compression", true, "Disable response compression for k8s restAPI in client-go. ")

	if err := cmd.Flags().Parse(os.Args); err != nil {
		return
	}

	ctrl.SetLogger(logger)

	// default to 3 seconds if they don't pass the env var
	globalHTTPTimeoutMS, err := kedautil.ResolveOsEnvInt("KEDA_HTTP_DEFAULT_TIMEOUT", 3000)
	if err != nil {
		logger.Error(err, "Invalid KEDA_HTTP_DEFAULT_TIMEOUT")
		return
	}

	controllerMaxReconciles, err := kedautil.ResolveOsEnvInt("KEDA_METRICS_CTRL_MAX_RECONCILES", 1)
	if err != nil {
		logger.Error(err, "Invalid KEDA_METRICS_CTRL_MAX_RECONCILES")
		return
	}

	err = printWelcomeMsg(cmd)
	if err != nil {
		return
	}

	kedaProvider, stopCh, err := cmd.makeProvider(ctx, time.Duration(globalHTTPTimeoutMS)*time.Millisecond, controllerMaxReconciles)
	if err != nil {
		logger.Error(err, "making provider")
		return
	}
	cmd.WithExternalMetrics(kedaProvider)

	logger.Info(cmd.Message)
	if err = cmd.Run(stopCh); err != nil {
		return
	}
}
