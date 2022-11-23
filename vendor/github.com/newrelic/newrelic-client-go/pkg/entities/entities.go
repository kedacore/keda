// Package entities provides a programmatic API for interacting with New Relic One entities.
package entities

import (
	"github.com/newrelic/newrelic-client-go/internal/http"
	"github.com/newrelic/newrelic-client-go/pkg/config"
	"github.com/newrelic/newrelic-client-go/pkg/logging"
)

// Entities is used to communicate with the New Relic Entities product.
type Entities struct {
	client http.Client
	logger logging.Logger
}

// New returns a new client for interacting with New Relic One entities.
func New(config config.Config) Entities {
	return Entities{
		client: http.NewClient(config),
		logger: config.GetLogger(),
	}
}
