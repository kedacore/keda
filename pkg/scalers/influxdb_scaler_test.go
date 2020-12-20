package scalers

import (
	"testing"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

var testInfluxDBResolvedEnv = map[string]string{
	"INFLUX_ORG":   "influx_org",
	"INFLUX_TOKEN": "myToken",
}

type parseInfluxDBMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

type influxDBMetricIdentifier struct {
	metadataTestData *parseInfluxDBMetadataTestData
	name             string
}

var testInfluxDBMetadata = []parseInfluxDBMetadataTestData{
	// nothing passed
	{map[string]string{}, true},
	// everything is passed in verbatim
	{map[string]string{"serverURL": "https://influxdata.com", "metricName": "influx_metric", "organizationName": "influx_org", "query": "from(bucket: hello)", "thresholdValue": "10", "authToken": "myToken"}, false},
	// everything is passed in (environment variables)
	{map[string]string{"serverURL": "https://influxdata.com", "organizationNameFromEnv": "INFLUX_ORG", "query": "from(bucket: hello)", "thresholdValue": "10", "authTokenFromEnv": "INFLUX_TOKEN"}, false},
	// no serverURL passed
	{map[string]string{"metricName": "influx_metric", "organizationName": "influx_org", "query": "from(bucket: hello)", "thresholdValue": "10", "authToken": "myToken"}, true},
	// no organization name passed
	{map[string]string{"serverURL": "https://influxdata.com", "metricName": "influx_metric", "query": "from(bucket: hello)", "thresholdValue": "10", "authToken": "myToken"}, true},
	// no query passed
	{map[string]string{"serverURL": "https://influxdata.com", "organizationName": "influx_org", "thresholdValue": "10", "authToken": "myToken"}, true},
	// no threshold value passed
	{map[string]string{"serverURL": "https://influxdata.com", "organizationName": "influx_org", "query": "from(bucket: hello)", "authToken": "myToken"}, true},
	// no auth token passed
	{map[string]string{"serverURL": "https://influxdata.com", "organizationName": "influx_org", "query": "from(bucket: hello)", "thresholdValue": "10"}, true}}

var influxDBMetricIdentifiers = []influxDBMetricIdentifier{
	{&testInfluxDBMetadata[1], "influxdb-influx_metric-influx_org"},
	{&testInfluxDBMetadata[2], "influxdb-https---xxx-influx_org"},
}

func TestInfluxDBParseMetadata(t *testing.T) {
	testCaseNum := 1
	for _, testData := range testInfluxDBMetadata {
		_, err := parseInfluxDBMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, ResolvedEnv: testInfluxDBResolvedEnv})
		if err != nil && !testData.isError {
			t.Errorf("Expected success but got error for unit test # %v", testCaseNum)
		}
		if testData.isError && err == nil {
			t.Errorf("Expected error but got success for unit test #%v", testCaseNum)
		}
		testCaseNum++
	}
}

func TestInfluxDBGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range influxDBMetricIdentifiers {
		meta, err := parseInfluxDBMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, ResolvedEnv: testInfluxDBResolvedEnv})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockInfluxDBScaler := influxDBScaler{influxdb2.NewClient("https://influxdata.com", "myToken"), meta}

		metricSpec := mockInfluxDBScaler.GetMetricSpecForScaling()
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}
