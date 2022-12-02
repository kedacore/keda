package workflows

import (
	"github.com/newrelic/newrelic-client-go/internal/http"
	"github.com/newrelic/newrelic-client-go/pkg/config"
	"github.com/newrelic/newrelic-client-go/pkg/infrastructure"
	"github.com/newrelic/newrelic-client-go/pkg/logging"
)

// Workflows are used to communicate with New Relic Workflows.
type Workflows struct {
	client      http.Client
	config      config.Config
	infraClient http.Client
	logger      logging.Logger
	pager       http.Pager
}

// New is used to create a new Workflows' client instance.
func New(config config.Config) Workflows {
	infraConfig := config

	infraClient := http.NewClient(infraConfig)
	infraClient.SetAuthStrategy(&http.PersonalAPIKeyCapableV2Authorizer{})
	infraClient.SetErrorValue(&infrastructure.ErrorResponse{})

	client := http.NewClient(config)
	client.SetAuthStrategy(&http.PersonalAPIKeyCapableV2Authorizer{})

	pkg := Workflows{
		client:      client,
		config:      config,
		infraClient: infraClient,
		logger:      config.GetLogger(),
		pager:       &http.LinkHeaderPager{},
	}

	return pkg
}
