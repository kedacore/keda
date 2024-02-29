package gcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildMQLQuery(t *testing.T) {
	for _, tc := range []struct {
		name         string
		resourceType string
		metric       string
		resourceName string
		aggregation  string

		expected string
		isError  bool
	}{
		{
			"topic with aggregation",
			"topic", "pubsub.googleapis.com/topic/x", "mytopic", "count",
			"fetch pubsub_topic | metric 'pubsub.googleapis.com/topic/x' | filter (resource.project_id == 'myproject' && resource.topic_id == 'mytopic')" +
				" | within 5m | align delta(3m) | every 3m | group_by [], count(value)",
			false,
		},
		{
			"topic without aggregation",
			"topic", "pubsub.googleapis.com/topic/x", "mytopic", "",
			"fetch pubsub_topic | metric 'pubsub.googleapis.com/topic/x' | filter (resource.project_id == 'myproject' && resource.topic_id == 'mytopic')" +
				" | within 2m",
			false,
		},
		{
			"subscription with aggregation",
			"subscription", "pubsub.googleapis.com/subscription/x", "mysubscription", "percentile99",
			"fetch pubsub_subscription | metric 'pubsub.googleapis.com/subscription/x' | filter (resource.project_id == 'myproject' && resource.subscription_id == 'mysubscription')" +
				" | within 5m | align delta(3m) | every 3m | group_by [], percentile(value, 99)",
			false,
		},
		{
			"subscription without aggregation",
			"subscription", "pubsub.googleapis.com/subscription/x", "mysubscription", "",
			"fetch pubsub_subscription | metric 'pubsub.googleapis.com/subscription/x' | filter (resource.project_id == 'myproject' && resource.subscription_id == 'mysubscription')" +
				" | within 2m",
			false,
		},
		{
			"invalid percentile",
			"topic", "pubsub.googleapis.com/topic/x", "mytopic", "percentile101",
			"invalid percentile value: 101",
			true,
		},
		{
			"unsupported aggregation function",
			"topic", "pubsub.googleapis.com/topic/x", "mytopic", "max",
			"unsupported aggregation function: max",
			true,
		},
	} {
		s := &StackDriverClient{}
		t.Run(tc.name, func(t *testing.T) {
			q, err := s.BuildMQLQuery("myproject", tc.resourceType, tc.metric, tc.resourceName, tc.aggregation)
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
