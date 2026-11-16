package scalers

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-logr/logr"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type parseArangoDBMetadataTestData struct {
	metadata    map[string]string
	authParams  map[string]string
	raisesError bool
}

var testArangoDBMetadata = []parseArangoDBMetadataTestData{
	// No metadata
	{
		metadata:    map[string]string{},
		authParams:  map[string]string{},
		raisesError: true,
	},
	// missing query
	{
		metadata:    map[string]string{"endpoints": "https://localhost:8529", "collection": "demo", "queryValue": "12", "dbName": "test"},
		authParams:  map[string]string{},
		raisesError: true,
	},
	// with metric name
	{
		metadata:    map[string]string{"endpoints": "https://localhost:8529", "query": `FOR t IN testCollection FILTER t.cook_time == '3 hours' RETURN t`, "collection": "demo", "queryValue": "12", "dbName": "test"},
		authParams:  map[string]string{},
		raisesError: false,
	},
	// from trigger auth
	{
		metadata:    map[string]string{"endpoints": "https://localhost:8529", "query": `FOR t IN testCollection FILTER t.cook_time == '3 hours' RETURN t`, "collection": "demo", "queryValue": "12"},
		authParams:  map[string]string{"dbName": "test", "username": "sample", "password": "secure"},
		raisesError: false,
	},
	// wrong activationQueryValue
	{
		metadata:    map[string]string{"endpoints": "https://localhost:8529", "query": `FOR t IN testCollection FILTER t.cook_time == '3 hours' RETURN t`, "collection": "demo", "queryValue": "12", "activationQueryValue": "aa", "dbName": "test"},
		authParams:  map[string]string{},
		raisesError: true,
	},
	// with connectionLimit
	{
		metadata:    map[string]string{"endpoints": "https://localhost:8529", "query": `FOR t IN testCollection FILTER t.cook_time == '3 hours' RETURN t`, "collection": "demo", "queryValue": "12", "connectionLimit": "2", "dbName": "test"},
		authParams:  map[string]string{},
		raisesError: false,
	},
	// connectionLimit of zero, which means no limit like an unset value
	{
		metadata:    map[string]string{"endpoints": "https://localhost:8529", "query": `FOR t IN testCollection FILTER t.cook_time == '3 hours' RETURN t`, "collection": "demo", "queryValue": "12", "connectionLimit": "0", "dbName": "test"},
		authParams:  map[string]string{},
		raisesError: false,
	},
	// negative connectionLimit, which net/http would treat as no limit
	{
		metadata:    map[string]string{"endpoints": "https://localhost:8529", "query": `FOR t IN testCollection FILTER t.cook_time == '3 hours' RETURN t`, "collection": "demo", "queryValue": "12", "connectionLimit": "-1", "dbName": "test"},
		authParams:  map[string]string{},
		raisesError: true,
	},
}

type arangoDBAuthMetadataTestData struct {
	metadata    map[string]string
	authParams  map[string]string
	raisesError bool
}

var testArangoDBAuthMetadata = []arangoDBAuthMetadataTestData{
	// success bearer default
	{map[string]string{"endpoints": "https://http://34.162.13.192:8529,https://34.162.13.193:8529", "collection": "demo", "query": "FOR d IN myCollection RETURN d", "queryValue": "1", "dbName": "testdb", "authModes": "bearer"}, map[string]string{"bearerToken": "dummy-token"}, false},
	// fail bearerAuth with no token
	{map[string]string{"endpoints": "https://http://34.162.13.192:8529,https://34.162.13.193:8529", "collection": "demo", "query": "FOR d IN myCollection RETURN d", "queryValue": "1", "dbName": "testdb", "authModes": "bearer"}, map[string]string{}, true},
	// success basicAuth
	{map[string]string{"endpoints": "https://http://34.162.13.192:8529,https://34.162.13.193:8529", "collection": "demo", "query": "FOR d IN myCollection RETURN d", "queryValue": "1", "dbName": "testdb", "authModes": "basic"}, map[string]string{"username": "user", "password": "pass"}, false},
	// fail basicAuth with no username
	{map[string]string{"endpoints": "https://http://34.162.13.192:8529,https://34.162.13.193:8529", "collection": "demo", "query": "FOR d IN myCollection RETURN d", "queryValue": "1", "dbName": "testdb", "authModes": "basic"}, map[string]string{}, true},
	// success basicAuth with no password
	{map[string]string{"endpoints": "https://http://34.162.13.192:8529,https://34.162.13.193:8529", "collection": "demo", "query": "FOR d IN myCollection RETURN d", "queryValue": "1", "dbName": "testdb", "authModes": "basic"}, map[string]string{"username": "user"}, false},
}

