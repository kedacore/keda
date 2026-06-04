//go:build integration
// +build integration

package scalers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

const (
	intTestProject  = "test-project"
	intTestInstance = "test-instance"
	intTestDatabase = "test-db"
	intTestTable    = "jobs"
)

// requireEmulator skips the test if SPANNER_EMULATOR_HOST is not set.
func requireEmulator(t *testing.T) (grpcAddr, httpAddr string) {
	t.Helper()
	host := os.Getenv("SPANNER_EMULATOR_HOST")
	if host == "" {
		t.Skip("SPANNER_EMULATOR_HOST not set — skipping integration test")
	}
	// HTTP gateway is on port 9020 by convention (grpc on 9010)
	httpHost := strings.Replace(host, "9010", "9020", 1)
	return host, "http://" + httpHost
}

func restPost(t *testing.T, url string, body interface{}) {
	t.Helper()
	data, _ := json.Marshal(body)
	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		t.Logf("POST %s: %v", url, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		t.Logf("POST %s → %d: %s", url, resp.StatusCode, string(b))
	}
}

// setupEmulator creates instance + database + table via the REST gateway,
// then inserts test rows via the Go Spanner client.
func setupEmulator(t *testing.T, httpBase string) {
	t.Helper()

	// Create instance
	restPost(t,
		fmt.Sprintf("%s/v1/projects/%s/instances", httpBase, intTestProject),
		map[string]interface{}{
			"instanceId": intTestInstance,
			"instance": map[string]interface{}{
				"config":      fmt.Sprintf("projects/%s/instanceConfigs/emulator-config", intTestProject),
				"displayName": "test",
				"nodeCount":   1,
			},
		},
	)

	// Create database
	restPost(t,
		fmt.Sprintf("%s/v1/projects/%s/instances/%s/databases",
			httpBase, intTestProject, intTestInstance),
		map[string]interface{}{
			"createStatement": fmt.Sprintf("CREATE DATABASE `%s`", intTestDatabase),
			"extraStatements": []string{
				fmt.Sprintf(`CREATE TABLE %s (
					id      INT64 NOT NULL,
					status  STRING(64) NOT NULL,
					created TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true)
				) PRIMARY KEY (id)`, intTestTable),
			},
		},
	)

	// Wait for DDL to apply
	time.Sleep(500 * time.Millisecond)

	// Insert rows via Go client (SPANNER_EMULATOR_HOST already set → no auth needed)
	ctx := context.Background()
	dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		intTestProject, intTestInstance, intTestDatabase)

	// SPANNER_EMULATOR_HOST is set → SDK uses insecure connection automatically
	client, err := spanner.NewClient(ctx, dbPath)
	if err != nil {
		t.Fatalf("spanner.NewClient: %v", err)
	}
	defer client.Close()

	// Clear then insert deterministic test data
	_, _ = client.Apply(ctx, []*spanner.Mutation{
		spanner.Delete(intTestTable, spanner.AllKeys()),
	})

	_, err = client.Apply(ctx, []*spanner.Mutation{
		spanner.Insert(intTestTable, []string{"id", "status", "created"},
			[]interface{}{int64(1), "pending", spanner.CommitTimestamp}),
		spanner.Insert(intTestTable, []string{"id", "status", "created"},
			[]interface{}{int64(2), "pending", spanner.CommitTimestamp}),
		spanner.Insert(intTestTable, []string{"id", "status", "created"},
			[]interface{}{int64(3), "done", spanner.CommitTimestamp}),
	})
	if err != nil {
		t.Fatalf("insert test rows: %v", err)
	}
}

// spannerWriteRows inserts/deletes rows directly so tests can mutate state
// while the scaler is alive — simulating real workload changes.
func spannerWriteRows(t *testing.T, mutations []*spanner.Mutation) {
	t.Helper()
	ctx := context.Background()
	dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		intTestProject, intTestInstance, intTestDatabase)
	client, err := spanner.NewClient(ctx, dbPath)
	if err != nil {
		t.Fatalf("spannerWriteRows NewClient: %v", err)
	}
	defer client.Close()
	if _, err := client.Apply(ctx, mutations); err != nil {
		t.Fatalf("spannerWriteRows Apply: %v", err)
	}
}

func newIntegrationScaler(t *testing.T, triggerIndex int, query, targetValue, activationValue string) Scaler {
	t.Helper()
	config := &scalersconfig.ScalerConfig{
		TriggerIndex: triggerIndex,
		TriggerMetadata: map[string]string{
			"projectId":       intTestProject,
			"instanceId":      intTestInstance,
			"databaseId":      intTestDatabase,
			"query":           query,
			"targetValue":     targetValue,
			"activationValue": activationValue,
		},
		AuthParams:  map[string]string{},
		ResolvedEnv: map[string]string{},
	}
	scaler, err := NewGcpSpannerScaler(config)
	if err != nil {
		t.Fatalf("NewGcpSpannerScaler: %v", err)
	}
	return scaler
}

// ---- Tests ---------------------------------------------------------------

