package scalers

import (
	"testing"
)

var testPubSubResolvedEnv = map[string]string{
	"SAMPLE_CREDS": "{}",
}

type parsePubSubMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

var testPubSubMetadata = []parsePubSubMetadataTestData{
	{map[string]string{}, true},
	// all properly formed
	{map[string]string{"subscriptionName": "mysubscription", "subscriptionSize": "7", "credentials": "SAMPLE_CREDS"}, false},
	// missing subscriptionName
	{map[string]string{"subscriptionName": "", "subscriptionSize": "7", "credentials": "SAMPLE_CREDS"}, true},
	// missing credentials
	{map[string]string{"subscriptionName": "mysubscription", "subscriptionSize": "7", "credentials": ""}, true},
	// incorrect credentials
	{map[string]string{"subscriptionName": "mysubscription", "subscriptionSize": "7", "credentials": "WRONG_CREDS"}, true},
	// malformed subscriptionSize
	{map[string]string{"subscriptionName": "mysubscription", "subscriptionSize": "AA", "credentials": "SAMPLE_CREDS"}, true},
}

func TestPubSubParseMetadata(t *testing.T) {
	for _, testData := range testPubSubMetadata {
		_, err := parsePubSubMetadata(testData.metadata, testPubSubResolvedEnv)
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}
