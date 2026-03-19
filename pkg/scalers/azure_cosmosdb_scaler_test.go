package scalers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	v2 "k8s.io/api/autoscaling/v2"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

var testCosmosDBResolvedEnv = map[string]string{
	"COSMOS_CONNECTION": "AccountEndpoint=https://test.documents.azure.com:443/;AccountKey=dGVzdGtleQ==",
}

type parseCosmosDBMetadataTestData struct {
	name        string
	metadata    map[string]string
	isError     bool
	resolvedEnv map[string]string
	authParams  map[string]string
	podIdentity kedav1alpha1.PodIdentityProvider
}

type cosmosDBMetricIdentifier struct {
	name             string
	metadataTestData *parseCosmosDBMetadataTestData
	triggerIndex     int
	metricName       string
}

var testCosmosDBMetadata = []parseCosmosDBMetadataTestData{
	{
		name:        "nothing passed",
		metadata:    map[string]string{},
		isError:     true,
		resolvedEnv: testCosmosDBResolvedEnv,
		authParams:  map[string]string{},
		podIdentity: "",
	},
	{
		name: "properly formed with connection string",
		metadata: map[string]string{
			"connectionFromEnv": "COSMOS_CONNECTION",
			"databaseId":        "testdb",
			"containerId":       "testcontainer",
			"leaseDatabaseId":   "testdb",
			"leaseContainerId":  "leases",
			"processorName":     "testprocessor",
		},
		isError:     false,
		resolvedEnv: testCosmosDBResolvedEnv,
		authParams:  map[string]string{},
		podIdentity: "",
	},
	{
		name: "missing database id",
		metadata: map[string]string{
			"connectionFromEnv": "COSMOS_CONNECTION",
			"containerId":       "testcontainer",
			"leaseDatabaseId":   "testdb",
			"leaseContainerId":  "leases",
			"processorName":     "testprocessor",
		},
		isError:     true,
		resolvedEnv: testCosmosDBResolvedEnv,
		authParams:  map[string]string{},
		podIdentity: "",
	},
	{
		name: "missing container id",
		metadata: map[string]string{
			"connectionFromEnv": "COSMOS_CONNECTION",
			"databaseId":        "testdb",
			"leaseDatabaseId":   "testdb",
			"leaseContainerId":  "leases",
			"processorName":     "testprocessor",
		},
		isError:     true,
		resolvedEnv: testCosmosDBResolvedEnv,
		authParams:  map[string]string{},
		podIdentity: "",
	},
	{
		name: "missing lease database id",
		metadata: map[string]string{
			"connectionFromEnv": "COSMOS_CONNECTION",
			"databaseId":        "testdb",
			"containerId":       "testcontainer",
			"leaseContainerId":  "leases",
			"processorName":     "testprocessor",
		},
		isError:     true,
		resolvedEnv: testCosmosDBResolvedEnv,
		authParams:  map[string]string{},
		podIdentity: "",
	},
	{
		name: "missing lease container id",
		metadata: map[string]string{
			"connectionFromEnv": "COSMOS_CONNECTION",
			"databaseId":        "testdb",
			"containerId":       "testcontainer",
			"leaseDatabaseId":   "testdb",
			"processorName":     "testprocessor",
		},
		isError:     true,
		resolvedEnv: testCosmosDBResolvedEnv,
		authParams:  map[string]string{},
		podIdentity: "",
	},
	{
		name: "missing processor name",
		metadata: map[string]string{
			"connectionFromEnv": "COSMOS_CONNECTION",
			"databaseId":        "testdb",
			"containerId":       "testcontainer",
			"leaseDatabaseId":   "testdb",
			"leaseContainerId":  "leases",
		},
		isError:     true,
		resolvedEnv: testCosmosDBResolvedEnv,
		authParams:  map[string]string{},
		podIdentity: "",
	},
	{
		name: "missing connection and key",
		metadata: map[string]string{
			"databaseId":       "testdb",
			"containerId":      "testcontainer",
			"leaseDatabaseId":  "testdb",
			"leaseContainerId": "leases",
			"processorName":    "testprocessor",
		},
		isError:     true,
		resolvedEnv: map[string]string{},
		authParams:  map[string]string{},
		podIdentity: "",
	},
	{
		name: "connection from authParams",
		metadata: map[string]string{
			"databaseId":       "testdb",
			"containerId":      "testcontainer",
			"leaseDatabaseId":  "testdb",
			"leaseContainerId": "leases",
			"processorName":    "testprocessor",
		},
		isError:     false,
		resolvedEnv: map[string]string{},
		authParams: map[string]string{
			"connection": "AccountEndpoint=https://test.documents.azure.com:443/;AccountKey=dGVzdGtleQ==",
		},
		podIdentity: kedav1alpha1.PodIdentityProviderNone,
	},
	{
		name: "endpoint with key",
		metadata: map[string]string{
			"endpoint":         "https://test.documents.azure.com:443/",
			"databaseId":       "testdb",
			"containerId":      "testcontainer",
			"leaseDatabaseId":  "testdb",
			"leaseContainerId": "leases",
			"processorName":    "testprocessor",
		},
		isError:     false,
		resolvedEnv: map[string]string{},
		authParams: map[string]string{
			"cosmosDBKey": "dGVzdGtleQ==",
		},
		podIdentity: "",
	},
	{
		name: "podIdentity azure-workload with endpoint",
		metadata: map[string]string{
			"endpoint":         "https://test.documents.azure.com:443/",
			"databaseId":       "testdb",
			"containerId":      "testcontainer",
			"leaseDatabaseId":  "testdb",
			"leaseContainerId": "leases",
			"processorName":    "testprocessor",
		},
		isError:     false,
		resolvedEnv: map[string]string{},
		authParams: map[string]string{
			"cosmosDBKey": "dGVzdGtleQ==",
		},
		podIdentity: kedav1alpha1.PodIdentityProviderAzureWorkload,
	},
	{
		name: "podIdentity azure-workload without endpoint or connection",
		metadata: map[string]string{
			"databaseId":       "testdb",
			"containerId":      "testcontainer",
			"leaseDatabaseId":  "testdb",
			"leaseContainerId": "leases",
			"processorName":    "testprocessor",
		},
		isError:     true,
		resolvedEnv: map[string]string{},
		authParams:  map[string]string{},
		podIdentity: kedav1alpha1.PodIdentityProviderAzureWorkload,
	},
	{
		name: "separate lease connection",
		metadata: map[string]string{
			"connectionFromEnv":      "COSMOS_CONNECTION",
			"leaseConnectionFromEnv": "COSMOS_CONNECTION",
			"databaseId":             "testdb",
			"containerId":            "testcontainer",
			"leaseDatabaseId":        "testdb",
			"leaseContainerId":       "leases",
			"processorName":          "testprocessor",
		},
		isError:     false,
		resolvedEnv: testCosmosDBResolvedEnv,
		authParams:  map[string]string{},
		podIdentity: "",
	},
	{
		name: "invalid changeFeedLagThreshold",
		metadata: map[string]string{
			"connectionFromEnv":      "COSMOS_CONNECTION",
			"databaseId":             "testdb",
			"containerId":            "testcontainer",
			"leaseDatabaseId":        "testdb",
			"leaseContainerId":       "leases",
			"processorName":          "testprocessor",
			"changeFeedLagThreshold": "invalid",
		},
		isError:     true,
		resolvedEnv: testCosmosDBResolvedEnv,
		authParams:  map[string]string{},
		podIdentity: "",
	},
	{
		name: "invalid activationChangeFeedLagThreshold",
		metadata: map[string]string{
			"connectionFromEnv":                "COSMOS_CONNECTION",
			"databaseId":                       "testdb",
			"containerId":                      "testcontainer",
			"leaseDatabaseId":                  "testdb",
			"leaseContainerId":                 "leases",
			"processorName":                    "testprocessor",
			"activationChangeFeedLagThreshold": "invalid",
		},
		isError:     true,
		resolvedEnv: testCosmosDBResolvedEnv,
		authParams:  map[string]string{},
		podIdentity: "",
	},
}

