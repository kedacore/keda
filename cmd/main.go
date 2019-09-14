package main

import (
	"flag"
	"time"

	log "github.com/sirupsen/logrus"
	adapter "github.com/kedacore/keda/pkg/adapter"
	"github.com/kedacore/keda/pkg/controller"
	"github.com/kedacore/keda/pkg/handler"
	"github.com/kedacore/keda/pkg/kubernetes"
	"github.com/kedacore/keda/pkg/signals"
	"k8s.io/apiserver/pkg/util/logs"

	// workaround go dep management system
	_ "golang.org/x/tools/imports"
	_ "k8s.io/code-generator/pkg/util"
	_ "k8s.io/gengo/parser"
)

const kedaVersion = "0.0.1"

var (
	// GitCommit is set by the build using -ldflags "-X main.GitCommit=$GIT_COMMIT"
	GitCommit string
	logLevel  = flag.String("log-level", "info", "Options are debug, info, warning, error, fatal, or panic. (default info)")
)

func main() {
	printVersion()
	logs.InitLogs()
	defer logs.FlushLogs()

	kedaClient, kubeClient, err := kubernetes.GetClients()
	if err != nil {
		panic(err)
	}

	ctx := signals.Context()
	scaleHandler := handler.NewScaleHandler(kedaClient, kubeClient)
	go controller.NewController(kedaClient, kubeClient, scaleHandler).Run(ctx)
	if err := adapter.NewAdapter(scaleHandler).Run(ctx.Done()); err != nil {
		log.Fatalf("unable to run custom metrics adapter: %v", err)
	}

	shutdownDuration := 5 * time.Second
	log.Infof("allowing %s for graceful shutdown to complete", shutdownDuration)
	<-time.After(shutdownDuration)
}

func printVersion() {
	log.Infof("Keda version: %s", kedaVersion)
	log.Infof("Git commit: %s", GitCommit)
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
