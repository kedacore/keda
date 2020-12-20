package scalers

import (
	"errors"
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
		{
			name:     "with address",
			metadata: map[string]string{"stream": "my-stream", "consumerGroup": "my-stream-consumer-group", "pendingEntriesCount": "5", "addressFromEnv": "REDIS_SERVICE", "passwordFromEnv": "REDIS_PASSWORD", "databaseIndex": "0", "enableTLS": "true"},
			resolvedEnv: map[string]string{
				"REDIS_SERVICE":  "myredis:6379",
				"REDIS_PASSWORD": "foobarred",
			},
			authParams: nil,
		},

		{
			name:     "with host and port",
			metadata: map[string]string{"stream": "my-stream", "consumerGroup": "my-stream-consumer-group", "pendingEntriesCount": "15", "hostFromEnv": "REDIS_HOST", "port": "REDIS_PORT", "passwordFromEnv": "REDIS_PASSWORD", "databaseIndex": "0", "enableTLS": "false"},
			resolvedEnv: map[string]string{
				"REDIS_HOST":     "myredis",
				"REDIS_PORT":     "6379",
				"REDIS_PASSWORD": "foobarred",
			},
			authParams: authParams,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(te *testing.T) {
			m, err := parseRedisStreamsMetadata(&ScalerConfig{TriggerMetadata: tc.metadata, ResolvedEnv: tc.resolvedEnv, AuthParams: tc.authParams}, parseRedisAddress)
			assert.Nil(t, err)
			assert.Equal(t, m.streamName, tc.metadata[streamNameMetadata])
			assert.Equal(t, m.consumerGroupName, tc.metadata[consumerGroupNameMetadata])
			assert.Equal(t, strconv.Itoa(m.targetPendingEntriesCount), tc.metadata[pendingEntriesCountMetadata])
			if authParams != nil {
				// if authParam is used
				assert.Equal(t, m.connectionInfo.password, authParams[passwordMetadata])
			} else {
				// if metadata is used to pass password env var name
				assert.Equal(t, m.connectionInfo.password, tc.resolvedEnv[tc.metadata[passwordMetadata]])
			}
			assert.Equal(t, strconv.Itoa(m.databaseIndex), tc.metadata[databaseIndexMetadata])
			b, err := strconv.ParseBool(tc.metadata[enableTLSMetadata])
			assert.Nil(t, err)
			assert.Equal(t, m.connectionInfo.enableTLS, b)
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
		// missing mandatory metadata
		{"missing address as well as host/port", map[string]string{"stream": "my-stream", "pendingEntriesCount": "5", "consumerGroup": "my-stream-consumer-group"}, resolvedEnvMap},

		{"host present but missing port", map[string]string{"stream": "my-stream", "pendingEntriesCount": "5", "consumerGroup": "my-stream-consumer-group", "host": "REDIS_HOST"}, resolvedEnvMap},

		{"port present but missing host", map[string]string{"stream": "my-stream", "pendingEntriesCount": "5", "consumerGroup": "my-stream-consumer-group", "port": "REDIS_PORT"}, resolvedEnvMap},

		{"missing stream", map[string]string{"pendingEntriesCount": "5", "consumerGroup": "my-stream-consumer-group", "address": "REDIS_HOST"}, resolvedEnvMap},

		{"missing consumerGroup", map[string]string{"stream": "my-stream", "pendingEntriesCount": "5", "address": "REDIS_HOST"}, resolvedEnvMap},

		{"missing pendingEntriesCount", map[string]string{"stream": "my-stream", "consumerGroup": "my-stream-consumer-group", "address": "REDIS_HOST"}, resolvedEnvMap},

		// invalid value for respective fields
		{"invalid pendingEntriesCount", map[string]string{"stream": "my-stream", "consumerGroup": "my-stream-consumer-group", "pendingEntriesCount": "junk", "host": "REDIS_HOST", "port": "REDIS_PORT", "databaseIndex": "0", "enableTLS": "false"}, resolvedEnvMap},

		{"invalid databaseIndex", map[string]string{"stream": "my-stream", "consumerGroup": "my-stream-consumer-group", "pendingEntriesCount": "15", "address": "REDIS_SERVER", "databaseIndex": "junk", "enableTLS": "false"}, resolvedEnvMap},

		{"invalid enableTLS", map[string]string{"stream": "my-stream", "consumerGroup": "my-stream-consumer-group", "pendingEntriesCount": "15", "address": "REDIS_SERVER", "databaseIndex": "1", "enableTLS": "no"}, resolvedEnvMap},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(te *testing.T) {
			_, err := parseRedisStreamsMetadata(&ScalerConfig{TriggerMetadata: tc.metadata, ResolvedEnv: tc.resolvedEnv, AuthParams: map[string]string{}}, parseRedisAddress)
			assert.NotNil(t, err)
		})
	}
}

