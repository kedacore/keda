package scalers

import (
	"github.com/go-redis/redis"
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
		_, err := parseRedisMetadata(testData.metadata, testRedisResolvedEnv, testData.authParams)
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
		meta, err := parseRedisMetadata(testData.metadataTestData.metadata, testRedisResolvedEnv, testData.metadataTestData.authParams)
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockRedisScaler := redisScaler{meta, &redis.Client{}}

		metricSpec := mockRedisScaler.GetMetricSpecForScaling()
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}
