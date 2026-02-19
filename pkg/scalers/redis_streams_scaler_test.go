package scalers

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

func TestParseRedisStreamsMetadata(t *testing.T) {
	type testCase struct {
		name        string
		metadata    map[string]string
		resolvedEnv map[string]string
		authParams  map[string]string
	}

	authParams := map[string]string{"username": "foobarred", "password": "foobarred"}

	testCasesPending := []testCase{
		{
			name:     "with address",
			metadata: map[string]string{"stream": "my-stream", "consumerGroup": "my-stream-consumer-group", "pendingEntriesCount": "5", "addressFromEnv": "REDIS_SERVICE", "usernameFromEnv": "REDIS_USERNAME", "passwordFromEnv": "REDIS_PASSWORD", "databaseIndex": "0", "enableTLS": "true"},
			resolvedEnv: map[string]string{
				"REDIS_SERVICE":  "myredis:6379",
				"REDIS_USERNAME": "foobarred",
				"REDIS_PASSWORD": "foobarred",
			},
			authParams: nil,
		},

		{
			name:     "with host and port",
			metadata: map[string]string{"stream": "my-stream", "consumerGroup": "my-stream-consumer-group", "pendingEntriesCount": "15", "hostFromEnv": "REDIS_HOST", "port": "REDIS_PORT", "usernameFromEnv": "REDIS_USERNAME", "passwordFromEnv": "REDIS_PASSWORD", "databaseIndex": "0", "enableTLS": "false"},
			resolvedEnv: map[string]string{
				"REDIS_HOST":     "myredis",
				"REDIS_PORT":     "6379",
				"REDIS_USERNAME": "foobarred",
				"REDIS_PASSWORD": "foobarred",
			},
			authParams: authParams,
		},
	}

	for _, tc := range testCasesPending {
		t.Run(tc.name, func(te *testing.T) {
			m, err := parseRedisStreamsMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: tc.metadata, ResolvedEnv: tc.resolvedEnv, AuthParams: tc.authParams})
			assert.Nil(t, err)
			assert.Equal(t, tc.metadata[streamNameMetadata], m.StreamName)
			assert.Equal(t, tc.metadata[consumerGroupNameMetadata], m.ConsumerGroupName)
			assert.Equal(t, tc.metadata[pendingEntriesCountMetadata], strconv.FormatInt(m.TargetPendingEntriesCount, 10))
			if authParams != nil {
				// if authParam is used
				assert.Equal(t, authParams[usernameMetadata], m.ConnectionInfo.Username)
				assert.Equal(t, authParams[passwordMetadata], m.ConnectionInfo.Password)
			} else {
				// if metadata is used to pass credentials' env var names
				assert.Equal(t, tc.resolvedEnv[tc.metadata[usernameMetadata]], m.ConnectionInfo.Username)
				assert.Equal(t, tc.resolvedEnv[tc.metadata[passwordMetadata]], m.ConnectionInfo.Password)
			}

			assert.Equal(t, tc.metadata[databaseIndexMetadata], strconv.Itoa(m.DatabaseIndex))
			b, err := strconv.ParseBool(tc.metadata[enableTLSMetadata])
			assert.Nil(t, err)
			assert.Equal(t, b, m.ConnectionInfo.EnableTLS)
		})
	}

	testCasesLag := []testCase{
		{
			name:     "with address",
			metadata: map[string]string{"stream": "my-stream", "consumerGroup": "my-stream-consumer-group", "lagCount": "5", "activationLagCount": "3", "addressFromEnv": "REDIS_SERVICE", "usernameFromEnv": "REDIS_USERNAME", "passwordFromEnv": "REDIS_PASSWORD", "databaseIndex": "0", "enableTLS": "true"},
			resolvedEnv: map[string]string{
				"REDIS_SERVICE":  "myredis:6379",
				"REDIS_USERNAME": "foobarred",
				"REDIS_PASSWORD": "foobarred",
			},
			authParams: nil,
		},

		{
			name:     "with host and port",
			metadata: map[string]string{"stream": "my-stream", "consumerGroup": "my-stream-consumer-group", "lagCount": "2", "activationLagCount": "3", "hostFromEnv": "REDIS_HOST", "port": "REDIS_PORT", "usernameFromEnv": "REDIS_USERNAME", "passwordFromEnv": "REDIS_PASSWORD", "databaseIndex": "0", "enableTLS": "false"},
			resolvedEnv: map[string]string{
				"REDIS_HOST":     "myredis",
				"REDIS_PORT":     "6379",
				"REDIS_USERNAME": "foobarred",
				"REDIS_PASSWORD": "foobarred",
			},
			authParams: authParams,
		},
	}

	for _, tc := range testCasesLag {
		t.Run(tc.name, func(te *testing.T) {
			m, err := parseRedisStreamsMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: tc.metadata, ResolvedEnv: tc.resolvedEnv, AuthParams: tc.authParams})
			assert.Nil(t, err)
			assert.Equal(t, m.StreamName, tc.metadata[streamNameMetadata])
			assert.Equal(t, m.ConsumerGroupName, tc.metadata[consumerGroupNameMetadata])
			assert.Equal(t, strconv.FormatInt(m.TargetLag, 10), tc.metadata[lagMetadata])
			if authParams != nil {
				// if authParam is used
				assert.Equal(t, m.ConnectionInfo.Username, authParams[usernameMetadata])
				assert.Equal(t, m.ConnectionInfo.Password, authParams[passwordMetadata])
			} else {
				// if metadata is used to pass credentials' env var names
				assert.Equal(t, m.ConnectionInfo.Username, tc.resolvedEnv[tc.metadata[usernameMetadata]])
				assert.Equal(t, m.ConnectionInfo.Password, tc.resolvedEnv[tc.metadata[passwordMetadata]])
			}

			assert.Equal(t, strconv.Itoa(m.DatabaseIndex), tc.metadata[databaseIndexMetadata])
			b, err := strconv.ParseBool(tc.metadata[enableTLSMetadata])
			assert.Nil(t, err)
			assert.Equal(t, m.ConnectionInfo.EnableTLS, b)
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
		{"missing address as well as host/port", map[string]string{"stream": "my-stream", "pendingEntriesCount": "5", "lagCount": "5", "consumerGroup": "my-stream-consumer-group"}, resolvedEnvMap},

		{"host present but missing port", map[string]string{"stream": "my-stream", "pendingEntriesCount": "5", "lagCount": "5", "consumerGroup": "my-stream-consumer-group", "host": "REDIS_HOST"}, resolvedEnvMap},

		{"port present but missing host", map[string]string{"stream": "my-stream", "pendingEntriesCount": "5", "lagCount": "5", "consumerGroup": "my-stream-consumer-group", "port": "REDIS_PORT"}, resolvedEnvMap},

		{"missing stream", map[string]string{"pendingEntriesCount": "5", "consumerGroup": "my-stream-consumer-group", "address": "REDIS_HOST"}, resolvedEnvMap},

		// invalid value for respective fields
		{"invalid lag", map[string]string{"stream": "my-stream", "consumerGroup": "my-stream-consumer-group", "pendingEntriesCount": "5", "lagCount": "junk", "host": "REDIS_HOST", "port": "REDIS_PORT", "databaseIndex": "0", "enableTLS": "false"}, resolvedEnvMap},

		{"invalid pendingEntriesCount", map[string]string{"stream": "my-stream", "consumerGroup": "my-stream-consumer-group", "pendingEntriesCount": "junk", "host": "REDIS_HOST", "port": "REDIS_PORT", "databaseIndex": "0", "enableTLS": "false"}, resolvedEnvMap},

		{"invalid streamLength", map[string]string{"stream": "my-stream", "streamLength": "junk", "host": "REDIS_HOST", "port": "REDIS_PORT", "databaseIndex": "0", "enableTLS": "false"}, resolvedEnvMap},

		{"invalid databaseIndex", map[string]string{"stream": "my-stream", "consumerGroup": "my-stream-consumer-group", "pendingEntriesCount": "15", "address": "REDIS_SERVER", "databaseIndex": "junk", "enableTLS": "false"}, resolvedEnvMap},

		{"invalid enableTLS", map[string]string{"stream": "my-stream", "consumerGroup": "my-stream-consumer-group", "pendingEntriesCount": "15", "address": "REDIS_SERVER", "databaseIndex": "1", "enableTLS": "no"}, resolvedEnvMap},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(te *testing.T) {
			_, err := parseRedisStreamsMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: tc.metadata, ResolvedEnv: tc.resolvedEnv, AuthParams: map[string]string{}})
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
		triggerIndex     int
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
		meta, err := parseRedisStreamsMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, ResolvedEnv: map[string]string{"REDIS_SERVICE": "my-address"}, AuthParams: testData.metadataTestData.authParams, TriggerIndex: testData.triggerIndex})
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
			name: "invalid pending entries count",
			metadata: map[string]string{
				"stream":              "my-stream",
				"hosts":               "a, b, c",
				"ports":               "1, 2, 3",
				"consumerGroup":       "consumer1",
				"pendingEntriesCount": "invalid",
			},
			wantMeta: nil,
			wantErr:  ErrRedisStreamParse,
		},
		{
			name: "invalid lag",
			metadata: map[string]string{
				"stream":              "my-stream",
				"hosts":               "a, b, c",
				"ports":               "1, 2, 3",
				"consumerGroup":       "consumer1",
				"pendingEntriesCount": "5",
				"lagCount":            "junk",
			},
			wantMeta: nil,
			wantErr:  ErrRedisStreamParse,
		},
		{
			name: "address is defined in auth params",
			metadata: map[string]string{
				"stream":             "my-stream",
				"lagCount":           "6",
				"activationLagCount": "3",
				"consumerGroup":      "consumer1",
			},
			authParams: map[string]string{
				"addresses": ":7001, :7002",
			},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 0,
				TargetLag:                 6,
				ActivationLagCount:        3,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{":7001", ":7002"},
				},
				scaleFactor: lagFactor,
			},
			wantErr: nil,
		},
		{
			name: "zero activation lag count with lag count is allowed",
			metadata: map[string]string{
				"stream":             "my-stream",
				"lagCount":           "7",
				"activationLagCount": "0",
				"consumerGroup":      "consumer1",
			},
			authParams: map[string]string{
				"addresses": ":7001, :7002",
			},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 0,
				TargetLag:                 7,
				ActivationLagCount:        0,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{":7001", ":7002"},
				},
				scaleFactor: lagFactor,
			},
			wantErr: nil,
		},
		{
			name: "address is defined in auth params",
			metadata: map[string]string{
				"stream":              "my-stream",
				"pendingEntriesCount": "5",
				"consumerGroup":       "consumer1",
			},
			authParams: map[string]string{
				"addresses": ":7001, :7002",
			},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 5,
				ActivationLagCount:        0,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{":7001", ":7002"},
				},
				scaleFactor: xPendingFactor,
			},
			wantErr: nil,
		},
		{
			name: "hosts and ports given in auth params",
			metadata: map[string]string{
				"stream":             "my-stream",
				"lagCount":           "6",
				"activationLagCount": "3",
				"consumerGroup":      "consumer1",
			},
			authParams: map[string]string{
				"hosts": "   a, b,    c ",
				"ports": "1, 2, 3",
			},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 0,
				TargetLag:                 6,
				ActivationLagCount:        3,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{"a:1", "b:2", "c:3"},
					Hosts:     []string{"a", "b", "c"},
					Ports:     []string{"1", "2", "3"},
				},
				scaleFactor: lagFactor,
			},
			wantErr: nil,
		},
		{
			name: "hosts and ports given in auth params",
			metadata: map[string]string{
				"stream":              "my-stream",
				"pendingEntriesCount": "5",
				"consumerGroup":       "consumer1",
			},
			authParams: map[string]string{
				"hosts": "   a, b,    c ",
				"ports": "1, 2, 3",
			},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 5,
				ActivationLagCount:        0,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{"a:1", "b:2", "c:3"},
					Hosts:     []string{"a", "b", "c"},
					Ports:     []string{"1", "2", "3"},
				},
				scaleFactor: xPendingFactor,
			},
			wantErr: nil,
		},
		{
			name: "username given in authParams",
			metadata: map[string]string{
				"hosts":              "a, b, c",
				"ports":              "1, 2, 3",
				"stream":             "my-stream",
				"lagCount":           "7",
				"activationLagCount": "3",
				"consumerGroup":      "consumer1",
			},
			authParams: map[string]string{
				"username": "username",
			},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 0,
				TargetLag:                 7,
				ActivationLagCount:        3,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{"a:1", "b:2", "c:3"},
					Hosts:     []string{"a", "b", "c"},
					Ports:     []string{"1", "2", "3"},
					Username:  "username",
				},
				scaleFactor: lagFactor,
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
				"consumerGroup":       "consumer1",
			},
			authParams: map[string]string{
				"username": "username",
			},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 5,
				ActivationLagCount:        0,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{"a:1", "b:2", "c:3"},
					Hosts:     []string{"a", "b", "c"},
					Ports:     []string{"1", "2", "3"},
					Username:  "username",
				},
				scaleFactor: xPendingFactor,
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
				"consumerGroup":       "consumer1",
				"username":            "username",
			},
			authParams: map[string]string{},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 5,
				ActivationLagCount:        0,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{"a:1", "b:2", "c:3"},
					Hosts:     []string{"a", "b", "c"},
					Ports:     []string{"1", "2", "3"},
					Username:  "username",
				},
				scaleFactor: xPendingFactor,
			},
			wantErr: nil,
		},
		{
			name: "username given in metadata from env",
			metadata: map[string]string{
				"hosts":              "a, b, c",
				"ports":              "1, 2, 3",
				"stream":             "my-stream",
				"lagCount":           "7",
				"activationLagCount": "3",
				"consumerGroup":      "consumer1",
				"usernameFromEnv":    "REDIS_USERNAME",
			},
			authParams:  map[string]string{},
			resolvedEnv: testRedisResolvedEnv,
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 0,
				TargetLag:                 7,
				ActivationLagCount:        3,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{"a:1", "b:2", "c:3"},
					Hosts:     []string{"a", "b", "c"},
					Ports:     []string{"1", "2", "3"},
					Username:  "none",
				},
				scaleFactor: lagFactor,
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
				"consumerGroup":       "consumer1",
				"usernameFromEnv":     "REDIS_USERNAME",
			},
			authParams:  map[string]string{},
			resolvedEnv: testRedisResolvedEnv,
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 5,
				ActivationLagCount:        0,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{"a:1", "b:2", "c:3"},
					Hosts:     []string{"a", "b", "c"},
					Ports:     []string{"1", "2", "3"},
					Username:  "none",
				},
				scaleFactor: xPendingFactor,
			},
			wantErr: nil,
		},
		{
			name: "password given in authParams",
			metadata: map[string]string{
				"hosts":              "a, b, c",
				"ports":              "1, 2, 3",
				"stream":             "my-stream",
				"lagCount":           "7",
				"activationLagCount": "3",
				"consumerGroup":      "consumer1",
			},
			authParams: map[string]string{
				"password": "password",
			},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 0,
				TargetLag:                 7,
				ActivationLagCount:        3,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{"a:1", "b:2", "c:3"},
					Hosts:     []string{"a", "b", "c"},
					Ports:     []string{"1", "2", "3"},
					Password:  "password",
				},
				scaleFactor: lagFactor,
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
				"consumerGroup":       "consumer1",
			},
			authParams: map[string]string{
				"password": "password",
			},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 5,
				ActivationLagCount:        0,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{"a:1", "b:2", "c:3"},
					Hosts:     []string{"a", "b", "c"},
					Ports:     []string{"1", "2", "3"},
					Password:  "password",
				},
				scaleFactor: xPendingFactor,
			},
			wantErr: nil,
		},
		{
			name: "password given in metadata from env",
			metadata: map[string]string{
				"hosts":              "a, b, c",
				"ports":              "1, 2, 3",
				"stream":             "my-stream",
				"lagCount":           "7",
				"activationLagCount": "3",
				"consumerGroup":      "consumer1",
				"passwordFromEnv":    "REDIS_PASSWORD",
			},
			authParams:  map[string]string{},
			resolvedEnv: testRedisResolvedEnv,
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 0,
				TargetLag:                 7,
				ActivationLagCount:        3,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{"a:1", "b:2", "c:3"},
					Hosts:     []string{"a", "b", "c"},
					Ports:     []string{"1", "2", "3"},
					Password:  "none",
				},
				scaleFactor: lagFactor,
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
				"consumerGroup":       "consumer1",
				"passwordFromEnv":     "REDIS_PASSWORD",
			},
			authParams:  map[string]string{},
			resolvedEnv: testRedisResolvedEnv,
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 5,
				ActivationLagCount:        0,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{"a:1", "b:2", "c:3"},
					Hosts:     []string{"a", "b", "c"},
					Ports:     []string{"1", "2", "3"},
					Password:  "none",
				},
				scaleFactor: xPendingFactor,
			},
			wantErr: nil,
		},
		{
			name: "tls enabled without setting unsafeSsl",
			metadata: map[string]string{
				"hosts":              "a, b, c",
				"ports":              "1, 2, 3",
				"stream":             "my-stream",
				"lagCount":           "7",
				"activationLagCount": "3",
				"consumerGroup":      "consumer1",
				"enableTLS":          "true",
			},
			authParams: map[string]string{
				"password": "password",
			},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 0,
				TargetLag:                 7,
				ActivationLagCount:        3,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{"a:1", "b:2", "c:3"},
					Hosts:     []string{"a", "b", "c"},
					Ports:     []string{"1", "2", "3"},
					Password:  "password",
					EnableTLS: true,
					UnsafeSsl: false,
				},
				scaleFactor: lagFactor,
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
				"consumerGroup":       "consumer1",
				"enableTLS":           "true",
			},
			authParams: map[string]string{
				"password": "password",
			},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 5,
				ActivationLagCount:        0,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{"a:1", "b:2", "c:3"},
					Hosts:     []string{"a", "b", "c"},
					Ports:     []string{"1", "2", "3"},
					Password:  "password",
					EnableTLS: true,
					UnsafeSsl: false,
				},
				scaleFactor: xPendingFactor,
			},
			wantErr: nil,
		},
		{
			name: "tls enabled with unsafeSsl true",
			metadata: map[string]string{
				"hosts":              "a, b, c",
				"ports":              "1, 2, 3",
				"stream":             "my-stream",
				"lagCount":           "7",
				"activationLagCount": "3",
				"consumerGroup":      "consumer1",
				"enableTLS":          "true",
				"unsafeSsl":          "true",
			},
			authParams: map[string]string{
				"password": "password",
			},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 0,
				TargetLag:                 7,
				ActivationLagCount:        3,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{"a:1", "b:2", "c:3"},
					Hosts:     []string{"a", "b", "c"},
					Ports:     []string{"1", "2", "3"},
					Password:  "password",
					EnableTLS: true,
					UnsafeSsl: true,
				},
				scaleFactor: lagFactor,
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
				"consumerGroup":       "consumer1",
				"enableTLS":           "true",
				"unsafeSsl":           "true",
			},
			authParams: map[string]string{
				"password": "password",
			},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 5,
				ActivationLagCount:        0,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{"a:1", "b:2", "c:3"},
					Hosts:     []string{"a", "b", "c"},
					Ports:     []string{"1", "2", "3"},
					Password:  "password",
					EnableTLS: true,
					UnsafeSsl: true,
				},
				scaleFactor: xPendingFactor,
			},
			wantErr: nil,
		},
		{
			name: "tls in auth param",
			metadata: map[string]string{
				"hosts":               "a, b, c",
				"ports":               "1, 2, 3",
				"stream":              "my-stream",
				"pendingEntriesCount": "5",
				"consumerGroup":       "consumer1",
			},
			authParams: map[string]string{
				"password":    "password",
				"tls":         "enable",
				"ca":          "caaa",
				"cert":        "ceert",
				"key":         "keey",
				"keyPassword": "keeyPassword",
			},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 5,
				ActivationLagCount:        0,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses:   []string{"a:1", "b:2", "c:3"},
					Hosts:       []string{"a", "b", "c"},
					Ports:       []string{"1", "2", "3"},
					Password:    "password",
					EnableTLS:   true,
					Ca:          "caaa",
					Cert:        "ceert",
					Key:         "keey",
					KeyPassword: "keeyPassword",
				},
				scaleFactor: xPendingFactor,
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
				StreamName:         "my-stream",
				TargetStreamLength: 5,
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{":7001", ":7002"},
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
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 5,
				TargetLag:                 0,
				ActivationLagCount:        0,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{":7001", ":7002"},
				},
				scaleFactor: xPendingFactor,
			},
			wantErr: nil,
		},
	}

	for _, testCase := range cases {
		c := testCase
		t.Run(c.name, func(t *testing.T) {
			config := &scalersconfig.ScalerConfig{
				TriggerMetadata: c.metadata,
				ResolvedEnv:     c.resolvedEnv,
				AuthParams:      c.authParams,
			}
			meta, err := parseRedisStreamsMetadata(config)
			if c.wantErr != nil {
				assert.ErrorContains(t, err, c.wantErr.Error())
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
				"lagCount":            "invalid",
				"activationLagCount":  "3",
			},
			wantMeta: nil,
			wantErr:  ErrRedisStreamParse,
		},
		{
			name: "address is defined in auth params",
			metadata: map[string]string{
				"stream":             "my-stream",
				"lagCount":           "7",
				"activationLagCount": "3",
				"consumerGroup":      "consumer1",
			},
			authParams: map[string]string{
				"addresses": ":7001, :7002",
			},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 0,
				TargetLag:                 7,
				ActivationLagCount:        3,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{":7001", ":7002"},
				},
				scaleFactor: lagFactor,
			},
			wantErr: nil,
		},
		{
			name: "address is defined in auth params",
			metadata: map[string]string{
				"stream":              "my-stream",
				"pendingEntriesCount": "5",
				"consumerGroup":       "consumer1",
			},
			authParams: map[string]string{
				"addresses": ":7001, :7002",
			},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 5,
				ActivationLagCount:        0,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{":7001", ":7002"},
				},
				scaleFactor: xPendingFactor,
			},
			wantErr: nil,
		},
		{
			name: "hosts and ports given in auth params",
			metadata: map[string]string{
				"stream":             "my-stream",
				"lagCount":           "7",
				"activationLagCount": "3",
				"consumerGroup":      "consumer1",
			},
			authParams: map[string]string{
				"hosts": "   a, b,    c ",
				"ports": "1, 2, 3",
			},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 0,
				TargetLag:                 7,
				ActivationLagCount:        3,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{"a:1", "b:2", "c:3"},
					Hosts:     []string{"a", "b", "c"},
					Ports:     []string{"1", "2", "3"},
				},
				scaleFactor: lagFactor,
			},
			wantErr: nil,
		},
		{
			name: "hosts and ports given in auth params",
			metadata: map[string]string{
				"stream":              "my-stream",
				"pendingEntriesCount": "5",
				"consumerGroup":       "consumer1",
			},
			authParams: map[string]string{
				"hosts": "   a, b,    c ",
				"ports": "1, 2, 3",
			},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 5,
				ActivationLagCount:        0,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{"a:1", "b:2", "c:3"},
					Hosts:     []string{"a", "b", "c"},
					Ports:     []string{"1", "2", "3"},
				},
				scaleFactor: xPendingFactor,
			},
			wantErr: nil,
		},
		{
			name: "username given in authParams",
			metadata: map[string]string{
				"hosts":              "a, b, c",
				"ports":              "1, 2, 3",
				"stream":             "my-stream",
				"lagCount":           "7",
				"activationLagCount": "3",
				"consumerGroup":      "consumer1",
			},
			authParams: map[string]string{
				"username": "username",
			},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 0,
				TargetLag:                 7,
				ActivationLagCount:        3,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{"a:1", "b:2", "c:3"},
					Hosts:     []string{"a", "b", "c"},
					Ports:     []string{"1", "2", "3"},
					Username:  "username",
				},
				scaleFactor: lagFactor,
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
				"consumerGroup":       "consumer1",
			},
			authParams: map[string]string{
				"username": "username",
			},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 5,
				ActivationLagCount:        0,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{"a:1", "b:2", "c:3"},
					Hosts:     []string{"a", "b", "c"},
					Ports:     []string{"1", "2", "3"},
					Username:  "username",
				},
				scaleFactor: xPendingFactor,
			},
			wantErr: nil,
		},
		{
			name: "username given in metadata",
			metadata: map[string]string{
				"hosts":              "a, b, c",
				"ports":              "1, 2, 3",
				"stream":             "my-stream",
				"lagCount":           "7",
				"activationLagCount": "3",
				"consumerGroup":      "consumer1",
				"username":           "username",
			},
			authParams: map[string]string{},
			wantMeta: &redisStreamsMetadata{
				StreamName:         "my-stream",
				TargetLag:          7,
				ActivationLagCount: 3,
				ConsumerGroupName:  "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{"a:1", "b:2", "c:3"},
					Hosts:     []string{"a", "b", "c"},
					Ports:     []string{"1", "2", "3"},
					Username:  "username",
				},
				scaleFactor: lagFactor,
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
				"consumerGroup":       "consumer1",
				"username":            "username",
			},
			authParams: map[string]string{},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 5,
				ActivationLagCount:        0,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{"a:1", "b:2", "c:3"},
					Hosts:     []string{"a", "b", "c"},
					Ports:     []string{"1", "2", "3"},
					Username:  "username",
				},
				scaleFactor: xPendingFactor,
			},
			wantErr: nil,
		},
		{
			name: "username given in metadata from env",
			metadata: map[string]string{
				"hosts":              "a, b, c",
				"ports":              "1, 2, 3",
				"stream":             "my-stream",
				"lagCount":           "7",
				"activationLagCount": "3",
				"consumerGroup":      "consumer1",
				"usernameFromEnv":    "REDIS_USERNAME",
			},
			authParams:  map[string]string{},
			resolvedEnv: testRedisResolvedEnv,
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 0,
				TargetLag:                 7,
				ActivationLagCount:        3,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{"a:1", "b:2", "c:3"},
					Hosts:     []string{"a", "b", "c"},
					Ports:     []string{"1", "2", "3"},
					Username:  "none",
				},
				scaleFactor: lagFactor,
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
				"consumerGroup":       "consumer1",
				"usernameFromEnv":     "REDIS_USERNAME",
			},
			authParams:  map[string]string{},
			resolvedEnv: testRedisResolvedEnv,
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 5,
				ActivationLagCount:        0,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{"a:1", "b:2", "c:3"},
					Hosts:     []string{"a", "b", "c"},
					Ports:     []string{"1", "2", "3"},
					Username:  "none",
				},
				scaleFactor: xPendingFactor,
			},
			wantErr: nil,
		},
		{
			name: "password given in authParams",
			metadata: map[string]string{
				"hosts":              "a, b, c",
				"ports":              "1, 2, 3",
				"stream":             "my-stream",
				"lagCount":           "7",
				"activationLagCount": "3",
				"consumerGroup":      "consumer1",
			},
			authParams: map[string]string{
				"password": "password",
			},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 0,
				TargetLag:                 7,
				ActivationLagCount:        3,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{"a:1", "b:2", "c:3"},
					Hosts:     []string{"a", "b", "c"},
					Ports:     []string{"1", "2", "3"},
					Password:  "password",
				},
				scaleFactor: lagFactor,
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
				"consumerGroup":       "consumer1",
			},
			authParams: map[string]string{
				"password": "password",
			},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 5,
				ActivationLagCount:        0,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{"a:1", "b:2", "c:3"},
					Hosts:     []string{"a", "b", "c"},
					Ports:     []string{"1", "2", "3"},
					Password:  "password",
				},
				scaleFactor: xPendingFactor,
			},
			wantErr: nil,
		},
		{
			name: "password given in metadata from env",
			metadata: map[string]string{
				"hosts":              "a, b, c",
				"ports":              "1, 2, 3",
				"stream":             "my-stream",
				"lagCount":           "7",
				"activationLagCount": "3",
				"consumerGroup":      "consumer1",
				"passwordFromEnv":    "REDIS_PASSWORD",
			},
			authParams:  map[string]string{},
			resolvedEnv: testRedisResolvedEnv,
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 0,
				TargetLag:                 7,
				ActivationLagCount:        3,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{"a:1", "b:2", "c:3"},
					Hosts:     []string{"a", "b", "c"},
					Ports:     []string{"1", "2", "3"},
					Password:  "none",
				},
				scaleFactor: lagFactor,
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
				"consumerGroup":       "consumer1",
				"passwordFromEnv":     "REDIS_PASSWORD",
			},
			authParams:  map[string]string{},
			resolvedEnv: testRedisResolvedEnv,
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 5,
				ActivationLagCount:        0,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{"a:1", "b:2", "c:3"},
					Hosts:     []string{"a", "b", "c"},
					Ports:     []string{"1", "2", "3"},
					Password:  "none",
				},
				scaleFactor: xPendingFactor,
			},
			wantErr: nil,
		},
		{
			name: "sentinelUsername given in authParams",
			metadata: map[string]string{
				"hosts":              "a, b, c",
				"ports":              "1, 2, 3",
				"stream":             "my-stream",
				"lagCount":           "7",
				"activationLagCount": "3",
				"consumerGroup":      "consumer1",
			},
			authParams: map[string]string{
				"sentinelUsername": "sentinelUsername",
			},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 0,
				TargetLag:                 7,
				ActivationLagCount:        3,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses:        []string{"a:1", "b:2", "c:3"},
					Hosts:            []string{"a", "b", "c"},
					Ports:            []string{"1", "2", "3"},
					SentinelUsername: "sentinelUsername",
				},
				scaleFactor: lagFactor,
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
				"consumerGroup":       "consumer1",
			},
			authParams: map[string]string{
				"sentinelUsername": "sentinelUsername",
			},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 5,
				ActivationLagCount:        0,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses:        []string{"a:1", "b:2", "c:3"},
					Hosts:            []string{"a", "b", "c"},
					Ports:            []string{"1", "2", "3"},
					SentinelUsername: "sentinelUsername",
				},
				scaleFactor: xPendingFactor,
			},
			wantErr: nil,
		},
		{
			name: "sentinelUsername given in metadata",
			metadata: map[string]string{
				"hosts":              "a, b, c",
				"ports":              "1, 2, 3",
				"stream":             "my-stream",
				"lagCount":           "7",
				"activationLagCount": "3",
				"consumerGroup":      "consumer1",
				"sentinelUsername":   "sentinelUsername",
			},
			authParams: map[string]string{},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 0,
				TargetLag:                 7,
				ActivationLagCount:        3,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses:        []string{"a:1", "b:2", "c:3"},
					Hosts:            []string{"a", "b", "c"},
					Ports:            []string{"1", "2", "3"},
					SentinelUsername: "sentinelUsername",
				},
				scaleFactor: lagFactor,
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
				"consumerGroup":       "consumer1",
				"sentinelUsername":    "sentinelUsername",
			},
			authParams: map[string]string{},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 5,
				ActivationLagCount:        0,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses:        []string{"a:1", "b:2", "c:3"},
					Hosts:            []string{"a", "b", "c"},
					Ports:            []string{"1", "2", "3"},
					SentinelUsername: "sentinelUsername",
				},
				scaleFactor: xPendingFactor,
			},
			wantErr: nil,
		},
		{
			name: "sentinelUsername given in metadata from env",
			metadata: map[string]string{
				"hosts":                   "a, b, c",
				"ports":                   "1, 2, 3",
				"stream":                  "my-stream",
				"lagCount":                "7",
				"activationLagCount":      "3",
				"consumerGroup":           "consumer1",
				"sentinelUsernameFromEnv": "REDIS_SENTINEL_USERNAME",
			},
			authParams:  map[string]string{},
			resolvedEnv: testRedisResolvedEnv,
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 0,
				TargetLag:                 7,
				ActivationLagCount:        3,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses:        []string{"a:1", "b:2", "c:3"},
					Hosts:            []string{"a", "b", "c"},
					Ports:            []string{"1", "2", "3"},
					SentinelUsername: "none",
				},
				scaleFactor: lagFactor,
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
				"consumerGroup":           "consumer1",
				"sentinelUsernameFromEnv": "REDIS_SENTINEL_USERNAME",
			},
			authParams:  map[string]string{},
			resolvedEnv: testRedisResolvedEnv,
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 5,
				ActivationLagCount:        0,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses:        []string{"a:1", "b:2", "c:3"},
					Hosts:            []string{"a", "b", "c"},
					Ports:            []string{"1", "2", "3"},
					SentinelUsername: "none",
				},
				scaleFactor: xPendingFactor,
			},
			wantErr: nil,
		},
		{
			name: "sentinelPassword given in authParams",
			metadata: map[string]string{
				"hosts":              "a, b, c",
				"ports":              "1, 2, 3",
				"stream":             "my-stream",
				"lagCount":           "7",
				"activationLagCount": "3",
				"consumerGroup":      "consumer1",
			},
			authParams: map[string]string{
				"sentinelPassword": "sentinelPassword",
			},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 0,
				TargetLag:                 7,
				ActivationLagCount:        3,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses:        []string{"a:1", "b:2", "c:3"},
					Hosts:            []string{"a", "b", "c"},
					Ports:            []string{"1", "2", "3"},
					SentinelPassword: "sentinelPassword",
				},
				scaleFactor: lagFactor,
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
				"consumerGroup":       "consumer1",
			},
			authParams: map[string]string{
				"sentinelPassword": "sentinelPassword",
			},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 5,
				ActivationLagCount:        0,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses:        []string{"a:1", "b:2", "c:3"},
					Hosts:            []string{"a", "b", "c"},
					Ports:            []string{"1", "2", "3"},
					SentinelPassword: "sentinelPassword",
				},
				scaleFactor: xPendingFactor,
			},
			wantErr: nil,
		},
		{
			name: "sentinelPassword given in metadata from env",
			metadata: map[string]string{
				"hosts":                   "a, b, c",
				"ports":                   "1, 2, 3",
				"stream":                  "my-stream",
				"lagCount":                "7",
				"activationLagCount":      "3",
				"consumerGroup":           "consumer1",
				"sentinelPasswordFromEnv": "REDIS_SENTINEL_PASSWORD",
			},
			authParams:  map[string]string{},
			resolvedEnv: testRedisResolvedEnv,
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 0,
				TargetLag:                 7,
				ActivationLagCount:        3,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses:        []string{"a:1", "b:2", "c:3"},
					Hosts:            []string{"a", "b", "c"},
					Ports:            []string{"1", "2", "3"},
					SentinelPassword: "none",
				},
				scaleFactor: lagFactor,
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
				"consumerGroup":           "consumer1",
				"sentinelPasswordFromEnv": "REDIS_SENTINEL_PASSWORD",
			},
			authParams:  map[string]string{},
			resolvedEnv: testRedisResolvedEnv,
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 5,
				ActivationLagCount:        0,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses:        []string{"a:1", "b:2", "c:3"},
					Hosts:            []string{"a", "b", "c"},
					Ports:            []string{"1", "2", "3"},
					SentinelPassword: "none",
				},
				scaleFactor: xPendingFactor,
			},
			wantErr: nil,
		},
		{
			name: "sentinelMaster given in authParams",
			metadata: map[string]string{
				"hosts":              "a, b, c",
				"ports":              "1, 2, 3",
				"stream":             "my-stream",
				"lagCount":           "7",
				"activationLagCount": "3",
				"consumerGroup":      "consumer1",
			},
			authParams: map[string]string{
				"sentinelMaster": "sentinelMaster",
			},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 0,
				TargetLag:                 7,
				ActivationLagCount:        3,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses:      []string{"a:1", "b:2", "c:3"},
					Hosts:          []string{"a", "b", "c"},
					Ports:          []string{"1", "2", "3"},
					SentinelMaster: "sentinelMaster",
				},
				scaleFactor: lagFactor,
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
				"consumerGroup":       "consumer1",
			},
			authParams: map[string]string{
				"sentinelMaster": "sentinelMaster",
			},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 5,
				ActivationLagCount:        0,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses:      []string{"a:1", "b:2", "c:3"},
					Hosts:          []string{"a", "b", "c"},
					Ports:          []string{"1", "2", "3"},
					SentinelMaster: "sentinelMaster",
				},
				scaleFactor: xPendingFactor,
			},
			wantErr: nil,
		},
		{
			name: "sentinelMaster given in metadata",
			metadata: map[string]string{
				"hosts":              "a, b, c",
				"ports":              "1, 2, 3",
				"stream":             "my-stream",
				"lagCount":           "7",
				"activationLagCount": "3",
				"consumerGroup":      "consumer1",
				"sentinelMaster":     "sentinelMaster",
			},
			authParams: map[string]string{},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 0,
				TargetLag:                 7,
				ActivationLagCount:        3,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses:      []string{"a:1", "b:2", "c:3"},
					Hosts:          []string{"a", "b", "c"},
					Ports:          []string{"1", "2", "3"},
					SentinelMaster: "sentinelMaster",
				},
				scaleFactor: lagFactor,
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
				"consumerGroup":       "consumer1",
				"sentinelMaster":      "sentinelMaster",
			},
			authParams: map[string]string{},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 5,
				ActivationLagCount:        0,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses:      []string{"a:1", "b:2", "c:3"},
					Hosts:          []string{"a", "b", "c"},
					Ports:          []string{"1", "2", "3"},
					SentinelMaster: "sentinelMaster",
				},
				scaleFactor: xPendingFactor,
			},
			wantErr: nil,
		},
		{
			name: "sentinelMaster given in metadata from env",
			metadata: map[string]string{
				"hosts":                 "a, b, c",
				"ports":                 "1, 2, 3",
				"stream":                "my-stream",
				"lagCount":              "7",
				"activationLagCount":    "3",
				"consumerGroup":         "consumer1",
				"sentinelMasterFromEnv": "REDIS_SENTINEL_MASTER",
			},
			authParams:  map[string]string{},
			resolvedEnv: testRedisResolvedEnv,
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 0,
				TargetLag:                 7,
				ActivationLagCount:        3,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses:      []string{"a:1", "b:2", "c:3"},
					Hosts:          []string{"a", "b", "c"},
					Ports:          []string{"1", "2", "3"},
					SentinelMaster: "none",
				},
				scaleFactor: lagFactor,
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
				"consumerGroup":         "consumer1",
				"sentinelMasterFromEnv": "REDIS_SENTINEL_MASTER",
			},
			authParams:  map[string]string{},
			resolvedEnv: testRedisResolvedEnv,
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 5,
				ActivationLagCount:        0,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses:      []string{"a:1", "b:2", "c:3"},
					Hosts:          []string{"a", "b", "c"},
					Ports:          []string{"1", "2", "3"},
					SentinelMaster: "none",
				},
				scaleFactor: xPendingFactor,
			},
			wantErr: nil,
		},
		{
			name: "tls enabled without setting unsafeSsl",
			metadata: map[string]string{
				"hosts":              "a, b, c",
				"ports":              "1, 2, 3",
				"stream":             "my-stream",
				"lagCount":           "7",
				"activationLagCount": "3",
				"consumerGroup":      "consumer1",
				"enableTLS":          "true",
			},
			authParams: map[string]string{
				"password": "password",
			},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 0,
				TargetLag:                 7,
				ActivationLagCount:        3,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{"a:1", "b:2", "c:3"},
					Hosts:     []string{"a", "b", "c"},
					Ports:     []string{"1", "2", "3"},
					Password:  "password",
					EnableTLS: true,
					UnsafeSsl: false,
				},
				scaleFactor: lagFactor,
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
				"consumerGroup":       "consumer1",
				"enableTLS":           "true",
			},
			authParams: map[string]string{
				"password": "password",
			},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 5,
				ActivationLagCount:        0,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{"a:1", "b:2", "c:3"},
					Hosts:     []string{"a", "b", "c"},
					Ports:     []string{"1", "2", "3"},
					Password:  "password",
					EnableTLS: true,
					UnsafeSsl: false,
				},
				scaleFactor: xPendingFactor,
			},
			wantErr: nil,
		},
		{
			name: "tls enabled with unsafeSsl true",
			metadata: map[string]string{
				"hosts":              "a, b, c",
				"ports":              "1, 2, 3",
				"stream":             "my-stream",
				"lagCount":           "7",
				"activationLagCount": "3",
				"consumerGroup":      "consumer1",
				"enableTLS":          "true",
				"unsafeSsl":          "true",
			},
			authParams: map[string]string{
				"password": "password",
			},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 0,
				TargetLag:                 7,
				ActivationLagCount:        3,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{"a:1", "b:2", "c:3"},
					Hosts:     []string{"a", "b", "c"},
					Ports:     []string{"1", "2", "3"},
					Password:  "password",
					EnableTLS: true,
					UnsafeSsl: true,
				},
				scaleFactor: lagFactor,
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
				"consumerGroup":       "consumer1",
				"enableTLS":           "true",
				"unsafeSsl":           "true",
			},
			authParams: map[string]string{
				"password": "password",
			},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 5,
				ActivationLagCount:        0,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{"a:1", "b:2", "c:3"},
					Hosts:     []string{"a", "b", "c"},
					Ports:     []string{"1", "2", "3"},
					Password:  "password",
					EnableTLS: true,
					UnsafeSsl: true,
				},
				scaleFactor: xPendingFactor,
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
				StreamName:         "my-stream",
				TargetStreamLength: 15,
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{"a:1"},
					Hosts:     []string{"a"},
					Ports:     []string{"1"},
					Password:  "",
					EnableTLS: false,
					UnsafeSsl: false,
				},
				scaleFactor: xLengthFactor,
			},
			wantErr: nil,
		},
		{
			name: "streamLength, pendingEntriesCount and consumerGroup passed",
			metadata: map[string]string{
				"hosts":              "a",
				"ports":              "1",
				"stream":             "my-stream",
				"streamLength":       "15",
				"lagCount":           "70",
				"activationLagCount": "3",
				"consumerGroup":      "consumer1",
			},
			authParams: map[string]string{},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 0,
				TargetLag:                 70,
				ActivationLagCount:        3,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{"a:1"},
					Hosts:     []string{"a"},
					Ports:     []string{"1"},
					Password:  "",
					EnableTLS: false,
					UnsafeSsl: false,
				},
				scaleFactor: lagFactor,
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
				"consumerGroup":       "consumer1",
			},
			authParams: map[string]string{},
			wantMeta: &redisStreamsMetadata{
				StreamName:                "my-stream",
				TargetPendingEntriesCount: 5,
				ActivationLagCount:        0,
				ConsumerGroupName:         "consumer1",
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{"a:1"},
					Hosts:     []string{"a"},
					Ports:     []string{"1"},
					Password:  "",
					EnableTLS: false,
					UnsafeSsl: false,
				},
				scaleFactor: xPendingFactor,
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
				StreamName:         "my-stream",
				TargetStreamLength: 15,
				ConnectionInfo: redisConnectionInfo{
					Addresses: []string{"a:1"},
					Hosts:     []string{"a"},
					Ports:     []string{"1"},
					Password:  "",
					EnableTLS: false,
					UnsafeSsl: false,
				},
				scaleFactor: xLengthFactor,
			},
			wantErr: nil,
		},
	}

	for _, testCase := range cases {
		c := testCase
		t.Run(c.name, func(t *testing.T) {
			config := &scalersconfig.ScalerConfig{
				TriggerMetadata: c.metadata,
				ResolvedEnv:     c.resolvedEnv,
				AuthParams:      c.authParams,
			}
			meta, err := parseRedisStreamsMetadata(config)
			if c.wantErr != nil {
				assert.ErrorContains(t, err, c.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, c.wantMeta, meta)
		})
	}
}

