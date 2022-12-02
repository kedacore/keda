// Package users provides a programmatic API for interacting with New Relic users.
package users

import (
	"github.com/newrelic/newrelic-client-go/internal/http"
	"github.com/newrelic/newrelic-client-go/pkg/config"
	"github.com/newrelic/newrelic-client-go/pkg/logging"
)

// Users is used to interact with New Relic users.
type Users struct {
	client http.Client
	logger logging.Logger
}

// New returns a new client for interacting with New Relic users.
func New(config config.Config) Users {
	return Users{
		client: http.NewClient(config),
		logger: config.GetLogger(),
	}
}
