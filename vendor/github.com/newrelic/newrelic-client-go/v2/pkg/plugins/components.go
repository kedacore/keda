package plugins

import (
	"fmt"
	"strconv"
	"time"
)

// ListComponentsParams represents a set of filters to be
// used when querying New Relic applications.
type ListComponentsParams struct {
	Name         string `url:"filter[name],omitempty"`
	IDs          []int  `url:"filter[ids],omitempty,comma"`
	PluginID     int    `url:"filter[plugin_id],omitempty"`
	HealthStatus bool   `url:"health_status,omitempty"`
}

// ListComponents is used to retrieve the components associated with
// a New Relic account.
func (p *Plugins) ListComponents(params *ListComponentsParams) ([]*Component, error) {
	c := []*Component{}
	nextURL := p.config.Region().RestURL("components.json")

	for nextURL != "" {
		response := componentsResponse{}
		resp, err := p.client.Get(nextURL, &params, &response)

		if err != nil {
			return nil, err
		}

		c = append(c, response.Components...)

		paging := p.pager.Parse(resp)
		nextURL = paging.Next
	}

	return c, nil

}

// GetComponent is used to retrieve a specific New Relic component.
func (p *Plugins) GetComponent(componentID int) (*Component, error) {
	response := componentResponse{}
	url := fmt.Sprintf("/components/%d.json", componentID)

	_, err := p.client.Get(p.config.Region().RestURL(url), nil, &response)

	if err != nil {
		return nil, err
	}

	return &response.Component, nil
}

// ListComponentMetricsParams represents a set of parameters to be
// used when querying New Relic component metrics.
type ListComponentMetricsParams struct {
	// Name allows for filtering the returned list of metrics by name.
	Name string `url:"name,omitempty"`
}

// ListComponentMetrics is used to retrieve the metrics for a specific New Relic component.
func (p *Plugins) ListComponentMetrics(componentID int, params *ListComponentMetricsParams) ([]*ComponentMetric, error) {
	m := []*ComponentMetric{}
	response := componentMetricsResponse{}
	nextURL := p.config.Region().RestURL("components", strconv.Itoa(componentID), "metrics.json")

	for nextURL != "" {
		resp, err := p.client.Get(nextURL, &params, &response)

		if err != nil {
			return nil, err
		}

		m = append(m, response.Metrics...)

		paging := p.pager.Parse(resp)
		nextURL = paging.Next
	}

	return m, nil
}

// GetComponentMetricDataParams represents a set of parameters to be
// used when querying New Relic component metric data.
type GetComponentMetricDataParams struct {
	// Names allows retrieval of specific metrics by name.
	// At least one metric name is required.
	Names []string `url:"names[],omitempty"`

	// Values allows retrieval of specific metric values.
	Values []string `url:"values[],omitempty"`

	// From specifies a begin time for the query.
	From *time.Time `url:"from,omitempty"`

	// To specifies an end time for the query.
	To *time.Time `url:"to,omitempty"`

	// Period represents the period of timeslices in seconds.
	Period int `url:"period,omitempty"`

	// Summarize will summarize the data when set to true.
	Summarize bool `url:"summarize,omitempty"`

	// Raw will return unformatted raw values when set to true.
	Raw bool `url:"raw,omitempty"`
}

// GetComponentMetricData is used to retrieve the metric timeslice data for a specific component metric.
func (p *Plugins) GetComponentMetricData(componentID int, params *GetComponentMetricDataParams) ([]*Metric, error) {
	m := []*Metric{}
	response := componentMetricDataResponse{}
	nextURL := p.config.Region().RestURL("components", strconv.Itoa(componentID), "metrics/data.json")

	for nextURL != "" {
		resp, err := p.client.Get(nextURL, &params, &response)

		if err != nil {
			return nil, err
		}

		m = append(m, response.MetricData.Metrics...)

		paging := p.pager.Parse(resp)
		nextURL = paging.Next
	}

	return m, nil
}

type componentsResponse struct {
	Components []*Component `json:"components,omitempty"`
}

type componentResponse struct {
	Component Component `json:"component,omitempty"`
}

type componentMetricsResponse struct {
	Metrics []*ComponentMetric `json:"metrics,omitempty"`
}

type componentMetricDataResponse struct {
	MetricData struct {
		Metrics []*Metric `json:"metrics,omitempty"`
	} `json:"metric_data,omitempty"`
}
