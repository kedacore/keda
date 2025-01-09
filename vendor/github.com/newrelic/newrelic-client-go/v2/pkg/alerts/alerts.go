package alerts

import (
	"github.com/newrelic/newrelic-client-go/v2/internal/http"
	"github.com/newrelic/newrelic-client-go/v2/pkg/config"
	"github.com/newrelic/newrelic-client-go/v2/pkg/infrastructure"
	"github.com/newrelic/newrelic-client-go/v2/pkg/logging"
)

// Alerts is used to communicate with New Relic Alerts.
type Alerts struct {
	client      http.Client
	config      config.Config
	infraClient http.Client
	logger      logging.Logger
	pager       http.Pager
}

// New is used to create a new Alerts client instance.
func New(config config.Config) Alerts {
	infraConfig := config

	infraClient := http.NewClient(infraConfig)
	infraClient.SetAuthStrategy(&http.PersonalAPIKeyCapableV2Authorizer{})
	infraClient.SetErrorValue(&infrastructure.ErrorResponse{})

	client := http.NewClient(config)
	client.SetAuthStrategy(&http.PersonalAPIKeyCapableV2Authorizer{})

	pkg := Alerts{
		client:      client,
		config:      config,
		infraClient: infraClient,
		logger:      config.GetLogger(),
		pager:       &http.LinkHeaderPager{},
	}

	return pkg
}
