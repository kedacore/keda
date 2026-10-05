package scalers

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"testing"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

// mockNoRowsDriver is a database/sql driver that returns no rows (sql.ErrNoRows)
type mockNoRowsDriver struct{}

func (d *mockNoRowsDriver) Open(name string) (driver.Conn, error) {
	return &mockNoRowsConn{}, nil
}

type mockNoRowsConn struct{}

func (c *mockNoRowsConn) Prepare(query string) (driver.Stmt, error) {
	return &mockNoRowsStmt{}, nil
}

func (c *mockNoRowsConn) Close() error { return nil }

func (c *mockNoRowsConn) Begin() (driver.Tx, error) {
	return nil, errors.New("not implemented")
}

type mockNoRowsStmt struct{}

func (s *mockNoRowsStmt) Close() error { return nil }

func (s *mockNoRowsStmt) NumInput() int { return -1 }

func (s *mockNoRowsStmt) Exec(args []driver.Value) (driver.Result, error) {
	return nil, errors.New("not implemented")
}

func (s *mockNoRowsStmt) Query(args []driver.Value) (driver.Rows, error) {
	return &mockNoRowsRows{}, nil
}

type mockNoRowsRows struct{}

func (r *mockNoRowsRows) Columns() []string { return []string{"value"} }

func (r *mockNoRowsRows) Close() error { return nil }

func (r *mockNoRowsRows) Next(dest []driver.Value) error {
	return io.EOF
}

// mockResultDriver returns a fixed float64 value on query
type mockResultDriver struct{}

func (d *mockResultDriver) Open(name string) (driver.Conn, error) {
	return &mockResultConn{}, nil
}

type mockResultConn struct{}

func (c *mockResultConn) Prepare(query string) (driver.Stmt, error) {
	return &mockResultStmt{}, nil
}

func (c *mockResultConn) Close() error { return nil }

func (c *mockResultConn) Begin() (driver.Tx, error) {
	return nil, errors.New("not implemented")
}

type mockResultStmt struct{}

func (s *mockResultStmt) Close() error { return nil }

func (s *mockResultStmt) NumInput() int { return -1 }

func (s *mockResultStmt) Exec(args []driver.Value) (driver.Result, error) {
	return nil, errors.New("not implemented")
}

func (s *mockResultStmt) Query(args []driver.Value) (driver.Rows, error) {
	return &mockResultRows{}, nil
}

type mockResultRows struct {
	called bool
}

func (r *mockResultRows) Columns() []string { return []string{"value"} }

func (r *mockResultRows) Close() error { return nil }

func (r *mockResultRows) Next(dest []driver.Value) error {
	if r.called {
		return io.EOF
	}
	r.called = true
	dest[0] = float64(42)
	return nil
}

