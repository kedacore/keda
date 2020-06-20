package scalers

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseRedisStreamsMetadata(t *testing.T) {

	type testCase struct {
		name        string
		metadata    map[string]string
		resolvedEnv map[string]string
		authParams  map[string]string
	}

	authParams := map[string]string{"password": "foobarred"}

	testCases := []testCase{
		{"with address", map[string]string{"stream": "my-stream", "consumerGroup": "my-stream-consumer-group", "pendingEntriesCount": "5", "address": "REDIS_SERVICE", "password": "REDIS_PASSWORD", "databaseIndex": "0", "enableTLS": "true"}, map[string]string{
			"REDIS_SERVICE":  "myredis:6379",
			"REDIS_PASSWORD": "foobarred",
		}, nil},

		{"with host and port", map[string]string{"stream": "my-stream", "consumerGroup": "my-stream-consumer-group", "pendingEntriesCount": "15", "host": "REDIS_HOST", "port": "REDIS_PORT", "databaseIndex": "0", "enableTLS": "false"}, map[string]string{
			"REDIS_HOST":     "myredis",
			"REDIS_PORT":     "6379",
			"REDIS_PASSWORD": "foobarred",
		}, authParams},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(te *testing.T) {
			m, err := parseRedisStreamsMetadata(tc.metadata, tc.resolvedEnv, tc.authParams)
			assert.Nil(t, err)
			assert.Equal(t, m.streamName, tc.metadata[streamNameMetadata])
			assert.Equal(t, m.consumerGroupName, tc.metadata[consumerGroupNameMetadata])
			assert.Equal(t, strconv.Itoa(m.targetPendingEntriesCount), tc.metadata[pendingEntriesCountMetadata])
			if authParams != nil {
				//if authParam is used
				assert.Equal(t, m.password, authParams[passwordMetadata])
			} else {
				//if metadata is used to pass password env var name
				assert.Equal(t, m.password, tc.resolvedEnv[tc.metadata[passwordMetadata]])
			}
			assert.Equal(t, strconv.Itoa(m.databaseIndex), tc.metadata[databaseIndexMetadata])
			b, err := strconv.ParseBool(tc.metadata[enableTLSMetadata])
			assert.Nil(t, err)
			assert.Equal(t, m.enableTLS, b)
		})
	}
}

func TestParseRedisStreamsMetadataForInvalidCases(t *testing.T) {
	resolvedEnvMap := map[string]string{
		"REDIS_SERVER":   "myredis:6379",
		"REDIS_HOST":     "myredis",
		"REDIS_PORT":     "6379",
		"REDIS_PASSWORD": "",
	}
	type testCase struct {
		name        string
		metadata    map[string]string
		resolvedEnv map[string]string
	}

	testCases := []testCase{
		//missing mandatory metadata
		{"missing address as well as host/port", map[string]string{"stream": "my-stream", "pendingEntriesCount": "5", "consumerGroup": "my-stream-consumer-group"}, resolvedEnvMap},

		{"host present but missing port", map[string]string{"stream": "my-stream", "pendingEntriesCount": "5", "consumerGroup": "my-stream-consumer-group", "host": "REDIS_HOST"}, resolvedEnvMap},

		{"port present but missing host", map[string]string{"stream": "my-stream", "pendingEntriesCount": "5", "consumerGroup": "my-stream-consumer-group", "port": "REDIS_PORT"}, resolvedEnvMap},

		{"missing stream", map[string]string{"pendingEntriesCount": "5", "consumerGroup": "my-stream-consumer-group", "address": "REDIS_HOST"}, resolvedEnvMap},

		{"missing consumerGroup", map[string]string{"stream": "my-stream", "pendingEntriesCount": "5", "address": "REDIS_HOST"}, resolvedEnvMap},

		{"missing pendingEntriesCount", map[string]string{"stream": "my-stream", "consumerGroup": "my-stream-consumer-group", "address": "REDIS_HOST"}, resolvedEnvMap},

		//invalid value for respective fields
		{"invalid pendingEntriesCount", map[string]string{"stream": "my-stream", "consumerGroup": "my-stream-consumer-group", "pendingEntriesCount": "junk", "host": "REDIS_HOST", "port": "REDIS_PORT", "databaseIndex": "0", "enableTLS": "false"}, resolvedEnvMap},

		{"invalid databaseIndex", map[string]string{"stream": "my-stream", "consumerGroup": "my-stream-consumer-group", "pendingEntriesCount": "15", "address": "REDIS_SERVER", "databaseIndex": "junk", "enableTLS": "false"}, resolvedEnvMap},

		{"invalid enableTLS", map[string]string{"stream": "my-stream", "consumerGroup": "my-stream-consumer-group", "pendingEntriesCount": "15", "address": "REDIS_SERVER", "databaseIndex": "1", "enableTLS": "no"}, resolvedEnvMap},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(te *testing.T) {
			_, err := parseRedisStreamsMetadata(tc.metadata, tc.resolvedEnv, map[string]string{})
			assert.NotNil(t, err)
		})
	}
}

type redisStreamsTestMetadata struct {
	metadata   map[string]string
	isError    bool
	authParams map[string]string
}
