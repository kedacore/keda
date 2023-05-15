package scalers

import (
	"context"
	"strconv"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
)

func TestParseRedisStreamsMetadata(t *testing.T) {
	type testCase struct {
		name        string
		metadata    map[string]string
		resolvedEnv map[string]string
		authParams  map[string]string
	}

	authParams := map[string]string{"username": "foobarred", "password": "foobarred"}

	testCases := []testCase{
		{
			name:     "with address",
			metadata: map[string]string{"stream": "my-stream", "consumerGroup": "my-stream-consumer-group", "pendingEntriesCount": "5", "lag": "5", "addressFromEnv": "REDIS_SERVICE", "usernameFromEnv": "REDIS_USERNAME", "passwordFromEnv": "REDIS_PASSWORD", "databaseIndex": "0", "enableTLS": "true"},
			resolvedEnv: map[string]string{
				"REDIS_SERVICE":  "myredis:6379",
				"REDIS_USERNAME": "foobarred",
				"REDIS_PASSWORD": "foobarred",
			},
			authParams: nil,
		},

		{
			name:     "with host and port",
			metadata: map[string]string{"stream": "my-stream", "consumerGroup": "my-stream-consumer-group", "pendingEntriesCount": "15", "lag": "2", "hostFromEnv": "REDIS_HOST", "port": "REDIS_PORT", "usernameFromEnv": "REDIS_USERNAME", "passwordFromEnv": "REDIS_PASSWORD", "databaseIndex": "0", "enableTLS": "false"},
			resolvedEnv: map[string]string{
				"REDIS_HOST":     "myredis",
				"REDIS_PORT":     "6379",
				"REDIS_USERNAME": "foobarred",
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
			assert.Equal(t, strconv.FormatInt(m.targetLag, 10), tc.metadata[lagMetadata])
			if authParams != nil {
				// if authParam is used
				assert.Equal(t, m.connectionInfo.username, authParams[usernameMetadata])
				assert.Equal(t, m.connectionInfo.password, authParams[passwordMetadata])
			} else {
				// if metadata is used to pass credentials' env var names
				assert.Equal(t, m.connectionInfo.username, tc.resolvedEnv[tc.metadata[usernameMetadata]])
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
		{"missing address as well as host/port", map[string]string{"stream": "my-stream", "pendingEntriesCount": "5", "lag": "5", "consumerGroup": "my-stream-consumer-group"}, resolvedEnvMap},

		{"host present but missing port", map[string]string{"stream": "my-stream", "pendingEntriesCount": "5", "lag": "5", "consumerGroup": "my-stream-consumer-group", "host": "REDIS_HOST"}, resolvedEnvMap},

		{"port present but missing host", map[string]string{"stream": "my-stream", "pendingEntriesCount": "5", "lag": "5", "consumerGroup": "my-stream-consumer-group", "port": "REDIS_PORT"}, resolvedEnvMap},

		{"missing stream", map[string]string{"pendingEntriesCount": "5", "consumerGroup": "my-stream-consumer-group", "address": "REDIS_HOST"}, resolvedEnvMap},

		// invalid value for respective fields
		{"invalid lag", map[string]string{"stream": "my-stream", "consumerGroup": "my-stream-consumer-group", "pendingEntriesCount": "5", "lag": "junk", "host": "REDIS_HOST", "port": "REDIS_PORT", "databaseIndex": "0", "enableTLS": "false"}, resolvedEnvMap},

		{"invalid streamLength", map[string]string{"stream": "my-stream", "streamLength": "junk", "host": "REDIS_HOST", "port": "REDIS_PORT", "databaseIndex": "0", "enableTLS": "false"}, resolvedEnvMap},

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
		scalerIndex      int
		name             string
	}

	var redisStreamsTestData = []redisStreamsTestMetadata{
		{
			metadata:   map[string]string{"stream": "my-stream", "consumerGroup": "my-stream-consumer-group", "pendingEntriesCount": "5", "address": "REDIS_SERVICE", "password": "REDIS_PASSWORD", "databaseIndex": "0", "enableTLS": "true"},
			authParams: nil,
		},
	}

	var redisStreamMetricIdentifiers = []redisStreamsMetricIdentifier{
		{&redisStreamsTestData[0], 0, "s0-redis-streams-my-stream"},
		{&redisStreamsTestData[0], 1, "s1-redis-streams-my-stream"},
	}

	for _, testData := range redisStreamMetricIdentifiers {
		meta, err := parseRedisStreamsMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, ResolvedEnv: map[string]string{"REDIS_SERVICE": "my-address"}, AuthParams: testData.metadataTestData.authParams, ScalerIndex: testData.scalerIndex}, parseRedisAddress)
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		closeFn := func() error { return nil }
		getPendingEntriesCountFn := func(ctx context.Context) (int64, error) { return -1, nil }
		mockRedisStreamsScaler := redisStreamsScaler{"", meta, closeFn, getPendingEntriesCountFn, logr.Discard()}

		metricSpec := mockRedisStreamsScaler.GetMetricSpecForScaling(context.Background())
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
			wantErr:  ErrRedisNoAddresses,
		},
		{
			name: "unequal number of hosts/ports",
			metadata: map[string]string{
				"hosts": "a, b, c",
				"ports": "1, 2",
			},
			wantMeta: nil,
			wantErr:  ErrRedisUnequalHostsAndPorts,
		},
		{
			name: "no stream name",
			metadata: map[string]string{
				"hosts":               "a, b, c",
				"ports":               "1, 2, 3",
				"pendingEntriesCount": "5",
			},
			wantMeta: nil,
			wantErr:  ErrRedisMissingStreamName,
		},
		{
			name: "invalid lag",
			metadata: map[string]string{
				"stream":              "my-stream",
				"hosts":               "a, b, c",
				"ports":               "1, 2, 3",
				"consumerGroup":       "consumer1",
				"pendingEntriesCount": "5",
				"lag":                 "junk",
			},
			wantMeta: nil,
			wantErr:  strconv.ErrSyntax,
		},
		{
			name: "address is defined in auth params",
			metadata: map[string]string{
				"stream":              "my-stream",
				"pendingEntriesCount": "5",
				"lag":                 "6",
				"consumerGroup":       "consumer1",
			},
			authParams: map[string]string{
				"addresses": ":7001, :7002",
			},
			wantMeta: &redisStreamsMetadata{
				streamName:                "my-stream",
				targetPendingEntriesCount: 0, 
				targetLag:                 6,
				consumerGroupName:         "consumer1",
				connectionInfo: redisConnectionInfo{
					addresses: []string{":7001", ":7002"},
				},
				scaleFactor: xLagFactor,
			},
			wantErr: nil,
		},
		{
			name: "hosts and ports given in auth params",
			metadata: map[string]string{
				"stream":              "my-stream",
				"pendingEntriesCount": "5",
				"lag":                 "6",
				"consumerGroup":       "consumer1",
			},
			authParams: map[string]string{
				"hosts": "   a, b,    c ",
				"ports": "1, 2, 3",
			},
			wantMeta: &redisStreamsMetadata{
				streamName:                "my-stream",
				targetPendingEntriesCount: 0,
				targetLag:                 6,
				consumerGroupName:         "consumer1",
				connectionInfo: redisConnectionInfo{
					addresses: []string{"a:1", "b:2", "c:3"},
					hosts:     []string{"a", "b", "c"},
					ports:     []string{"1", "2", "3"},
				},
				scaleFactor: xLagFactor,
			},
			wantErr: nil,
		},
		{
			name: "username given in authParams",
			metadata: map[string]string{
				"hosts":               "a, b, c",
				"ports":               "1, 2, 3",
				"stream":              "my-stream",
				"pendingEntriesCount": "5",
				"lag":                 "7",
				"consumerGroup":       "consumer1",
			},
			authParams: map[string]string{
				"username": "username",
			},
			wantMeta: &redisStreamsMetadata{
				streamName:                "my-stream",
				targetPendingEntriesCount: 0,
				targetLag:                 7,
				consumerGroupName:         "consumer1",
				connectionInfo: redisConnectionInfo{
					addresses: []string{"a:1", "b:2", "c:3"},
					hosts:     []string{"a", "b", "c"},
					ports:     []string{"1", "2", "3"},
					username:  "username",
				},
				scaleFactor: xLagFactor,
			},
			wantErr: nil,
		},
		{
			name: "username given in metadata",
			metadata: map[string]string{
				"hosts":               "a, b, c",
				"ports":               "1, 2, 3",
				"stream":              "my-stream",
				"pendingEntriesCount": "5",
				"lag":                 "7",
				"consumerGroup":       "consumer1",
				"username":            "username",
			},
			authParams: map[string]string{},
			wantMeta: &redisStreamsMetadata{
				streamName:                "my-stream",
				targetPendingEntriesCount: 0,
				targetLag:                 7,
				consumerGroupName:         "consumer1",
				connectionInfo: redisConnectionInfo{
					addresses: []string{"a:1", "b:2", "c:3"},
					hosts:     []string{"a", "b", "c"},
					ports:     []string{"1", "2", "3"},
					username:  "username",
				},
				scaleFactor: xLagFactor,
			},
			wantErr: nil,
		},
		{
			name: "username given in metadata from env",
			metadata: map[string]string{
				"hosts":               "a, b, c",
				"ports":               "1, 2, 3",
				"stream":              "my-stream",
				"pendingEntriesCount": "5",
				"lag":                 "7",
				"consumerGroup":       "consumer1",
				"usernameFromEnv":     "REDIS_USERNAME",
			},
			authParams:  map[string]string{},
			resolvedEnv: testRedisResolvedEnv,
			wantMeta: &redisStreamsMetadata{
				streamName:                "my-stream",
				targetPendingEntriesCount: 0,
				targetLag:                 7,
				consumerGroupName:         "consumer1",
				connectionInfo: redisConnectionInfo{
					addresses: []string{"a:1", "b:2", "c:3"},
					hosts:     []string{"a", "b", "c"},
					ports:     []string{"1", "2", "3"},
					username:  "none",
				},
				scaleFactor: xLagFactor,
			},
			wantErr: nil,
		},
		{
			name: "password given in authParams",
			metadata: map[string]string{
				"hosts":               "a, b, c",
				"ports":               "1, 2, 3",
				"stream":              "my-stream",
				"pendingEntriesCount": "5",
				"lag":                 "7",
				"consumerGroup":       "consumer1",
			},
			authParams: map[string]string{
				"password": "password",
			},
			wantMeta: &redisStreamsMetadata{
				streamName:                "my-stream",
				targetPendingEntriesCount: 0,
				targetLag:                 7,
				consumerGroupName:         "consumer1",
				connectionInfo: redisConnectionInfo{
					addresses: []string{"a:1", "b:2", "c:3"},
					hosts:     []string{"a", "b", "c"},
					ports:     []string{"1", "2", "3"},
					password:  "password",
				},
				scaleFactor: xLagFactor,
			},
			wantErr: nil,
		},
		{
			name: "password given in metadata from env",
			metadata: map[string]string{
				"hosts":               "a, b, c",
				"ports":               "1, 2, 3",
				"stream":              "my-stream",
				"pendingEntriesCount": "5",
				"lag":                 "7",
				"consumerGroup":       "consumer1",
				"passwordFromEnv":     "REDIS_PASSWORD",
			},
			authParams:  map[string]string{},
			resolvedEnv: testRedisResolvedEnv,
			wantMeta: &redisStreamsMetadata{
				streamName:                "my-stream",
				targetPendingEntriesCount: 0,
				targetLag:                 7,
				consumerGroupName:         "consumer1",
				connectionInfo: redisConnectionInfo{
					addresses: []string{"a:1", "b:2", "c:3"},
					hosts:     []string{"a", "b", "c"},
					ports:     []string{"1", "2", "3"},
					password:  "none",
				},
				scaleFactor: xLagFactor,
			},
			wantErr: nil,
		},
		{
			name: "tls enabled without setting unsafeSsl",
			metadata: map[string]string{
				"hosts":               "a, b, c",
				"ports":               "1, 2, 3",
				"stream":              "my-stream",
				"pendingEntriesCount": "5",
				"lag":                 "7",
				"consumerGroup":       "consumer1",
				"enableTLS":           "true",
			},
			authParams: map[string]string{
				"password": "password",
			},
			wantMeta: &redisStreamsMetadata{
				streamName:                "my-stream",
				targetPendingEntriesCount: 0,
				targetLag:                 7,
				consumerGroupName:         "consumer1",
				connectionInfo: redisConnectionInfo{
					addresses: []string{"a:1", "b:2", "c:3"},
					hosts:     []string{"a", "b", "c"},
					ports:     []string{"1", "2", "3"},
					password:  "password",
					enableTLS: true,
					unsafeSsl: false,
				},
				scaleFactor: xLagFactor,
			},
			wantErr: nil,
		},
		{
			name: "tls enabled with unsafeSsl true",
			metadata: map[string]string{
				"hosts":               "a, b, c",
				"ports":               "1, 2, 3",
				"stream":              "my-stream",
				"pendingEntriesCount": "5",
				"lag":                 "7",
				"consumerGroup":       "consumer1",
				"enableTLS":           "true",
				"unsafeSsl":           "true",
			},
			authParams: map[string]string{
				"password": "password",
			},
			wantMeta: &redisStreamsMetadata{
				streamName:                "my-stream",
				targetPendingEntriesCount: 0,
				targetLag:                 7,
				consumerGroupName:         "consumer1",
				connectionInfo: redisConnectionInfo{
					addresses: []string{"a:1", "b:2", "c:3"},
					hosts:     []string{"a", "b", "c"},
					ports:     []string{"1", "2", "3"},
					password:  "password",
					enableTLS: true,
					unsafeSsl: true,
				},
				scaleFactor: xLagFactor,
			},
			wantErr: nil,
		},
		{
			name: "stream is provided",
			metadata: map[string]string{
				"stream": "my-stream",
			},
			authParams: map[string]string{
				"addresses": ":7001, :7002",
			},
			wantMeta: &redisStreamsMetadata{
				streamName:         "my-stream",
				targetStreamLength: 5,
				connectionInfo: redisConnectionInfo{
					addresses: []string{":7001", ":7002"},
				},
				scaleFactor: xLengthFactor,
			},
			wantErr: nil,
		},
		{
			name: "stream, consumerGroup is provided",
			metadata: map[string]string{
				"stream":        "my-stream",
				"consumerGroup": "consumer1",
			},
			authParams: map[string]string{
				"addresses": ":7001, :7002",
			},
			wantMeta: &redisStreamsMetadata{
				streamName:                "my-stream",
				targetPendingEntriesCount: 5,
				targetLag:                 0,
				consumerGroupName:         "consumer1",
				connectionInfo: redisConnectionInfo{
					addresses: []string{":7001", ":7002"},
				},
				scaleFactor: xPendingFactor,
			},
			wantErr: nil,
		},
	}

	for _, testCase := range cases {
		c := testCase
		t.Run(c.name, func(t *testing.T) {
			config := &ScalerConfig{
				TriggerMetadata: c.metadata,
				ResolvedEnv:     c.resolvedEnv,
				AuthParams:      c.authParams,
			}
			meta, err := parseRedisStreamsMetadata(config, parseRedisClusterAddress)
			if c.wantErr != nil {
				assert.ErrorIs(t, err, c.wantErr)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, c.wantMeta, meta)
		})
	}
}