var cosmosDBMetricIdentifiers = []cosmosDBMetricIdentifier{
	{
		name:             "properly formed metric",
		metadataTestData: &testCosmosDBMetadata[1],
		triggerIndex:     0,
		metricName:       "s0-azure-cosmosdb-leases-testprocessor",
	},
	{
		name:             "endpoint with key metric",
		metadataTestData: &testCosmosDBMetadata[9],
		triggerIndex:     1,
		metricName:       "s1-azure-cosmosdb-leases-testprocessor",
	},
}

func TestCosmosDBParseMetadata(t *testing.T) {
	for _, testData := range testCosmosDBMetadata {
		t.Run(testData.name, func(t *testing.T) {
			config := &scalersconfig.ScalerConfig{
				TriggerMetadata: testData.metadata,
				ResolvedEnv:     testData.resolvedEnv,
				AuthParams:      testData.authParams,
				PodIdentity:     kedav1alpha1.AuthPodIdentity{Provider: testData.podIdentity},
			}

			_, err := parseAzureCosmosDBMetadata(config)
			if err != nil && !testData.isError {
				t.Errorf("Expected success but got error: %v", err)
			}
			if testData.isError && err == nil {
				t.Errorf("Expected error but got success. testData: %v", testData)
			}
		})
	}
}

func TestCosmosDBGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range cosmosDBMetricIdentifiers {
		t.Run(testData.name, func(t *testing.T) {
			config := &scalersconfig.ScalerConfig{
				TriggerMetadata: testData.metadataTestData.metadata,
				ResolvedEnv:     testData.metadataTestData.resolvedEnv,
				AuthParams:      testData.metadataTestData.authParams,
				PodIdentity:     kedav1alpha1.AuthPodIdentity{Provider: testData.metadataTestData.podIdentity},
				TriggerIndex:    testData.triggerIndex,
			}

			meta, err := parseAzureCosmosDBMetadata(config)
			if err != nil {
				t.Fatal("Could not parse metadata:", err)
			}

			mockScaler := azureCosmosDBScaler{
				metadata:   meta,
				logger:     logr.Discard(),
				metricType: v2.AverageValueMetricType,
			}

			metricSpec := mockScaler.GetMetricSpecForScaling(context.Background())
			metricName := metricSpec[0].External.Metric.Name
			assert.Equal(t, testData.metricName, metricName)
		})
	}
}

func TestCosmosDBConnectionStringParsing(t *testing.T) {
	testCases := []struct {
		name             string
		connectionStr    string
		expectError      bool
		expectedEndpoint string
	}{
		{
			name:             "valid connection string",
			connectionStr:    "AccountEndpoint=https://test.documents.azure.com:443/;AccountKey=dGVzdGtleQ==",
			expectError:      false,
			expectedEndpoint: "https://test.documents.azure.com:443/",
		},
		{
			name:          "missing endpoint",
			connectionStr: "AccountKey=dGVzdGtleQ==",
			expectError:   true,
		},
		{
			name:          "missing key",
			connectionStr: "AccountEndpoint=https://test.documents.azure.com:443/",
			expectError:   true,
		},
		{
			name:          "empty string",
			connectionStr: "",
			expectError:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			endpoint, key, err := parseCosmosDBConnectionString(tc.connectionStr)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedEndpoint, endpoint)
				assert.NotEmpty(t, key)
			}
		})
	}
}

func TestExtractLSNFromSessionToken(t *testing.T) {
	testCases := []struct {
		name        string
		token       string
		expectedLSN string
	}{
		{
			name:        "simple format",
			token:       "0:123",
			expectedLSN: "123",
		},
		{
			name:        "compound format with global LSN",
			token:       "0:1#100#2",
			expectedLSN: "100",
		},
		{
			name:        "two segments",
			token:       "5:42#999",
			expectedLSN: "999",
		},
		{
			name:        "empty token",
			token:       "",
			expectedLSN: "",
		},
		{
			name:        "no colon",
			token:       "justanumber",
			expectedLSN: "justanumber",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			lsn := extractLSNFromSessionToken(tc.token)
			assert.Equal(t, tc.expectedLSN, lsn)
		})
	}
}