func TestActivityCount(t *testing.T) {
	// Test to make sure GetMetricsAndActivity returns true for isActive
	// when the lag count is greater than activationLagCount and false
	// when it is less.
	type testCase struct {
		name        string
		metadata    map[string]string
		resolvedEnv map[string]string
		authParams  map[string]string
		wantMeta    *redisStreamsMetadata
		wantErr     error
	}
	c := testCase{
		name: "sentinelMaster given in metadata from env",
		metadata: map[string]string{
			"hosts":              "a, b, c",
			"ports":              "1, 2, 3",
			"stream":             "my-stream",
			"lagCount":           "7",
			"activationLagCount": "3",
			"consumerGroup":      "consumer1",
		},
		authParams:  map[string]string{},
		resolvedEnv: testRedisResolvedEnv,
		wantMeta: &redisStreamsMetadata{
			StreamName:                "my-stream",
			TargetPendingEntriesCount: 0,
			TargetLag:                 7,
			ActivationLagCount:        3,
			ConsumerGroupName:         "consumer1",
			ConnectionInfo: redisConnectionInfo{
				Addresses: []string{"a:1", "b:2", "c:3"},
				Hosts:     []string{"a", "b", "c"},
				Ports:     []string{"1", "2", "3"},
			},
			scaleFactor: lagFactor,
		},
		wantErr: nil,
	}
	t.Run(c.name, func(t *testing.T) {
		config := &scalersconfig.ScalerConfig{
			TriggerMetadata: c.metadata,
			ResolvedEnv:     c.resolvedEnv,
			AuthParams:      c.authParams,
		}
		meta, err := parseRedisStreamsMetadata(config)
		if c.wantErr != nil {
			assert.ErrorIs(t, err, c.wantErr)
		} else {
			assert.NoError(t, err)
		}
		assert.Equal(t, c.wantMeta, meta)
		ctx := context.Background()
		metricType, err := GetMetricTargetType(config)
		logger := InitializeLogger(config, "redis_streams_scaler")
		closeFn := func() error {
			return nil
		}

		entriesCountFn := func(ctx context.Context) (int64, error) {
			return 0, nil // Initiall, there is a lag of 0.
		}

		scaler := &redisStreamsScaler{
			metricType:        metricType,
			metadata:          meta,
			closeFn:           closeFn,
			getEntriesCountFn: entriesCountFn,
			logger:            logger,
		}

		if err != nil {
			t.Logf("Scaler error: %s", err)
		}

		// When the lag is 0, the scaler should be inactive.
		metricSpec := scaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		_, isActive, err := scaler.GetMetricsAndActivity(ctx, metricName)

		if err != nil {
			t.Logf("Error when running GetMetricsAndActivity: %s", err)
		}

		assert.Equal(t, isActive, false, "redis scaler shouldn't be active when lag is less than activation")

		scaler.getEntriesCountFn = func(ctx context.Context) (int64, error) {
			return 4, nil // Simulate having a lag of 4, one more than the activation value.
		}
		_, isActive, err = scaler.GetMetricsAndActivity(ctx, metricName)

		if err != nil {
			t.Logf("Error when running GetMetricsAndActivity: %s", err)
		}

		assert.Equal(t, isActive, true, "redis scaler should be active when lag is greater than activation")
	})
}

