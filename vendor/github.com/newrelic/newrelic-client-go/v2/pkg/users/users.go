// Package users provides a programmatic API for interacting with New Relic users.
package users

import (
	"github.com/newrelic/newrelic-client-go/v2/internal/http"
	"github.com/newrelic/newrelic-client-go/v2/pkg/config"
	"github.com/newrelic/newrelic-client-go/v2/pkg/logging"
)

// Users is used to interact with New Relic users.
type Users struct {
	client http.Client
	logger logging.Logger
	config config.Config
}

// New returns a new client for interacting with New Relic users.
func New(config config.Config) Users {
	return Users{
		client: http.NewClient(config),
		logger: config.GetLogger(),
		config: config,
	}
}
