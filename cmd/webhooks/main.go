/*
Copyright 2023 The KEDA Authors

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
	"os"

	"github.com/spf13/pflag"
	apimachineryruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/k8s"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
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

func main() {
	var metricsAddr string
	var probeAddr string
	var webhooksClientRequestQPS float32
	var webhooksClientRequestBurst int
	var certDir string
	var tlsMinVersion string
	pflag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	pflag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	pflag.Float32Var(&webhooksClientRequestQPS, "kube-api-qps", 20.0, "Set the QPS rate for throttling requests sent to the apiserver")
	pflag.IntVar(&webhooksClientRequestBurst, "kube-api-burst", 30, "Set the burst for throttling requests sent to the apiserver")
	pflag.StringVar(&certDir, "cert-dir", "/certs", "Webhook certificates dir to use. Defaults to /certs")
	pflag.StringVar(&tlsMinVersion, "tls-min-version", "1.3", "Minimum TLS version")

	opts := zap.Options{}
	opts.BindFlags(flag.CommandLine)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	ctx := ctrl.SetupSignalHandler()

	cfg := ctrl.GetConfigOrDie()
	cfg.QPS = webhooksClientRequestQPS
	cfg.Burst = webhooksClientRequestBurst

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:                 scheme,
		LeaderElection:         false,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		CertDir:                certDir,
	})
	if err != nil {
		setupLog.Error(err, "unable to start admission webhooks")
		os.Exit(1)
	}

	//+kubebuilder:scaffold:builder

	_, kubeVersion, err := k8s.InitScaleClient(mgr)
	if err != nil {
		setupLog.Error(err, "unable to init scale client")
		os.Exit(1)
	}

	kedautil.PrintWelcome(setupLog, kubeVersion, "admission webhooks")

	setupWebhook(mgr, tlsMinVersion)

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running admission webhooks")
		os.Exit(1)
	}
}

func setupWebhook(mgr manager.Manager, tlsMinVersion string) {
	// setup webhooks
	if err := (&kedav1alpha1.ScaledObject{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "ScaledObject")
		os.Exit(1)
	}

	setupLog.V(1).Info("setting up webhook server")
	hookServer := mgr.GetWebhookServer()
	hookServer.TLSMinVersion = tlsMinVersion
}
