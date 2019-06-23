package scalers

import (
	"testing"
)

var testRedisResolvedEnv = map[string]string{
	"REDIS_HOST":     "none",
	"REDIS_PASSWORD": "none",
}

type parseRedisMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

var testRedisMetadata = []parseRedisMetadataTestData{
	// nothing passed
	{map[string]string{}, true},
	// properly formed listName
	{map[string]string{"listName": "mylist", "listLength": "10", "address": "REDIS_HOST", "password": "REDIS_PASSWORD"}, false},
	// properly formed listName, empty address
	{map[string]string{"listName": "mylist", "listLength": "10", "address": "", "password": ""}, true},
	// improperly formed listLength
	{map[string]string{"listName": "mylist", "listLength": "AA", "address": "REDIS_HOST", "password": ""}, true},
	// address does not resolve
	{map[string]string{"listName": "mylist", "listLength": "0", "address": "REDIS_WRONG", "password": ""}, true},
}

func TestRedisParseMetadata(t *testing.T) {
	for _, testData := range testRedisMetadata {
		_, err := parseRedisMetadata(testData.metadata, testRedisResolvedEnv)
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}
