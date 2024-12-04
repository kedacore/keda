// Package nrqldroprules provides a programmatic API for interacting configuring New Relc NRQL Drop Rules.
package nrqldroprules

import (
	"github.com/newrelic/newrelic-client-go/v2/internal/http"
	"github.com/newrelic/newrelic-client-go/v2/pkg/config"
	"github.com/newrelic/newrelic-client-go/v2/pkg/logging"
)

// NrqlDropRules is used to interact with New Relic accounts.
type Nrqldroprules struct {
	client http.Client
	logger logging.Logger
}

// New returns a new client for interacting with New Relic accounts.
func New(config config.Config) Nrqldroprules {
	return Nrqldroprules{
		client: http.NewClient(config),
		logger: config.GetLogger(),
	}
}
