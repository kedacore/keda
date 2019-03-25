package adapter

import (
	"github.com/Azure/Kore/pkg/handler"
	koreprov "github.com/Azure/Kore/pkg/provider"
	log "github.com/Sirupsen/logrus"
	basecmd "github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/cmd"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/provider"
)

type KoreAdapter struct {
	basecmd.AdapterBase

	// Message is printed on succesful startup
	Message string
}

func (a *KoreAdapter) NewExternalMetricsProvider(scaleHandler *handler.ScaleHandler) provider.ExternalMetricsProvider {
	client, err := a.DynamicClient()
	if err != nil {
		log.Fatalf("unable to construct dynamic client: %v", err)
	}

	mapper, err := a.RESTMapper()
	if err != nil {
		log.Fatalf("unable to construct discovery REST mapper: %v", err)
	}

	provider := koreprov.NewProvider(client, mapper, scaleHandler)
	a.WithExternalMetrics(provider)
	return provider
}
