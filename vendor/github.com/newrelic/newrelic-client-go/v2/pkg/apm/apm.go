// Package apm provides a programmatic API for interacting with the New Relic APM product.
package apm

import (
	"github.com/newrelic/newrelic-client-go/v2/internal/http"
	"github.com/newrelic/newrelic-client-go/v2/pkg/config"
)

// APM is used to communicate with the New Relic APM product.
type APM struct {
	client http.Client
	config config.Config
	pager  http.Pager
}

// New is used to create a new APM client instance.
func New(config config.Config) APM {
	client := http.NewClient(config)
	client.SetAuthStrategy(&http.PersonalAPIKeyCapableV2Authorizer{})

	pkg := APM{
		client: client,
		config: config,
		pager:  &http.LinkHeaderPager{},
	}

	return pkg
}