type arangoDBMetricIdentifier struct {
	metadataTestData *parseArangoDBMetadataTestData
	triggerIndex     int
	name             string
}

var arangoDBMetricIdentifiers = []arangoDBMetricIdentifier{
	{metadataTestData: &testArangoDBMetadata[2], triggerIndex: 0, name: "s0-arangodb"},
	{metadataTestData: &testArangoDBMetadata[2], triggerIndex: 1, name: "s1-arangodb"},
}

func TestParseArangoDBMetadata(t *testing.T) {
	for idx, testData := range testArangoDBMetadata {
		_, err := parseArangoDBMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
		if err != nil && !testData.raisesError {
			t.Errorf("Test %v: expected success but got error: %s", idx, err)
		}
		if err == nil && testData.raisesError {
			t.Error("Expected error but got success")
		}
	}
}

func TestArangoDBScalerAuthParams(t *testing.T) {
	for _, testData := range testArangoDBAuthMetadata {
		meta, err := parseArangoDBMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams})

		if err != nil && !testData.raisesError {
			t.Error("Expected success but got error", err)
		}
		if testData.raisesError && err == nil {
			t.Error("Expected error but got success")
		}

		if err == nil {
			if meta.ArangoDBAuth.EnabledBasicAuth() && !strings.Contains(testData.metadata["authModes"], "basic") {
				t.Error("wrong auth mode detected")
			}
		}
	}
}

func TestArangoDBGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range arangoDBMetricIdentifiers {
		meta, err := parseArangoDBMetadata(&scalersconfig.ScalerConfig{
			AuthParams:      testData.metadataTestData.authParams,
			TriggerMetadata: testData.metadataTestData.metadata,
			TriggerIndex:    testData.triggerIndex,
		})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockArangoDBScaler := arangoDBScaler{"", meta, nil, logr.Discard()}

		metricSpec := mockArangoDBScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

// TestArangoDBClientConnectionLimit checks that connectionLimit caps the
// connections the scaler's client opens to an ArangoDB endpoint.
func TestArangoDBClientConnectionLimit(t *testing.T) {
	tests := []struct {
		name            string
		connectionLimit string
		wantMin         int
		wantMax         int
	}{
		{name: "limit of one", connectionLimit: "1", wantMin: 1, wantMax: 1},
		{name: "limit of two", connectionLimit: "2", wantMin: 1, wantMax: 2},
		{name: "no limit", connectionLimit: "", wantMin: 2, wantMax: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mu sync.Mutex
			var open, maxOpen int
			server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				// Keep each request busy, so that concurrent requests need their own connection.
				time.Sleep(50 * time.Millisecond)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"server":"arango","version":"3.12.4","license":"community"}`))
			}))
			server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
				mu.Lock()
				defer mu.Unlock()
				switch state {
				case http.StateNew:
					open++
					maxOpen = max(maxOpen, open)
				case http.StateClosed, http.StateHijacked:
					open--
				}
			}
			server.Start()
			defer server.Close()

			metadata := map[string]string{
				"endpoints":  server.URL,
				"dbName":     "test",
				"collection": "demo",
				"query":      "FOR d IN demo RETURN d",
				"queryValue": "1",
			}
			if tt.connectionLimit != "" {
				metadata["connectionLimit"] = tt.connectionLimit
			}
			meta, err := parseArangoDBMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: metadata, AuthParams: map[string]string{}})
			if err != nil {
				t.Fatal("Could not parse metadata:", err)
			}
			client, err := getNewArangoDBClient(meta)
			if err != nil {
				t.Fatal("Could not create client:", err)
			}

			var wg sync.WaitGroup
			for range 5 {
				wg.Go(func() {
					if _, err := client.Version(context.Background()); err != nil {
						t.Error("Version() error:", err)
					}
				})
			}
			wg.Wait()

			mu.Lock()
			defer mu.Unlock()
			if maxOpen < tt.wantMin || maxOpen > tt.wantMax {
				t.Errorf("open connections = %d, want between %d and %d", maxOpen, tt.wantMin, tt.wantMax)
			}
		})
	}
}
