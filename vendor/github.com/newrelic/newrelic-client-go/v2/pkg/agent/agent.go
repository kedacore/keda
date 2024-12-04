package agent

import (
	"github.com/newrelic/newrelic-client-go/v2/internal/http"
	"github.com/newrelic/newrelic-client-go/v2/pkg/config"
	"github.com/newrelic/newrelic-client-go/v2/pkg/logging"
)

type Agent struct {
	client http.Client
	logger logging.Logger
}

func New(config config.Config) Agent {
	return Agent{
		client: http.NewClient(config),
		logger: config.GetLogger(),
	}
}