func TestParseRedisSentinelStreamsMetadata(t *testing.T) {
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
			wantErr:  ErrRedisNoAddresses,
		},
		{
			name: "unequal number of hosts/ports",
			metadata: map[string]string{
				"hosts": "a, b, c",
				"ports": "1, 2",
			},
			wantMeta: nil,
			wantErr:  ErrRedisUnequalHostsAndPorts,
		},
		{
			name: "no stream name",
			metadata: map[string]string{
				"hosts":               "a, b, c",
				"ports":               "1, 2, 3",
				"pendingEntriesCount": "5",
			},
			wantMeta: nil,
			wantErr:  ErrRedisMissingStreamName,
		},
		{
			name: "invalid lag count",
			metadata: map[string]string{
				"stream":              "my-stream",
				"hosts":               "a, b, c",
				"ports":               "1, 2, 3",
				"consumerGroup":       "consumer1",
				"pendingEntriesCount": "5",
				"lag":                 "invalid",
			},
			wantMeta: nil,
			wantErr:  strconv.ErrSyntax,
		},
		{
			name: "address is defined in auth params",
			metadata: map[string]string{
				"stream":              "my-stream",
				"pendingEntriesCount": "5",
				"lag":                 "7",
				"consumerGroup":       "consumer1",
			},
			authParams: map[string]string{
				"addresses": ":7001, :7002",
			},
			wantMeta: &redisStreamsMetadata{
				streamName:                "my-stream",
				targetPendingEntriesCount: 0,
				targetLag:                 7,
				consumerGroupName:         "consumer1",
				connectionInfo: redisConnectionInfo{
					addresses: []string{":7001", ":7002"},
				},
				scaleFactor: xLagFactor,
			},
			wantErr: nil,
		},
		{
			name: "hosts and ports given in auth params",
			metadata: map[string]string{
				"stream":              "my-stream",
				"pendingEntriesCount": "5",
				"lag":                 "7",
				"consumerGroup":       "consumer1",
			},
			authParams: map[string]string{
				"hosts": "   a, b,    c ",
				"ports": "1, 2, 3",
			},
			wantMeta: &redisStreamsMetadata{
				streamName:                "my-stream",
				targetPendingEntriesCount: 0,
				targetLag:                 7,
				consumerGroupName:         "consumer1",
				connectionInfo: redisConnectionInfo{
					addresses: []string{"a:1", "b:2", "c:3"},
					hosts:     []string{"a", "b", "c"},
					ports:     []string{"1", "2", "3"},
				},
				scaleFactor: xLagFactor,
			},
			wantErr: nil,
		},
		{
			name: "username given in authParams",
			metadata: map[string]string{
				"hosts":               "a, b, c",
				"ports":               "1, 2, 3",
				"stream":              "my-stream",
				"pendingEntriesCount": "5",
				"lag":                 "7",
				"consumerGroup":       "consumer1",
			},
			authParams: map[string]string{
				"username": "username",
			},
			wantMeta: &redisStreamsMetadata{
				streamName:                "my-stream",
				targetPendingEntriesCount: 0,
				targetLag:                 7,
				consumerGroupName:         "consumer1",
				connectionInfo: redisConnectionInfo{
					addresses: []string{"a:1", "b:2", "c:3"},
					hosts:     []string{"a", "b", "c"},
					ports:     []string{"1", "2", "3"},
					username:  "username",
				},
				scaleFactor: xLagFactor,
			},
			wantErr: nil,
		},
		{
			name: "username given in metadata",
			metadata: map[string]string{
				"hosts":               "a, b, c",
				"ports":               "1, 2, 3",
				"stream":              "my-stream",
				"pendingEntriesCount": "5",
				"lag":                 "7",
				"consumerGroup":       "consumer1",
				"username":            "username",
			},
			authParams: map[string]string{},
			wantMeta: &redisStreamsMetadata{
				streamName:                "my-stream",
				targetPendingEntriesCount: 0,
				targetLag:                 7,
				consumerGroupName:         "consumer1",
				connectionInfo: redisConnectionInfo{
					addresses: []string{"a:1", "b:2", "c:3"},
					hosts:     []string{"a", "b", "c"},
					ports:     []string{"1", "2", "3"},
					username:  "username",
				},
				scaleFactor: xLagFactor,
			},
			wantErr: nil,
		},
		{
			name: "username given in metadata from env",
			metadata: map[string]string{
				"hosts":               "a, b, c",
				"ports":               "1, 2, 3",
				"stream":              "my-stream",
				"pendingEntriesCount": "5",
				"lag":                 "7",
				"consumerGroup":       "consumer1",
				"usernameFromEnv":     "REDIS_USERNAME",
			},
			authParams:  map[string]string{},
			resolvedEnv: testRedisResolvedEnv,
			wantMeta: &redisStreamsMetadata{
				streamName:                "my-stream",
				targetPendingEntriesCount: 0,
				targetLag:                 7,
				consumerGroupName:         "consumer1",
				connectionInfo: redisConnectionInfo{
					addresses: []string{"a:1", "b:2", "c:3"},
					hosts:     []string{"a", "b", "c"},
					ports:     []string{"1", "2", "3"},
					username:  "none",
				},
				scaleFactor: xLagFactor,
			},
			wantErr: nil,
		},
		{
			name: "password given in authParams",
			metadata: map[string]string{
				"hosts":               "a, b, c",
				"ports":               "1, 2, 3",
				"stream":              "my-stream",
				"pendingEntriesCount": "5",
				"lag":                 "7",
				"consumerGroup":       "consumer1",
			},
			authParams: map[string]string{
				"password": "password",
			},
			wantMeta: &redisStreamsMetadata{
				streamName:                "my-stream",
				targetPendingEntriesCount: 0,
				targetLag:                 7,
				consumerGroupName:         "consumer1",
				connectionInfo: redisConnectionInfo{
					addresses: []string{"a:1", "b:2", "c:3"},
					hosts:     []string{"a", "b", "c"},
					ports:     []string{"1", "2", "3"},
					password:  "password",
				},
				scaleFactor: xLagFactor,
			},
			wantErr: nil,
		},
		{
			name: "password given in metadata from env",
			metadata: map[string]string{
				"hosts":               "a, b, c",
				"ports":               "1, 2, 3",
				"stream":              "my-stream",
				"pendingEntriesCount": "5",
				"lag":                 "7",
				"consumerGroup":       "consumer1",
				"passwordFromEnv":     "REDIS_PASSWORD",
			},
			authParams:  map[string]string{},
			resolvedEnv: testRedisResolvedEnv,
			wantMeta: &redisStreamsMetadata{
				streamName:                "my-stream",
				targetPendingEntriesCount: 0,
				targetLag:                 7,
				consumerGroupName:         "consumer1",
				connectionInfo: redisConnectionInfo{
					addresses: []string{"a:1", "b:2", "c:3"},
					hosts:     []string{"a", "b", "c"},
					ports:     []string{"1", "2", "3"},
					password:  "none",
				},
				scaleFactor: xLagFactor,
			},
			wantErr: nil,
		},
		{
			name: "sentinelUsername given in authParams",
			metadata: map[string]string{
				"hosts":               "a, b, c",
				"ports":               "1, 2, 3",
				"stream":              "my-stream",
				"pendingEntriesCount": "5",
				"lag":                 "7",
				"consumerGroup":       "consumer1",
			},
			authParams: map[string]string{
				"sentinelUsername": "sentinelUsername",
			},
			wantMeta: &redisStreamsMetadata{
				streamName:                "my-stream",
				targetPendingEntriesCount: 0,
				targetLag:                 7,
				consumerGroupName:         "consumer1",
				connectionInfo: redisConnectionInfo{
					addresses:        []string{"a:1", "b:2", "c:3"},
					hosts:            []string{"a", "b", "c"},
					ports:            []string{"1", "2", "3"},
					sentinelUsername: "sentinelUsername",
				},
				scaleFactor: xLagFactor,
			},
			wantErr: nil,
		},
		{
			name: "sentinelUsername given in metadata",
			metadata: map[string]string{
				"hosts":               "a, b, c",
				"ports":               "1, 2, 3",
				"stream":              "my-stream",
				"pendingEntriesCount": "5",
				"lag":                 "7",
				"consumerGroup":       "consumer1",
				"sentinelUsername":    "sentinelUsername",
			},
			authParams: map[string]string{},
			wantMeta: &redisStreamsMetadata{
				streamName:                "my-stream",
				targetPendingEntriesCount: 0,
				targetLag:                 7,
				consumerGroupName:         "consumer1",
				connectionInfo: redisConnectionInfo{
					addresses:        []string{"a:1", "b:2", "c:3"},
					hosts:            []string{"a", "b", "c"},
					ports:            []string{"1", "2", "3"},
					sentinelUsername: "sentinelUsername",
				},
				scaleFactor: xLagFactor,
			},
			wantErr: nil,
		},
		{
			name: "sentinelUsername given in metadata from env",
			metadata: map[string]string{
				"hosts":                   "a, b, c",
				"ports":                   "1, 2, 3",
				"stream":                  "my-stream",
				"pendingEntriesCount":     "5",
				"lag":                     "7",
				"consumerGroup":           "consumer1",
				"sentinelUsernameFromEnv": "REDIS_SENTINEL_USERNAME",
			},
			authParams:  map[string]string{},
			resolvedEnv: testRedisResolvedEnv,
			wantMeta: &redisStreamsMetadata{
				streamName:                "my-stream",
				targetPendingEntriesCount: 0,
				targetLag:                 7,
				consumerGroupName:         "consumer1",
				connectionInfo: redisConnectionInfo{
					addresses:        []string{"a:1", "b:2", "c:3"},
					hosts:            []string{"a", "b", "c"},
					ports:            []string{"1", "2", "3"},
					sentinelUsername: "none",
				},
				scaleFactor: xLagFactor,
			},
			wantErr: nil,
		},
		{
			name: "sentinelPassword given in authParams",
			metadata: map[string]string{
				"hosts":               "a, b, c",
				"ports":               "1, 2, 3",
				"stream":              "my-stream",
				"pendingEntriesCount": "5",
				"lag":                 "7",
				"consumerGroup":       "consumer1",
			},
			authParams: map[string]string{
				"sentinelPassword": "sentinelPassword",
			},
			wantMeta: &redisStreamsMetadata{
				streamName:                "my-stream",
				targetPendingEntriesCount: 0,
				targetLag:                 7,
				consumerGroupName:         "consumer1",
				connectionInfo: redisConnectionInfo{
					addresses:        []string{"a:1", "b:2", "c:3"},
					hosts:            []string{"a", "b", "c"},
					ports:            []string{"1", "2", "3"},
					sentinelPassword: "sentinelPassword",
				},
				scaleFactor: xLagFactor,
			},
			wantErr: nil,
		},
		{
			name: "sentinelPassword given in metadata from env",
			metadata: map[string]string{
				"hosts":                   "a, b, c",
				"ports":                   "1, 2, 3",
				"stream":                  "my-stream",
				"pendingEntriesCount":     "5",
				"lag":                     "7",
				"consumerGroup":           "consumer1",
				"sentinelPasswordFromEnv": "REDIS_SENTINEL_PASSWORD",
			},
			authParams:  map[string]string{},
			resolvedEnv: testRedisResolvedEnv,
			wantMeta: &redisStreamsMetadata{
				streamName:                "my-stream",
				targetPendingEntriesCount: 0,
				targetLag:                 7,
				consumerGroupName:         "consumer1",
				connectionInfo: redisConnectionInfo{
					addresses:        []string{"a:1", "b:2", "c:3"},
					hosts:            []string{"a", "b", "c"},
					ports:            []string{"1", "2", "3"},
					sentinelPassword: "none",
				},
				scaleFactor: xLagFactor,
			},
			wantErr: nil,
		},
		{
			name: "sentinelMaster given in authParams",
			metadata: map[string]string{
				"hosts":               "a, b, c",
				"ports":               "1, 2, 3",
				"stream":              "my-stream",
				"pendingEntriesCount": "5",
				"lag":                 "7",
				"consumerGroup":       "consumer1",
			},
			authParams: map[string]string{
				"sentinelMaster": "sentinelMaster",
			},
			wantMeta: &redisStreamsMetadata{
				streamName:                "my-stream",
				targetPendingEntriesCount: 0,
				targetLag:                 7,
				consumerGroupName:         "consumer1",
				connectionInfo: redisConnectionInfo{
					addresses:      []string{"a:1", "b:2", "c:3"},
					hosts:          []string{"a", "b", "c"},
					ports:          []string{"1", "2", "3"},
					sentinelMaster: "sentinelMaster",
				},
				scaleFactor: xLagFactor,
			},
			wantErr: nil,
		},
		{
			name: "sentinelMaster given in metadata",
			metadata: map[string]string{
				"hosts":               "a, b, c",
				"ports":               "1, 2, 3",
				"stream":              "my-stream",
				"pendingEntriesCount": "5",
				"lag":                 "7",
				"consumerGroup":       "consumer1",
				"sentinelMaster":      "sentinelMaster",
			},
			authParams: map[string]string{},
			wantMeta: &redisStreamsMetadata{
				streamName:                "my-stream",
				targetPendingEntriesCount: 0,
				targetLag:                 7,
				consumerGroupName:         "consumer1",
				connectionInfo: redisConnectionInfo{
					addresses:      []string{"a:1", "b:2", "c:3"},
					hosts:          []string{"a", "b", "c"},
					ports:          []string{"1", "2", "3"},
					sentinelMaster: "sentinelMaster",
				},
				scaleFactor: xLagFactor,
			},
			wantErr: nil,
		},
		{
			name: "sentinelMaster given in metadata from env",
			metadata: map[string]string{
				"hosts":                 "a, b, c",
				"ports":                 "1, 2, 3",
				"stream":                "my-stream",
				"pendingEntriesCount":   "5",
				"lag":                   "7",
				"consumerGroup":         "consumer1",
				"sentinelMasterFromEnv": "REDIS_SENTINEL_MASTER",
			},
			authParams:  map[string]string{},
			resolvedEnv: testRedisResolvedEnv,
			wantMeta: &redisStreamsMetadata{
				streamName:                "my-stream",
				targetPendingEntriesCount: 0,
				targetLag:                 7,
				consumerGroupName:         "consumer1",
				connectionInfo: redisConnectionInfo{
					addresses:      []string{"a:1", "b:2", "c:3"},
					hosts:          []string{"a", "b", "c"},
					ports:          []string{"1", "2", "3"},
					sentinelMaster: "none",
				},
				scaleFactor: xLagFactor,
			},
			wantErr: nil,
		},
		{
			name: "tls enabled without setting unsafeSsl",
			metadata: map[string]string{
				"hosts":               "a, b, c",
				"ports":               "1, 2, 3",
				"stream":              "my-stream",
				"pendingEntriesCount": "5",
				"lag":                 "7",
				"consumerGroup":       "consumer1",
				"enableTLS":           "true",
			},
			authParams: map[string]string{
				"password": "password",
			},
			wantMeta: &redisStreamsMetadata{
				streamName:                "my-stream",
				targetPendingEntriesCount: 0,
				targetLag:                 7,
				consumerGroupName:         "consumer1",
				connectionInfo: redisConnectionInfo{
					addresses: []string{"a:1", "b:2", "c:3"},
					hosts:     []string{"a", "b", "c"},
					ports:     []string{"1", "2", "3"},
					password:  "password",
					enableTLS: true,
					unsafeSsl: false,
				},
				scaleFactor: xLagFactor,
			},
			wantErr: nil,
		},
		{
			name: "tls enabled with unsafeSsl true",
			metadata: map[string]string{
				"hosts":               "a, b, c",
				"ports":               "1, 2, 3",
				"stream":              "my-stream",
				"pendingEntriesCount": "5",
				"lag":                 "7",
				"consumerGroup":       "consumer1",
				"enableTLS":           "true",
				"unsafeSsl":           "true",
			},
			authParams: map[string]string{
				"password": "password",
			},
			wantMeta: &redisStreamsMetadata{
				streamName:                "my-stream",
				targetPendingEntriesCount: 0,
				targetLag:                 7,
				consumerGroupName:         "consumer1",
				connectionInfo: redisConnectionInfo{
					addresses: []string{"a:1", "b:2", "c:3"},
					hosts:     []string{"a", "b", "c"},
					ports:     []string{"1", "2", "3"},
					password:  "password",
					enableTLS: true,
					unsafeSsl: true,
				},
				scaleFactor: xLagFactor,
			},
			wantErr: nil,
		},
		{
			name: "streamLength passed",
			metadata: map[string]string{
				"hosts":        "a",
				"ports":        "1",
				"stream":       "my-stream",
				"streamLength": "15",
			},
			authParams: map[string]string{},
			wantMeta: &redisStreamsMetadata{
				streamName:         "my-stream",
				targetStreamLength: 15,
				connectionInfo: redisConnectionInfo{
					addresses: []string{"a:1"},
					hosts:     []string{"a"},
					ports:     []string{"1"},
					password:  "",
					enableTLS: false,
					unsafeSsl: false,
				},
				scaleFactor: xLengthFactor,
			},
			wantErr: nil,
		},
		{
			name: "streamLength, pendingEntriesCount and consumerGroup passed",
			metadata: map[string]string{
				"hosts":               "a",
				"ports":               "1",
				"stream":              "my-stream",
				"streamLength":        "15",
				"pendingEntriesCount": "5",
				"lag":                 "70",
				"consumerGroup":       "consumer1",
			},
			authParams: map[string]string{},
			wantMeta: &redisStreamsMetadata{
				streamName:                "my-stream",
				targetPendingEntriesCount: 0,
				targetLag:                 70,
				consumerGroupName:         "consumer1",
				connectionInfo: redisConnectionInfo{
					addresses: []string{"a:1"},
					hosts:     []string{"a"},
					ports:     []string{"1"},
					password:  "",
					enableTLS: false,
					unsafeSsl: false,
				},
				scaleFactor: xLagFactor,
			},
			wantErr: nil,
		},
		{
			name: "streamLength and pendingEntriesCount passed",
			metadata: map[string]string{
				"hosts":               "a",
				"ports":               "1",
				"stream":              "my-stream",
				"streamLength":        "15",
				"pendingEntriesCount": "30",
			},
			authParams: map[string]string{},
			wantMeta: &redisStreamsMetadata{
				streamName:         "my-stream",
				targetStreamLength: 15,
				connectionInfo: redisConnectionInfo{
					addresses: []string{"a:1"},
					hosts:     []string{"a"},
					ports:     []string{"1"},
					password:  "",
					enableTLS: false,
					unsafeSsl: false,
				},
				scaleFactor: xLengthFactor,
			},
			wantErr: nil,
		},
	}

	for _, testCase := range cases {
		c := testCase
		t.Run(c.name, func(t *testing.T) {
			config := &ScalerConfig{
				TriggerMetadata: c.metadata,
				ResolvedEnv:     c.resolvedEnv,
				AuthParams:      c.authParams,
			}
			meta, err := parseRedisStreamsMetadata(config, parseRedisSentinelAddress)
			if c.wantErr != nil {
				assert.ErrorIs(t, err, c.wantErr)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, c.wantMeta, meta)
		})
	}
}
