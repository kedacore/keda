package customeradministration

import (
	"github.com/newrelic/newrelic-client-go/v2/internal/http"
	"github.com/newrelic/newrelic-client-go/v2/pkg/config"
	"github.com/newrelic/newrelic-client-go/v2/pkg/logging"
)

type Customeradministration struct {
	client http.Client
	logger logging.Logger
	config config.Config
}

func New(config config.Config) Customeradministration {
	client := http.NewClient(config)

	pkg := Customeradministration{
		client: client,
		logger: config.GetLogger(),
		config: config,
	}
	return pkg
}