func TestExtractItemLSN(t *testing.T) {
	testCases := []struct {
		name        string
		item        string
		expectedLSN int64
		expectError bool
	}{
		{
			name:        "numeric LSN",
			item:        `{"_lsn": 1234}`,
			expectedLSN: 1234,
		},
		{
			name:        "string LSN",
			item:        `{"_lsn": "5678"}`,
			expectedLSN: 5678,
		},
		{
			name:        "missing LSN",
			item:        `{"id": "doc1"}`,
			expectedLSN: 0,
			expectError: true,
		},
		{
			name:        "invalid JSON",
			item:        `not json`,
			expectedLSN: -1,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			lsn, err := extractItemLSN(json.RawMessage(tc.item))
			if tc.expectError {
				assert.True(t, err != nil || lsn <= 0)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedLSN, lsn)
			}
		})
	}
}

func TestCosmosDBAuthTokenGeneration(t *testing.T) {
	token := generateCosmosDBAuthToken("get", "docs", "dbs/testdb/colls/testcol", "thu, 01 jan 2024 00:00:00 gmt", "dGVzdGtleQ==")
	assert.Contains(t, token, "type%3Dmaster%26ver%3D1.0%26sig%3D")
}

func TestCosmosDBLeaseParsingDotNetFormat(t *testing.T) {
	// Realistic .NET SDK lease documents have: version=0, FeedRange, Mode, properties fields.
	// The scaler must parse LeaseToken and ContinuationToken and ignore the extra fields.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dbs/testdb/colls/leases/docs":
			// Return raw JSON matching actual .NET SDK lease format
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Documents":[
				{
					"id": "host1.documents.azure.com_abc==_def=..6",
					"version": 0,
					"_etag": "\"08000b63-0000-0800-0000-69a8c6640000\"",
					"LeaseToken": "6",
					"FeedRange": {"Range": {"min": "36DB6DB6DB6DB6DB6DB6DB6DB6DB6DB6", "max": "FF"}},
					"Owner": "dotnet-host1",
					"ContinuationToken": "\"511\"",
					"properties": {},
					"timestamp": "2026-03-04T23:55:16.5233511Z",
					"Mode": "Incremental Feed",
					"_rid": "abc123",
					"_self": "dbs/abc/colls/def/docs/ghi",
					"_ts": 1772668516
				},
				{
					"id": "host1.documents.azure.com_abc==_def=..3",
					"version": 0,
					"LeaseToken": "3",
					"FeedRange": {"Range": {"min": "0", "max": "36DB6DB6DB6DB6DB6DB6DB6DB6DB6DB6"}},
					"Owner": "dotnet-host1",
					"ContinuationToken": "\"248\"",
					"properties": {},
					"Mode": "Incremental Feed"
				},
				{
					"id": ".metadata.lease",
					"version": 0,
					"Owner": "",
					"properties": {}
				}
			]}`))
		case "/dbs/testdb/colls/data/docs":
			pkRangeID := r.Header.Get("x-ms-documentdb-partitionkeyrangeid")
			switch pkRangeID {
			case "6":
				// Partition 6 has lag: sessionLSN=600, itemLSN=512, lag=89
				w.Header().Set("x-ms-session-token", "6:0#600")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"Documents":[{"id":"doc1","_lsn":512}]}`))
			case "3":
				// Partition 3 is caught up
				w.Header().Set("x-ms-session-token", "3:0#248")
				w.WriteHeader(http.StatusNotModified)
			}
		}
	}))
	defer server.Close()

	client := &cosmosDBClient{
		httpClient:       &http.Client{},
		dataEndpoint:     server.URL,
		dataKey:          "dGVzdGtleQ==",
		leaseEndpoint:    server.URL,
		leaseKey:         "dGVzdGtleQ==",
		databaseID:       "testdb",
		containerID:      "data",
		leaseDatabaseID:  "testdb",
		leaseContainerID: "leases",
	}

	totalLag, _, err := client.estimateLag(context.Background())
	assert.NoError(t, err)
	// Only partition 6 has lag; partition 3 is caught up; metadata doc is filtered
	assert.Equal(t, int64(89), totalLag)
}

