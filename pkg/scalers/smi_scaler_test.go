package scalers

import (
	"testing"
)

type smiMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

var testSmiMetadata = []smiMetadataTestData{
	{map[string]string{"metricName": "p50_response_latency", "metricValue": "100"}, false},
	{map[string]string{"metricName": "p50_response_latency", "metricValue": "100m"}, false},
	{map[string]string{"metricName": "p50_response_latency", "metricValue": "100u"}, false},
	{map[string]string{"metricName": "p50_response_latency", "metricValue": ""}, true},
	{map[string]string{"metricName": "p50_response_latency", "metricValue": "100x"}, true},
	{map[string]string{"metricName": "", "metricValue": "100"}, true},
	{map[string]string{"metricName": "", "metricValue": ""}, true},
}

func TestSmiMetadata(t *testing.T) {
	for i, testData := range testSmiMetadata {
		_, err := parseSmiMetadata(testData.metadata)
		if err != nil && !testData.isError {
			t.Errorf("[test %v] Expected success but got error: %v", i, err)
		}
		if testData.isError && err == nil {
			t.Errorf("[test %v] Expected error but got success", i)
		}
	}
}
