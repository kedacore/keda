package region

import (
	"os"
	"strings"
)

const (
	// US represents New Relic's US-based production deployment.
	US Name = "US"

	// EU represents New Relic's EU-based production deployment.
	EU Name = "EU"

	// Staging represents New Relic's US-based staging deployment.
	// This is for internal New Relic use only.
	Staging Name = "Staging"

	// Local represents a local development environment.
	Local Name = "Local"
)

// Regions defines the service URLs that make up the various environments.
var Regions = map[Name]*Region{
	US: {
		name:                  "US",
		infrastructureBaseURL: "https://infra-api.newrelic.com/v2",
		insightsBaseURL:       "https://insights-collector.newrelic.com/v1",
		insightsKeysBaseURL:   "https://insights.newrelic.com/internal_api/1",
		logsBaseURL:           "https://log-api.newrelic.com/log/v1",
		nerdGraphBaseURL:      "https://api.newrelic.com/graphql",
		restBaseURL:           "https://api.newrelic.com/v2",
		syntheticsBaseURL:     "https://synthetics.newrelic.com/synthetics/api",
		metricsBaseURL:        "https://metric-api.newrelic.com/metric/v1",
		blobServiceBaseURL:    "https://blob-api.service.newrelic.com/v1/e",
	},
	EU: {
		name:                  "EU",
		infrastructureBaseURL: "https://infra-api.eu.newrelic.com/v2",
		insightsBaseURL:       "https://insights-collector.eu01.nr-data.net/v1",
		insightsKeysBaseURL:   "https://insights.eu.newrelic.com/internal_api/1",
		logsBaseURL:           "https://log-api.eu.newrelic.com/log/v1",
		nerdGraphBaseURL:      "https://api.eu.newrelic.com/graphql",
		restBaseURL:           "https://api.eu.newrelic.com/v2",
		syntheticsBaseURL:     "https://synthetics.eu.newrelic.com/synthetics/api",
		metricsBaseURL:        "https://metric-api.eu.newrelic.com/metric/v1",
		blobServiceBaseURL:    "https://blob-api.service.eu.newrelic.com/v1/e",
	},
	Staging: {
		name:                  "Staging",
		infrastructureBaseURL: "https://staging-infra-api.newrelic.com/v2",
		insightsBaseURL:       "https://staging-insights-collector.newrelic.com/v1",
		insightsKeysBaseURL:   "https://staging-insights.newrelic.com/internal_api/1",
		logsBaseURL:           "https://staging-log-api.newrelic.com/log/v1",
		nerdGraphBaseURL:      "https://staging-api.newrelic.com/graphql",
		restBaseURL:           "https://staging-api.newrelic.com/v2",
		syntheticsBaseURL:     "https://staging-synthetics.newrelic.com/synthetics/api",
		metricsBaseURL:        "https://staging-metric-api.newrelic.com/metric/v1",
		blobServiceBaseURL:    "https://blob-api.staging-service.newrelic.com/v1/e",
	},
	Local: {
		name:                  "Local",
		infrastructureBaseURL: "http://localhost:3000/v2",
		insightsBaseURL:       "http://localhost:3000/v1",
		insightsKeysBaseURL:   "http://localhost:3000/internal_api/1",
		logsBaseURL:           "http://localhost:3000/log/v1",
		nerdGraphBaseURL:      "http://localhost:3000/graphql",
		restBaseURL:           "http://localhost:3000/v2",
		syntheticsBaseURL:     "http://localhost:3000/synthetics/api",
		metricsBaseURL:        "http://localhost:3000/metric/v1",
		// the following is just a placeholder, is not actually intended to work
		blobServiceBaseURL: "https:/localhost:3000/blob/v1/e",
	},
}

// Default represents the region returned if nothing was specified
const Default Name = US

// Parse takes a Region string and returns a RegionType
func Parse(r string) (Name, error) {
	switch strings.ToLower(r) {
	case "us":
		return US, nil
	case "eu":
		return EU, nil
	case "staging":
		return Staging, nil
	case "local":
		return Local, nil
	default:
		return "", UnknownError{Message: r}
	}
}

func Get(r Name) (*Region, error) {
	if reg, ok := Regions[r]; ok {
		ret := *reg // Make a copy
		if val := os.Getenv("NEW_RELIC_INFRASTRUCTURE_BASE_URL"); val != "" {
			ret.infrastructureBaseURL = val
		}
		if val := os.Getenv("NEW_RELIC_INSIGHTS_BASE_URL"); val != "" {
			ret.insightsBaseURL = val
		}
		if val := os.Getenv("NEW_RELIC_INSIGHTS_KEY_BASE_URL"); val != "" {
			ret.insightsKeysBaseURL = val
		}
		if val := os.Getenv("NEW_RELIC_LOGS_BASE_URL"); val != "" {
			ret.logsBaseURL = val
		}
		if val := os.Getenv("NEW_RELIC_NERDGRAPH_BASE_URL"); val != "" {
			ret.nerdGraphBaseURL = val
		}
		if val := os.Getenv("NEW_RELIC_REST_BASE_URL"); val != "" {
			ret.restBaseURL = val
		}
		if val := os.Getenv("NEW_RELIC_SYNTHETICS_BASE_URL"); val != "" {
			ret.syntheticsBaseURL = val
		}
		if val := os.Getenv("NEW_RELIC_METRICS_BASE_URL"); val != "" {
			ret.metricsBaseURL = val
		}
		return &ret, nil
	}

	return Regions[Default], UnknownUsingDefaultError{Message: r.String()}
}
