package apm

import (
	"context"
	"fmt"
)

// ApplicationInstanceLinks represents all the links for a New Relic application instance.
type ApplicationInstanceLinks struct {
	Application     int `json:"application,omitempty"`
	ApplicationHost int `json:"application_host,omitempty"`
}

// ApplicationInstance represents information about a New Relic application instance.
type ApplicationInstance struct {
	ID              int                       `json:"id,omitempty"`
	ApplicationName string                    `json:"application_name,omitempty"`
	Host            string                    `json:"host,omitempty"`
	Port            int                       `json:"port,omitempty"`
	Language        string                    `json:"language,omitempty"`
	HealthStatus    string                    `json:"health_status,omitempty"`
	Summary         ApplicationSummary        `json:"application_summary,omitempty"`
	EndUserSummary  ApplicationEndUserSummary `json:"end_user_summary,omitempty"`
	Links           ApplicationInstanceLinks  `json:"links,omitempty"`
}

// ListApplicationInstancesParams represents a set of filters to be
// used when querying New Relic application instances.
type ListApplicationInstancesParams struct {
	Hostname string `url:"filter[hostname],omitempty"`
	IDs      []int  `url:"filter[ids],omitempty,comma"`
}

// ListApplicationInstances is used to retrieve New Relic application instances.
func (a *APM) ListApplicationInstances(applicationID int, params *ListApplicationInstancesParams) ([]*ApplicationInstance, error) {
	return a.ListApplicationInstancesWithContext(context.Background(), applicationID, params)
}

// ListApplicationInstancesWithContext is used to retrieve New Relic application instances.
func (a *APM) ListApplicationInstancesWithContext(ctx context.Context, applicationID int, params *ListApplicationInstancesParams) ([]*ApplicationInstance, error) {
	instances := []*ApplicationInstance{}
	url := fmt.Sprintf("/applications/%d/instances.json", applicationID)
	nextURL := a.config.Region().RestURL(url)

	for nextURL != "" {
		response := applicationInstancesResponse{}
		resp, err := a.client.GetWithContext(ctx, nextURL, &params, &response)

		if err != nil {
			return nil, err
		}

		instances = append(instances, response.ApplicationInstances...)

		paging := a.pager.Parse(resp)
		nextURL = paging.Next
	}

	return instances, nil
}

// GetApplicationInstance is used to retrieve a specific New Relic application instance.
func (a *APM) GetApplicationInstance(applicationID int, instanceID int) (*ApplicationInstance, error) {
	return a.GetApplicationInstanceWithContext(context.Background(), applicationID, instanceID)
}

// GetApplicationInstanceWithContext is used to retrieve a specific New Relic application instance.
func (a *APM) GetApplicationInstanceWithContext(ctx context.Context, applicationID int, instanceID int) (*ApplicationInstance, error) {
	response := applicationInstanceResponse{}
	url := fmt.Sprintf("/applications/%d/instances/%d.json", applicationID, instanceID)

	_, err := a.client.GetWithContext(ctx, a.config.Region().RestURL(url), nil, &response)

	if err != nil {
		return nil, err
	}

	return response.ApplicationInstance, nil
}

type applicationInstancesResponse struct {
	ApplicationInstances []*ApplicationInstance `json:"application_instances,omitempty"`
}

type applicationInstanceResponse struct {
	ApplicationInstance *ApplicationInstance `json:"application_instance,omitempty"`
}
