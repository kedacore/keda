// Package region describes the operational regions defined for New Relic
//
// Regions are geographical locations where the New Relic platform operates
// and this package provides an abstraction layer for handling them within
// the New Relic Client and underlying APIs
package region

import (
	"errors"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

// Name is the name of a New Relic region
type Name string

// Region represents the members of the Region enumeration.
type Region struct {
	name                  string
	infrastructureBaseURL string
	insightsBaseURL       string
	logsBaseURL           string
	nerdGraphBaseURL      string
	restBaseURL           string
	syntheticsBaseURL     string
	insightsKeysBaseURL   string
}

// String returns a human readable value for the specified Region Name
func (n Name) String() string {
	return string(n)
}

// String returns a human readable value for the specified Region
func (r *Region) String() string {
	if r != nil && r.name != "" {
		return r.name
	}

	return "(Unknown)"
}

//
// NerdGraph - the future
//

// SetNerdGraphBaseURL Allows overriding the NerdGraph Base URL
func (r *Region) SetNerdGraphBaseURL(url string) {
	if r != nil && url != "" {
		r.nerdGraphBaseURL = url
	}
}

// NerdGraphURL returns the Full URL for Infrastructure REST API Calls, with any additional path elements appended
func (r *Region) NerdGraphURL(path ...string) string {
	if r == nil {
		log.Errorf("call to nil region.NerdGraphURL")
		return ""
	}

	url, err := concatURLPaths(r.nerdGraphBaseURL, path)
	if err != nil {
		log.Errorf("unable to make URL with error: %s", err)
		return r.nerdGraphBaseURL
	}

	return url
}

//
// REST
//

// SetRestBaseURL Allows overriding the REST Base URL
func (r *Region) SetRestBaseURL(url string) {
	if r != nil && url != "" {
		r.restBaseURL = url
	}
}

// RestURL returns the Full URL for REST API Calls, with any additional path elements appended
func (r *Region) RestURL(path ...string) string {
	if r == nil {
		log.Errorf("call to nil region.RestURL")
		return ""
	}

	url, err := concatURLPaths(r.restBaseURL, path)
	if err != nil {
		log.Errorf("unable to make URL with error: %s", err)
		return r.restBaseURL
	}

	return url
}

//
// Infrastructure
//

// SetInfrastructureBaseURL Allows overriding the Infrastructure Base URL
func (r *Region) SetInfrastructureBaseURL(url string) {
	if r != nil && url != "" {
		r.infrastructureBaseURL = url
	}
}

// InfrastructureURL returns the Full URL for Infrastructure REST API Calls, with any additional path elements appended
func (r *Region) InfrastructureURL(path ...string) string {
	if r == nil {
		log.Errorf("call to nil region.InfrastructureURL")
		return ""
	}

	url, err := concatURLPaths(r.infrastructureBaseURL, path)
	if err != nil {
		log.Errorf("unable to make URL with error: %s", err)
		return r.infrastructureBaseURL
	}

	return url
}

//
// Synthetics
//

// SetSyntheticsBaseURL Allows overriding the Synthetics Base URL
func (r *Region) SetSyntheticsBaseURL(url string) {
	if r != nil && url != "" {
		r.syntheticsBaseURL = url
	}
}

// SyntheticsURL returns the Full URL for Synthetics REST API Calls, with any additional path elements appended
func (r *Region) SyntheticsURL(path ...string) string {
	if r == nil {
		log.Errorf("call to nil region.SyntheticsURL")
		return ""
	}

	url, err := concatURLPaths(r.syntheticsBaseURL, path)
	if err != nil {
		log.Errorf("unable to make URL with error: %s", err)
		return r.syntheticsBaseURL
	}

	return url
}

//
// Insights Insert URL
//

func (r *Region) SetInsightsBaseURL(url string) {
	if r != nil && url != "" {
		r.insightsBaseURL = url
	}
}

// InsightsURL returns the Full URL for Insights custom insert API calls
func (r *Region) InsightsURL(accountID int) string {
	if r == nil {
		log.Errorf("call to nil region.SyntheticsURL")
		return ""
	}
	if accountID < 1 {
		log.Errorf("invalid account ID: %d", accountID)
		return ""
	}

	return fmt.Sprintf("%s/accounts/%d/events", r.insightsBaseURL, accountID)
}

// concatURLPaths is a helper function for the URL builders below
func concatURLPaths(host string, path []string) (string, error) {
	if host == "" {
		return "", errors.New("host can not be empty")
	}

	elements := make([]string, len(path)+1)
	elements[0] = strings.TrimSuffix(host, "/")

	for k, v := range path {
		elements[k+1] = strings.Trim(v, "/")
	}

	return strings.Join(elements, "/"), nil
}

//
// Insights Keys
//

func (r *Region) SetInsightsKeysBaseURL(url string) {
	if r != nil && url != "" {
		r.insightsKeysBaseURL = url
	}
}

// InsightsURL returns the Full URL for Insights custom insert API calls
func (r *Region) InsightsKeysURL(accountID int, path string) string {
	if r == nil {
		log.Errorf("call to nil region.InsightsKeysURL")
		return ""
	}
	if accountID < 1 {
		log.Errorf("invalid account ID: %d", accountID)
		return ""
	}

	return fmt.Sprintf("%s/accounts/%d/%s", r.insightsKeysBaseURL, accountID, path)
}

//
// Logs
//

// SetLogsBaseURL Allows overriding the Logs Base URL
func (r *Region) SetLogsBaseURL(url string) {
	if r != nil && url != "" {
		r.logsBaseURL = url
	}
}

// LogsURL returns the Full URL for the Log API
func (r *Region) LogsURL() string {
	if r == nil {
		log.Errorf("call to nil region.LogsURL")
		return ""
	}

	return r.logsBaseURL
}
