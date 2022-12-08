package alerts

import (
	"context"

	"github.com/newrelic/newrelic-client-go/internal/serialization"
)

// AlertEvent response struct
type AlertEvent struct {
	ID            int                      `json:"id"`
	EventType     string                   `json:"event_type"`
	Product       string                   `json:"product"`
	EntityType    string                   `json:"entity_type"`
	EntityGroupID int                      `json:"entity_group_id"`
	EntityID      int                      `json:"entity_id"`
	Priority      string                   `json:"priority"`
	Description   string                   `json:"description"`
	Timestamp     *serialization.EpochTime `json:"timestamp"`
	IncidentID    int                      `json:"incident_id"`
}

// ListAlertEventsParams represents a set of filters to be used
// when querying New Relic alert events
type ListAlertEventsParams struct {
	Product       string `url:"filter[product],omitempty"`
	EntityType    string `url:"filter[entity_type],omitempty"`
	EntityGroupID int    `url:"filter[entity_group_id],omitempty"`
	EntityID      int    `url:"filter[entity_id],omitempty"`
	EventType     string `url:"filter[event_type],omitempty"`
	IncidentID    int    `url:"filter[incident_id],omitempty"`
	Page          int    `url:"page,omitempty"`
}

// ListAlertEvents is used to retrieve New Relic alert events
func (a *Alerts) ListAlertEvents(params *ListAlertEventsParams) ([]*AlertEvent, error) {
	return a.ListAlertEventsWithContext(context.Background(), params)
}

// ListAlertEventsWithContext is used to retrieve New Relic alert events
func (a *Alerts) ListAlertEventsWithContext(ctx context.Context, params *ListAlertEventsParams) ([]*AlertEvent, error) {
	alertEvents := []*AlertEvent{}
	nextURL := a.config.Region().RestURL("alerts_events.json")

	for nextURL != "" {
		response := alertEventsResponse{}
		resp, err := a.client.GetWithContext(ctx, nextURL, &params, &response)

		if err != nil {
			return nil, err
		}

		alertEvents = append(alertEvents, response.AlertEvents...)

		paging := a.pager.Parse(resp)
		nextURL = paging.Next
	}

	return alertEvents, nil
}

type alertEventsResponse struct {
	AlertEvents []*AlertEvent `json:"alert_events,omitempty"`
}
