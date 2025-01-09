package testhelpers

import (
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/newrelic/newrelic-client-go/v2/pkg/config"
	"github.com/newrelic/newrelic-client-go/v2/pkg/region"
)

const (
	HTTPTimeout    = 60 * time.Second                                  // HTTPTimeout increases the timeout for integration tests
	LicenseKey     = "APMLicenseKey"                                   // LicenseKey used in mock configs
	LogLevel       = "debug"                                           // LogLevel used in mock configs
	PersonalAPIKey = "personalAPIKey"                                  // PersonalAPIKey used in mock configs (from Environment for Integration tests)
	UserAgent      = "newrelic/newrelic-client-go (automated testing)" // UserAgent used in mock configs
)

// NewTestConfig returns a fully saturated configration with modified BaseURLs
// for all endpoints based on the test server passed in
func NewTestConfig(t *testing.T, testServer *httptest.Server) config.Config {
	cfg := config.New()

	// Set some defaults from Testing constants
	cfg.LogLevel = LogLevel
	cfg.PersonalAPIKey = PersonalAPIKey
	cfg.UserAgent = UserAgent
	cfg.LicenseKey = LicenseKey

	if testServer != nil {
		cfg.Region().SetInfrastructureBaseURL(testServer.URL)
		cfg.Region().SetInsightsBaseURL(testServer.URL)
		cfg.Region().SetNerdGraphBaseURL(testServer.URL)
		cfg.Region().SetRestBaseURL(testServer.URL)
		cfg.Region().SetSyntheticsBaseURL(testServer.URL)
		cfg.Region().SetLogsBaseURL(testServer.URL)
	}

	return cfg
}

// NewIntegrationTestConfig grabs environment vars for required fields or skips the test.
// returns a fully saturated configuration
func NewIntegrationTestConfig(t *testing.T) config.Config {
	envPersonalAPIKey := os.Getenv("NEW_RELIC_API_KEY")
	envInsightsInsertKey := os.Getenv("NEW_RELIC_INSIGHTS_INSERT_KEY")
	envLicenseKey := os.Getenv("NEW_RELIC_LICENSE_KEY")
	envRegion := os.Getenv("NEW_RELIC_REGION")
	envLogLevel := os.Getenv("NEW_RELIC_LOG_LEVEL")

	if envPersonalAPIKey == "" {
		t.Skipf("acceptance testing requires NEW_RELIC_API_KEY")
	}

	cfg := config.New()

	// Set some defaults
	if envLogLevel != "" {
		cfg.LogLevel = envLogLevel
	} else {
		cfg.LogLevel = LogLevel
	}
	cfg.Logger = cfg.GetLogger()

	// HTTP Settings
	timeout := HTTPTimeout
	cfg.Timeout = &timeout
	cfg.UserAgent = UserAgent

	// Auth
	cfg.PersonalAPIKey = envPersonalAPIKey
	cfg.InsightsInsertKey = envInsightsInsertKey
	cfg.LicenseKey = envLicenseKey

	if envRegion != "" {
		regName, err := region.Parse(envRegion)
		assert.NoError(t, err)

		reg, err := region.Get(regName)
		assert.NoError(t, err)

		err = cfg.SetRegion(reg)
		assert.NoError(t, err)
	}

	return cfg
}
