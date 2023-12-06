package scalers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type parseMQLQueryTestCase struct {
	name         string
	resourceType string
	metric       string
	resourceName string
	aggregation  string

	expected string
	isError  bool
}

var testMQLQueries = []parseMQLQueryTestCase{
	{
		"topic with aggregation",
		"topic", "pubsub.googleapis.com/topic/x", "mytopic", "count",
		"fetch pubsub_topic | metric 'pubsub.googleapis.com/topic/x' | filter (resource.topic_id == 'mytopic')" +
			" | within 5m | align delta(3m) | every 3m | group_by [], count(value)",
		false,
	},
	{
		"topic without aggregation",
		"topic", "pubsub.googleapis.com/topic/x", "mytopic", "",
		"fetch pubsub_topic | metric 'pubsub.googleapis.com/topic/x' | filter (resource.topic_id == 'mytopic')" +
			" | within 1m",
		false,
	},
	{
		"subscription with aggregation",
		"subscription", "pubsub.googleapis.com/subscription/x", "mysubscription", "percentile99",
		"fetch pubsub_subscription | metric 'pubsub.googleapis.com/subscription/x' | filter (resource.subscription_id == 'mysubscription')" +
			" | within 5m | align delta(3m) | every 3m | group_by [], percentile(value, 99)",
		false,
	},
	{
		"subscription without aggregation",
		"subscription", "pubsub.googleapis.com/subscription/x", "mysubscription", "",
		"fetch pubsub_subscription | metric 'pubsub.googleapis.com/subscription/x' | filter (resource.subscription_id == 'mysubscription')" +
			" | within 1m",
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
}

func TestBuildMQLQuery(t *testing.T) {
	for _, tc := range testMQLQueries {
		t.Run(tc.name, func(t *testing.T) {
			q, err := buildMQLQuery(tc.resourceType, tc.metric, tc.resourceName, tc.aggregation)
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