func TestSpannerIntegration_GetMetricsAndActivity(t *testing.T) {
	_, httpBase := requireEmulator(t)
	setupEmulator(t, httpBase)

	scaler := newIntegrationScaler(t, 0,
		fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE status = 'pending'", intTestTable),
		"1", "0",
	)
	defer scaler.Close(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	metricName := "s0-gcp-spanner-test-instance-test-db-test-project"
	metrics, active, err := scaler.GetMetricsAndActivity(ctx, metricName)
	if err != nil {
		t.Fatalf("GetMetricsAndActivity: %v", err)
	}

	if len(metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(metrics))
	}

	// 2 rows with status='pending' were inserted
	gotMilli := metrics[0].Value.MilliValue()
	wantMilli := int64(2 * 1000)
	if gotMilli != wantMilli {
		t.Errorf("metric value: got %d milli, want %d milli", gotMilli, wantMilli)
	}

	if !active {
		t.Error("expected scaler to be active (count=2 > activationValue=0)")
	}

	t.Logf("OK: metric=%s value=%s active=%v", metricName, metrics[0].Value.String(), active)
}

func TestSpannerIntegration_InactiveWhenZero(t *testing.T) {
	_, httpBase := requireEmulator(t)
	setupEmulator(t, httpBase)

	scaler := newIntegrationScaler(t, 1,
		fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE status = 'nonexistent'", intTestTable),
		"1", "0",
	)
	defer scaler.Close(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	metrics, active, err := scaler.GetMetricsAndActivity(ctx,
		"s1-gcp-spanner-test-instance-test-db-test-project")
	if err != nil {
		t.Fatalf("GetMetricsAndActivity: %v", err)
	}

	gotMilli := metrics[0].Value.MilliValue()
	if gotMilli != 0 {
		t.Errorf("expected 0 milli, got %d", gotMilli)
	}
	if active {
		t.Error("expected scaler inactive when count=0 == activationValue=0")
	}

	t.Log("OK: scaler correctly inactive when query returns 0")
}

func TestSpannerIntegration_GetMetricSpecForScaling(t *testing.T) {
	requireEmulator(t)

	scaler := newIntegrationScaler(t, 0,
		fmt.Sprintf("SELECT COUNT(*) FROM %s", intTestTable),
		"5", "0",
	)
	defer scaler.Close(context.Background())

	specs := scaler.GetMetricSpecForScaling(context.Background())
	if len(specs) != 1 {
		t.Fatalf("expected 1 metric spec, got %d", len(specs))
	}

	want := "s0-gcp-spanner-test-instance-test-db-test-project"
	got := specs[0].External.Metric.Name
	if got != want {
		t.Errorf("metric name: got %q, want %q", got, want)
	}

	t.Logf("OK: metric spec name=%s", got)
}

// TestSpannerIntegration_ScalingSimulation simulates the KEDA polling loop:
// it mutates the Spanner table between polls and asserts that the scaler's
// metric value and activity flag track the real workload.
//
//   targetValue=2  activationValue=1
//
//   step 0: 0 pending  → value=0, active=false  (below activation)
//   step 1: 2 pending  → value=2, active=true   (at target → 1 replica)
//   step 2: 6 pending  → value=6, active=true   (3× target → 3 replicas)
//   step 3: 1 pending  → value=1, active=false  (== activation threshold → inactive)
//   step 4: 0 pending  → value=0, active=false  (scale-to-zero)
func TestSpannerIntegration_ScalingSimulation(t *testing.T) {
	_, httpBase := requireEmulator(t)
	setupEmulator(t, httpBase) // ensures instance/db/table exist; clears + seeds 2 pending rows

	const (
		targetValue     = 2.0
		activationValue = 1.0
	)

	scaler := newIntegrationScaler(t, 0,
		fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE status = 'pending'", intTestTable),
		fmt.Sprintf("%.0f", targetValue),
		fmt.Sprintf("%.0f", activationValue),
	)
	defer scaler.Close(context.Background())

	metricName := "s0-gcp-spanner-test-instance-test-db-test-project"

	poll := func(t *testing.T) (int64, bool) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		metrics, active, err := scaler.GetMetricsAndActivity(ctx, metricName)
		if err != nil {
			t.Fatalf("GetMetricsAndActivity: %v", err)
		}
		return metrics[0].Value.MilliValue() / 1000, active
	}

	type step struct {
		desc       string
		mutations  []*spanner.Mutation
		wantValue  int64
		wantActive bool
		// wantReplicas = ceil(wantValue / targetValue), clamped to ≥0
		wantReplicas int
	}

	steps := []step{
		{
			desc: "step 0: clear all pending → scale-to-zero",
			mutations: []*spanner.Mutation{
				spanner.Delete(intTestTable, spanner.AllKeys()),
			},
			wantValue: 0, wantActive: false, wantReplicas: 0,
		},
		{
			desc: "step 1: add 2 pending → at target, 1 replica",
			mutations: []*spanner.Mutation{
				spanner.Insert(intTestTable, []string{"id", "status", "created"},
					[]interface{}{int64(10), "pending", spanner.CommitTimestamp}),
				spanner.Insert(intTestTable, []string{"id", "status", "created"},
					[]interface{}{int64(11), "pending", spanner.CommitTimestamp}),
			},
			wantValue: 2, wantActive: true, wantReplicas: 1,
		},
		{
			desc: "step 2: add 4 more pending → 6 total, 3 replicas",
			mutations: []*spanner.Mutation{
				spanner.Insert(intTestTable, []string{"id", "status", "created"},
					[]interface{}{int64(20), "pending", spanner.CommitTimestamp}),
				spanner.Insert(intTestTable, []string{"id", "status", "created"},
					[]interface{}{int64(21), "pending", spanner.CommitTimestamp}),
				spanner.Insert(intTestTable, []string{"id", "status", "created"},
					[]interface{}{int64(22), "pending", spanner.CommitTimestamp}),
				spanner.Insert(intTestTable, []string{"id", "status", "created"},
					[]interface{}{int64(23), "pending", spanner.CommitTimestamp}),
			},
			wantValue: 6, wantActive: true, wantReplicas: 3,
		},
		{
			desc: "step 3: mark 5 rows done → 1 pending, == activationValue → inactive",
			mutations: []*spanner.Mutation{
				spanner.Update(intTestTable, []string{"id", "status"}, []interface{}{int64(10), "done"}),
				spanner.Update(intTestTable, []string{"id", "status"}, []interface{}{int64(11), "done"}),
				spanner.Update(intTestTable, []string{"id", "status"}, []interface{}{int64(20), "done"}),
				spanner.Update(intTestTable, []string{"id", "status"}, []interface{}{int64(21), "done"}),
				spanner.Update(intTestTable, []string{"id", "status"}, []interface{}{int64(22), "done"}),
			},
			wantValue: 1, wantActive: false, wantReplicas: 0,
		},
		{
			desc: "step 4: mark last row done → 0 pending, scale-to-zero",
			mutations: []*spanner.Mutation{
				spanner.Update(intTestTable, []string{"id", "status"}, []interface{}{int64(23), "done"}),
			},
			wantValue: 0, wantActive: false, wantReplicas: 0,
		},
	}

	for _, s := range steps {
		t.Run(s.desc, func(t *testing.T) {
			spannerWriteRows(t, s.mutations)

			gotValue, gotActive := poll(t)

			if gotValue != s.wantValue {
				t.Errorf("value: got %d, want %d", gotValue, s.wantValue)
			}
			if gotActive != s.wantActive {
				t.Errorf("active: got %v, want %v", gotActive, s.wantActive)
			}

			// When active=false KEDA overrides HPA and scales to 0.
			// When active=true: replicas = ceil(value / targetValue).
			gotReplicas := 0
			if gotActive {
				gotReplicas = int((gotValue + int64(targetValue) - 1) / int64(targetValue))
			}
			if gotReplicas != s.wantReplicas {
				t.Errorf("replicas: got %d, want %d", gotReplicas, s.wantReplicas)
			}

			t.Logf("OK: value=%d active=%v → %d replicas", gotValue, gotActive, gotReplicas)
		})
	}
}

