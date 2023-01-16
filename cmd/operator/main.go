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
	"time"

	"github.com/spf13/pflag"
	apimachineryruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	kedacontrollers "github.com/kedacore/keda/v2/controllers/keda"
	"github.com/kedacore/keda/v2/pkg/certificates"
	"github.com/kedacore/keda/v2/pkg/k8s"
	"github.com/kedacore/keda/v2/pkg/metricsservice"
	"github.com/kedacore/keda/v2/pkg/scaling"
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
	var probeAddr string
	var metricsServiceAddr string
	var enableLeaderElection bool
	var adapterClientRequestQPS float32
	var adapterClientRequestBurst int
	var disableCompression bool
	var certSecretName string
	var certDir string
	var operatorServiceName string
	var metricsServerServiceName string
	var webhooksServiceName string
	var enableCertRotation bool
	var validatingWebhookName string
	pflag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	pflag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	pflag.StringVar(&metricsServiceAddr, "metrics-service-bind-address", ":9666", "The address the gRPRC Metrics Service endpoint binds to.")
	pflag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	pflag.Float32Var(&adapterClientRequestQPS, "kube-api-qps", 20.0, "Set the QPS rate for throttling requests sent to the apiserver")
	pflag.IntVar(&adapterClientRequestBurst, "kube-api-burst", 30, "Set the burst for throttling requests sent to the apiserver")
	pflag.BoolVar(&disableCompression, "disable-compression", true, "Disable response compression for k8s restAPI in client-go. ")
	pflag.StringVar(&certSecretName, "cert-secret-name", "kedaorg-certs", "KEDA certificates secret name. Defaults to kedaorg-certs")
	pflag.StringVar(&certDir, "cert-dir", "/certs", "Webhook certificates dir to use. Defaults to /certs")
	pflag.StringVar(&operatorServiceName, "operator-service-name", "keda-operator", "Operator service name. Defaults to keda-operator")
	pflag.StringVar(&metricsServerServiceName, "metrics-server-service-name", "keda-metrics-apiserver", "Metrics server service name. Defaults to keda-metrics-apiserver")
	pflag.StringVar(&webhooksServiceName, "webhooks-service-name", "keda-admission-webhooks", "Webhook service name. Defaults to keda-admission-webhooks")
	pflag.BoolVar(&enableCertRotation, "enable-cert-rotation", false, "enable automatic generation and rotation of TLS certificates/keys")
	pflag.StringVar(&validatingWebhookName, "validating-webhook-name", "keda-admission", "ValidatingWebhookConfiguration name. Defaults to keda-admission")
	opts := zap.Options{}
	opts.BindFlags(flag.CommandLine)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	ctx := ctrl.SetupSignalHandler()
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
	cfg.DisableCompression = disableCompression

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
		setupLog.Error(err, "invalid KEDA_HTTP_DEFAULT_TIMEOUT")
		os.Exit(1)
	}

	scaledObjectMaxReconciles, err := kedautil.ResolveOsEnvInt("KEDA_SCALEDOBJECT_CTRL_MAX_RECONCILES", 5)
	if err != nil {
		setupLog.Error(err, "invalid KEDA_SCALEDOBJECT_CTRL_MAX_RECONCILES")
		os.Exit(1)
	}

	scaledJobMaxReconciles, err := kedautil.ResolveOsEnvInt("KEDA_SCALEDJOB_CTRL_MAX_RECONCILES", 1)
	if err != nil {
		setupLog.Error(err, "invalid KEDA_SCALEDJOB_CTRL_MAX_RECONCILES")
		os.Exit(1)
	}

	globalHTTPTimeout := time.Duration(globalHTTPTimeoutMS) * time.Millisecond
	eventRecorder := mgr.GetEventRecorderFor("keda-operator")

	kubeClientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		setupLog.Error(err, "Unable to create kube clientset")
		os.Exit(1)
	}
	objectNamespace, err := kedautil.GetClusterObjectNamespace()
	if err != nil {
		setupLog.Error(err, "Unable to get cluster object namespace")
		os.Exit(1)
	}
	// the namespaced kubeInformerFactory is used to restrict secret informer to only list/watch secrets in KEDA cluster object namespace,
	// refer to https://github.com/kedacore/keda/issues/3668
	kubeInformerFactory := kubeinformers.NewSharedInformerFactoryWithOptions(kubeClientset, 1*time.Hour, kubeinformers.WithNamespace(objectNamespace))
	secretInformer := kubeInformerFactory.Core().V1().Secrets()

	scaleClient, kubeVersion, err := k8s.InitScaleClient(mgr)
	if err != nil {
		setupLog.Error(err, "unable to init scale client")
		os.Exit(1)
	}

	scaledHandler := scaling.NewScaleHandler(mgr.GetClient(), scaleClient, mgr.GetScheme(), globalHTTPTimeout, eventRecorder, secretInformer.Lister())

	if err = (&kedacontrollers.ScaledObjectReconciler{
		Client:       mgr.GetClient(),
		Scheme:       mgr.GetScheme(),
		Recorder:     eventRecorder,
		ScaleClient:  scaleClient,
		ScaleHandler: scaledHandler,
	}).SetupWithManager(mgr, controller.Options{MaxConcurrentReconciles: scaledObjectMaxReconciles}); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ScaledObject")
		os.Exit(1)
	}
	if err = (&kedacontrollers.ScaledJobReconciler{
		Client:            mgr.GetClient(),
		Scheme:            mgr.GetScheme(),
		GlobalHTTPTimeout: globalHTTPTimeout,
		Recorder:          eventRecorder,
		SecretsLister:     secretInformer.Lister(),
		SecretsSynced:     secretInformer.Informer().HasSynced,
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

	certReady := make(chan struct{})
	if enableCertRotation {
		certManager := certificates.CertManager{
			SecretName:            certSecretName,
			CertDir:               certDir,
			OperatorService:       operatorServiceName,
			MetricsServerService:  metricsServerServiceName,
			WebhookService:        webhooksServiceName,
			CAName:                "KEDA",
			CAOrganization:        "KEDAORG",
			ValidatingWebhookName: validatingWebhookName,
			APIServiceName:        "v1beta1.external.metrics.k8s.io",
			Logger:                setupLog,
			Ready:                 certReady,
		}
		if err := certManager.AddCertificateRotation(ctx, mgr); err != nil {
			setupLog.Error(err, "unable to set up cert rotation")
			os.Exit(1)
		}
	} else {
		close(certReady)
	}

	grpcServer := metricsservice.NewGrpcServer(&scaledHandler, metricsServiceAddr, certDir, certReady)
	if err := mgr.Add(&grpcServer); err != nil {
		setupLog.Error(err, "unable to set up Metrics Service gRPC server")
		os.Exit(1)
	}

	kedautil.PrintWelcome(setupLog, kubeVersion, "manager")

	kubeInformerFactory.Start(ctx.Done())

	if ok := cache.WaitForCacheSync(ctx.Done(), secretInformer.Informer().HasSynced); !ok {
		setupLog.Error(nil, "failed to wait Secrets cache synced")
		os.Exit(1)
	}

	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