func init() {
	sql.Register("clickhouse-mock-norows", &mockNoRowsDriver{})
	sql.Register("clickhouse-mock-result", &mockResultDriver{})
}

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
		metadata:    map[string]string{"query": "SELECT COUNT(*) FROM table", "targetQueryValue": "5", "host": "test_host", "port": "9000", "username": "test_user", "password": "test_pass", "database": "test_db"},
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
	// TLS enabled with cert and key from authParams
	{
		metadata:    map[string]string{"query": "SELECT COUNT(*) FROM table", "targetQueryValue": "5"},
		authParams:  map[string]string{"connectionString": "clickhouse://user:pass@localhost:9000/default", "tls": "true", "cert": "test-cert", "key": "test-key"},
		resolvedEnv: map[string]string{},
		raisesError: false,
	},
	// TLS enabled with cert but no key
	{
		metadata:    map[string]string{"query": "SELECT COUNT(*) FROM table", "targetQueryValue": "5"},
		authParams:  map[string]string{"connectionString": "clickhouse://user:pass@localhost:9000/default", "tls": "true", "cert": "test-cert"},
		resolvedEnv: map[string]string{},
		raisesError: true,
	},
	// TLS enabled with key but no cert
	{
		metadata:    map[string]string{"query": "SELECT COUNT(*) FROM table", "targetQueryValue": "5"},
		authParams:  map[string]string{"connectionString": "clickhouse://user:pass@localhost:9000/default", "tls": "true", "key": "test-key"},
		resolvedEnv: map[string]string{},
		raisesError: true,
	},
	// TLS enabled without cert or key (just server-side TLS)
	{
		metadata:    map[string]string{"query": "SELECT COUNT(*) FROM table", "targetQueryValue": "5"},
		authParams:  map[string]string{"connectionString": "clickhouse://user:pass@localhost:9000/default", "tls": "true"},
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

func TestBuildClickHouseDSN(t *testing.T) {
	testData := []struct {
		name      string
		meta      *clickhouseMetadata
		expected  string
		expectErr bool
	}{
		{
			name: "connection string without TLS",
			meta: &clickhouseMetadata{
				ConnectionString: "clickhouse://localhost:9000/default",
			},
			expected: "clickhouse://localhost:9000/default",
		},
		{
			name: "connection string with TLS",
			meta: &clickhouseMetadata{
				ConnectionString: "clickhouse://localhost:9000/default",
				TLS:              true,
			},
			expected: "clickhouse://localhost:9000/default?secure=true",
		},
		{
			name: "connection string with TLS and unsafe SSL",
			meta: &clickhouseMetadata{
				ConnectionString: "clickhouse://localhost:9000/default",
				TLS:              true,
				UnsafeSsl:        true,
			},
			expected: "clickhouse://localhost:9000/default?secure=true&skip_verify=true",
		},
		{
			name: "host/port with TLS",
			meta: &clickhouseMetadata{
				Host:     "localhost",
				Port:     "9000",
				Database: "default",
				Username: "default",
				Password: "",
				TLS:      true,
			},
			expected: "clickhouse://default:@localhost:9000/default?secure=true",
		},
		{
			name: "host/port with TLS and unsafe SSL",
			meta: &clickhouseMetadata{
				Host:      "localhost",
				Port:      "9000",
				Database:  "default",
				Username:  "default",
				Password:  "",
				TLS:       true,
				UnsafeSsl: true,
			},
			expected: "clickhouse://default:@localhost:9000/default?secure=true&skip_verify=true",
		},
		{
			name: "invalid connection string",
			meta: &clickhouseMetadata{
				ConnectionString: "://\x00invalid",
			},
			expectErr: true,
		},
	}

	for _, tt := range testData {
		t.Run(tt.name, func(t *testing.T) {
			result, err := buildClickHouseDSN(tt.meta)
			if tt.expectErr {
				if err == nil {
					t.Error("Expected an error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
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

func TestClickHouseGetQueryResult(t *testing.T) {
	t.Run("no rows", func(t *testing.T) {
		db, err := sql.Open("clickhouse-mock-norows", "")
		if err != nil {
			t.Fatalf("failed to open mock db: %v", err)
		}
		defer db.Close()

		scaler := &clickhouseScaler{
			metricType: v2.AverageValueMetricType,
			metadata: &clickhouseMetadata{
				Query:            "SELECT COUNT(*) FROM table",
				TargetQueryValue: 10,
			},
			connection: db,
			logger:     logr.Discard(),
		}

		value, err := scaler.getQueryResult(context.Background())
		if err != nil {
			t.Errorf("Expected nil error for no rows, got: %v", err)
		}
		if value != 0 {
			t.Errorf("Expected 0 for no rows, got: %v", value)
		}
	})

	t.Run("normal result", func(t *testing.T) {
		db, err := sql.Open("clickhouse-mock-result", "")
		if err != nil {
			t.Fatalf("failed to open mock db: %v", err)
		}
		defer db.Close()

		scaler := &clickhouseScaler{
			metricType: v2.AverageValueMetricType,
			metadata: &clickhouseMetadata{
				Query:            "SELECT COUNT(*) FROM table",
				TargetQueryValue: 10,
			},
			connection: db,
			logger:     logr.Discard(),
		}

		value, err := scaler.getQueryResult(context.Background())
		if err != nil {
			t.Errorf("Expected nil error for normal result, got: %v", err)
		}
		if value != 42 {
			t.Errorf("Expected 42, got: %v", value)
		}
	})
}
