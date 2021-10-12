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
	"runtime"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	openapinamer "k8s.io/apiserver/pkg/endpoints/openapi"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	basecmd "sigs.k8s.io/custom-metrics-apiserver/pkg/cmd"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"

	generatedopenapi "github.com/kedacore/keda/v2/adapter/generated/openapi"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	kedacontrollers "github.com/kedacore/keda/v2/controllers/keda"
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
	prometheusMetricsPort     int
	prometheusMetricsPath     string
	adapterClientRequestQPS   float32
	adapterClientRequestBurst int
)

func (a *Adapter) makeProvider(globalHTTPTimeout time.Duration) (provider.MetricsProvider, <-chan struct{}, error) {
	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if cfg != nil {
		cfg.QPS = adapterClientRequestQPS
		cfg.Burst = adapterClientRequestBurst
	}

	if err != nil {
		logger.Error(err, "failed to get the config")
		return nil, nil, fmt.Errorf("failed to get the config (%s)", err)
	}

	scheme := scheme.Scheme
	if err := appsv1.SchemeBuilder.AddToScheme(scheme); err != nil {
		logger.Error(err, "failed to add apps/v1 scheme to runtime scheme")
		return nil, nil, fmt.Errorf("failed to add apps/v1 scheme to runtime scheme (%s)", err)
	}
	if err := kedav1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		logger.Error(err, "failed to add keda scheme to runtime scheme")
		return nil, nil, fmt.Errorf("failed to add keda scheme to runtime scheme (%s)", err)
	}

	kubeclient, err := client.New(cfg, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		logger.Error(err, "unable to construct new client")
		return nil, nil, fmt.Errorf("unable to construct new client (%s)", err)
	}

	broadcaster := record.NewBroadcaster()
	recorder := broadcaster.NewRecorder(scheme, corev1.EventSource{Component: "keda-metrics-adapter"})
	handler := scaling.NewScaleHandler(kubeclient, nil, scheme, globalHTTPTimeout, recorder)

	namespace, err := getWatchNamespace()
	if err != nil {
		logger.Error(err, "failed to get watch namespace")
		return nil, nil, fmt.Errorf("failed to get watch namespace (%s)", err)
	}

	prometheusServer := &prommetrics.PrometheusMetricServer{}
	go func() { prometheusServer.NewServer(fmt.Sprintf(":%v", prometheusMetricsPort), prometheusMetricsPath) }()
	stopCh := make(chan struct{})
	if err := runScaledObjectController(scheme, namespace, handler, logger, stopCh); err != nil {
		return nil, nil, err
	}

	return kedaprovider.NewProvider(logger, handler, kubeclient, namespace), stopCh, nil
}

func runScaledObjectController(scheme *k8sruntime.Scheme, namespace string, scaleHandler scaling.ScaleHandler, logger logr.Logger, stopCh chan<- struct{}) error {
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:    scheme,
		Namespace: namespace,
	})
	if err != nil {
		return err
	}

	if err := (&kedacontrollers.MetricsScaledObjectReconciler{
		ScaleHandler: scaleHandler,
	}).SetupWithManager(mgr, controller.Options{}); err != nil {
		return err
	}

	go func() {
		if err := mgr.Start(context.Background()); err != nil {
			logger.Error(err, "controller-runtime encountered an error")
			stopCh <- struct{}{}
			close(stopCh)
		}
	}()

	return nil
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

	cmd.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(generatedopenapi.GetOpenAPIDefinitions, openapinamer.NewDefinitionNamer(scheme.Scheme))
	cmd.OpenAPIConfig.Info.Title = "keda-adapter"
	cmd.OpenAPIConfig.Info.Version = "1.0.0"

	cmd.Flags().StringVar(&cmd.Message, "msg", "starting adapter...", "startup message")
	cmd.Flags().AddGoFlagSet(flag.CommandLine) // make sure we get the klog flags
	cmd.Flags().IntVar(&prometheusMetricsPort, "metrics-port", 9022, "Set the port to expose prometheus metrics")
	cmd.Flags().StringVar(&prometheusMetricsPath, "metrics-path", "/metrics", "Set the path for the prometheus metrics endpoint")
	cmd.Flags().Float32Var(&adapterClientRequestQPS, "kube-api-qps", 20.0, "Set the QPS rate for throttling requests sent to the apiserver")
	cmd.Flags().IntVar(&adapterClientRequestBurst, "kube-api-burst", 30, "Set the burst for throttling requests sent to the apiserver")
	if err := cmd.Flags().Parse(os.Args); err != nil {
		return
	}

	ctrl.SetLogger(logger)

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

	kedaProvider, stopCh, err := cmd.makeProvider(time.Duration(globalHTTPTimeoutMS) * time.Millisecond)
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
