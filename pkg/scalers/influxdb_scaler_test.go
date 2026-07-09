package scalers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/InfluxCommunity/influxdb3-go/v2/influxdb3"
	"github.com/go-logr/logr"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	api "github.com/influxdata/influxdb-client-go/v2/api"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
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
	// 2 everything is passed in verbatim (authToken via authParams)
	{map[string]string{"serverURL": "https://influxdata.com", "metricName": "influx_metric", "organizationName": "influx_org", "query": "from(bucket: hello)", "thresholdValue": "10", "unsafeSsl": "false"}, false, map[string]string{"authToken": "myToken"}},
	// 3 everything is passed in (environment variables)
	{map[string]string{"serverURL": "https://influxdata.com", "organizationNameFromEnv": "INFLUX_ORG", "query": "from(bucket: hello)", "thresholdValue": "10", "authTokenFromEnv": "INFLUX_TOKEN", "unsafeSsl": "false"}, false, map[string]string{}},
	// 4 no serverURL passed
	{map[string]string{"metricName": "influx_metric", "organizationName": "influx_org", "query": "from(bucket: hello)", "thresholdValue": "10", "unsafeSsl": "false"}, true, map[string]string{"authToken": "myToken"}},
	// 5 no organization name passed
	{map[string]string{"serverURL": "https://influxdata.com", "metricName": "influx_metric", "query": "from(bucket: hello)", "thresholdValue": "10", "unsafeSsl": "false"}, true, map[string]string{"authToken": "myToken"}},
	// 6 no query passed
	{map[string]string{"serverURL": "https://influxdata.com", "organizationName": "influx_org", "thresholdValue": "10", "unsafeSsl": "false"}, true, map[string]string{"authToken": "myToken"}},
	// 7 no threshold value passed
	{map[string]string{"serverURL": "https://influxdata.com", "organizationName": "influx_org", "query": "from(bucket: hello)", "unsafeSsl": "false"}, true, map[string]string{"authToken": "myToken"}},
	// 8 no auth token passed (optional, for unauthenticated instances)
	{map[string]string{"serverURL": "https://influxdata.com", "organizationName": "influx_org", "query": "from(bucket: hello)", "thresholdValue": "10", "unsafeSsl": "false"}, false, map[string]string{}},
	// 9 authToken, organizationName, and serverURL are defined in authParams
	{map[string]string{"query": "from(bucket: hello)", "thresholdValue": "10", "unsafeSsl": "false"}, false, map[string]string{"serverURL": "https://influxdata.com", "organizationName": "influx_org", "authToken": "myToken"}},
	// 10 no unsafeSsl value passed
	{map[string]string{"serverURL": "https://influxdata.com", "metricName": "influx_metric", "organizationName": "influx_org", "query": "from(bucket: hello)", "thresholdValue": "10"}, false, map[string]string{"authToken": "myToken"}},
	// 11 wrong activationThreshold value
	{map[string]string{"serverURL": "https://influxdata.com", "metricName": "influx_metric", "organizationName": "influx_org", "query": "from(bucket: hello)", "thresholdValue": "10", "activationThresholdValue": "aa", "unsafeSsl": "false"}, true, map[string]string{"authToken": "myToken"}},
	// 12 unsupported influxVersion
	{map[string]string{"serverURL": "https://influxdata.com", "influxVersion": "1", "database": "test", "metricKey": "mymetric", "metricName": "influx_metric", "organizationName": "influx_org", "query": "SELECT \"water_level\" FROM \"h2o_feet\" WHERE \"location\"='coyote_creek' ORDER BY time DESC LIMIT 1;", "thresholdValue": "10", "unsafeSsl": "false"}, true, map[string]string{"authToken": "myToken"}},
	// 13 valid influxVersion but no database
	{map[string]string{"serverURL": "https://influxdata.com", "influxVersion": "3", "metricKey": "mymetric", "metricName": "influx_metric", "organizationName": "influx_org", "query": "SELECT \"water_level\" FROM \"h2o_feet\" WHERE \"location\"='coyote_creek' ORDER BY time DESC LIMIT 1;", "thresholdValue": "10", "unsafeSsl": "false"}, true, map[string]string{"authToken": "myToken"}},
	// 14 influxVersion 3 with all required values
	{map[string]string{"serverURL": "https://influxdata.com", "influxVersion": "3", "database": "test", "metricKey": "mymetric", "metricName": "influx_metric", "organizationName": "influx_org", "query": "SELECT \"water_level\" FROM \"h2o_feet\" WHERE \"location\"='coyote_creek' ORDER BY time DESC LIMIT 1;", "thresholdValue": "10", "unsafeSsl": "false"}, false, map[string]string{"authToken": "myToken"}},
	// 15 influxVersion 3 with queryType InfluxQL
	{map[string]string{"serverURL": "https://influxdata.com", "influxVersion": "3", "database": "test", "metricKey": "mymetric", "queryType": "InfluxQL", "metricName": "influx_metric", "organizationName": "influx_org", "query": "SELECT \"water_level\" FROM \"h2o_feet\" WHERE \"location\"='coyote_creek' ORDER BY time DESC LIMIT 1;", "thresholdValue": "10", "unsafeSsl": "false"}, false, map[string]string{"authToken": "myToken"}},
	// 16 influxVersion 3 with no metricKey
	{map[string]string{"serverURL": "https://influxdata.com", "influxVersion": "3", "database": "test", "queryType": "InfluxQL", "metricName": "influx_metric", "organizationName": "influx_org", "query": "SELECT \"water_level\" FROM \"h2o_feet\" WHERE \"location\"='coyote_creek' ORDER BY time DESC LIMIT 1;", "thresholdValue": "10", "unsafeSsl": "false"}, true, map[string]string{"authToken": "myToken"}},
	// 17 influxVersion 3 with queryType FlightSQL
	{map[string]string{"serverURL": "https://influxdata.com", "influxVersion": "3", "database": "test", "metricKey": "mymetric", "queryType": "FlightSQL", "metricName": "influx_metric", "organizationName": "influx_org", "query": "SELECT \"water_level\" FROM \"h2o_feet\" WHERE \"location\"='coyote_creek' ORDER BY time DESC LIMIT 1;", "thresholdValue": "10", "unsafeSsl": "false"}, false, map[string]string{"authToken": "myToken"}},
	// 18 influxVersion 3 with no organization
	{map[string]string{"serverURL": "https://influxdata.com", "influxVersion": "3", "database": "test", "metricKey": "mymetric", "queryType": "FlightSQL", "metricName": "influx_metric", "query": "SELECT \"water_level\" FROM \"h2o_feet\" WHERE \"location\"='coyote_creek' ORDER BY time DESC LIMIT 1;", "thresholdValue": "10", "unsafeSsl": "false"}, false, map[string]string{"authToken": "myToken"}},
	// 19 deprecated authToken in triggerMetadata errors
	{map[string]string{"serverURL": "https://influxdata.com", "organizationName": "influx_org", "query": "from(bucket: hello)", "thresholdValue": "10", "authToken": "myToken", "unsafeSsl": "false"}, true, map[string]string{}},
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

