// Package apiaccess provides a programmatic API for interacting with New Relic API keys
package apiaccess

import (
	"github.com/newrelic/newrelic-client-go/v2/internal/http"
	"github.com/newrelic/newrelic-client-go/v2/pkg/config"
	"github.com/newrelic/newrelic-client-go/v2/pkg/logging"
)

// APIAccess is used to communicate with the New Relic APIKeys product.
type APIAccess struct {
	insightsKeysClient http.Client
	client             http.Client
	config             config.Config
	logger             logging.Logger
}

// New returns a new client for interacting with New Relic One entities.
func New(config config.Config) APIAccess {
	insightsKeysClient := http.NewClient(config)
	insightsKeysClient.SetAuthStrategy(&http.PersonalAPIKeyCapableV2Authorizer{})
	return APIAccess{
		client:             http.NewClient(config),
		insightsKeysClient: insightsKeysClient,
		config:             config,
		logger:             config.GetLogger(),
	}
}
