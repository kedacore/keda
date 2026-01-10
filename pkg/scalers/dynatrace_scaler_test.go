package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	"github.com/stretchr/testify/assert"
)

type dynatraceMetadataTestData struct {
	metadata   map[string]string
	authParams map[string]string
	errorCase  bool
}

type dynatraceMetricIdentifier struct {
	metadataTestData *dynatraceMetadataTestData
	triggerIndex     int
	name             string
}

var testDynatraceMetadata = []dynatraceMetadataTestData{
	{map[string]string{}, map[string]string{}, true},
	// all properly formed for metricSelector
	{map[string]string{"threshold": "100", "from": "now-3d", "metricSelector": "MyCustomEvent:filter(eq(\"someProperty\",\"someValue\")):count:splitBy(\"dt.entity.process_group\"):fold"}, map[string]string{"host": "http://dummy:1234", "token": "dummy"}, false},
	// all properly formed for query
	{map[string]string{"threshold": "100", "query": "dql-query"}, map[string]string{"host": "http://dummy:1234", "token": "dummy"}, false},
	// malformed threshold
	{map[string]string{"threshold": "abc", "from": "now-3d", "metricSelector": "MyCustomEvent:filter(eq(\"someProperty\",\"someValue\")):count:splitBy(\"dt.entity.process_group\"):fold"}, map[string]string{"host": "http://dummy:1234", "token": "dummy"}, true},
	// malformed activationThreshold
	{map[string]string{"activationThreshold": "abc", "threshold": "100", "from": "now-3d", "metricSelector": "MyCustomEvent:filter(eq(\"someProperty\",\"someValue\")):count:splitBy(\"dt.entity.process_group\"):fold"}, map[string]string{"host": "http://dummy:1234", "token": "dummy"}, true},
	// missing threshold
	{map[string]string{"metricSelector": "MyCustomEvent:filter(eq(\"someProperty\",\"someValue\")):count:splitBy(\"dt.entity.process_group\"):fold"}, map[string]string{"host": "http://dummy:1234", "token": "dummy"}, true},
	// missing metricsSelector and query
	{map[string]string{"threshold": "100"}, map[string]string{"host": "http://dummy:1234", "token": "dummy"}, true},
	// set metricsSelector and query
	{map[string]string{"threshold": "100", "metricSelector": "selector", "query": "query"}, map[string]string{"host": "http://dummy:1234", "token": "dummy"}, true},
	// set and query and from
	{map[string]string{"threshold": "100", "from": "now-3d", "query": "dql-query"}, map[string]string{"host": "http://dummy:1234", "token": "dummy"}, true},
	// missing token (must come from auth params)
	{map[string]string{"token": "foo", "threshold": "100", "from": "now-3d", "metricSelector": "MyCustomEvent:filter(eq(\"someProperty\",\"someValue\")):count:splitBy(\"dt.entity.process_group\"):fold"}, map[string]string{"host": "http://dummy:1234"}, true},
}

var dynatraceMetricIdentifiers = []dynatraceMetricIdentifier{
	{&testDynatraceMetadata[1], 0, "s0-dynatrace"},
	{&testDynatraceMetadata[1], 1, "s1-dynatrace"},
}

func TestDynatraceParseMetadata(t *testing.T) {
	for _, testData := range testDynatraceMetadata {
		_, err := parseDynatraceMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
		if err != nil && !testData.errorCase {
			fmt.Printf("X: %s", testData.metadata)
			t.Error("Expected success but got error", err)
		}
		if testData.errorCase && err == nil {
			fmt.Printf("X: %s", testData.metadata)
			t.Error("Expected error but got success")
		}
	}
}
func TestDynatraceGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range dynatraceMetricIdentifiers {
		meta, err := parseDynatraceMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.metadataTestData.authParams, TriggerIndex: testData.triggerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockDynatraceScaler := dynatraceScaler{
			metadata:   meta,
			httpClient: nil,
		}

		metricSpec := mockDynatraceScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

func TestDynatraceGetMetricByQuery(t *testing.T) {
	testCases := []struct {
		name                string
		executeResponseFail bool
		pollResponseFail    bool
		pollResponseAfter   int
		metricValue         float64
		isError             bool
	}{
		{
			name:                "value returned successfully on first poll",
			executeResponseFail: false,
			pollResponseFail:    false,
			pollResponseAfter:   0,
			metricValue:         100.1,
			isError:             false,
		},
		{
			name:                "value returned successfully on second poll",
			executeResponseFail: false,
			pollResponseFail:    false,
			pollResponseAfter:   1,
			metricValue:         200.2,
			isError:             false,
		},
		{
			name:                "excute fail",
			executeResponseFail: true,
			isError:             true,
		},
		{
			name:             "pooling fail",
			pollResponseFail: true,
			isError:          true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pollingCount := 0
			var apiStub = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/platform/storage/query/v1/query:execute" {
					if tt.executeResponseFail {
						http.Error(w, "Bad Request", http.StatusBadRequest)
						return
					} else {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusAccepted)
						bytes, err := json.Marshal(dynatraceExecuteQueryResponse{
							State:        "RUNNING",
							RequestToken: "token",
						})
						assert.NoError(t, err)
						_, err = w.Write(bytes)
						assert.NoError(t, err)
					}
				}
				if r.URL.Path == "/platform/storage/query/v1/query:poll" {
					if tt.pollResponseFail {
						http.Error(w, "Bad Request", http.StatusBadRequest)
						return
					} else {
						pollingCount++
						if pollingCount > tt.pollResponseAfter {
							w.Header().Set("Content-Type", "application/json")
							w.WriteHeader(http.StatusOK)
							bytes, err := json.Marshal(dynatraceQueryResponse{
								State: "SUCCEEDED",
								Result: struct {
									Records []struct {
										R float64 "json:\"r\""
									} "json:\"records\""
								}{Records: []struct {
									R float64 "json:\"r\""
								}{{R: tt.metricValue}}},
							})
							assert.NoError(t, err)
							_, err = w.Write(bytes)
							assert.NoError(t, err)
						} else {
							w.Header().Set("Content-Type", "application/json")
							w.WriteHeader(http.StatusOK)
							bytes, err := json.Marshal(dynatraceQueryResponse{
								State: "RUNNING",
							})
							assert.NoError(t, err)
							_, err = w.Write(bytes)
							assert.NoError(t, err)
						}

					}
				}
			}))

			metadata := map[string]string{"threshold": "100", "query": "dql-query", "host": apiStub.URL}
			auth := map[string]string{"token": "123ws"}
			scaler, err := NewDynatraceScaler(&scalersconfig.ScalerConfig{TriggerMetadata: metadata, AuthParams: auth, TriggerIndex: 0, GlobalHTTPTimeout: time.Minute})
			if err != nil {
				t.Fatal("Could not start scaler:", err)
			}

			metric, _, err := scaler.GetMetricsAndActivity(t.Context(), "dummy")
			if tt.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.metricValue, metric[0].Value.AsFloat64Slow())
			}
		})
	}
}
