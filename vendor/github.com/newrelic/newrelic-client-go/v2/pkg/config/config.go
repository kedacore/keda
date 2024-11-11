// Package config provides cross-cutting configuration support for the newrelic-client-go project.
package config

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/newrelic/newrelic-client-go/v2/internal/version"
	"github.com/newrelic/newrelic-client-go/v2/pkg/logging"
	"github.com/newrelic/newrelic-client-go/v2/pkg/region"
)

// Config contains all the configuration data for the API Client.
type Config struct {
	// LicenseKey to authenticate Log API requests
	// see: https://docs.newrelic.com/docs/accounts/accounts-billing/account-setup/new-relic-license-key
	LicenseKey string

	// PersonalAPIKey to authenticate API requests
	// see: https://docs.newrelic.com/docs/apis/get-started/intro-apis/types-new-relic-api-keys#personal-api-key
	PersonalAPIKey string

	// AdminAPIKey to authenticate API requests
	// Deprecated.  Use a Personal API key for authentication.
	// see: https://docs.newrelic.com/docs/apis/get-started/intro-apis/types-new-relic-api-keys#admin
	AdminAPIKey string

	// InsightsInsertKey to send custom events to Insights
	InsightsInsertKey string

	// region of the New Relic platform to use
	region *region.Region

	// Timeout is the client timeout for HTTP requests.
	Timeout *time.Duration

	// HTTPTransport allows customization of the client's underlying transport.
	HTTPTransport http.RoundTripper

	// Compression used in sending data in HTTP requests.
	Compression CompressionType

	// UserAgent updates the default user agent string used by the client.
	UserAgent string

	// ServiceName is for New Relic internal use only.
	ServiceName string

	// LogLevel can be one of the following values:
	// "panic", "fatal", "error", "warn", "info", "debug", "trace"
	LogLevel string

	// LogJSON toggles formatting of log entries in JSON format.
	LogJSON bool

	// Logger allows customization of the client's underlying logger.
	Logger logging.Logger
}

// New creates a default configuration and returns it
func New() Config {
	reg, _ := region.Get(region.Default)

	return Config{
		region:      reg,
		LogLevel:    "info",
		Compression: Compression.None,
	}
}

func (cfg *Config) Init(opts []ConfigOption) error {
	// Loop through config options
	for _, fn := range opts {
		if nil != fn {
			if err := fn(cfg); err != nil {
				return err
			}
		}
	}

	if cfg.PersonalAPIKey == "" && cfg.AdminAPIKey == "" && cfg.InsightsInsertKey == "" {
		return errors.New("must use at least one of: ConfigPersonalAPIKey, ConfigAdminAPIKey, ConfigInsightsInsertKey")
	}

	if cfg.Logger == nil {
		cfg.Logger = cfg.GetLogger()
	}

	return nil
}

// Region returns the region configuration struct
// if one has not been set, use the default region
func (c *Config) Region() *region.Region {
	if c.region == nil {
		reg, _ := region.Get(region.Default)
		c.region = reg
	}

	return c.region
}

// SetRegion configures the region
func (c *Config) SetRegion(reg *region.Region) error {
	if reg == nil {
		return region.ErrorNil()
	}

	c.region = reg

	return nil
}

func (c *Config) SetServiceName(serviceName string) error {
	if c.ServiceName == "" {
		c.ServiceName = serviceName
	}

	customServiceName := os.Getenv("NEW_RELIC_SERVICE_NAME")
	if customServiceName != "" {
		c.ServiceName = fmt.Sprintf("%s|%s", customServiceName, c.ServiceName)
	}

	return nil
}

// GetLogger returns a logger instance based on the config values.
func (c *Config) GetLogger() logging.Logger {
	if c.Logger != nil {
		return c.Logger
	}

	l := logging.NewLogrusLogger()
	l.SetDefaultFields(map[string]string{"newrelic-client-go": version.Version})
	l.SetLogJSON(c.LogJSON)
	l.SetLevel(c.LogLevel)

	// c.Logger = l

	return l
}
