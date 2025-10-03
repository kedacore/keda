// Package nrqldroprules provides a programmatic API for interacting configuring New Relic NRQL Drop Rules.
//
// Deprecated: This package is deprecated, as NRQL Drop Rules shall reach their end-of-life on January 7, 2026.
// It will be removed in a future major version. Switch to the new `pipelinecontrol` package to use Pipeline Cloud Rules, the new alternative to Drop Rules.
// See the README.md of this package for more details.

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
