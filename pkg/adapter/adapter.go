package adapter

import (
	"flag"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/kedacore/keda/pkg/handler"
	kedaprov "github.com/kedacore/keda/pkg/provider"
	basecmd "github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/cmd"
)

type Adapter struct {
	basecmd.AdapterBase

	// Message is printed on successful startup
	Message string
}

func NewAdapter(scaleHandler *handler.ScaleHandler) *Adapter {
	a := &Adapter{}
	a.Flags().StringVar(&a.Message, "msg", "starting adapter...", "startup message")
	a.Flags().AddGoFlagSet(flag.CommandLine)
	a.Flags().Parse(os.Args)
	client, err := a.DynamicClient()
	if err != nil {
		log.Fatalf("unable to construct dynamic client: %v", err)
	}

	mapper, err := a.RESTMapper()
	if err != nil {
		log.Fatalf("unable to construct discovery REST mapper: %v", err)
	}

	provider := kedaprov.NewProvider(client, mapper, scaleHandler)
	a.WithExternalMetrics(provider)
	return a
}
