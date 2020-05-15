package scalers

import (
	"testing"
)

type parseCronMetadataTestData struct {
	metadata   map[string]string
	isError    bool
}

// A complete valid metadata example for reference
var validCronMetadata = map[string]string{
	"timezone"       : "America/New_York",
	"start"          : "0 0/30 * * *",
	"end"            : "0 15/30 * * *",
	"metricName"     : "replicacount",
	"desiredReplicas": "10",
}

var testCronMetadata = []parseCronMetadataTestData{
	{map[string]string{}, true},
	{validCronMetadata, false},
	{map[string]string{"timezone": "America/New_York", "start": "0 0/30 * * *", "end": "0 15/30 * * *", "metricName": "", "desiredReplicas": "10"}, true},
	{map[string]string{"start": "0 0/30 * * *", "end": "0 15/30 * * *", "metricName": "replicacount", "desiredReplicas": "10"}, true},
}

func TestCronParseMetadata(t *testing.T) {
	for _, testData := range testCronMetadata {
		_, err := parseCronMetadata(testData.metadata, map[string]string{})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}