func TestIsRedisKeyNotFoundError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"exact redis message", fmt.Errorf("ERR no such key"), true},
		{"valkey mixed-case variant", fmt.Errorf("ERR No Such Key"), true},
		{"message embedded in longer string", fmt.Errorf("some prefix: err no such key (reading stream)"), true},
		{"NOGROUP error (not a missing-key error)", fmt.Errorf("NOGROUP No such consumer group"), false},
		{"unrelated error", fmt.Errorf("WRONGTYPE Operation against a key holding the wrong kind of value"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isRedisKeyNotFoundError(tc.err))
		})
	}
}

func TestIsRedisNoGroupError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"standard NOGROUP message", fmt.Errorf("NOGROUP No such consumer group 'mygroup' for key name 'mystream'"), true},
		{"lowercase nogroup", fmt.Errorf("nogroup consumer group not found"), true},
		{"NOGROUP embedded in longer string", fmt.Errorf("ERR: NOGROUP -consumer group missing"), true},
		{"no such key error (not a nogroup error)", fmt.Errorf("ERR no such key"), false},
		{"unrelated error", fmt.Errorf("WRONGTYPE Operation against a key holding the wrong kind of value"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isRedisNoGroupError(tc.err))
		})
	}
}

// TestRedisStreamsGetMetricsAndActivityErrorPaths verifies that:
// - a zero count from a missing key/group is treated as inactive with no error,
// - a count above the activation threshold is active,
// - a genuine redis error is propagated.
func TestRedisStreamsGetMetricsAndActivityErrorPaths(t *testing.T) {
	config := &scalersconfig.ScalerConfig{
		TriggerMetadata: map[string]string{
			"stream":              "my-stream",
			"consumerGroup":       "my-group",
			"pendingEntriesCount": "5",
			"address":             "myredis:6379",
		},
		AuthParams:  map[string]string{},
		ResolvedEnv: map[string]string{},
	}
	meta, err := parseRedisStreamsMetadata(config)
	assert.NoError(t, err)

	metricType, err := GetMetricTargetType(config)
	assert.NoError(t, err)

	logger := InitializeLogger(config, "redis_streams_scaler")
	ctx := context.Background()

	cases := []struct {
		name          string
		countFnReturn int64
		countFnErr    error
		wantActive    bool
		wantErr       bool
	}{
		{
			name:          "missing key or group (count=0, no error) → inactive, no error",
			countFnReturn: 0,
			countFnErr:    nil,
			wantActive:    false,
			wantErr:       false,
		},
		{
			name:          "pending count above activation threshold (default 0) → active",
			countFnReturn: 10,
			countFnErr:    nil,
			wantActive:    true,
			wantErr:       false,
		},
		{
			name:          "genuine redis error is propagated",
			countFnReturn: -1,
			countFnErr:    fmt.Errorf("WRONGTYPE Operation against a key"),
			wantActive:    false,
			wantErr:       true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			scaler := &redisStreamsScaler{
				metricType: metricType,
				metadata:   meta,
				closeFn:    func() error { return nil },
				getEntriesCountFn: func(_ context.Context) (int64, error) {
					return tc.countFnReturn, tc.countFnErr
				},
				logger: logger,
			}
			metricSpec := scaler.GetMetricSpecForScaling(ctx)
			metricName := metricSpec[0].External.Metric.Name
			_, isActive, err := scaler.GetMetricsAndActivity(ctx, metricName)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.wantActive, isActive)
		})
	}
}
