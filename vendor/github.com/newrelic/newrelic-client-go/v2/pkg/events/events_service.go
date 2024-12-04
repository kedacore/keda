package events

import (
	"context"

	"github.com/newrelic/newrelic-client-go/v2/pkg/config"
)

// EventsAPI provides an interface to enable mocking the service implementation.
// You should use this interface to invoke methods and substitute with a mock having a compatible implementation for unit testing your usages.
type EventsAPI interface {
	CreateEvent(accountID int, event interface{}) error
	CreateEventWithContext(ctx context.Context, accountID int, event interface{}) error
}

// Provides the EventsAPI operations for *Events
func NewEventsService(opts ...config.ConfigOption) (*Events, error) {
	cfg := config.New()

	err := cfg.Init(opts)
	if err != nil {
		return nil, err
	}

	events := New(cfg)
	return &events, nil
}
