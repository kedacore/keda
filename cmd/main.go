package main

import (
	"flag"
	"time"

	adapter "github.com/Azure/Kore/pkg/adapter"
	"github.com/Azure/Kore/pkg/controller"
	"github.com/Azure/Kore/pkg/handler"
	"github.com/Azure/Kore/pkg/kubernetes"
	"github.com/Azure/Kore/pkg/signals"
	log "github.com/Sirupsen/logrus"
	"k8s.io/apiserver/pkg/util/logs"

	// workaround go dep management system
	_ "golang.org/x/tools/imports"
	_ "k8s.io/code-generator/pkg/util"
	_ "k8s.io/gengo/parser"
)

var (
	logLevel = flag.String("log-level", "info", "Options are debug, info, warning, error, fatal, or panic. (default info)")
)

func main() {

	ctx := signals.Context()
	logs.InitLogs()
	defer logs.FlushLogs()

	adapter := &adapter.KoreAdapter{}
	adapter.Flags().StringVar(&adapter.Message, "msg", "starting adapter...", "startup message")
	adapter.Flags().AddGoFlagSet(flag.CommandLine)

	parsedLogLevel, err := log.ParseLevel(*logLevel)
	if err == nil {
		log.SetLevel(parsedLogLevel)
		log.Infof("Log level set to: %s", parsedLogLevel)
	} else {
		log.Fatalf("Invalid value for --log-level: %s", *logLevel)
	}

	koreClient, kubeClient, err := kubernetes.GetClients()
	if err != nil {
		panic(err)
	}
	scaleHandler := handler.NewScaleHandler(koreClient, kubeClient)
	adapter.NewExternalMetricsProvider(scaleHandler)
	go controller.NewController(koreClient, kubeClient, scaleHandler).Run(ctx)
	if err := adapter.Run(ctx.Done()); err != nil {
		log.Fatalf("unable to run custom metrics adapter: %v", err)
	}

	shutdownDuration := 5 * time.Second
	log.Infof("allowing %s for graceful shutdown to complete", shutdownDuration)
	<-time.After(shutdownDuration)
}
