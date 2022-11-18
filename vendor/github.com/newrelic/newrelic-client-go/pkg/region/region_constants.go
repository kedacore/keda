package region

import (
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
		return &ret, nil
	}

	return Regions[Default], UnknownUsingDefaultError{Message: r.String()}
}
