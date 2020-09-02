package scalers

import (
	"testing"
)

var metricsAPIResolvedEnv = map[string]string{}
var authParams = map[string]string{}

type metricsAPIMetadataTestData struct {
	metadata    map[string]string
	raisesError bool
}

var testMetricsAPIMetadata = []metricsAPIMetadataTestData{
	// No metadata
	{metadata: map[string]string{}, raisesError: true},
	// OK
	{metadata: map[string]string{"url": "http://dummy:1230/api/v1/", "valueLocation": "metric", "targetValue": "42"}, raisesError: false},
	// Target not an int
	{metadata: map[string]string{"url": "http://dummy:1230/api/v1/", "valueLocation": "metric", "targetValue": "aa"}, raisesError: true},
	// Missing metric name
	{metadata: map[string]string{"url": "http://dummy:1230/api/v1/", "targetValue": "aa"}, raisesError: true},
	// Missing url
	{metadata: map[string]string{"valueLocation": "metric", "targetValue": "aa"}, raisesError: true},
	// Missing targetValue
	{metadata: map[string]string{"url": "http://dummy:1230/api/v1/", "valueLocation": "metric"}, raisesError: true},
}

func TestParseMetricsAPIMetadata(t *testing.T) {
	for _, testData := range testMetricsAPIMetadata {
		_, err := metricsAPIMetadata(metricsAPIResolvedEnv, testData.metadata, authParams)
		if err != nil && !testData.raisesError {
			t.Error("Expected success but got error", err)
		}
		if err == nil && testData.raisesError {
			t.Error("Expected error but got success")
		}
	}
}

func TestGetValueFromResponse(t *testing.T) {
	d := []byte(`{"components":[{"id": "82328e93e", "tasks": 32}],"count":2.43}`)
	v, err := GetValueFromResponse(d, "components.0.tasks")
	if err != nil {
		t.Error("Expected success but got error", err)
	}
	if v != 32 {
		t.Errorf("Expected %d got %d", 32, v)
	}

	v, err = GetValueFromResponse(d, "count")
	if err != nil {
		t.Error("Expected success but got error", err)
	}
	if v != 2 {
		t.Errorf("Expected %d got %d", 2, v)
	}

}
