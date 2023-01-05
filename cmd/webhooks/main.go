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
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/open-policy-agent/cert-controller/pkg/rotator"
	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachineryruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/k8s"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
	"github.com/kedacore/keda/v2/version"
	//+kubebuilder:scaffold:imports
)

var webhooks = []rotator.WebhookInfo{
	{
		Name: "keda-admission",
		Type: rotator.Validating,
	},
}

var (
	scheme         = apimachineryruntime.NewScheme()
	setupLog       = ctrl.Log.WithName("setup")
	serviceName    = "keda-admission-webhooks"
	caName         = "kedaorg-ca"
	caOrganization = "kedaorg"
	// DNSName is <service name>.<namespace>.svc
	dnsName = fmt.Sprintf("%s.%s.svc", serviceName, kedautil.GetPodNamespace())
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
	var webhookCertDir string
	var webhookSecretName string
	var enableCertRotation bool
	var tlsMinVersion string
	pflag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	pflag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	pflag.Float32Var(&webhooksClientRequestQPS, "kube-api-qps", 20.0, "Set the QPS rate for throttling requests sent to the apiserver")
	pflag.IntVar(&webhooksClientRequestBurst, "kube-api-burst", 30, "Set the burst for throttling requests sent to the apiserver")
	pflag.StringVar(&webhookCertDir, "webhooks-cert-dir", "/certs", "Webhook certificates dir to use. Defaults to /certs")
	pflag.StringVar(&webhookSecretName, "webhooks-cert-secret-name", "kedaorg-admission-webhooks-certs", "Webhook certificates secret name. Defaults to kedaorg-admission-webhooks-certs")
	pflag.BoolVar(&enableCertRotation, "enable-cert-rotation", false, "enable automatic generation and rotation of webhook TLS certificates/keys")
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
		CertDir:                webhookCertDir,
	})
	if err != nil {
		setupLog.Error(err, "unable to start admission webhooks")
		os.Exit(1)
	}

	// Make sure certs are generated and valid if cert rotation is enabled.
	setupFinished := make(chan struct{})
	if enableCertRotation {
		ensureSecret(ctx, mgr, webhookSecretName)
		setupLog.V(1).Info("setting up cert rotation")
		if err := rotator.AddRotator(mgr, &rotator.CertRotator{
			SecretKey: types.NamespacedName{
				Namespace: kedautil.GetPodNamespace(),
				Name:      webhookSecretName,
			},
			CertDir:                webhookCertDir,
			CAName:                 caName,
			CAOrganization:         caOrganization,
			DNSName:                dnsName,
			IsReady:                setupFinished,
			Webhooks:               webhooks,
			RestartOnSecretRefresh: true,
		}); err != nil {
			setupLog.Error(err, "unable to set up cert rotation")
			os.Exit(1)
		}
	} else {
		close(setupFinished)
	}

	//+kubebuilder:scaffold:builder

	_, kubeVersion, err := k8s.InitScaleClient(mgr)
	if err != nil {
		setupLog.Error(err, "unable to init scale client")
		os.Exit(1)
	}

	setupLog.Info("Starting admission webhooks")
	setupLog.Info(fmt.Sprintf("KEDA Version: %s", version.Version))
	setupLog.Info(fmt.Sprintf("Git Commit: %s", version.GitCommit))
	setupLog.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	setupLog.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	setupLog.Info(fmt.Sprintf("Running on Kubernetes %s", kubeVersion.PrettyVersion), "version", kubeVersion.Version)

	go setupWebhook(mgr, tlsMinVersion, setupFinished)

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

func ensureSecret(ctx context.Context, mgr manager.Manager, secretName string) {
	secrets := &corev1.SecretList{}
	kedaNamespace := kedautil.GetPodNamespace()
	opt := &client.ListOptions{
		Namespace: kedaNamespace,
	}

	err := mgr.GetAPIReader().List(ctx, secrets, opt)
	if err != nil {
		setupLog.Error(err, "unable to check secrets")
		os.Exit(1)
	}

	exists := false
	for _, secret := range secrets.Items {
		if secret.Name == secretName {
			exists = true
			break
		}
	}
	if !exists {
		secret := &corev1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Name:      secretName,
				Namespace: kedaNamespace,
				Labels: map[string]string{
					"app":                         "keda-admission-webhooks",
					"app.kubernetes.io/name":      "keda-admission-webhooks",
					"app.kubernetes.io/component": "admission-webhooks",
					"app.kubernetes.io/part-of":   "keda",
				},
			},
		}
		err = mgr.GetClient().Create(ctx, secret)
		if err != nil {
			setupLog.Error(err, "unable to create certificates secret")
			os.Exit(1)
		}
		setupLog.V(1).Info(fmt.Sprintf("created the secret %s to store cert-controller certificates", secretName))
	}
}

func setupWebhook(mgr manager.Manager, tlsMinVersion string, setupFinished chan struct{}) {
	// Block until the setup (certificate generation) finishes.
	<-setupFinished

	// setup webhooks
	if err := (&kedav1alpha1.ScaledObject{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "ScaledObject")
		os.Exit(1)
	}

	setupLog.V(1).Info("setting up webhook server")
	hookServer := mgr.GetWebhookServer()
	hookServer.TLSMinVersion = tlsMinVersion
}
