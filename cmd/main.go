package main

import (
	"crypto/tls"
	"flag"
	"net/http"
	"os"
	"time"

	"github.com/Azure/Kore/pkg/controller"
	"github.com/Azure/Kore/pkg/handler"
	"github.com/Azure/Kore/pkg/kubernetes"
	koreprov "github.com/Azure/Kore/pkg/provider"
	"github.com/Azure/Kore/pkg/signals"
	log "github.com/Sirupsen/logrus"
	"github.com/golang/glog"
	basecmd "github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/cmd"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/provider"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/util/logs"

	// workaround go dep management system
	_ "golang.org/x/tools/imports"
	_ "k8s.io/code-generator/pkg/util"
	_ "k8s.io/gengo/parser"
)

var (
	disableTLSVerification = flag.Bool("disableTLSVerification", false, "Disable TLS certificate verification")
)

type KoreAdapter struct {
	basecmd.AdapterBase

	// Message is printed on succesful startup
	Message string
}

func (a *KoreAdapter) makeProviderOrDie(scaleHandler *handler.ScaleHandler) provider.ExternalMetricsProvider {
	client, err := a.DynamicClient()
	if err != nil {
		glog.Fatalf("unable to construct dynamic client: %v", err)
	}

	mapper, err := a.RESTMapper()
	if err != nil {
		glog.Fatalf("unable to construct discovery REST mapper: %v", err)
	}

	return koreprov.NewProvider(client, mapper, scaleHandler)
}

func main() {

	ctx := signals.Context()
	logs.InitLogs()
	defer logs.FlushLogs()

	//basecmd.AdapterBase{Name: "scale-controller"}, "Started metrics server in scale controller"
	cmd := &KoreAdapter{}
	cmd.Flags().StringVar(&cmd.Message, "msg", "starting adapter...", "startup message")
	cmd.Flags().AddGoFlagSet(flag.CommandLine) // make sure we get the glog flags
	cmd.Flags().Parse(os.Args)

	if *disableTLSVerification {
		log.Infof("Setting TLSClientConfig InsecureSkipVerify to true because --disableTLSVerification was passed")
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	koreClient, kubeClient, err := kubernetes.GetClients()
	if err != nil {
		panic(err)
	}
	scaleHandler := handler.NewScaleHandler(koreClient, kubeClient)
	provider := cmd.makeProviderOrDie(scaleHandler)
	cmd.WithExternalMetrics(provider)
	glog.Infof(cmd.Message)
	go controller.NewController(koreClient, kubeClient, scaleHandler).Run(ctx)
	if err := cmd.Run(wait.NeverStop); err != nil {
		glog.Fatalf("unable to run custom metrics adapter: %v", err)
	}

	shutdownDuration := 5 * time.Second
	log.Infof("allowing %s for graceful shutdown to complete", shutdownDuration)
	<-time.After(shutdownDuration)
}
