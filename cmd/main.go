package main

import (
	"time"

	"github.com/Azure/Kore/pkg/controller"
	"github.com/Azure/Kore/pkg/kubernetes"
	"github.com/Azure/Kore/pkg/signals"
	log "github.com/Sirupsen/logrus"
)

func main() {
	koreClient, kubeClient, err := kubernetes.GetClients()
	if err != nil {
		panic(err)
	}

	ctx := signals.Context()
	controller.NewController(koreClient, kubeClient).Run(ctx)

	shutdownDuration := 5 * time.Second
	log.Infof("allowing %s for graceful shutdown to complete", shutdownDuration)
	<-time.After(shutdownDuration)
}
