package scalers

import "testing"

type parseCronMetadataTestData struct {
	metadata   map[string]string
	isError    bool
}

// A complete valid metadata example for reference
var validCronMetadata = map[string]string{
	"startTime"      : "1589435100",
	"endTime"        : "1589435700",
	"metricName"     : "replicacount",
	"desiredReplicas": "10",
}

var testCronMetadata = []parseCronMetadataTestData{
	{map[string]string{}, true},
	{validCronMetadata, false},
	{map[string]string{"startTime": "1589435100", "endTime": "1589435700", "metricName": "", "desiredReplicas": "10"}, true},
	{map[string]string{"startTime": "1589435400", "endTime": "1589435100", "metricName": "", "desiredReplicas": "10"}, true},
}

func TestCronParseMetadata(t *testing.T) {
	_, errNoDepl := parseCronMetadata("", "", validCronMetadata, map[string]string{})
	if errNoDepl == nil {
		t.Error("Expected success but got error", errNoDepl)
	}
	for _, testData := range testCronMetadata {
		_, err := parseCronMetadata("nginx-deployment-basic", "keda", testData.metadata, map[string]string{})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}
