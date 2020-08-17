package scalers

import (
	"fmt"
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
	// address is defined in the authParams
	{map[string]string{"listName": "mylist", "listLength": "0"}, false, map[string]string{"address": "localhost:6379"}},
	// host and port is defined in the authParams
	{map[string]string{"listName": "mylist", "listLength": "0"}, false, map[string]string{"host": "localhost", "port": "6379"}},
	// host only is defined in the authParams
	{map[string]string{"listName": "mylist", "listLength": "0"}, true, map[string]string{"host": "localhost"}}}

func TestRedisParseMetadata(t *testing.T) {
	testCaseNum := 1
	for _, testData := range testRedisMetadata {
		_, err := parseRedisMetadata(testData.metadata, testRedisResolvedEnv, testData.authParams)
		if err != nil && !testData.isError {
			t.Error(fmt.Sprintf("Expected success but got error for unit test # %v", testCaseNum), err)
		}
		if testData.isError && err == nil {
			t.Error(fmt.Sprintf("Expected error but got success for unit test #%v", testCaseNum))
		}
		testCaseNum++
	}
}
