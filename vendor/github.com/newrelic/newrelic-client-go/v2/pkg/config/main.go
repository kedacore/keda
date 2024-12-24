package config

import (
	"errors"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/newrelic/newrelic-client-go/v2/pkg/logging"
	"github.com/newrelic/newrelic-client-go/v2/pkg/region"
)

// ConfigOption configures the Config when provided to NewApplication.
// https://docs.newrelic.com/docs/apis/get-started/intro-apis/types-new-relic-api-keys
type ConfigOption func(*Config) error

// ConfigPersonalAPIKey sets the New Relic Admin API key this client will use.
// This key should be used to create a client instance.
// https://docs.newrelic.com/docs/apis/get-started/intro-apis/types-new-relic-api-keys
func ConfigPersonalAPIKey(apiKey string) ConfigOption {
	return func(cfg *Config) error {
		cfg.PersonalAPIKey = apiKey
		return nil
	}
}

// ConfigInsightsInsertKey sets the New Relic Insights insert key this client will use.
// https://docs.newrelic.com/docs/apis/get-started/intro-apis/types-new-relic-api-keys
func ConfigInsightsInsertKey(insightsInsertKey string) ConfigOption {
	return func(cfg *Config) error {
		cfg.InsightsInsertKey = insightsInsertKey
		return nil
	}
}

// ConfigAdminAPIKey sets the New Relic Admin API key this client will use.
// Deprecated.  Use a personal API key for authentication.
// https://docs.newrelic.com/docs/apis/get-started/intro-apis/types-new-relic-api-keys
func ConfigAdminAPIKey(adminAPIKey string) ConfigOption {
	return func(cfg *Config) error {
		cfg.AdminAPIKey = adminAPIKey
		return nil
	}
}

// ConfigRegion sets the New Relic Region this client will use.
func ConfigRegion(r string) ConfigOption {
	return func(cfg *Config) error {
		// We can ignore this error since we will be defaulting in the next step
		regName, _ := region.Parse(r)

		reg, err := region.Get(regName)
		if err != nil {
			if _, ok := err.(region.UnknownUsingDefaultError); ok {
				// If region wasn't provided, output a warning message
				// indicating the default region "US" is being used.
				log.Warn(err)
				return nil
			}

			return err
		}

		err = cfg.SetRegion(reg)

		return err
	}
}

// ConfigHTTPTimeout sets the timeout for HTTP requests.
func ConfigHTTPTimeout(t time.Duration) ConfigOption {
	return func(cfg *Config) error {
		var timeout = &t
		cfg.Timeout = timeout
		return nil
	}
}

// ConfigHTTPTransport sets the HTTP Transporter.
func ConfigHTTPTransport(transport http.RoundTripper) ConfigOption {
	return func(cfg *Config) error {
		if transport != nil {
			cfg.HTTPTransport = transport
			return nil
		}

		return errors.New("HTTP Transport can not be nil")
	}
}

// ConfigUserAgent sets the HTTP UserAgent for API requests.
func ConfigUserAgent(ua string) ConfigOption {
	return func(cfg *Config) error {
		if ua != "" {
			cfg.UserAgent = ua
			return nil
		}

		return errors.New("user-agent can not be empty")
	}
}

// ConfigServiceName sets the service name logged
func ConfigServiceName(name string) ConfigOption {
	return func(cfg *Config) error {
		if name != "" {
			cfg.ServiceName = name
		}

		return nil
	}
}

// ConfigBaseURL sets the base URL used to make requests to the REST API V2.
func ConfigBaseURL(url string) ConfigOption {
	return func(cfg *Config) error {
		if url != "" {
			cfg.Region().SetRestBaseURL(url)
			return nil
		}

		return errors.New("base URL can not be empty")
	}
}

// ConfigInfrastructureBaseURL sets the base URL used to make requests to the Infrastructure API.
func ConfigInfrastructureBaseURL(url string) ConfigOption {
	return func(cfg *Config) error {
		if url != "" {
			cfg.Region().SetInfrastructureBaseURL(url)
			return nil
		}

		return errors.New("infrastructure base URL can not be empty")
	}
}

// ConfigSyntheticsBaseURL sets the base URL used to make requests to the Synthetics API.
func ConfigSyntheticsBaseURL(url string) ConfigOption {
	return func(cfg *Config) error {
		if url != "" {
			cfg.Region().SetSyntheticsBaseURL(url)
			return nil
		}

		return errors.New("synthetics base URL can not be empty")
	}
}

// ConfigNerdGraphBaseURL sets the base URL used to make requests to the NerdGraph API.
func ConfigNerdGraphBaseURL(url string) ConfigOption {
	return func(cfg *Config) error {
		if url != "" {
			cfg.Region().SetNerdGraphBaseURL(url)
			return nil
		}

		return errors.New("nerdgraph base URL can not be empty")
	}
}

// ConfigLogLevel sets the log level for the client.
func ConfigLogLevel(logLevel string) ConfigOption {
	return func(cfg *Config) error {
		if logLevel != "" {
			cfg.LogLevel = logLevel
			return nil
		}

		return errors.New("log level can not be empty")
	}
}

// ConfigLogJSON toggles JSON formatting on for the logger if set to true.
func ConfigLogJSON(logJSON bool) ConfigOption {
	return func(cfg *Config) error {
		cfg.LogJSON = logJSON
		return nil
	}
}

// ConfigLogger can be used to customize the client's logger.
// Custom loggers must conform to the logging.Logger interface.
func ConfigLogger(logger logging.Logger) ConfigOption {
	return func(cfg *Config) error {
		if logger != nil {
			cfg.Logger = logger
			return nil
		}

		return errors.New("logger can not be nil")
	}
}
