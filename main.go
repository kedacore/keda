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
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/spf13/pflag"
	apimachineryruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	kedacontrollers "github.com/kedacore/keda/v2/controllers/keda"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
	"github.com/kedacore/keda/v2/version"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = apimachineryruntime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(kedav1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
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
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var adapterClientRequestQPS float32
	var adapterClientRequestBurst int
	pflag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	pflag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	pflag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	pflag.Float32Var(&adapterClientRequestQPS, "kube-api-qps", 20.0, "Set the QPS rate for throttling requests sent to the apiserver")
	pflag.IntVar(&adapterClientRequestBurst, "kube-api-burst", 30, "Set the burst for throttling requests sent to the apiserver")
	opts := zap.Options{}
	opts.BindFlags(flag.CommandLine)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	pflag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	namespace, err := getWatchNamespace()
	if err != nil {
		setupLog.Error(err, "failed to get watch namespace")
		os.Exit(1)
	}

	leaseDuration, err := kedautil.ResolveOsEnvDuration("KEDA_OPERATOR_LEADER_ELECTION_LEASE_DURATION")
	if err != nil {
		setupLog.Error(err, "invalid KEDA_OPERATOR_LEADER_ELECTION_LEASE_DURATION")
		os.Exit(1)
	}

	renewDeadline, err := kedautil.ResolveOsEnvDuration("KEDA_OPERATOR_LEADER_ELECTION_RENEW_DEADLINE")
	if err != nil {
		setupLog.Error(err, "invalid KEDA_OPERATOR_LEADER_ELECTION_RENEW_DEADLINE")
		os.Exit(1)
	}

	retryPeriod, err := kedautil.ResolveOsEnvDuration("KEDA_OPERATOR_LEADER_ELECTION_RETRY_PERIOD")
	if err != nil {
		setupLog.Error(err, "invalid KEDA_OPERATOR_LEADER_ELECTION_RETRY_PERIOD")
		os.Exit(1)
	}

	cfg := ctrl.GetConfigOrDie()
	cfg.QPS = adapterClientRequestQPS
	cfg.Burst = adapterClientRequestBurst

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "operator.keda.sh",
		LeaseDuration:          leaseDuration,
		RenewDeadline:          renewDeadline,
		RetryPeriod:            retryPeriod,
		Namespace:              namespace,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// default to 3 seconds if they don't pass the env var
	globalHTTPTimeoutMS, err := kedautil.ResolveOsEnvInt("KEDA_HTTP_DEFAULT_TIMEOUT", 3000)
	if err != nil {
		setupLog.Error(err, "Invalid KEDA_HTTP_DEFAULT_TIMEOUT")
		os.Exit(1)
	}

	scaledObjectMaxReconciles, err := kedautil.ResolveOsEnvInt("KEDA_SCALEDOBJECT_CTRL_MAX_RECONCILES", 5)
	if err != nil {
		setupLog.Error(err, "Invalid KEDA_SCALEDOBJECT_CTRL_MAX_RECONCILES")
		os.Exit(1)
	}

	scaledJobMaxReconciles, err := kedautil.ResolveOsEnvInt("KEDA_SCALEDJOB_CTRL_MAX_RECONCILES", 1)
	if err != nil {
		setupLog.Error(err, "Invalid KEDA_SCALEDJOB_CTRL_MAX_RECONCILES")
		os.Exit(1)
	}

	globalHTTPTimeout := time.Duration(globalHTTPTimeoutMS) * time.Millisecond
	eventRecorder := mgr.GetEventRecorderFor("keda-operator")

	if err = (&kedacontrollers.ScaledObjectReconciler{
		Client:            mgr.GetClient(),
		Scheme:            mgr.GetScheme(),
		GlobalHTTPTimeout: globalHTTPTimeout,
		Recorder:          eventRecorder,
	}).SetupWithManager(mgr, controller.Options{MaxConcurrentReconciles: scaledObjectMaxReconciles}); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ScaledObject")
		os.Exit(1)
	}
	if err = (&kedacontrollers.ScaledJobReconciler{
		Client:            mgr.GetClient(),
		Scheme:            mgr.GetScheme(),
		GlobalHTTPTimeout: globalHTTPTimeout,
		Recorder:          eventRecorder,
	}).SetupWithManager(mgr, controller.Options{MaxConcurrentReconciles: scaledJobMaxReconciles}); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ScaledJob")
		os.Exit(1)
	}
	if err = (&kedacontrollers.TriggerAuthenticationReconciler{
		Client:        mgr.GetClient(),
		EventRecorder: eventRecorder,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "TriggerAuthentication")
		os.Exit(1)
	}
	if err = (&kedacontrollers.ClusterTriggerAuthenticationReconciler{
		Client:        mgr.GetClient(),
		EventRecorder: eventRecorder,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ClusterTriggerAuthentication")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("Starting manager")
	setupLog.Info(fmt.Sprintf("KEDA Version: %s", version.Version))
	setupLog.Info(fmt.Sprintf("Git Commit: %s", version.GitCommit))
	setupLog.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	setupLog.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
