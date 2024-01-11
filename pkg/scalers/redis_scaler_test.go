package scalers

import (
	"context"
	"strconv"
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
	enableTLS  bool
}

type redisMetricIdentifier struct {
	metadataTestData *parseRedisMetadataTestData
	triggerIndex     int
	name             string
}

var testRedisMetadata = []parseRedisMetadataTestData{
	// nothing passed
	{map[string]string{}, true, map[string]string{}, false},
	// properly formed listName
	{map[string]string{"listName": "mylist", "listLength": "10", "addressFromEnv": "REDIS_HOST", "passwordFromEnv": "REDIS_PASSWORD"}, false, map[string]string{}, false},
	// properly formed hostPort
	{map[string]string{"listName": "mylist", "listLength": "10", "hostFromEnv": "REDIS_HOST", "portFromEnv": "REDIS_PORT", "passwordFromEnv": "REDIS_PASSWORD"}, false, map[string]string{}, false},
	// properly formed hostPort
	{map[string]string{"listName": "mylist", "listLength": "10", "addressFromEnv": "REDIS_HOST", "host": "REDIS_HOST", "port": "REDIS_PORT", "passwordFromEnv": "REDIS_PASSWORD"}, false, map[string]string{}, false},
	// improperly formed hostPort
	{map[string]string{"listName": "mylist", "listLength": "10", "hostFromEnv": "REDIS_HOST", "passwordFromEnv": "REDIS_PASSWORD"}, true, map[string]string{}, false},
	// properly formed listName, empty address
	{map[string]string{"listName": "mylist", "listLength": "10", "address": "", "password": ""}, true, map[string]string{}, false},
	// improperly formed listLength
	{map[string]string{"listName": "mylist", "listLength": "AA", "addressFromEnv": "REDIS_HOST", "password": ""}, true, map[string]string{}, false},
	// improperly formed activationListLength
	{map[string]string{"listName": "mylist", "listLength": "1", "activationListLength": "AA", "addressFromEnv": "REDIS_HOST", "password": ""}, true, map[string]string{}, false},
	// address does not resolve
	{map[string]string{"listName": "mylist", "listLength": "0", "addressFromEnv": "REDIS_WRONG", "password": ""}, true, map[string]string{}, false},
	// password is defined in the authParams
	{map[string]string{"listName": "mylist", "listLength": "0", "addressFromEnv": "REDIS_WRONG"}, true, map[string]string{"password": ""}, false},
	// address is defined in the authParams
	{map[string]string{"listName": "mylist", "listLength": "0"}, false, map[string]string{"address": "localhost:6379"}, false},
	// host and port is defined in the authParams
	{map[string]string{"listName": "mylist", "listLength": "0"}, false, map[string]string{"host": "localhost", "port": "6379"}, false},
	// enableTLS, TLS defined in the authParams only
	{map[string]string{"listName": "mylist", "listLength": "0"}, false, map[string]string{"address": "localhost:6379", "tls": "enable", "ca": "caaa", "cert": "ceert", "key": "keey"}, true},
	// enableTLS, TLS cert/key and assumed public CA
	{map[string]string{"listName": "mylist", "listLength": "0"}, false, map[string]string{"address": "localhost:6379", "tls": "enable", "cert": "ceert", "key": "keey"}, true},
	// enableTLS, TLS cert/key + key password and assumed public CA
	{map[string]string{"listName": "mylist", "listLength": "0"}, false, map[string]string{"address": "localhost:6379", "tls": "enable", "cert": "ceert", "key": "keey", "keyPassword": "keeyPassword"}, true},
	// enableTLS, TLS CA only
	{map[string]string{"listName": "mylist", "listLength": "0"}, false, map[string]string{"address": "localhost:6379", "tls": "enable", "ca": "caaa"}, true},
	// enableTLS is enabled by metadata
	{map[string]string{"listName": "mylist", "listLength": "0", "enableTLS": "true"}, false, map[string]string{"address": "localhost:6379"}, true},
	// enableTLS is defined both in authParams and metadata
	{map[string]string{"listName": "mylist", "listLength": "0", "enableTLS": "true"}, true, map[string]string{"address": "localhost:6379", "tls": "disable"}, true},
	// host only is defined in the authParams
	{map[string]string{"listName": "mylist", "listLength": "0"}, true, map[string]string{"host": "localhost"}, false}}

var redisMetricIdentifiers = []redisMetricIdentifier{
	{&testRedisMetadata[1], 0, "s0-redis-mylist"},
	{&testRedisMetadata[1], 1, "s1-redis-mylist"},
}

