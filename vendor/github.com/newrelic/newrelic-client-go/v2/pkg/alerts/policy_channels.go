package alerts

import (
	"context"
	"strconv"
)

// PolicyChannels represents an association of alert channels to a specific alert policy.
type PolicyChannels struct {
	ID         int   `json:"id,omitempty"`
	ChannelIDs []int `json:"channel_ids,omitempty"`
}

// UpdatePolicyChannels updates a policy by adding the specified notification channels.
func (a *Alerts) UpdatePolicyChannels(policyID int, channelIDs []int) (*PolicyChannels, error) {
	return a.UpdatePolicyChannelsWithContext(context.Background(), policyID, channelIDs)
}

// UpdatePolicyChannelsWithContext updates a policy by adding the specified notification channels.
func (a *Alerts) UpdatePolicyChannelsWithContext(ctx context.Context, policyID int, channelIDs []int) (*PolicyChannels, error) {
	channelIDStrings := make([]string, len(channelIDs))

	for i, channelID := range channelIDs {
		channelIDStrings[i] = strconv.Itoa(channelID)
	}

	queryParams := updatePolicyChannelsParams{
		PolicyID:   policyID,
		ChannelIDs: channelIDs,
	}

	resp := updatePolicyChannelsResponse{}

	_, err := a.client.PutWithContext(ctx, a.config.Region().RestURL("/alerts_policy_channels.json"), &queryParams, nil, &resp)

	if err != nil {
		return nil, err
	}

	return &resp.Policy, nil
}

// DeletePolicyChannel deletes a notification channel from an alert policy.
// This method returns a response containing the Channel that was deleted from the policy.
func (a *Alerts) DeletePolicyChannel(policyID int, channelID int) (*Channel, error) {
	return a.DeletePolicyChannelWithContext(context.Background(), policyID, channelID)
}

// DeletePolicyChannelWithContext deletes a notification channel from an alert policy.
// This method returns a response containing the Channel that was deleted from the policy.
func (a *Alerts) DeletePolicyChannelWithContext(ctx context.Context, policyID int, channelID int) (*Channel, error) {
	queryParams := deletePolicyChannelsParams{
		PolicyID:  policyID,
		ChannelID: channelID,
	}

	resp := deletePolicyChannelResponse{}

	_, err := a.client.DeleteWithContext(ctx, a.config.Region().RestURL("/alerts_policy_channels.json"), &queryParams, &resp)

	if err != nil {
		return nil, err
	}

	return &resp.Channel, nil
}

type updatePolicyChannelsParams struct {
	PolicyID   int   `url:"policy_id,omitempty"`
	ChannelIDs []int `url:"channel_ids,comma"`
}

type deletePolicyChannelsParams struct {
	PolicyID  int `url:"policy_id,omitempty"`
	ChannelID int `url:"channel_id,omitempty"`
}

type updatePolicyChannelsResponse struct {
	Policy PolicyChannels `json:"policy,omitempty"`
}

type deletePolicyChannelResponse struct {
	Channel Channel           `json:"channel,omitempty"`
	Links   map[string]string `json:"channel.policy_ids,omitempty"`
}
