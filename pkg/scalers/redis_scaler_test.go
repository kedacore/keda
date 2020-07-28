package scalers

import (
	"testing"
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
	{map[string]string{"listName": "mylist", "listLength": "10", "address": "REDIS_HOST", "password": "REDIS_PASSWORD"}, false, map[string]string{}},
	// properly formed hostPort
	{map[string]string{"listName": "mylist", "listLength": "10", "host": "REDIS_HOST", "port": "REDIS_PORT", "password": "REDIS_PASSWORD"}, false, map[string]string{}},
	// properly formed hostPort
	{map[string]string{"listName": "mylist", "listLength": "10", "address": "REDIS_HOST", "host": "REDIS_HOST", "port": "REDIS_PORT", "password": "REDIS_PASSWORD"}, false, map[string]string{}},
	// improperly formed hostPort
	{map[string]string{"listName": "mylist", "listLength": "10", "host": "REDIS_HOST", "password": "REDIS_PASSWORD"}, true, map[string]string{}},
	// properly formed listName, empty address
	{map[string]string{"listName": "mylist", "listLength": "10", "address": "", "password": ""}, true, map[string]string{}},
	// improperly formed listLength
	{map[string]string{"listName": "mylist", "listLength": "AA", "address": "REDIS_HOST", "password": ""}, true, map[string]string{}},
	// address does not resolve
	{map[string]string{"listName": "mylist", "listLength": "0", "address": "REDIS_WRONG", "password": ""}, true, map[string]string{}},
	// password is defined in the authParams
	{map[string]string{"listName": "mylist", "listLength": "0", "address": "REDIS_WRONG"}, true, map[string]string{"password": ""}},
}

var redisMetricIdentifiers = []redisMetricIdentifier{
	{&testRedisMetadata[1], "redis-mylist"},
}

func TestRedisParseMetadata(t *testing.T) {
	for _, testData := range testRedisMetadata {
		_, err := parseRedisMetadata(testData.metadata, testRedisResolvedEnv, testData.authParams)
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

func TestRedisGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range redisMetricIdentifiers {
		meta, err := parseRedisMetadata(testData.metadataTestData.metadata, testRedisResolvedEnv, testData.metadataTestData.authParams)
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockRedisScaler := redisScaler{meta}

		metricSpec := mockRedisScaler.GetMetricSpecForScaling()
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}