func TestCosmosDBLeaseParsingJavaFormat(t *testing.T) {
	// Realistic Java SDK lease documents: no version field, no FeedRange/Mode/properties.
	// The scaler must parse these identically.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dbs/testdb/colls/leases/docs":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Documents":[
				{
					"id": "myhost.documents.azure.com_changefeed-estimator_data..2",
					"_etag": "\"0100baf0-0000-0800-0000-69a8c5560000\"",
					"LeaseToken": "2",
					"ContinuationToken": "\"248\"",
					"timestamp": "2026-03-04T23:50:46.219570110Z",
					"Owner": "java-host1",
					"_rid": "5jBSAKD6NqgELTEBAAAAAA==",
					"_ts": 1772668246
				},
				{
					"id": "myhost.documents.azure.com_changefeed-estimator_data..5",
					"LeaseToken": "5",
					"ContinuationToken": "\"100\"",
					"Owner": "java-host2"
				},
				{
					"id": ".lock",
					"_etag": "\"abc\"",
					"Owner": ""
				}
			]}`))
		case "/dbs/testdb/colls/data/docs":
			pkRangeID := r.Header.Get("x-ms-documentdb-partitionkeyrangeid")
			switch pkRangeID {
			case "2":
				// Partition 2 has lag: sessionLSN=400, itemLSN=249, lag=152
				w.Header().Set("x-ms-session-token", "2:0#400")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"Documents":[{"id":"doc1","_lsn":249}]}`))
			case "5":
				// Partition 5 also has lag: sessionLSN=200, itemLSN=101, lag=100
				w.Header().Set("x-ms-session-token", "5:0#200")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"Documents":[{"id":"doc2","_lsn":101}]}`))
			}
		}
	}))
	defer server.Close()

	client := &cosmosDBClient{
		httpClient:       &http.Client{},
		dataEndpoint:     server.URL,
		dataKey:          "dGVzdGtleQ==",
		leaseEndpoint:    server.URL,
		leaseKey:         "dGVzdGtleQ==",
		databaseID:       "testdb",
		containerID:      "data",
		leaseDatabaseID:  "testdb",
		leaseContainerID: "leases",
	}

	totalLag, _, err := client.estimateLag(context.Background())
	assert.NoError(t, err)
	// Both partitions 2 and 5 have lag; lock doc is filtered out
	assert.Equal(t, int64(252), totalLag)
}

func TestCosmosDBLeaseParsingMixedFormats(t *testing.T) {
	// Edge case: lease container might contain docs from both SDKs (e.g. during migration).
	// The scaler should handle this gracefully since it only reads common fields.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dbs/testdb/colls/leases/docs":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Documents":[
				{
					"id": "dotnet-lease",
					"version": 0,
					"LeaseToken": "0",
					"ContinuationToken": "\"500\"",
					"Owner": "dotnet-host",
					"FeedRange": {"Range": {"min": "0", "max": "80"}},
					"Mode": "Incremental Feed"
				},
				{
					"id": "java-lease",
					"LeaseToken": "1",
					"ContinuationToken": "\"300\"",
					"Owner": "java-host"
				}
			]}`))
		case "/dbs/testdb/colls/data/docs":
			// Both partitions have lag
			w.Header().Set("x-ms-session-token", "0:0#700")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Documents":[{"id":"doc1","_lsn":550}]}`))
		}
	}))
	defer server.Close()

	client := &cosmosDBClient{
		httpClient:       &http.Client{},
		dataEndpoint:     server.URL,
		dataKey:          "dGVzdGtleQ==",
		leaseEndpoint:    server.URL,
		leaseKey:         "dGVzdGtleQ==",
		databaseID:       "testdb",
		containerID:      "data",
		leaseDatabaseID:  "testdb",
		leaseContainerID: "leases",
	}

	totalLag, _, err := client.estimateLag(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, int64(302), totalLag)
}

