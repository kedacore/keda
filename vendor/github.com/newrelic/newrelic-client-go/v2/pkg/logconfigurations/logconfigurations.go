package logconfigurations

// Package synthetics provides a programmatic API for interacting with the New Relic Synthetics product.
import (
	"github.com/newrelic/newrelic-client-go/v2/internal/http"
	"github.com/newrelic/newrelic-client-go/v2/pkg/config"
	"github.com/newrelic/newrelic-client-go/v2/pkg/logging"
)

// Obfuscation is used to communicate with the New Relic Obfuscation product.
type Logconfigurations struct {
	client http.Client
	logger logging.Logger
}

// New is used to create a new Obfuscation expression.
func New(config config.Config) Logconfigurations {
	client := http.NewClient(config)

	pkg := Logconfigurations{
		client: client,
		logger: config.GetLogger(),
	}

	return pkg
}