func TestRedisParseMetadata(t *testing.T) {
	testCaseNum := 0
	for _, testData := range testRedisMetadata {
		testCaseNum++
		meta, err := parseRedisMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, ResolvedEnv: testRedisResolvedEnv, AuthParams: testData.authParams}, parseRedisAddress)
		if err != nil && !testData.isError {
			t.Errorf("Expected success but got error for unit test # %v", testCaseNum)
		}
		if testData.isError && err == nil {
			t.Errorf("Expected error but got success for unit test #%v", testCaseNum)
		}
		if testData.isError {
			continue
		}
		if meta.connectionInfo.enableTLS != testData.enableTLS {
			t.Errorf("Expected enableTLS to be set to %v but got %v for unit test #%v\n", testData.enableTLS, meta.connectionInfo.enableTLS, testCaseNum)
		}
		if meta.connectionInfo.enableTLS {
			if meta.connectionInfo.ca != testData.authParams["ca"] {
				t.Errorf("Expected ca to be set to %v but got %v for unit test #%v\n", testData.authParams["ca"], meta.connectionInfo.enableTLS, testCaseNum)
			}
			if meta.connectionInfo.cert != testData.authParams["cert"] {
				t.Errorf("Expected cert to be set to %v but got %v for unit test #%v\n", testData.authParams["cert"], meta.connectionInfo.cert, testCaseNum)
			}
			if meta.connectionInfo.key != testData.authParams["key"] {
				t.Errorf("Expected key to be set to %v but got %v for unit test #%v\n", testData.authParams["key"], meta.connectionInfo.key, testCaseNum)
			}
			if meta.connectionInfo.keyPassword != testData.authParams["keyPassword"] {
				t.Errorf("Expected key to be set to %v but got %v for unit test #%v\n", testData.authParams["keyPassword"], meta.connectionInfo.key, testCaseNum)
			}
		}
	}
}

func TestRedisGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range redisMetricIdentifiers {
		meta, err := parseRedisMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, ResolvedEnv: testRedisResolvedEnv, AuthParams: testData.metadataTestData.authParams, TriggerIndex: testData.triggerIndex}, parseRedisAddress)
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
			name: "no list name",
			metadata: map[string]string{
				"hosts":      "a, b, c",
				"ports":      "1, 2, 3",
				"listLength": "5",
			},
			wantMeta: nil,
			wantErr:  ErrRedisNoListName,
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
			wantErr:  strconv.ErrSyntax,
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
		{
			name: "tls enabled without setting unsafeSsl",
			metadata: map[string]string{
				"listName":  "mylist",
				"enableTLS": "true",
			},
			authParams: map[string]string{
				"addresses": ":7001, :7002",
			},
			wantMeta: &redisMetadata{
				listLength: 5,
				listName:   "mylist",
				connectionInfo: redisConnectionInfo{
					addresses: []string{":7001", ":7002"},
					enableTLS: true,
					unsafeSsl: false,
				},
			},
			wantErr: nil,
		},
		{
			name: "tls enabled with unsafeSsl true",
			metadata: map[string]string{
				"listName":  "mylist",
				"enableTLS": "true",
				"unsafeSsl": "true",
			},
			authParams: map[string]string{
				"addresses": ":7001, :7002",
			},
			wantMeta: &redisMetadata{
				listLength: 5,
				listName:   "mylist",
				connectionInfo: redisConnectionInfo{
					addresses: []string{":7001", ":7002"},
					enableTLS: true,
					unsafeSsl: true,
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
				assert.ErrorIs(t, err, c.wantErr)
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
			name: "no list name",
			metadata: map[string]string{
				"hosts":      "a, b, c",
				"ports":      "1, 2, 3",
				"listLength": "5",
			},
			wantMeta: nil,
			wantErr:  ErrRedisNoListName,
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
			wantErr:  strconv.ErrSyntax,
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
		{
			name: "tls enabled without setting unsafeSsl",
			metadata: map[string]string{
				"listName":  "mylist",
				"enableTLS": "true",
			},
			authParams: map[string]string{
				"addresses": ":7001, :7002",
			},
			wantMeta: &redisMetadata{
				listLength: 5,
				listName:   "mylist",
				connectionInfo: redisConnectionInfo{
					addresses: []string{":7001", ":7002"},
					enableTLS: true,
					unsafeSsl: false,
				},
			},
			wantErr: nil,
		},
		{
			name: "tls enabled with unsafeSsl true",
			metadata: map[string]string{
				"listName":  "mylist",
				"enableTLS": "true",
				"unsafeSsl": "true",
			},
			authParams: map[string]string{
				"addresses": ":7001, :7002",
			},
			wantMeta: &redisMetadata{
				listLength: 5,
				listName:   "mylist",
				connectionInfo: redisConnectionInfo{
					addresses: []string{":7001", ":7002"},
					enableTLS: true,
					unsafeSsl: true,
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
				assert.ErrorIs(t, err, c.wantErr)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, c.wantMeta, meta)
		})
	}
}
