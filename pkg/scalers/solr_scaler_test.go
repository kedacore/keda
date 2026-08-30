package scalers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type parseSolrMetadataTestData struct {
	metadata   map[string]string
	isError    bool
	authParams map[string]string
}

type solrMetricIdentifier struct {
	metadataTestData *parseSolrMetadataTestData
	triggerIndex     int
	name             string
}

var testSolrMetadata = []parseSolrMetadataTestData{
	// nothing passed
	{map[string]string{}, true, map[string]string{}},
	// properly formed metadata
	{map[string]string{"host": "http://192.168.49.2:30217", "collection": "my_core", "query": "*:*", "targetQueryValue": "1"}, false, map[string]string{"username": "test_username", "password": "test_password"}},
	// no query passed
	{map[string]string{"host": "http://192.168.49.2:30217", "collection": "my_core", "targetQueryValue": "1"}, false, map[string]string{"username": "test_username", "password": "test_password"}},
	// no host passed
	{map[string]string{"collection": "my_core", "query": "*:*", "targetQueryValue": "1"}, true, map[string]string{"username": "test_username", "password": "test_password"}},
	// no collection passed
	{map[string]string{"host": "http://192.168.49.2:30217", "query": "*:*", "targetQueryValue": "1"}, true, map[string]string{"username": "test_username", "password": "test_password"}},
	// no targetQueryValue passed
	{map[string]string{"host": "http://192.168.49.2:30217", "collection": "my_core", "query": "*:*"}, true, map[string]string{"username": "test_username", "password": "test_password"}},
	// no username passed
	{map[string]string{"host": "http://192.168.49.2:30217", "collection": "my_core", "query": "*:*", "targetQueryValue": "1"}, true, map[string]string{"password": "test_password"}},
	// no password passed
	{map[string]string{"host": "http://192.168.49.2:30217", "collection": "my_core", "query": "*:*", "targetQueryValue": "1"}, true, map[string]string{"username": "test_username"}},
}

var solrMetricIdentifiers = []solrMetricIdentifier{
	{&testSolrMetadata[1], 0, "s0-solr"},
	{&testSolrMetadata[2], 1, "s1-solr"},
}

func TestSolrParseMetadata(t *testing.T) {
	testCaseNum := 1
	for _, testData := range testSolrMetadata {
		_, err := parseSolrMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
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
		meta, err := parseSolrMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, TriggerIndex: testData.triggerIndex, AuthParams: testData.metadataTestData.authParams})
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

// Solr reports failures as JSON when wt=json, and those bodies unmarshal cleanly into solrResponse
// with numFound left at 0. Without a status check the scaler reported a healthy queue length of 0
// and scaled the workload to zero during an outage or an auth rejection.
func TestSolrGetItemCountErrorsOnNon200(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"server error", http.StatusInternalServerError, `{"responseHeader":{"status":500},"error":{"msg":"no servers hosting shard","code":500}}`},
		{"unauthorized", http.StatusUnauthorized, `{"responseHeader":{"status":401},"error":{"msg":"require authentication","code":401}}`},
		{"collection missing", http.StatusNotFound, `{"responseHeader":{"status":404},"error":{"msg":"no such collection","code":404}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				// nosemgrep: no-direct-write-to-responsewriter
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()

			s, err := NewSolrScaler(&scalersconfig.ScalerConfig{
				TriggerMetadata: map[string]string{
					"host": server.URL, "collection": "my_core", "query": "*:*", "targetQueryValue": "1",
				},
				AuthParams: map[string]string{"username": "u", "password": "p"},
			})
			if err != nil {
				t.Fatalf("failed to create solr scaler: %v", err)
			}

			count, err := s.(*solrScaler).getItemCount(context.Background())
			if err == nil {
				t.Errorf("expected an error for HTTP %d, got count %v and nil error", tc.status, count)
			}
			if count != -1 {
				t.Errorf("expected the sentinel count -1 on error, got %v", count)
			}
		})
	}
}

// A literal JSON null body leaves the *solrResponse nil, which used to panic on dereference.
func TestSolrGetItemCountHandlesNullBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		// nosemgrep: no-direct-write-to-responsewriter
		_, _ = w.Write([]byte("null"))
	}))
	defer server.Close()

	s, err := NewSolrScaler(&scalersconfig.ScalerConfig{
		TriggerMetadata: map[string]string{
			"host": server.URL, "collection": "my_core", "query": "*:*", "targetQueryValue": "1",
		},
		AuthParams: map[string]string{"username": "u", "password": "p"},
	})
	if err != nil {
		t.Fatalf("failed to create solr scaler: %v", err)
	}

	if _, err := s.(*solrScaler).getItemCount(context.Background()); err == nil {
		t.Error("expected an error for a null response body, got nil")
	}
}
