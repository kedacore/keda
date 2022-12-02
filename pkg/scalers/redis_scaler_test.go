package scalers

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
)

var testRedisResolvedEnv = map[string]string{
	"REDIS_HOST":              "none",
	"REDIS_PORT":              "6379",
	"REDIS_USERNAME":          "none",
	"REDIS_PASSWORD":          "none",
	"REDIS_SENTINEL_MASTER":   "none",
	"REDIS_SENTINEL_USERNAME": "none",
	"REDIS_SENTINEL_PASSWORD": "none",
}

type parseRedisMetadataTestData struct {
	metadata   map[string]string
	isError    bool
	authParams map[string]string
}

type redisMetricIdentifier struct {
	metadataTestData *parseRedisMetadataTestData
	scalerIndex      int
	name             string
}

var testRedisMetadata = []parseRedisMetadataTestData{
	// nothing passed
	{map[string]string{}, true, map[string]string{}},
	// properly formed listName
	{map[string]string{"listName": "mylist", "listLength": "10", "addressFromEnv": "REDIS_HOST", "passwordFromEnv": "REDIS_PASSWORD"}, false, map[string]string{}},
	// properly formed hostPort
	{map[string]string{"listName": "mylist", "listLength": "10", "hostFromEnv": "REDIS_HOST", "portFromEnv": "REDIS_PORT", "passwordFromEnv": "REDIS_PASSWORD"}, false, map[string]string{}},
	// properly formed hostPort
	{map[string]string{"listName": "mylist", "listLength": "10", "addressFromEnv": "REDIS_HOST", "host": "REDIS_HOST", "port": "REDIS_PORT", "passwordFromEnv": "REDIS_PASSWORD"}, false, map[string]string{}},
	// improperly formed hostPort
	{map[string]string{"listName": "mylist", "listLength": "10", "hostFromEnv": "REDIS_HOST", "passwordFromEnv": "REDIS_PASSWORD"}, true, map[string]string{}},
	// properly formed listName, empty address
	{map[string]string{"listName": "mylist", "listLength": "10", "address": "", "password": ""}, true, map[string]string{}},
	// improperly formed listLength
	{map[string]string{"listName": "mylist", "listLength": "AA", "addressFromEnv": "REDIS_HOST", "password": ""}, true, map[string]string{}},
	// improperly formed activationListLength
	{map[string]string{"listName": "mylist", "listLength": "1", "activationListLength": "AA", "addressFromEnv": "REDIS_HOST", "password": ""}, true, map[string]string{}},
	// address does not resolve
	{map[string]string{"listName": "mylist", "listLength": "0", "addressFromEnv": "REDIS_WRONG", "password": ""}, true, map[string]string{}},
	// password is defined in the authParams
	{map[string]string{"listName": "mylist", "listLength": "0", "addressFromEnv": "REDIS_WRONG"}, true, map[string]string{"password": ""}},
	// address is defined in the authParams
	{map[string]string{"listName": "mylist", "listLength": "0"}, false, map[string]string{"address": "localhost:6379"}},
	// host and port is defined in the authParams
	{map[string]string{"listName": "mylist", "listLength": "0"}, false, map[string]string{"host": "localhost", "port": "6379"}},
	// host only is defined in the authParams
	{map[string]string{"listName": "mylist", "listLength": "0"}, true, map[string]string{"host": "localhost"}}}

var redisMetricIdentifiers = []redisMetricIdentifier{
	{&testRedisMetadata[1], 0, "s0-redis-mylist"},
	{&testRedisMetadata[1], 1, "s1-redis-mylist"},
}

func TestRedisParseMetadata(t *testing.T) {
	testCaseNum := 1
	for _, testData := range testRedisMetadata {
		_, err := parseRedisMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, ResolvedEnv: testRedisResolvedEnv, AuthParams: testData.authParams}, parseRedisAddress)
		if err != nil && !testData.isError {
			t.Errorf("Expected success but got error for unit test # %v", testCaseNum)
		}
		if testData.isError && err == nil {
			t.Errorf("Expected error but got success for unit test #%v", testCaseNum)
		}
		testCaseNum++
	}
}

func TestRedisGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range redisMetricIdentifiers {
		meta, err := parseRedisMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, ResolvedEnv: testRedisResolvedEnv, AuthParams: testData.metadataTestData.authParams, ScalerIndex: testData.scalerIndex}, parseRedisAddress)
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		closeFn := func() error { return nil }
		lengthFn := func(context.Context) (int64, error) { return -1, nil }
		mockRedisScaler := redisScaler{
			"",
			meta,
			closeFn,
			lengthFn,
			logr.Discard(),
		}

		metricSpec := mockRedisScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

func TestParseRedisClusterMetadata(t *testing.T) {
	cases := []struct {
		name        string
		metadata    map[string]string
		resolvedEnv map[string]string
		authParams  map[string]string
		wantMeta    *redisMetadata
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
			name: "no list name",
			metadata: map[string]string{
				"hosts":      "a, b, c",
				"ports":      "1, 2, 3",
				"listLength": "5",
			},
			wantMeta: nil,
			wantErr:  errors.New("no list name given"),
		},
		{
			name: "invalid list length",
			metadata: map[string]string{
				"hosts":      "a, b, c",
				"ports":      "1, 2, 3",
				"listName":   "mylist",
				"listLength": "invalid",
			},
			wantMeta: nil,
			wantErr:  errors.New("list length parsing error"),
		},
		{
			name: "address is defined in auth params",
			metadata: map[string]string{
				"listName": "mylist",
			},
			authParams: map[string]string{
				"addresses": ":7001, :7002",
			},
			wantMeta: &redisMetadata{
				listLength: 5,
				listName:   "mylist",
				connectionInfo: redisConnectionInfo{
					addresses: []string{":7001", ":7002"},
				},
			},
			wantErr: nil,
		},
		{
			name: "hosts and ports given in auth params",
			metadata: map[string]string{
				"listName": "mylist",
			},
			authParams: map[string]string{
				"hosts": "   a, b,    c ",
				"ports": "1, 2, 3",
			},
			wantMeta: &redisMetadata{
				listLength: 5,
				listName:   "mylist",
				connectionInfo: redisConnectionInfo{
					addresses: []string{"a:1", "b:2", "c:3"},
					hosts:     []string{"a", "b", "c"},
					ports:     []string{"1", "2", "3"},
				},
			},
			wantErr: nil,
		},
		{
			name: "username given in authParams",
			metadata: map[string]string{
				"hosts":    "a, b, c",
				"ports":    "1, 2, 3",
				"listName": "mylist",
			},
			authParams: map[string]string{
				"username": "username",
			},
			wantMeta: &redisMetadata{
				listLength: 5,
				listName:   "mylist",
				connectionInfo: redisConnectionInfo{
					addresses: []string{"a:1", "b:2", "c:3"},
					hosts:     []string{"a", "b", "c"},
					ports:     []string{"1", "2", "3"},
					username:  "username",
				},
			},
			wantErr: nil,
		},
		{
			name: "username given in metadata",
			metadata: map[string]string{
				"hosts":    "a, b, c",
				"ports":    "1, 2, 3",
				"listName": "mylist",
				"username": "username",
			},
			authParams: map[string]string{},
			wantMeta: &redisMetadata{
				listLength: 5,
				listName:   "mylist",
				connectionInfo: redisConnectionInfo{
					addresses: []string{"a:1", "b:2", "c:3"},
					hosts:     []string{"a", "b", "c"},
					ports:     []string{"1", "2", "3"},
					username:  "username",
				},
			},
			wantErr: nil,
		},
		{
			name: "username given in metadata from env",
			metadata: map[string]string{
				"hosts":           "a, b, c",
				"ports":           "1, 2, 3",
				"listName":        "mylist",
				"usernameFromEnv": "REDIS_USERNAME",
			},
			authParams:  map[string]string{},
			resolvedEnv: testRedisResolvedEnv,
			wantMeta: &redisMetadata{
				listLength: 5,
				listName:   "mylist",
				connectionInfo: redisConnectionInfo{
					addresses: []string{"a:1", "b:2", "c:3"},
					hosts:     []string{"a", "b", "c"},
					ports:     []string{"1", "2", "3"},
					username:  "none",
				},
			},
			wantErr: nil,
		},
		{
			name: "password given in authParams",
			metadata: map[string]string{
				"hosts":    "a, b, c",
				"ports":    "1, 2, 3",
				"listName": "mylist",
			},
			authParams: map[string]string{
				"password": "password",
			},
			wantMeta: &redisMetadata{
				listLength: 5,
				listName:   "mylist",
				connectionInfo: redisConnectionInfo{
					addresses: []string{"a:1", "b:2", "c:3"},
					hosts:     []string{"a", "b", "c"},
					ports:     []string{"1", "2", "3"},
					password:  "password",
				},
			},
			wantErr: nil,
		},
		{
			name: "password given in metadata from env",
			metadata: map[string]string{
				"hosts":           "a, b, c",
				"ports":           "1, 2, 3",
				"listName":        "mylist",
				"passwordFromEnv": "REDIS_PASSWORD",
			},
			authParams:  map[string]string{},
			resolvedEnv: testRedisResolvedEnv,
			wantMeta: &redisMetadata{
				listLength: 5,
				listName:   "mylist",
				connectionInfo: redisConnectionInfo{
					addresses: []string{"a:1", "b:2", "c:3"},
					hosts:     []string{"a", "b", "c"},
					ports:     []string{"1", "2", "3"},
					password:  "none",
				},
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
			meta, err := parseRedisMetadata(config, parseRedisClusterAddress)
			if c.wantErr != nil {
				assert.Contains(t, err.Error(), c.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, c.wantMeta, meta)
		})
	}
}

