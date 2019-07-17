package scalers

import (
	"testing"
)

var testExternalScalerResolvedEnv map[string]string

type parseExternalScalerMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

var testExternalScalerMetadata = []parseExternalScalerMetadataTestData{
	{map[string]string{}, true},
	// all properly formed
	{map[string]string{"serviceURI": "myservice", "test1": "7", "test2": "SAMPLE_CREDS"}, false},
	// missing serviceURI
	{map[string]string{"test1": "1", "test2": "SAMPLE_CREDS"}, true},
}

func TestExternalScalerParseMetadata(t *testing.T) {
	for _, testData := range testExternalScalerMetadata {
		_, err := parseExternalScalerMetadata(testData.metadata, testExternalScalerResolvedEnv)
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}
