package main

import (
	"github.com/Azure/Kore/pkg/scalers"
	"time"

	"github.com/Azure/Kore/pkg/controller"
	"github.com/Azure/Kore/pkg/kubernetes"
	"github.com/Azure/Kore/pkg/signals"
	log "github.com/Sirupsen/logrus"

	// workaround go dep management system
	_ "k8s.io/code-generator/pkg/util"
	_ "k8s.io/gengo/parser"
	_ "golang.org/x/tools/imports"
)

func main() {
	koreClient, kubeClient, err := kubernetes.GetClients()
	if err != nil {
		panic(err)
	}

	ctx := signals.Context()
	scaleHandler := scalers.NewScaleHandler(koreClient, kubeClient)
	controller.NewController(koreClient, kubeClient, scaleHandler).Run(ctx)

	shutdownDuration := 5 * time.Second
	log.Infof("allowing %s for graceful shutdown to complete", shutdownDuration)
	<-time.After(shutdownDuration)
}
