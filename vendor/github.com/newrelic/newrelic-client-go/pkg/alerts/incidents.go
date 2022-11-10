package alerts

import (
	"context"
	"fmt"

	"github.com/newrelic/newrelic-client-go/internal/serialization"
)

// Incident represents a New Relic alert incident.
type Incident struct {
	ID                 int                      `json:"id,omitempty"`
	OpenedAt           *serialization.EpochTime `json:"opened_at,omitempty"`
	ClosedAt           *serialization.EpochTime `json:"closed_at,omitempty"`
	IncidentPreference string                   `json:"incident_preference,omitempty"`
	Links              IncidentLink             `json:"links"`
}

// IncidentLink represents a link between a New Relic alert incident and its violations
type IncidentLink struct {
	Violations []int `json:"violations,omitempty"`
	PolicyID   int   `json:"policy_id"`
}

// ListIncidents returns all alert incidents.
func (a *Alerts) ListIncidents(onlyOpen bool, excludeViolations bool) ([]*Incident, error) {
	return a.ListIncidentsWithContext(context.Background(), onlyOpen, excludeViolations)
}

// ListIncidentsWithContext returns all alert incidents.
func (a *Alerts) ListIncidentsWithContext(ctx context.Context, onlyOpen bool, excludeViolations bool) ([]*Incident, error) {
	incidents := []*Incident{}
	queryParams := listIncidentsParams{
		OnlyOpen:          onlyOpen,
		ExcludeViolations: excludeViolations,
	}

	nextURL := a.config.Region().RestURL("/alerts_incidents.json")

	for nextURL != "" {
		incidentsResponse := alertIncidentsResponse{}
		resp, err := a.client.GetWithContext(ctx, nextURL, queryParams, &incidentsResponse)

		if err != nil {
			return nil, err
		}

		incidents = append(incidents, incidentsResponse.Incidents...)

		paging := a.pager.Parse(resp)
		nextURL = paging.Next
	}

	return incidents, nil
}

// AcknowledgeIncident acknowledges an existing incident.
func (a *Alerts) AcknowledgeIncident(id int) (*Incident, error) {
	return a.AcknowledgeIncidentWithContext(context.Background(), id)
}

// AcknowledgeIncidentWithContext acknowledges an existing incident.
func (a *Alerts) AcknowledgeIncidentWithContext(ctx context.Context, id int) (*Incident, error) {
	return a.updateIncident(ctx, id, "acknowledge")
}

// CloseIncident closes an existing open incident.
func (a *Alerts) CloseIncident(id int) (*Incident, error) {
	return a.CloseIncidentWithContext(context.Background(), id)
}

// CloseIncidentWithContext closes an existing open incident.
func (a *Alerts) CloseIncidentWithContext(ctx context.Context, id int) (*Incident, error) {
	return a.updateIncident(ctx, id, "close")
}

func (a *Alerts) updateIncident(ctx context.Context, id int, verb string) (*Incident, error) {
	response := alertIncidentResponse{}
	url := fmt.Sprintf("/alerts_incidents/%d/%s.json", id, verb)
	_, err := a.client.PutWithContext(ctx, a.config.Region().RestURL(url), nil, nil, &response)

	if err != nil {
		return nil, err
	}

	return &response.Incident, nil
}

type listIncidentsParams struct {
	OnlyOpen          bool `url:"only_open,omitempty"`
	ExcludeViolations bool `url:"exclude_violations,omitempty"`
}

type alertIncidentsResponse struct {
	Incidents []*Incident `json:"incidents,omitempty"`
}

type alertIncidentResponse struct {
	Incident Incident `json:"incident,omitempty"`
}
