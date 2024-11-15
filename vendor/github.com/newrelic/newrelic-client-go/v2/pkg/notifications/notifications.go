package notifications

import (
	"github.com/newrelic/newrelic-client-go/v2/internal/http"
	"github.com/newrelic/newrelic-client-go/v2/pkg/config"
	"github.com/newrelic/newrelic-client-go/v2/pkg/infrastructure"
	"github.com/newrelic/newrelic-client-go/v2/pkg/logging"
)

// Notifications are used to communicate with New Relic Notifications.
type Notifications struct {
	client      http.Client
	config      config.Config
	infraClient http.Client
	logger      logging.Logger
	pager       http.Pager
}

// New is used to create a new Notifications' client instance.
func New(config config.Config) Notifications {
	infraConfig := config

	infraClient := http.NewClient(infraConfig)
	infraClient.SetAuthStrategy(&http.PersonalAPIKeyCapableV2Authorizer{})
	infraClient.SetErrorValue(&infrastructure.ErrorResponse{})

	client := http.NewClient(config)
	client.SetAuthStrategy(&http.PersonalAPIKeyCapableV2Authorizer{})

	pkg := Notifications{
		client:      client,
		config:      config,
		infraClient: infraClient,
		logger:      config.GetLogger(),
		pager:       &http.LinkHeaderPager{},
	}

	return pkg
}