func TestCosmosDBLeaseParsingEPKBasedDotNet(t *testing.T) {
	// .NET SDK EPK-based leases (version=1) use FeedRange with EPK ranges.
	// ContinuationToken is still a quoted LSN for incremental feed mode.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dbs/testdb/colls/leases/docs":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Documents":[
				{
					"id": "host1..epk..0-AA",
					"version": 1,
					"LeaseToken": "0",
					"FeedRange": {"Range": {"min": "", "max": "AA"}},
					"Owner": "dotnet-host1",
					"ContinuationToken": "\"750\"",
					"Mode": "LatestVersion"
				},
				{
					"id": "host1..epk..AA-FF",
					"version": 1,
					"LeaseToken": "1",
					"FeedRange": {"Range": {"min": "AA", "max": "FF"}},
					"Owner": "dotnet-host1",
					"ContinuationToken": "\"320\"",
					"Mode": "LatestVersion"
				}
			]}`))
		case "/dbs/testdb/colls/data/docs":
			pkRangeID := r.Header.Get("x-ms-documentdb-partitionkeyrangeid")
			switch pkRangeID {
			case "0":
				w.Header().Set("x-ms-session-token", "0:0#900")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"Documents":[{"id":"doc1","_lsn":751}]}`))
			case "1":
				w.Header().Set("x-ms-session-token", "1:0#320")
				w.WriteHeader(http.StatusNotModified)
			}
		}
	}))
	defer server.Close()

	client := &cosmosDBClient{
		httpClient:       &http.Client{},
		dataEndpoint:     server.URL,
		dataKey:          "dGVzdGtleQ==",
		leaseEndpoint:    server.URL,
		leaseKey:         "dGVzdGtleQ==",
		databaseID:       "testdb",
		containerID:      "data",
		leaseDatabaseID:  "testdb",
		leaseContainerID: "leases",
	}

	totalLag, _, err := client.estimateLag(context.Background())
	assert.NoError(t, err)
	// Partition 0 has lag (900-751+1=150), partition 1 is caught up
	assert.Equal(t, int64(150), totalLag)
}

func TestCosmosDBLeaseParsingEPKBasedJava(t *testing.T) {
	// Java SDK EPK-based leases (version=1) may use Base64-encoded ContinuationTokens.
	// The scaler passes ContinuationToken as-is to If-None-Match, and Cosmos DB
	// recognizes its own tokens regardless of encoding.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dbs/testdb/colls/leases/docs":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Documents":[
				{
					"id": "java-epk-lease-0",
					"version": 1,
					"LeaseToken": "0",
					"ContinuationToken": "eyJWIjoiMiIsIlJpZCI6ImFiYz0iLCJDb250aW51YXRpb24iOlt7InRva2VuIjoiXCI1MDBcIiIsInJhbmdlIjp7Im1pbiI6IiIsIm1heCI6IkZGIn19XX0=",
					"Owner": "java-host1",
					"feedRange": {"min": "", "max": "FF"}
				},
				{
					"id": "java-epk-lease-1",
					"version": 1,
					"LeaseToken": "1",
					"ContinuationToken": "eyJWIjoiMiIsIlJpZCI6ImRlZj0iLCJDb250aW51YXRpb24iOlt7InRva2VuIjoiXCIyMDBcIiIsInJhbmdlIjp7Im1pbiI6IkZGIiwibWF4IjoiRkZGRiJ9fV19",
					"Owner": "java-host2",
					"feedRange": {"min": "FF", "max": "FFFF"}
				}
			]}`))
		case "/dbs/testdb/colls/data/docs":
			// Simulate Cosmos DB accepting Base64 continuation tokens and returning results
			w.Header().Set("x-ms-session-token", "0:0#600")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Documents":[{"id":"doc1","_lsn":501}]}`))
		}
	}))
	defer server.Close()

	client := &cosmosDBClient{
		httpClient:       &http.Client{},
		dataEndpoint:     server.URL,
		dataKey:          "dGVzdGtleQ==",
		leaseEndpoint:    server.URL,
		leaseKey:         "dGVzdGtleQ==",
		databaseID:       "testdb",
		containerID:      "data",
		leaseDatabaseID:  "testdb",
		leaseContainerID: "leases",
	}

	totalLag, _, err := client.estimateLag(context.Background())
	assert.NoError(t, err)
	// Both partitions have lag; Base64 tokens are passed through to the server
	assert.Equal(t, int64(200), totalLag)
}

