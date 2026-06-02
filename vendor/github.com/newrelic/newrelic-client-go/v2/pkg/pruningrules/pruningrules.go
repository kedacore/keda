// Package pruningrules provides a programmatic API for managing New Relic metric pruning rules.
// Pruning rules strip specified attributes from metric aggregates using the DROP_ATTRIBUTES_FROM_METRIC_AGGREGATES action.

package pruningrules

import (
	"github.com/newrelic/newrelic-client-go/v2/internal/http"
	"github.com/newrelic/newrelic-client-go/v2/pkg/config"
	"github.com/newrelic/newrelic-client-go/v2/pkg/logging"
)

// Pruningrules is used to interact with New Relic metric pruning rules.
type Pruningrules struct {
	client http.Client
	logger logging.Logger
}

// New returns a new client for managing metric pruning rules.
func New(config config.Config) Pruningrules {
	return Pruningrules{
		client: http.NewClient(config),
		logger: config.GetLogger(),
	}
}