func TestParseRedisSentinelMetadata(t *testing.T) {
	cases := []struct {
		name        string
		metadata    map[string]string
		resolvedEnv map[string]string
		authParams  map[string]string
		wantMeta    *redisMetadata
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
			name: "no list name",
			metadata: map[string]string{
				"hosts":      "a, b, c",
				"ports":      "1, 2, 3",
				"listLength": "5",
			},
			wantMeta: nil,
			wantErr:  errors.New("no list name given"),
		},
		{
			name: "invalid list length",
			metadata: map[string]string{
				"hosts":      "a, b, c",
				"ports":      "1, 2, 3",
				"listName":   "mylist",
				"listLength": "invalid",
			},
			wantMeta: nil,
			wantErr:  errors.New("list length parsing error"),
		},
		{
			name: "address is defined in auth params",
			metadata: map[string]string{
				"listName": "mylist",
			},
			authParams: map[string]string{
				"addresses": ":7001, :7002",
			},
			wantMeta: &redisMetadata{
				listLength: 5,
				listName:   "mylist",
				connectionInfo: redisConnectionInfo{
					addresses: []string{":7001", ":7002"},
				},
			},
			wantErr: nil,
		},
		{
			name: "hosts and ports given in auth params",
			metadata: map[string]string{
				"listName": "mylist",
			},
			authParams: map[string]string{
				"hosts": "   a, b,    c ",
				"ports": "1, 2, 3",
			},
			wantMeta: &redisMetadata{
				listLength: 5,
				listName:   "mylist",
				connectionInfo: redisConnectionInfo{
					addresses: []string{"a:1", "b:2", "c:3"},
					hosts:     []string{"a", "b", "c"},
					ports:     []string{"1", "2", "3"},
				},
			},
			wantErr: nil,
		},
		{
			name: "hosts and ports given in auth params",
			metadata: map[string]string{
				"listName": "mylist",
			},
			authParams: map[string]string{
				"hosts": "   a, b,    c ",
				"ports": "1, 2, 3",
			},
			wantMeta: &redisMetadata{
				listLength: 5,
				listName:   "mylist",
				connectionInfo: redisConnectionInfo{
					addresses: []string{"a:1", "b:2", "c:3"},
					hosts:     []string{"a", "b", "c"},
					ports:     []string{"1", "2", "3"},
				},
			},
			wantErr: nil,
		},
		{
			name: "username given in authParams",
			metadata: map[string]string{
				"hosts":    "a, b, c",
				"ports":    "1, 2, 3",
				"listName": "mylist",
			},
			authParams: map[string]string{
				"username": "username",
			},
			wantMeta: &redisMetadata{
				listLength: 5,
				listName:   "mylist",
				connectionInfo: redisConnectionInfo{
					addresses: []string{"a:1", "b:2", "c:3"},
					hosts:     []string{"a", "b", "c"},
					ports:     []string{"1", "2", "3"},
					username:  "username",
				},
			},
			wantErr: nil,
		},
		{
			name: "username given in metadata",
			metadata: map[string]string{
				"hosts":    "a, b, c",
				"ports":    "1, 2, 3",
				"listName": "mylist",
				"username": "username",
			},
			authParams: map[string]string{},
			wantMeta: &redisMetadata{
				listLength: 5,
				listName:   "mylist",
				connectionInfo: redisConnectionInfo{
					addresses: []string{"a:1", "b:2", "c:3"},
					hosts:     []string{"a", "b", "c"},
					ports:     []string{"1", "2", "3"},
					username:  "username",
				},
			},
			wantErr: nil,
		},
		{
			name: "username given in metadata from env",
			metadata: map[string]string{
				"hosts":           "a, b, c",
				"ports":           "1, 2, 3",
				"listName":        "mylist",
				"usernameFromEnv": "REDIS_USERNAME",
			},
			authParams:  map[string]string{},
			resolvedEnv: testRedisResolvedEnv,
			wantMeta: &redisMetadata{
				listLength: 5,
				listName:   "mylist",
				connectionInfo: redisConnectionInfo{
					addresses: []string{"a:1", "b:2", "c:3"},
					hosts:     []string{"a", "b", "c"},
					ports:     []string{"1", "2", "3"},
					username:  "none",
				},
			},
			wantErr: nil,
		},
		{
			name: "password given in authParams",
			metadata: map[string]string{
				"hosts":    "a, b, c",
				"ports":    "1, 2, 3",
				"listName": "mylist",
			},
			authParams: map[string]string{
				"password": "password",
			},
			wantMeta: &redisMetadata{
				listLength: 5,
				listName:   "mylist",
				connectionInfo: redisConnectionInfo{
					addresses: []string{"a:1", "b:2", "c:3"},
					hosts:     []string{"a", "b", "c"},
					ports:     []string{"1", "2", "3"},
					password:  "password",
				},
			},
			wantErr: nil,
		},
		{
			name: "password given in metadata from env",
			metadata: map[string]string{
				"hosts":           "a, b, c",
				"ports":           "1, 2, 3",
				"listName":        "mylist",
				"passwordFromEnv": "REDIS_PASSWORD",
			},
			authParams:  map[string]string{},
			resolvedEnv: testRedisResolvedEnv,
			wantMeta: &redisMetadata{
				listLength: 5,
				listName:   "mylist",
				connectionInfo: redisConnectionInfo{
					addresses: []string{"a:1", "b:2", "c:3"},
					hosts:     []string{"a", "b", "c"},
					ports:     []string{"1", "2", "3"},
					password:  "none",
				},
			},
			wantErr: nil,
		},
		{
			name: "sentinelUsername given in authParams",
			metadata: map[string]string{
				"hosts":    "a, b, c",
				"ports":    "1, 2, 3",
				"listName": "mylist",
			},
			authParams: map[string]string{
				"sentinelUsername": "sentinelUsername",
			},
			wantMeta: &redisMetadata{
				listLength: 5,
				listName:   "mylist",
				connectionInfo: redisConnectionInfo{
					addresses:        []string{"a:1", "b:2", "c:3"},
					hosts:            []string{"a", "b", "c"},
					ports:            []string{"1", "2", "3"},
					sentinelUsername: "sentinelUsername",
				},
			},
			wantErr: nil,
		},
		{
			name: "sentinelUsername given in metadata",
			metadata: map[string]string{
				"hosts":            "a, b, c",
				"ports":            "1, 2, 3",
				"listName":         "mylist",
				"sentinelUsername": "sentinelUsername",
			},
			authParams: map[string]string{},
			wantMeta: &redisMetadata{
				listLength: 5,
				listName:   "mylist",
				connectionInfo: redisConnectionInfo{
					addresses:        []string{"a:1", "b:2", "c:3"},
					hosts:            []string{"a", "b", "c"},
					ports:            []string{"1", "2", "3"},
					sentinelUsername: "sentinelUsername",
				},
			},
			wantErr: nil,
		},
		{
			name: "sentinelUsername given in metadata from env",
			metadata: map[string]string{
				"hosts":                   "a, b, c",
				"ports":                   "1, 2, 3",
				"listName":                "mylist",
				"sentinelUsernameFromEnv": "REDIS_SENTINEL_USERNAME",
			},
			authParams:  map[string]string{},
			resolvedEnv: testRedisResolvedEnv,
			wantMeta: &redisMetadata{
				listLength: 5,
				listName:   "mylist",
				connectionInfo: redisConnectionInfo{
					addresses:        []string{"a:1", "b:2", "c:3"},
					hosts:            []string{"a", "b", "c"},
					ports:            []string{"1", "2", "3"},
					sentinelUsername: "none",
				},
			},
			wantErr: nil,
		},
		{
			name: "sentinelPassword given in authParams",
			metadata: map[string]string{
				"hosts":    "a, b, c",
				"ports":    "1, 2, 3",
				"listName": "mylist",
			},
			authParams: map[string]string{
				"sentinelPassword": "sentinelPassword",
			},
			wantMeta: &redisMetadata{
				listLength: 5,
				listName:   "mylist",
				connectionInfo: redisConnectionInfo{
					addresses:        []string{"a:1", "b:2", "c:3"},
					hosts:            []string{"a", "b", "c"},
					ports:            []string{"1", "2", "3"},
					sentinelPassword: "sentinelPassword",
				},
			},
			wantErr: nil,
		},
		{
			name: "sentinelPassword given in metadata from env",
			metadata: map[string]string{
				"hosts":                   "a, b, c",
				"ports":                   "1, 2, 3",
				"listName":                "mylist",
				"sentinelPasswordFromEnv": "REDIS_SENTINEL_PASSWORD",
			},
			authParams:  map[string]string{},
			resolvedEnv: testRedisResolvedEnv,
			wantMeta: &redisMetadata{
				listLength: 5,
				listName:   "mylist",
				connectionInfo: redisConnectionInfo{
					addresses:        []string{"a:1", "b:2", "c:3"},
					hosts:            []string{"a", "b", "c"},
					ports:            []string{"1", "2", "3"},
					sentinelPassword: "none",
				},
			},
			wantErr: nil,
		},
		{
			name: "sentinelMaster given in authParams",
			metadata: map[string]string{
				"hosts":    "a, b, c",
				"ports":    "1, 2, 3",
				"listName": "mylist",
			},
			authParams: map[string]string{
				"sentinelMaster": "sentinelMaster",
			},
			wantMeta: &redisMetadata{
				listLength: 5,
				listName:   "mylist",
				connectionInfo: redisConnectionInfo{
					addresses:      []string{"a:1", "b:2", "c:3"},
					hosts:          []string{"a", "b", "c"},
					ports:          []string{"1", "2", "3"},
					sentinelMaster: "sentinelMaster",
				},
			},
			wantErr: nil,
		},
		{
			name: "sentinelMaster given in metadata",
			metadata: map[string]string{
				"hosts":          "a, b, c",
				"ports":          "1, 2, 3",
				"listName":       "mylist",
				"sentinelMaster": "sentinelMaster",
			},
			authParams: map[string]string{},
			wantMeta: &redisMetadata{
				listLength: 5,
				listName:   "mylist",
				connectionInfo: redisConnectionInfo{
					addresses:      []string{"a:1", "b:2", "c:3"},
					hosts:          []string{"a", "b", "c"},
					ports:          []string{"1", "2", "3"},
					sentinelMaster: "sentinelMaster",
				},
			},
			wantErr: nil,
		},
		{
			name: "sentinelMaster given in metadata from env",
			metadata: map[string]string{
				"hosts":                 "a, b, c",
				"ports":                 "1, 2, 3",
				"listName":              "mylist",
				"sentinelMasterFromEnv": "REDIS_SENTINEL_MASTER",
			},
			authParams:  map[string]string{},
			resolvedEnv: testRedisResolvedEnv,
			wantMeta: &redisMetadata{
				listLength: 5,
				listName:   "mylist",
				connectionInfo: redisConnectionInfo{
					addresses:      []string{"a:1", "b:2", "c:3"},
					hosts:          []string{"a", "b", "c"},
					ports:          []string{"1", "2", "3"},
					sentinelMaster: "none",
				},
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
			meta, err := parseRedisMetadata(config, parseRedisSentinelAddress)
			if c.wantErr != nil {
				assert.Contains(t, err.Error(), c.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, c.wantMeta, meta)
		})
	}
}
