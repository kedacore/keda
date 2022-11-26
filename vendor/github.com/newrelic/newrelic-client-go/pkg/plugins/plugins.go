// Package plugins provides a programmatic API for interacting with the New Relic Plugins product.
package plugins

import (
	"fmt"

	"github.com/newrelic/newrelic-client-go/internal/http"
	"github.com/newrelic/newrelic-client-go/pkg/config"
	"github.com/newrelic/newrelic-client-go/pkg/logging"
)

// Plugins is used to communicate with the New Relic Plugins product.
type Plugins struct {
	client http.Client
	config config.Config
	logger logging.Logger
	pager  http.Pager
}

// New is used to create a new Plugins client instance.
func New(config config.Config) Plugins {
	client := http.NewClient(config)
	client.SetAuthStrategy(&http.PersonalAPIKeyCapableV2Authorizer{})

	pkg := Plugins{
		client: client,
		config: config,
		logger: config.GetLogger(),
		pager:  &http.LinkHeaderPager{},
	}

	return pkg
}

// ListPluginsParams represents a set of query string parameters
// used as filters when querying New Relic plugins.
type ListPluginsParams struct {
	GUID     string `url:"filter[guid],omitempty"`
	IDs      []int  `url:"filter[ids],omitempty,comma"`
	Detailed bool   `url:"detailed,omitempty"`
}

// ListPlugins returns a list of Plugins associated with an account.
// If the query paramater `detailed=true` is provided, the plugins
// response objects will contain an additional `details` property
// with metadata pertaining to each plugin.
func (p *Plugins) ListPlugins(params *ListPluginsParams) ([]*Plugin, error) {
	results := []*Plugin{}
	nextURL := p.config.Region().RestURL("plugins.json")

	for nextURL != "" {
		response := pluginsResponse{}
		resp, err := p.client.Get(nextURL, &params, &response)

		if err != nil {
			return nil, err
		}

		results = append(results, response.Plugins...)

		paging := p.pager.Parse(resp)
		nextURL = paging.Next
	}

	return results, nil
}

// GetPluginParams represents a set of query string parameters
// to apply to the request.
type GetPluginParams struct {
	Detailed bool `url:"detailed,omitempty"`
}

// GetPlugin returns a plugin for a given account. If the query paramater `detailed=true`
// is provided, the response will contain an additional `details` property with
// metadata pertaining to the plugin.
func (p *Plugins) GetPlugin(id int, params *GetPluginParams) (*Plugin, error) {
	response := pluginResponse{}

	url := fmt.Sprintf("/plugins/%d.json", id)
	_, err := p.client.Get(p.config.Region().RestURL(url), &params, &response)

	if err != nil {
		return nil, err
	}

	return &response.Plugin, nil
}

type pluginsResponse struct {
	Plugins []*Plugin `json:"plugins,omitempty"`
}

type pluginResponse struct {
	Plugin Plugin `json:"plugin,omitempty"`
}
