package fleetcontrol

import (
	"github.com/newrelic/newrelic-client-go/v2/internal/http"
	"github.com/newrelic/newrelic-client-go/v2/pkg/config"
	"github.com/newrelic/newrelic-client-go/v2/pkg/infrastructure"
	"github.com/newrelic/newrelic-client-go/v2/pkg/logging"
)

// Alerts is used to communicate with New Relic Alerts.
type Fleetcontrol struct {
	client      http.Client
	logger      logging.Logger
	config      config.Config
	infraClient http.Client
	pager       http.Pager
}

// New is used to create a new Alerts client instance.
func New(config config.Config) Fleetcontrol {
	infraConfig := config

	infraClient := http.NewClient(infraConfig)
	infraClient.SetAuthStrategy(&http.PersonalAPIKeyCapableV2Authorizer{})
	infraClient.SetErrorValue(&infrastructure.ErrorResponse{})

	client := http.NewClient(config)
	client.SetAuthStrategy(&http.PersonalAPIKeyCapableV2Authorizer{})

	pkg := Fleetcontrol{
		client:      client,
		config:      config,
		infraClient: infraClient,
		logger:      config.GetLogger(),
		pager:       &http.LinkHeaderPager{},
	}

	return pkg
}
