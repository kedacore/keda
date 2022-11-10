package apm

import (
	"context"
	"fmt"
	"strconv"

	"github.com/newrelic/newrelic-client-go/internal/http"
)

// Deployment represents information about a New Relic application deployment.
type Deployment struct {
	Links       *DeploymentLinks `json:"links,omitempty"`
	ID          int              `json:"id,omitempty"`
	Revision    string           `json:"revision"`
	Changelog   string           `json:"changelog,omitempty"`
	Description string           `json:"description,omitempty"`
	User        string           `json:"user,omitempty"`
	Timestamp   string           `json:"timestamp,omitempty"`
}

// DeploymentLinks contain the application ID for the deployment.
type DeploymentLinks struct {
	ApplicationID int `json:"application,omitempty"`
}

// ListDeployments returns deployments for an application.
func (a *APM) ListDeployments(applicationID int) ([]*Deployment, error) {
	return a.ListDeploymentsWithContext(context.Background(), applicationID)
}

// ListDeploymentsWithContext returns deployments for an application.
func (a *APM) ListDeploymentsWithContext(ctx context.Context, applicationID int) ([]*Deployment, error) {
	deployments := []*Deployment{}
	nextURL := a.config.Region().RestURL("applications", strconv.Itoa(applicationID), "deployments.json")

	for nextURL != "" {
		response := deploymentsResponse{}
		req, err := a.client.NewRequest("GET", nextURL, nil, nil, &response)
		if err != nil {
			return nil, err
		}

		req.WithContext(ctx)
		req.SetAuthStrategy(&http.PersonalAPIKeyCapableV2Authorizer{})

		resp, err := a.client.Do(req)

		if err != nil {
			return nil, err
		}

		deployments = append(deployments, response.Deployments...)

		paging := a.pager.Parse(resp)
		nextURL = paging.Next
	}

	return deployments, nil
}

// CreateDeployment creates a deployment marker for an application.
func (a *APM) CreateDeployment(applicationID int, deployment Deployment) (*Deployment, error) {
	return a.CreateDeploymentWithContext(context.Background(), applicationID, deployment)
}

// CreateDeploymentWithContext creates a deployment marker for an application.
func (a *APM) CreateDeploymentWithContext(ctx context.Context, applicationID int, deployment Deployment) (*Deployment, error) {
	reqBody := deploymentRequestBody{
		Deployment: deployment,
	}
	resp := deploymentResponse{}

	url := a.config.Region().RestURL("applications", strconv.Itoa(applicationID), "deployments.json")
	req, err := a.client.NewRequest("POST", url, nil, &reqBody, &resp)
	if err != nil {
		return nil, err
	}

	req.WithContext(ctx)
	req.SetAuthStrategy(&http.PersonalAPIKeyCapableV2Authorizer{})

	_, err = a.client.Do(req)

	if err != nil {
		return nil, err
	}

	return &resp.Deployment, nil
}

// DeleteDeployment deletes a deployment marker for an application.
func (a *APM) DeleteDeployment(applicationID int, deploymentID int) (*Deployment, error) {
	return a.DeleteDeploymentWithContext(context.Background(), applicationID, deploymentID)
}

// DeleteDeploymentWithContext deletes a deployment marker for an application.
func (a *APM) DeleteDeploymentWithContext(ctx context.Context, applicationID int, deploymentID int) (*Deployment, error) {
	resp := deploymentResponse{}
	url := a.config.Region().RestURL("applications", strconv.Itoa(applicationID), "deployments", fmt.Sprintf("%d.json", deploymentID))

	req, err := a.client.NewRequest("DELETE", url, nil, nil, &resp)
	if err != nil {
		return nil, err
	}

	req.WithContext(ctx)
	req.SetAuthStrategy(&http.PersonalAPIKeyCapableV2Authorizer{})

	_, err = a.client.Do(req)

	if err != nil {
		return nil, err
	}

	return &resp.Deployment, nil
}

type deploymentsResponse struct {
	Deployments []*Deployment `json:"deployments,omitempty"`
}

type deploymentResponse struct {
	Deployment Deployment `json:"deployment,omitempty"`
}

type deploymentRequestBody struct {
	Deployment Deployment `json:"deployment,omitempty"`
}