var influxDBMetricIdentifiers = []influxDBMetricIdentifier{
	{&testInfluxDBMetadata[1], 0, "s0-influxdb-influx_org"},
	{&testInfluxDBMetadata[2], 1, "s1-influxdb-influx_org"},
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

var influxDBV3MetricIdentifiers = []influxDBMetricIdentifier{
	{&testInfluxDBMetadata[13], 0, "s0-influxdb-test"},
	{&testInfluxDBMetadata[14], 1, "s1-influxdb-test"},
}

func TestInfluxDBV3GetMetricSpecForScaling(t *testing.T) {
	for _, testData := range influxDBV3MetricIdentifiers {
		meta, err := parseInfluxDBMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, ResolvedEnv: testInfluxDBResolvedEnv, TriggerIndex: testData.triggerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		client, err := influxdb3.New(influxdb3.ClientConfig{Host: "https://influxdata.com", Token: "myToken"})
		if err != nil {
			t.Fatal("Error connecting to influx v3:", err)
		}
		mockInfluxDBScaler := influxDBScalerV3{client, "", meta, logr.Discard()}

		metricSpec := mockInfluxDBScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Errorf("Wrong External metric source name: %s, expected: %s", metricName, testData.name)
		}
	}
}

// newInfluxDBTestQueryAPI returns a QueryAPI backed by an httptest server that
// serves the given InfluxDB CSV response body for the flux query endpoint.
func newInfluxDBTestQueryAPI(t *testing.T, csvBody string) (api.QueryAPI, func()) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		fmt.Fprint(w, csvBody)
	}))
	client := influxdb2.NewClient(server.URL, "test-token")
	return client.QueryAPI("test-org"), server.Close
}

func TestQueryInfluxDBReturnsValue(t *testing.T) {
	csvBody := "#datatype,string,long,double\r\n" +
		",result,table,_value\r\n" +
		",_result,0,42.5\r\n\r\n"
	queryAPI, closeServer := newInfluxDBTestQueryAPI(t, csvBody)
	defer closeServer()

	value, err := queryInfluxDB(context.Background(), queryAPI, "from(bucket:\"test\")")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if value != 42.5 {
		t.Errorf("expected value 42.5, got: %v", value)
	}
}

// TestQueryInfluxDBReturnsErrorOnParseFailure guards against silently masking a
// query error as "no results found": the InfluxDB client surfaces server-side
// errors through Err() after Next() returns false, so queryInfluxDB must check
// it instead of assuming the result set is simply empty.
func TestQueryInfluxDBReturnsErrorOnParseFailure(t *testing.T) {
	// InfluxDB returns query errors as an error-annotated CSV table.
	csvBody := "#datatype,string,string\r\n" +
		",error,reference\r\n" +
		"invalid query syntax,897\r\n\r\n"
	queryAPI, closeServer := newInfluxDBTestQueryAPI(t, csvBody)
	defer closeServer()

	_, err := queryInfluxDB(context.Background(), queryAPI, "invalid")
	if err == nil {
		t.Fatal("expected an error when the query response cannot be parsed, got nil")
	}
	if strings.Contains(err.Error(), "no results found") {
		t.Errorf("query error was masked as an empty result: %v", err)
	}
}

func TestQueryInfluxDBReturnsErrorOnEmptyResult(t *testing.T) {
	queryAPI, closeServer := newInfluxDBTestQueryAPI(t, "\r\n")
	defer closeServer()

	_, err := queryInfluxDB(context.Background(), queryAPI, "empty")
	if err == nil || !strings.Contains(err.Error(), "no results found") {
		t.Errorf("expected \"no results found\" error for an empty result, got: %v", err)
	}
}
