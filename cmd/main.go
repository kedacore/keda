package main

import (
	"flag"
	"time"

	"github.com/Azure/Kore/pkg/handler"

	"github.com/Azure/Kore/pkg/controller"
	"github.com/Azure/Kore/pkg/kubernetes"
	"github.com/Azure/Kore/pkg/signals"
	log "github.com/Sirupsen/logrus"

	// workaround go dep management system
	_ "golang.org/x/tools/imports"
	_ "k8s.io/code-generator/pkg/util"
	_ "k8s.io/gengo/parser"
)

var (
	logLevel = flag.String("log-level", "info", "Options are debug, info, warning, error, fatal, or panic. (default info)")
)

func main() {
	koreClient, kubeClient, err := kubernetes.GetClients()
	if err != nil {
		panic(err)
	}

	ctx := signals.Context()
	scaleHandler := handler.NewScaleHandler(koreClient, kubeClient)
	controller.NewController(koreClient, kubeClient, scaleHandler).Run(ctx)

	shutdownDuration := 5 * time.Second
	log.Infof("allowing %s for graceful shutdown to complete", shutdownDuration)
	<-time.After(shutdownDuration)
}

func init() {
	flag.Parse()

	parsedLogLevel, err := log.ParseLevel(*logLevel)
	if err == nil {
		log.SetLevel(parsedLogLevel)
		log.Infof("Log level set to: %s", parsedLogLevel)
	} else {
		log.Fatalf("Invalid value for --log-level: %s", *logLevel)
	}
}
