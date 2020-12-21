package scalers

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testRedisResolvedEnv = map[string]string{
	"REDIS_HOST":     "none",
	"REDIS_PORT":     "6379",
	"REDIS_PASSWORD": "none",
}

type parseRedisMetadataTestData struct {
	metadata   map[string]string
	isError    bool
	authParams map[string]string
}

type redisMetricIdentifier struct {
	metadataTestData *parseRedisMetadataTestData
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
	{&testRedisMetadata[1], "redis-mylist"},
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
		meta, err := parseRedisMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, ResolvedEnv: testRedisResolvedEnv, AuthParams: testData.metadataTestData.authParams}, parseRedisAddress)
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		closeFn := func() error { return nil }
		lengthFn := func() (int64, error) { return -1, nil }
		mockRedisScaler := redisScaler{
			meta,
			closeFn,
			lengthFn,
		}

		metricSpec := mockRedisScaler.GetMetricSpecForScaling()
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
				targetListLength: 5,
				listName:         "mylist",
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
				targetListLength: 5,
				listName:         "mylist",
				connectionInfo: redisConnectionInfo{
					addresses: []string{"a:1", "b:2", "c:3"},
					hosts:     []string{"a", "b", "c"},
					ports:     []string{"1", "2", "3"},
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