func TestCosmosDBLagEstimation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dbs/testdb/colls/leases/docs":
			response := map[string]interface{}{
				"Documents": []map[string]interface{}{
					{
						"id":                "lease1",
						"LeaseToken":        "0",
						"ContinuationToken": `"1000"`,
						"Owner":             "testowner",
					},
					{
						"id":                "lease2",
						"LeaseToken":        "1",
						"ContinuationToken": `"2000"`,
						"Owner":             "testowner",
					},
					{
						// Metadata doc - should be filtered out
						"id":    "metadata",
						"Owner": "metadata",
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		case "/dbs/testdb/colls/testcontainer/docs":
			pkRangeID := r.Header.Get("x-ms-documentdb-partitionkeyrangeid")

			switch pkRangeID {
			case "0":
				// Partition with lag: sessionLSN=1100, itemLSN=1050, lag=51
				w.Header().Set("x-ms-session-token", "0:0#1100")
				w.Header().Set("Content-Type", "application/json")
				response := map[string]interface{}{
					"Documents": []map[string]interface{}{
						{"id": "item1", "_lsn": 1050},
					},
				}
				_ = json.NewEncoder(w).Encode(response)
			default:
				// Partition without lag (304 Not Modified)
				w.Header().Set("x-ms-session-token", "1:0#2000")
				w.WriteHeader(http.StatusNotModified)
			}
		}
	}))
	defer server.Close()

	client := &cosmosDBClient{
		httpClient:       &http.Client{},
		dataEndpoint:     server.URL,
		dataKey:          "dGVzdGtleQ==",
		leaseEndpoint:    server.URL,
		leaseKey:         "dGVzdGtleQ==",
		databaseID:       "testdb",
		containerID:      "testcontainer",
		leaseDatabaseID:  "testdb",
		leaseContainerID: "leases",
	}

	totalLag, _, err := client.estimateLag(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, int64(51), totalLag)
}

func TestCosmosDBLagEstimationEmptyLeases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"Documents": []map[string]interface{}{},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &cosmosDBClient{
		httpClient:       &http.Client{},
		leaseEndpoint:    server.URL,
		leaseKey:         "dGVzdGtleQ==",
		leaseDatabaseID:  "testdb",
		leaseContainerID: "leases",
	}

	totalLag, _, err := client.estimateLag(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, int64(0), totalLag)
}

