package datamanagement

import (
	"github.com/newrelic/newrelic-client-go/v2/internal/http"
	"github.com/newrelic/newrelic-client-go/v2/pkg/config"
	"github.com/newrelic/newrelic-client-go/v2/pkg/logging"
)

// Datamanagement is used to interact with New Relic cardinality and data management APIs.
type Datamanagement struct {
	client http.Client
	logger logging.Logger
}

// New returns a new client for managing cardinality limits.
func New(config config.Config) Datamanagement {
	client := http.NewClient(config)

	pkg := Datamanagement{
		client: client,
		logger: config.GetLogger(),
	}

	return pkg
}
