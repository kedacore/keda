package main

import (
	"crypto/tls"
	"net/http"
	"flag"
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

var (
	disableTLSVerification = flag.Bool("disableTLSVerification", false, "Disable TLS certificate verification")
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

func init() {
	flag.Parse()

	if *disableTLSVerification {
		log.Infof("Setting TLSClientConfig InsecureSkipVerify to true because --disableTLSVerification was passed")
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
}