// TestSpannerIntegration_EmptyRowsReturnZero confirms that a query returning no
// rows (iterator.Done immediately) is treated as value=0 and not as an error.
// This covers SELECT statements that may return an empty set depending on data,
// e.g. "SELECT id FROM jobs WHERE id = -1".
func TestSpannerIntegration_EmptyRowsReturnZero(t *testing.T) {
	_, httpBase := requireEmulator(t)
	setupEmulator(t, httpBase)

	// A query guaranteed to return no rows (no such status value exists).
	scaler := newIntegrationScaler(t, 2,
		fmt.Sprintf("SELECT id FROM %s WHERE status = 'impossible_value'", intTestTable),
		"1", "0",
	)
	defer scaler.Close(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	metrics, active, err := scaler.GetMetricsAndActivity(ctx,
		"s2-gcp-spanner-test-instance-test-db-test-project")
	if err != nil {
		t.Fatalf("expected no error for empty result set, got: %v", err)
	}
	if got := metrics[0].Value.MilliValue(); got != 0 {
		t.Errorf("expected value=0 for empty result, got %d milli", got)
	}
	if active {
		t.Error("expected scaler inactive when query returns no rows")
	}
	t.Log("OK: empty result set returns value=0, no error")
}

// TestSpannerIntegration_CloseIdempotent confirms that calling Close twice
// does not panic or return an error.
func TestSpannerIntegration_CloseIdempotent(t *testing.T) {
	requireEmulator(t)

	scaler := newIntegrationScaler(t, 0,
		fmt.Sprintf("SELECT COUNT(*) FROM %s", intTestTable),
		"5", "0",
	)

	ctx := context.Background()
	if err := scaler.Close(ctx); err != nil {
		t.Errorf("first Close: %v", err)
	}
	if err := scaler.Close(ctx); err != nil {
		t.Errorf("second Close: %v", err)
	}
	t.Log("OK: double Close is safe")
}
