// Package dashboards provides a programmatic API for interacting with New Relic dashboards.
package dashboards

import (
	"github.com/newrelic/newrelic-client-go/internal/http"
	"github.com/newrelic/newrelic-client-go/pkg/config"
	"github.com/newrelic/newrelic-client-go/pkg/logging"
)

// Dashboards is used to communicate with the New Relic Dashboards product.
type Dashboards struct {
	client http.Client
	config config.Config
	logger logging.Logger
	pager  http.Pager
}

// New is used to create a new Dashboards client instance.
func New(config config.Config) Dashboards {
	client := http.NewClient(config)
	client.SetAuthStrategy(&http.PersonalAPIKeyCapableV2Authorizer{})

	pkg := Dashboards{
		client: client,
		config: config,
		logger: config.GetLogger(),
		pager:  &http.LinkHeaderPager{},
	}

	return pkg
}
