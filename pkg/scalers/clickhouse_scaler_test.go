package scalers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type parseClickHouseMetadataTestData struct {
	metadata    map[string]string
	authParams  map[string]string
	resolvedEnv map[string]string
	raisesError bool
}

var testClickHouseResolvedEnv = map[string]string{
	"CLICKHOUSE_PASSWORD": "test_password",
	"CLICKHOUSE_CONN_STR": "clickhouse://user:pass@localhost:9000/default",
}

var testClickHouseMetadata = []parseClickHouseMetadataTestData{
	// No metadata
	{
		metadata:    map[string]string{},
		authParams:  map[string]string{},
		resolvedEnv: map[string]string{},
		raisesError: true,
	},
	// Missing query
	{
		metadata:    map[string]string{"targetQueryValue": "5"},
		authParams:  map[string]string{},
		resolvedEnv: map[string]string{},
		raisesError: true,
	},
	// Missing targetQueryValue when not AsMetricSource
	{
		metadata:    map[string]string{"query": "SELECT COUNT(*) FROM table"},
		authParams:  map[string]string{},
		resolvedEnv: map[string]string{},
		raisesError: true,
	},
	// Missing connectionString and host
	{
		metadata:    map[string]string{"query": "SELECT COUNT(*) FROM table", "targetQueryValue": "5"},
		authParams:  map[string]string{},
		resolvedEnv: map[string]string{},
		raisesError: true,
	},
	// connectionString from authParams
	{
		metadata:    map[string]string{"query": "SELECT COUNT(*) FROM table", "targetQueryValue": "5"},
		authParams:  map[string]string{"connectionString": "clickhouse://user:pass@localhost:9000/default"},
		resolvedEnv: map[string]string{},
		raisesError: false,
	},
	// Host/Port/Username/Password/Database provided separately
	{
		metadata:    map[string]string{"query": "SELECT COUNT(*) FROM table", "targetQueryValue": "5", "host": "test_host", "port": "8123", "username": "test_user", "password": "test_pass", "database": "test_db"},
		authParams:  map[string]string{},
		resolvedEnv: map[string]string{},
		raisesError: false,
	},
	// Host with default port, database, and username
	{
		metadata:    map[string]string{"query": "SELECT COUNT(*) FROM table", "targetQueryValue": "5", "host": "test_host"},
		authParams:  map[string]string{},
		resolvedEnv: map[string]string{},
		raisesError: false,
	},
	// Host with password from environment
	{
		metadata:    map[string]string{"query": "SELECT COUNT(*) FROM table", "targetQueryValue": "5", "host": "test_host", "password": "CLICKHOUSE_PASSWORD"},
		authParams:  map[string]string{},
		resolvedEnv: testClickHouseResolvedEnv,
		raisesError: false,
	},
	// Params from trigger authentication
	{
		metadata:    map[string]string{"query": "SELECT COUNT(*) FROM table", "targetQueryValue": "5"},
		authParams:  map[string]string{"host": "test_host", "port": "9000", "username": "test_user", "password": "test_pass", "database": "test_db"},
		resolvedEnv: map[string]string{},
		raisesError: false,
	},
	// AsMetricSource - targetQueryValue can be 0
	{
		metadata:    map[string]string{"query": "SELECT COUNT(*) FROM table", "targetQueryValue": "0"},
		authParams:  map[string]string{"connectionString": "clickhouse://user:pass@localhost:9000/default"},
		resolvedEnv: map[string]string{},
		raisesError: false,
	},
	// With activationTargetQueryValue
	{
		metadata:    map[string]string{"query": "SELECT COUNT(*) FROM table", "targetQueryValue": "5", "activationTargetQueryValue": "3"},
		authParams:  map[string]string{"connectionString": "clickhouse://user:pass@localhost:9000/default"},
		resolvedEnv: map[string]string{},
		raisesError: false,
	},
}

type clickHouseMetricIdentifier struct {
	metadataTestData *parseClickHouseMetadataTestData
	resolvedEnv      map[string]string
	authParam        map[string]string
	scaleIndex       int
	name             string
}

var clickHouseMetricIdentifiers = []clickHouseMetricIdentifier{
	{&testClickHouseMetadata[4], map[string]string{}, map[string]string{"connectionString": "clickhouse://user:pass@localhost:9000/default"}, 0, "s0-clickhouse"},
	{&testClickHouseMetadata[5], map[string]string{}, nil, 1, "s1-clickhouse"},
	{&testClickHouseMetadata[6], map[string]string{}, nil, 2, "s2-clickhouse"},
}

func TestParseClickHouseMetadata(t *testing.T) {
	for i, testData := range testClickHouseMetadata {
		config := &scalersconfig.ScalerConfig{
			ResolvedEnv:     testData.resolvedEnv,
			TriggerMetadata: testData.metadata,
			AuthParams:      testData.authParams,
			TriggerIndex:    0,
			AsMetricSource:  false,
		}
		// Special case for AsMetricSource test (index 9)
		if i == 9 {
			config.AsMetricSource = true
		}
		_, err := parseClickHouseMetadata(config)
		if err != nil && !testData.raisesError {
			t.Errorf("Test case %d: Expected success but got error: %v", i, err)
		}
		if err == nil && testData.raisesError {
			t.Errorf("Test case %d: Expected error but got success", i)
		}
	}
}

func TestClickHouseGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range clickHouseMetricIdentifiers {
		config := &scalersconfig.ScalerConfig{
			ResolvedEnv:     testData.resolvedEnv,
			TriggerMetadata: testData.metadataTestData.metadata,
			AuthParams:      testData.authParam,
			TriggerIndex:    testData.scaleIndex,
			AsMetricSource:  false,
		}
		meta, err := parseClickHouseMetadata(config)
		if err != nil {
			t.Fatalf("Could not parse metadata: %v", err)
		}
		// Create a mock scaler without actual database connection
		mockClickHouseScaler := &clickhouseScaler{
			metricType: v2.AverageValueMetricType,
			metadata:   meta,
			connection: nil, // Not needed for metric spec generation
			logger:     logr.Discard(),
		}

		metricSpec := mockClickHouseScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Errorf("Wrong External metric source name: expected '%s', got '%s'", testData.name, metricName)
		}
		// Verify metric spec structure
		if metricSpec[0].Type != externalMetricType {
			t.Errorf("Wrong metric spec type: expected '%s', got '%s'", externalMetricType, metricSpec[0].Type)
		}
		// Verify metric name generation
		expectedMetricName := GenerateMetricNameWithIndex(testData.scaleIndex, kedautil.NormalizeString("clickhouse"))
		if metricName != expectedMetricName {
			t.Errorf("Wrong metric name generation: expected '%s', got '%s'", expectedMetricName, metricName)
		}
	}
}
