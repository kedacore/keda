package apm

import (
	"context"
	"fmt"
)

// applicationsREST implements fetching Applications from the RESTv2 API
type applicationsREST struct {
	parent *APM
}

// list is used to retrieve New Relic applications.
func (a *applicationsREST) list(ctx context.Context, accountID int, params *ListApplicationsParams) ([]*Application, error) {
	apps := []*Application{}
	nextURL := a.parent.config.Region().RestURL("applications.json")

	for nextURL != "" {
		response := applicationsResponse{}
		resp, err := a.parent.client.GetWithContext(ctx, nextURL, &params, &response)

		if err != nil {
			return nil, err
		}

		apps = append(apps, response.Applications...)

		paging := a.parent.pager.Parse(resp)
		nextURL = paging.Next
	}

	return apps, nil
}

// get is used to retrieve a single New Relic application.
func (a *applicationsREST) get(ctx context.Context, accountID int, applicationID int) (*Application, error) {
	response := applicationResponse{}
	url := fmt.Sprintf("/applications/%d.json", applicationID)

	_, err := a.parent.client.GetWithContext(ctx, a.parent.config.Region().RestURL(url), nil, &response)

	if err != nil {
		return nil, err
	}

	return &response.Application, nil
}

// update is used to update a New Relic application's name and/or settings.
func (a *applicationsREST) update(ctx context.Context, accountID int, applicationID int, params UpdateApplicationParams) (*Application, error) {
	response := applicationResponse{}
	reqBody := updateApplicationRequest{
		Fields: updateApplicationFields(params),
	}
	url := fmt.Sprintf("/applications/%d.json", applicationID)

	_, err := a.parent.client.PutWithContext(ctx, a.parent.config.Region().RestURL(url), nil, &reqBody, &response)

	if err != nil {
		return nil, err
	}

	return &response.Application, nil
}

// remove is used to delete a New Relic application.
// This process will only succeed if the application is no longer reporting data.
func (a *applicationsREST) remove(ctx context.Context, accountID int, applicationID int) (*Application, error) {
	response := applicationResponse{}
	url := fmt.Sprintf("/applications/%d.json", applicationID)

	_, err := a.parent.client.DeleteWithContext(ctx, a.parent.config.Region().RestURL(url), nil, &response)

	if err != nil {
		return nil, err
	}

	return &response.Application, nil
}

type applicationsResponse struct {
	Applications []*Application `json:"applications,omitempty"`
}

type applicationResponse struct {
	Application Application `json:"application,omitempty"`
}

type updateApplicationRequest struct {
	Fields updateApplicationFields `json:"application"`
}

type updateApplicationFields struct {
	Name     string              `json:"name,omitempty"`
	Settings ApplicationSettings `json:"settings,omitempty"`
}