func TestCosmosDBLagEstimationAllPartitionsLagging(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dbs/testdb/colls/leases/docs":
			response := map[string]interface{}{
				"Documents": []map[string]interface{}{
					{"id": "lease1", "LeaseToken": "0", "ContinuationToken": `"100"`, "Owner": "owner1"},
					{"id": "lease2", "LeaseToken": "1", "ContinuationToken": `"200"`, "Owner": "owner2"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		case "/dbs/testdb/colls/testcontainer/docs":
			// Both partitions have lag
			w.Header().Set("x-ms-session-token", "0:0#500")
			w.Header().Set("Content-Type", "application/json")
			response := map[string]interface{}{
				"Documents": []map[string]interface{}{
					{"id": "item1", "_lsn": 400},
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	client := &cosmosDBClient{
		httpClient:       &http.Client{},
		dataEndpoint:     server.URL,
		dataKey:          "dGVzdGtleQ==",
		leaseEndpoint:    server.URL,
		leaseKey:         "dGVzdGtleQ==",
		databaseID:       "testdb",
		containerID:      "testcontainer",
		leaseDatabaseID:  "testdb",
		leaseContainerID: "leases",
	}

	totalLag, _, err := client.estimateLag(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, int64(202), totalLag)
}

func TestCosmosDBLagEstimationPartitionSplit(t *testing.T) {
	changeFeedCallCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dbs/testdb/colls/leases/docs":
			response := map[string]interface{}{
				"Documents": []map[string]interface{}{
					{"id": "lease1", "LeaseToken": "0", "ContinuationToken": `"100"`, "Owner": "owner1"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		case "/dbs/testdb/colls/testcontainer/docs":
			changeFeedCallCount++
			if changeFeedCallCount <= 1 {
				// First call returns 410 Gone (partition split)
				w.WriteHeader(http.StatusGone)
			} else {
				// Retry returns caught up
				w.Header().Set("x-ms-session-token", "0:0#100")
				w.WriteHeader(http.StatusNotModified)
			}
		}
	}))
	defer server.Close()

	client := &cosmosDBClient{
		httpClient:       &http.Client{},
		dataEndpoint:     server.URL,
		dataKey:          "dGVzdGtleQ==",
		leaseEndpoint:    server.URL,
		leaseKey:         "dGVzdGtleQ==",
		databaseID:       "testdb",
		containerID:      "testcontainer",
		leaseDatabaseID:  "testdb",
		leaseContainerID: "leases",
	}

	totalLag, _, err := client.estimateLag(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, int64(0), totalLag)
	// Should have retried: lease query + change feed (410) + lease query (retry) + change feed (304)
	assert.GreaterOrEqual(t, changeFeedCallCount, 2)
}

func TestCosmosDBGetMetricsAndActivity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dbs/testdb/colls/leases/docs":
			response := map[string]interface{}{
				"Documents": []map[string]interface{}{
					{"id": "lease1", "LeaseToken": "0", "ContinuationToken": `"100"`, "Owner": "owner1"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		case "/dbs/testdb/colls/testcontainer/docs":
			w.Header().Set("x-ms-session-token", "0:0#200")
			w.Header().Set("Content-Type", "application/json")
			response := map[string]interface{}{
				"Documents": []map[string]interface{}{
					{"id": "item1", "_lsn": 150},
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	scaler := &azureCosmosDBScaler{
		metricType: v2.AverageValueMetricType,
		metadata: &azureCosmosDBMetadata{
			DatabaseID:          "testdb",
			ContainerID:         "testcontainer",
			LeaseDatabaseID:     "testdb",
			LeaseContainerID:    "leases",
			ProcessorName:       "testprocessor",
			Threshold:           1,
			ActivationThreshold: 0,
		},
		cosmosClient: &cosmosDBClient{
			httpClient:       &http.Client{},
			dataEndpoint:     server.URL,
			dataKey:          "dGVzdGtleQ==",
			leaseEndpoint:    server.URL,
			leaseKey:         "dGVzdGtleQ==",
			databaseID:       "testdb",
			containerID:      "testcontainer",
			leaseDatabaseID:  "testdb",
			leaseContainerID: "leases",
		},
		logger: logr.Discard(),
	}

	metrics, isActive, err := scaler.GetMetricsAndActivity(context.Background(), "test-metric")
	assert.NoError(t, err)
	assert.True(t, isActive)
	assert.Len(t, metrics, 1)
}

func TestCosmosDBGetMetricsAndActivityNotActive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dbs/testdb/colls/leases/docs":
			response := map[string]interface{}{
				"Documents": []map[string]interface{}{
					{"id": "lease1", "LeaseToken": "0", "ContinuationToken": `"100"`, "Owner": "owner1"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		case "/dbs/testdb/colls/testcontainer/docs":
			// Caught up
			w.Header().Set("x-ms-session-token", "0:0#100")
			w.WriteHeader(http.StatusNotModified)
		}
	}))
	defer server.Close()

	scaler := &azureCosmosDBScaler{
		metricType: v2.AverageValueMetricType,
		metadata: &azureCosmosDBMetadata{
			DatabaseID:          "testdb",
			ContainerID:         "testcontainer",
			LeaseDatabaseID:     "testdb",
			LeaseContainerID:    "leases",
			ProcessorName:       "testprocessor",
			Threshold:           1,
			ActivationThreshold: 0,
		},
		cosmosClient: &cosmosDBClient{
			httpClient:       &http.Client{},
			dataEndpoint:     server.URL,
			dataKey:          "dGVzdGtleQ==",
			leaseEndpoint:    server.URL,
			leaseKey:         "dGVzdGtleQ==",
			databaseID:       "testdb",
			containerID:      "testcontainer",
			leaseDatabaseID:  "testdb",
			leaseContainerID: "leases",
		},
		logger: logr.Discard(),
	}

	metrics, isActive, err := scaler.GetMetricsAndActivity(context.Background(), "test-metric")
	assert.NoError(t, err)
	assert.False(t, isActive)
	assert.Len(t, metrics, 1)
}
