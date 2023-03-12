package scalers

import (
	"context"
	"net/http"
	"testing"
)

type parseSolrMetadataTestData struct {
	metadata   map[string]string
	isError    bool
	authParams map[string]string
}

type solrMetricIdentifier struct {
	metadataTestData *parseSolrMetadataTestData
	scalerIndex      int
	name             string
}

var testSolrMetadata = []parseSolrMetadataTestData{
	// nothing passed
	{map[string]string{}, true, map[string]string{}},
	// properly formed metadata
	{map[string]string{"host": "http://192.168.49.2:30217", "core": "mycore1", "query": "*:*", "targetQueryValue": "1", "username": "solr"}, false, map[string]string{"password": "U29sclJvY2tz"}},
	// no query passed
	{map[string]string{"host": "http://192.168.49.2:30217", "core": "mycore1", "targetQueryValue": "1", "username": "solr"}, false, map[string]string{"password": "U29sclJvY2tz"}},
	// no host passed
	{map[string]string{"core": "mycore1", "query": "*:*", "targetQueryValue": "1", "username": "solr"}, true, map[string]string{"password": "U29sclJvY2tz"}},
	// no core passed
	{map[string]string{"host": "http://192.168.49.2:30217", "query": "*:*", "targetQueryValue": "1", "username": "solr"}, true, map[string]string{"password": "U29sclJvY2tz"}},
	// no targetQueryValue passed
	{map[string]string{"host": "http://192.168.49.2:30217", "core": "mycore1", "query": "*:*", "username": "solr"}, true, map[string]string{"password": "U29sclJvY2tz"}},
	// no username passed
	{map[string]string{"host": "http://192.168.49.2:30217", "core": "mycore1", "query": "*:*", "targetQueryValue": "1"}, true, map[string]string{"password": "U29sclJvY2tz"}},
	// no password passed
	{map[string]string{"host": "http://192.168.49.2:30217", "core": "mycore1", "query": "*:*", "targetQueryValue": "1", "username": "solr"}, true, map[string]string{}},
}

var solrMetricIdentifiers = []solrMetricIdentifier{
	{&testSolrMetadata[1], 0, "s0-solr"},
	{&testSolrMetadata[2], 1, "s1-solr"},
}

func TestSolrParseMetadata(t *testing.T) {
	testCaseNum := 1
	for _, testData := range testSolrMetadata {
		_, err := parseSolrMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
		if err != nil && !testData.isError {
			t.Errorf("Expected success but got error for unit test # %v", testCaseNum)
		}
		if testData.isError && err == nil {
			t.Errorf("Expected error but got success for unit test # %v", testCaseNum)
		}
		testCaseNum++
	}
}

func TestSolrGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range solrMetricIdentifiers {
		ctx := context.Background()
		meta, err := parseSolrMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, ScalerIndex: testData.scalerIndex, AuthParams: testData.metadataTestData.authParams})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockSolrScaler := solrScaler{
			metadata:   meta,
			httpClient: http.DefaultClient,
		}

		metricSpec := mockSolrScaler.GetMetricSpecForScaling(ctx)
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Errorf("Wrong External metric source name: %s, expected: %s", metricName, testData.name)
		}
	}
}