type redisStreamsTestMetadata struct {
	metadata   map[string]string
	authParams map[string]string
}

func TestRedisStreamsGetMetricSpecForScaling(t *testing.T) {
	type redisStreamsMetricIdentifier struct {
		metadataTestData *redisStreamsTestMetadata
		name             string
	}

	var redisStreamsTestData = []redisStreamsTestMetadata{
		{
			metadata:   map[string]string{"stream": "my-stream", "consumerGroup": "my-stream-consumer-group", "pendingEntriesCount": "5", "address": "REDIS_SERVICE", "password": "REDIS_PASSWORD", "databaseIndex": "0", "enableTLS": "true"},
			authParams: nil,
		},
	}

	var redisStreamMetricIdentifiers = []redisStreamsMetricIdentifier{
		{&redisStreamsTestData[0], "redis-streams-my-stream-my-stream-consumer-group"},
	}

	for _, testData := range redisStreamMetricIdentifiers {
		meta, err := parseRedisStreamsMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, ResolvedEnv: map[string]string{"REDIS_SERVICE": "my-address"}, AuthParams: testData.metadataTestData.authParams}, parseRedisAddress)
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		closeFn := func() error { return nil }
		getPendingEntriesCountFn := func() (int64, error) { return -1, nil }
		mockRedisStreamsScaler := redisStreamsScaler{meta, closeFn, getPendingEntriesCountFn}

		metricSpec := mockRedisStreamsScaler.GetMetricSpecForScaling()
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

func TestParseRedisClusterStreamsMetadata(t *testing.T) {
	cases := []struct {
		name        string
		metadata    map[string]string
		resolvedEnv map[string]string
		authParams  map[string]string
		wantMeta    *redisStreamsMetadata
		wantErr     error
	}{
		{
			name:     "empty metadata",
			wantMeta: nil,
			wantErr:  errors.New("no addresses or hosts given. address should be a comma separated list of host:port or set the host/port values"),
		},
		{
			name: "unequal number of hosts/ports",
			metadata: map[string]string{
				"hosts": "a, b, c",
				"ports": "1, 2",
			},
			wantMeta: nil,
			wantErr:  errors.New("not enough hosts or ports given. number of hosts should be equal to the number of ports"),
		},
		{
			name: "no stream name",
			metadata: map[string]string{
				"hosts":               "a, b, c",
				"ports":               "1, 2, 3",
				"pendingEntriesCount": "5",
			},
			wantMeta: nil,
			wantErr:  errors.New("missing redis stream name"),
		},
		{
			name: "missing pending entries count",
			metadata: map[string]string{
				"hosts":  "a, b, c",
				"ports":  "1, 2, 3",
				"stream": "my-stream",
			},
			wantMeta: nil,
			wantErr:  errors.New("missing pending entries count"),
		},
		{
			name: "invalid pending entries count",
			metadata: map[string]string{
				"hosts":               "a, b, c",
				"ports":               "1, 2, 3",
				"pendingEntriesCount": "invalid",
			},
			wantMeta: nil,
			wantErr:  errors.New("error parsing pending entries count"),
		},
		{
			name: "address is defined in auth params",
			metadata: map[string]string{
				"stream":              "my-stream",
				"pendingEntriesCount": "10",
				"consumerGroup":       "consumer1",
			},
			authParams: map[string]string{
				"addresses": ":7001, :7002",
			},
			wantMeta: &redisStreamsMetadata{
				streamName:                "my-stream",
				targetPendingEntriesCount: 10,
				consumerGroupName:         "consumer1",
				connectionInfo: redisConnectionInfo{
					addresses: []string{":7001", ":7002"},
				},
			},
			wantErr: nil,
		},
		{
			name: "hosts and ports given in auth params",
			metadata: map[string]string{
				"stream":              "my-stream",
				"pendingEntriesCount": "10",
				"consumerGroup":       "consumer1",
			},
			authParams: map[string]string{
				"hosts": "   a, b,    c ",
				"ports": "1, 2, 3",
			},
			wantMeta: &redisStreamsMetadata{
				streamName:                "my-stream",
				targetPendingEntriesCount: 10,
				consumerGroupName:         "consumer1",
				connectionInfo: redisConnectionInfo{
					addresses: []string{"a:1", "b:2", "c:3"},
					hosts:     []string{"a", "b", "c"},
					ports:     []string{"1", "2", "3"},
				},
			},
			wantErr: nil,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			config := &ScalerConfig{
				TriggerMetadata: c.metadata,
				ResolvedEnv:     c.resolvedEnv,
				AuthParams:      c.authParams,
			}
			meta, err := parseRedisStreamsMetadata(config, parseRedisClusterAddress)
			if c.wantErr != nil {
				assert.Contains(t, err.Error(), c.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, c.wantMeta, meta)
		})
	}
}
