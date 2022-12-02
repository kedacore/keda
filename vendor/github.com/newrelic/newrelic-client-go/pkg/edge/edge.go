// Package edge provides a programmatic API for interacting with New Relic Edge.
package edge

import (
	"github.com/newrelic/newrelic-client-go/internal/http"
	"github.com/newrelic/newrelic-client-go/pkg/config"
	"github.com/newrelic/newrelic-client-go/pkg/logging"
)

// Edge is used to communicate with the New Relic Edge product.
type Edge struct {
	client http.Client
	logger logging.Logger
}

// New returns a new client for interacting with New Relic Edge.
func New(config config.Config) Edge {
	return Edge{
		client: http.NewClient(config),
		logger: config.GetLogger(),
	}
}
