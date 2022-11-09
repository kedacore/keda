package alerts

import (
	"context"
	"fmt"

	"github.com/newrelic/newrelic-client-go/pkg/errors"

	"github.com/newrelic/newrelic-client-go/internal/serialization"
)

// ChannelType specifies the channel type used when creating the alert channel.
type ChannelType string

var (
	// ChannelTypes enumerates the possible channel types for an alert channel.
	ChannelTypes = struct {
		Email     ChannelType
		OpsGenie  ChannelType
		PagerDuty ChannelType
		Slack     ChannelType
		User      ChannelType
		VictorOps ChannelType
		Webhook   ChannelType
	}{
		Email:     "email",
		OpsGenie:  "opsgenie",
		PagerDuty: "pagerduty",
		Slack:     "slack",
		User:      "user",
		VictorOps: "victorops",
		Webhook:   "webhook",
	}
)

// Channel represents a New Relic alert notification channel
type Channel struct {
	ID            int                  `json:"id,omitempty"`
	Name          string               `json:"name,omitempty"`
	Type          ChannelType          `json:"type,omitempty"`
	Configuration ChannelConfiguration `json:"configuration,omitempty"`
	Links         ChannelLinks         `json:"links,omitempty"`
}

// ChannelLinks represent the links between policies and alert channels
type ChannelLinks struct {
	PolicyIDs []int `json:"policy_ids,omitempty"`
}

// ChannelConfiguration represents a Configuration type within Channels
type ChannelConfiguration struct {
	Recipients            string `json:"recipients,omitempty"`
	IncludeJSONAttachment string `json:"include_json_attachment,omitempty"`
	AuthToken             string `json:"auth_token,omitempty"`
	APIKey                string `json:"api_key,omitempty"`
	Teams                 string `json:"teams,omitempty"`
	Tags                  string `json:"tags,omitempty"`
	URL                   string `json:"url,omitempty"`
	Channel               string `json:"channel,omitempty"`
	Key                   string `json:"key,omitempty"`
	RouteKey              string `json:"route_key,omitempty"`
	ServiceKey            string `json:"service_key,omitempty"`
	BaseURL               string `json:"base_url,omitempty"`
	AuthUsername          string `json:"auth_username,omitempty"`
	AuthPassword          string `json:"auth_password,omitempty"`
	PayloadType           string `json:"payload_type,omitempty"`
	Region                string `json:"region,omitempty"`
	UserID                string `json:"user_id,omitempty"`

	// Payload is unmarshaled to type map[string]string
	Payload serialization.MapStringInterface `json:"payload,omitempty"`

	// Headers is unmarshaled to type map[string]string
	Headers serialization.MapStringInterface `json:"headers,omitempty"`
}

// ListChannels returns all alert channels for a given account.
func (a *Alerts) ListChannels() ([]*Channel, error) {
	return a.ListChannelsWithContext(context.Background())
}

// ListChannelsWithContext returns all alert channels for a given account.
func (a *Alerts) ListChannelsWithContext(ctx context.Context) ([]*Channel, error) {
	alertChannels := []*Channel{}
	nextURL := a.config.Region().RestURL("/alerts_channels.json")

	for nextURL != "" {
		response := alertChannelsResponse{}
		resp, err := a.client.GetWithContext(ctx, nextURL, nil, &response)

		if err != nil {
			return nil, err
		}

		alertChannels = append(alertChannels, response.Channels...)

		paging := a.pager.Parse(resp)
		nextURL = paging.Next
	}

	return alertChannels, nil
}

// GetChannel returns a specific alert channel by ID for a given account.
func (a *Alerts) GetChannel(id int) (*Channel, error) {
	return a.GetChannelWithContext(context.Background(), id)
}

// GetChannelWithContext returns a specific alert channel by ID for a given account.
func (a *Alerts) GetChannelWithContext(ctx context.Context, id int) (*Channel, error) {
	channels, err := a.ListChannelsWithContext(ctx)
	if err != nil {
		return nil, err
	}

	for _, channel := range channels {
		if channel.ID == id {
			return channel, nil
		}
	}
	return nil, errors.NewNotFoundf("no channel found for id %d", id)
}

// CreateChannel creates an alert channel within a given account.
// The configuration options different based on channel type.
// For more information on the different configurations, please
// view the New Relic API documentation for this endpoint.
// Docs: https://docs.newrelic.com/docs/alerts/rest-api-alerts/new-relic-alerts-rest-api/rest-api-calls-new-relic-alerts#channels
func (a *Alerts) CreateChannel(channel Channel) (*Channel, error) {
	return a.CreateChannelWithContext(context.Background(), channel)
}

// CreateChannelWithContext creates an alert channel within a given account.
// The configuration options different based on channel type.
// For more information on the different configurations, please
// view the New Relic API documentation for this endpoint.
// Docs: https://docs.newrelic.com/docs/alerts/rest-api-alerts/new-relic-alerts-rest-api/rest-api-calls-new-relic-alerts#channels
func (a *Alerts) CreateChannelWithContext(ctx context.Context, channel Channel) (*Channel, error) {
	reqBody := alertChannelRequestBody{
		Channel: channel,
	}
	resp := alertChannelsResponse{}

	_, err := a.client.PostWithContext(ctx, a.config.Region().RestURL("alerts_channels.json"), nil, &reqBody, &resp)

	if err != nil {
		return nil, err
	}

	return resp.Channels[0], nil
}

// DeleteChannel deletes the alert channel with the specified ID.
func (a *Alerts) DeleteChannel(id int) (*Channel, error) {
	return a.DeleteChannelWithContext(context.Background(), id)
}

// DeleteChannelWithContext deletes the alert channel with the specified ID.
func (a *Alerts) DeleteChannelWithContext(ctx context.Context, id int) (*Channel, error) {
	resp := alertChannelResponse{}
	url := fmt.Sprintf("/alerts_channels/%d.json", id)
	_, err := a.client.DeleteWithContext(ctx, a.config.Region().RestURL(url), nil, &resp)

	if err != nil {
		return nil, err
	}

	return &resp.Channel, nil
}

type alertChannelsResponse struct {
	Channels []*Channel `json:"channels,omitempty"`
}

type alertChannelResponse struct {
	Channel Channel `json:"channel,omitempty"`
}

type alertChannelRequestBody struct {
	Channel Channel `json:"channel"`
}
