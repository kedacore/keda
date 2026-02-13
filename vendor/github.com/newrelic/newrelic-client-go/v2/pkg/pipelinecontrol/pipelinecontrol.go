package pipelinecontrol

import (
	"github.com/newrelic/newrelic-client-go/v2/internal/http"
	"github.com/newrelic/newrelic-client-go/v2/pkg/config"
	"github.com/newrelic/newrelic-client-go/v2/pkg/logging"
)

type Pipelinecontrol struct {
	client http.Client
	logger logging.Logger
	config config.Config
}

func New(config config.Config) Pipelinecontrol {
	client := http.NewClient(config)
	pkg := Pipelinecontrol{
		client: client,
		logger: config.GetLogger(),
		config: config,
	}
	return pkg
}
