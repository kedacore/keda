package http

import (
	"errors"
)

var (
	// ErrNotFound is returned when the resource was not found in New Relic.
	ErrNotFound = errors.New("newrelic: Resource not found")

	// ErrClassTooManyRequests is returned in json messages when client is sending more RPM than the service limit.
	// (RPM = Requests Per Minute)
	ErrClassTooManyRequests = errors.New("TOO_MANY_REQUESTS")
)
