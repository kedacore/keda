package scalers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"

	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
)

var testInfluxDBResolvedEnv = map[string]string{
	"INFLUX_ORG":   "influx_org",
	"INFLUX_TOKEN": "myToken",
}

type parseInfluxDBMetadataTestData struct {
	metadata   map[string]string
	isError    bool
	authParams map[string]string
}

type influxDBMetricIdentifier struct {
	metadataTestData *parseInfluxDBMetadataTestData
	triggerIndex     int
	name             string
}

var testInfluxDBMetadata = []parseInfluxDBMetadataTestData{
	// 1 nothing passed
	{map[string]string{}, true, map[string]string{}},
	// 2 everything is passed in verbatim
	{map[string]string{"serverURL": "https://influxdata.com", "metricName": "influx_metric", "organizationName": "influx_org", "query": "from(bucket: hello)", "thresholdValue": "10", "authToken": "myToken", "unsafeSsl": "false"}, false, map[string]string{}},
	// 3 everything is passed in (environment variables)
	{map[string]string{"serverURL": "https://influxdata.com", "organizationNameFromEnv": "INFLUX_ORG", "query": "from(bucket: hello)", "thresholdValue": "10", "authTokenFromEnv": "INFLUX_TOKEN", "unsafeSsl": "false"}, false, map[string]string{}},
	// 4 no serverURL passed
	{map[string]string{"metricName": "influx_metric", "organizationName": "influx_org", "query": "from(bucket: hello)", "thresholdValue": "10", "authToken": "myToken", "unsafeSsl": "false"}, true, map[string]string{}},
	// 5 no organization name passed
	{map[string]string{"serverURL": "https://influxdata.com", "metricName": "influx_metric", "query": "from(bucket: hello)", "thresholdValue": "10", "authToken": "myToken", "unsafeSsl": "false"}, true, map[string]string{}},
	// 6 no query passed
	{map[string]string{"serverURL": "https://influxdata.com", "organizationName": "influx_org", "thresholdValue": "10", "authToken": "myToken", "unsafeSsl": "false"}, true, map[string]string{}},
	// 7 no threshold value passed
	{map[string]string{"serverURL": "https://influxdata.com", "organizationName": "influx_org", "query": "from(bucket: hello)", "authToken": "myToken", "unsafeSsl": "false"}, true, map[string]string{}},
	// 8 no auth token passed
	{map[string]string{"serverURL": "https://influxdata.com", "organizationName": "influx_org", "query": "from(bucket: hello)", "thresholdValue": "10", "unsafeSsl": "false"}, true, map[string]string{}},
	// 9 authToken, organizationName, and serverURL are defined in authParams
	{map[string]string{"query": "from(bucket: hello)", "thresholdValue": "10", "unsafeSsl": "false"}, false, map[string]string{"serverURL": "https://influxdata.com", "organizationName": "influx_org", "authToken": "myToken"}},
	// 10 no unsafeSsl value passed
	{map[string]string{"serverURL": "https://influxdata.com", "metricName": "influx_metric", "organizationName": "influx_org", "query": "from(bucket: hello)", "thresholdValue": "10", "authToken": "myToken"}, false, map[string]string{}},
	// 11 wrong activationThreshold valuequeryInfluxDB
	{map[string]string{"serverURL": "https://influxdata.com", "metricName": "influx_metric", "organizationName": "influx_org", "query": "from(bucket: hello)", "thresholdValue": "10", "activationThresholdValue": "aa", "authToken": "myToken", "unsafeSsl": "false"}, true, map[string]string{}},
}

var influxDBMetricIdentifiers = []influxDBMetricIdentifier{
	{&testInfluxDBMetadata[1], 0, "s0-influxdb-influx_org"},
	{&testInfluxDBMetadata[2], 1, "s1-influxdb-influx_org"},
}

func TestInfluxDBParseMetadata(t *testing.T) {
	testCaseNum := 1
	for _, testData := range testInfluxDBMetadata {
		_, err := parseInfluxDBMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, ResolvedEnv: testInfluxDBResolvedEnv, AuthParams: testData.authParams})
		if err != nil && !testData.isError {
			t.Errorf("Expected success but got error for unit test # %v", testCaseNum)
		}
		if testData.isError && err == nil {
			t.Errorf("Expected error but got success for unit test # %v", testCaseNum)
		}
		testCaseNum++
	}
}

func TestInfluxDBGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range influxDBMetricIdentifiers {
		meta, err := parseInfluxDBMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, ResolvedEnv: testInfluxDBResolvedEnv, TriggerIndex: testData.triggerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockInfluxDBScaler := influxDBScaler{influxdb2.NewClient("https://influxdata.com", "myToken"), "", meta, logr.Discard()}

		metricSpec := mockInfluxDBScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Errorf("Wrong External metric source name: %s, expected: %s", metricName, testData.name)
		}
	}
}
