package main

import (
	"flag"
	"os"

	"github.com/kedacore/keda/pkg/handler"
	kedaprovider "github.com/kedacore/keda/pkg/provider"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
	"k8s.io/klog/klogr"

	kedav1alpha1 "github.com/kedacore/keda/pkg/apis/keda/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	basecmd "github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/cmd"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/provider"
)

// Adapter creates External Metrics Provider
type Adapter struct {
	basecmd.AdapterBase

	// Message is printed on succesful startup
	Message string
}

var logger = klogr.New().WithName("keda_metrics_adapter")

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

	handler := handler.NewScaleHandler(kubeclient, scheme)

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		logger.Error(err, "failed to get watch namespace")
		os.Exit(1)
	}

	return kedaprovider.NewProvider(logger, handler, kubeclient, namespace)
}

func main() {
	defer klog.Flush()

	cmd := &Adapter{}
	cmd.Flags().StringVar(&cmd.Message, "msg", "starting adapter...", "startup message")
	cmd.Flags().AddGoFlagSet(flag.CommandLine) // make sure we get the klog flags
	cmd.Flags().Parse(os.Args)

	kedaProvider := cmd.makeProviderOrDie()
	cmd.WithExternalMetrics(kedaProvider)

	logger.Info(cmd.Message)
	if err := cmd.Run(wait.NeverStop); err != nil {
		logger.Error(err, "unable to run external metrics adapter")
		os.Exit(1)
	}
}
