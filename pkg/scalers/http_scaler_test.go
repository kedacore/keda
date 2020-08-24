package scalers

import (
	"testing"
)

var testHTTPResolvedEnv = map[string]string{}

type parseHTTPMetadataTestData struct {
	metadata   map[string]string
	raisesError bool
}

var testHTTPMetadata = []parseHTTPMetadataTestData{
	// No metadata
	{metadata: map[string]string{}, raisesError: true},
	// OK
	{metadata: map[string]string{"apiURL": "http://dummy:1230/api/v1/", "metricName": "metric", "targetValue": "42"}, raisesError: false},
	// Target not an int
	{metadata: map[string]string{"apiURL": "http://dummy:1230/api/v1/", "metricName": "metric", "targetValue": "aa"}, raisesError: true},
	// Missing metric name
	{metadata: map[string]string{"apiURL": "http://dummy:1230/api/v1/", "targetValue": "aa"}, raisesError: true},
	// Missing apiURL
	{metadata: map[string]string{"metricName": "metric", "targetValue": "aa"}, raisesError: true},
	// Missing targetValue
	{metadata: map[string]string{"apiURL": "http://dummy:1230/api/v1/", "metricName": "metric"}, raisesError: true},
}

func TestParseHTTPMetadata(t *testing.T) {
	for _, testData := range testHTTPMetadata {
		_, err := parseHTTPMetadata(testHTTPResolvedEnv, testData.metadata, map[string]string{})
		if err != nil && !testData.raisesError {
			t.Error("Expected success but got error", err)
		}
		if err == nil && testData.raisesError {
			t.Error("Expected error but got success")
		}
	}
}
