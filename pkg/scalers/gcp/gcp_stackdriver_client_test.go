package gcp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBuildMQLQuery(t *testing.T) {
	for _, tc := range []struct {
		name         string
		resourceType string
		metric       string
		resourceName string
		aggregation  string
		timeHorizon  time.Duration

		expected string
		isError  bool
	}{
		{
			"topic with aggregation",
			"topic", "pubsub.googleapis.com/topic/x", "mytopic", "count", time.Minute,
			"fetch pubsub_topic | metric 'pubsub.googleapis.com/topic/x' | filter (resource.project_id == 'myproject' && resource.topic_id == 'mytopic')" +
				" | within 1m0s | align delta(3m0s) | every 3m0s | group_by [], count(value)",
			false,
		},
		{
			"topic without aggregation",
			"topic", "pubsub.googleapis.com/topic/x", "mytopic", "", time.Duration(0),
			"fetch pubsub_topic | metric 'pubsub.googleapis.com/topic/x' | filter (resource.project_id == 'myproject' && resource.topic_id == 'mytopic')" +
				" | within 2m0s",
			false,
		},
		{
			"subscription with aggregation",
			"subscription", "pubsub.googleapis.com/subscription/x", "mysubscription", "percentile99", time.Duration(0),
			"fetch pubsub_subscription | metric 'pubsub.googleapis.com/subscription/x' | filter (resource.project_id == 'myproject' && resource.subscription_id == 'mysubscription')" +
				" | within 5m0s | align delta(3m0s) | every 3m0s | group_by [], percentile(value, 99)",
			false,
		},
		{
			"subscription without aggregation",
			"subscription", "pubsub.googleapis.com/subscription/x", "mysubscription", "", time.Minute * 4,
			"fetch pubsub_subscription | metric 'pubsub.googleapis.com/subscription/x' | filter (resource.project_id == 'myproject' && resource.subscription_id == 'mysubscription')" +
				" | within 4m0s",
			false,
		},
		{
			"invalid percentile",
			"topic", "pubsub.googleapis.com/topic/x", "mytopic", "percentile101", time.Minute,
			"invalid percentile value: 101",
			true,
		},
		{
			"unsupported aggregation function",
			"topic", "pubsub.googleapis.com/topic/x", "mytopic", "max", time.Duration(0),
			"unsupported aggregation function: max",
			true,
		},
	} {
		s := &StackDriverClient{}
		t.Run(tc.name, func(t *testing.T) {
			q, err := s.BuildMQLQuery("myproject", tc.resourceType, tc.metric, tc.resourceName, tc.aggregation, tc.timeHorizon)
			if tc.isError {
				assert.Error(t, err)
				assert.Equal(t, tc.expected, err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, q)
			}
		})
	}
}

func TestGetActualProjectID(t *testing.T) {
	// There are three ways to get projectID
	// This is ordered from highest priority to lowest priority
	pidFromMetadata := "myproject0"
	pidFromClient := "myproject1"
	pidFromClientCreds := "myproject2"

	for _, tc := range []struct {
		name      string
		projectID string
		client    *StackDriverClient
		expected  string
	}{
		{
			"all projectID present",
			pidFromMetadata,
			&StackDriverClient{
				projectID: pidFromClient,
				credentials: GoogleApplicationCredentials{
					ProjectID: pidFromClientCreds,
				},
			},
			pidFromMetadata,
		},
		{
			"both projectID from metadata and client present",
			pidFromMetadata,
			&StackDriverClient{
				projectID: pidFromClient,
			},
			pidFromMetadata,
		},
		{
			"both projectID from metadata and client credentials present",
			pidFromMetadata,
			&StackDriverClient{
				credentials: GoogleApplicationCredentials{
					ProjectID: pidFromClientCreds,
				},
			},
			pidFromMetadata,
		},
		{
			"both projectID from client and client credentials present",
			"",
			&StackDriverClient{
				projectID: pidFromClient,
				credentials: GoogleApplicationCredentials{
					ProjectID: pidFromClientCreds,
				},
			},
			pidFromClient,
		},
		{
			"projectID from metadata only",
			pidFromMetadata,
			&StackDriverClient{},
			pidFromMetadata,
		},
		{
			"projectID from client only",
			"",
			&StackDriverClient{
				projectID: pidFromClient,
			},
			pidFromClient,
		},
		{
			"projectID from client credentials only",
			"",
			&StackDriverClient{
				projectID: "",
				credentials: GoogleApplicationCredentials{
					ProjectID: pidFromClientCreds,
				},
			},
			pidFromClientCreds,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pid := getActualProjectID(tc.client, tc.projectID)
			assert.Equal(t, pid, tc.expected)
		})
	}
}
