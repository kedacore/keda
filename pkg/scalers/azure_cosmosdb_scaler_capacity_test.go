package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
	v2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/fallback"
	"github.com/kedacore/keda/v2/pkg/metricscollector"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

func cosmosDBCapacityConfig(endpoint string) *scalersconfig.ScalerConfig {
	return &scalersconfig.ScalerConfig{
		ScalableObjectName:      "cosmos-capacity",
		ScalableObjectNamespace: "default",
		ScalableObjectType:      "ScaledObject",
		TriggerMetadata: map[string]string{
			"connection": fmt.Sprintf("AccountEndpoint=%s;AccountKey=dGVzdGtleQ==", endpoint),
			"databaseId": "testdb", "containerId": "data",
			"leaseDatabaseId": "testdb", "leaseContainerId": "leases",
			"processorName": "testprocessor",
		},
	}
}

func TestCosmosDBCapacityMetadata(t *testing.T) {
	tests := []struct {
		name       string
		capacity   string
		ha         string
		metricType v2.MetricTargetType
		wantError  string
	}{
		{name: "legacy defaults"},
		{name: "legacy Value", metricType: v2.ValueMetricType},
		{name: "capacity", capacity: "4"},
		{name: "HA", capacity: "4", ha: "true"},
		{name: "disabled HA", ha: "false"},
		{name: "capacity Value", capacity: "4", metricType: v2.ValueMetricType, wantError: "requires metricType AverageValue"},
		{name: "zero", capacity: "0", wantError: "must be greater than zero"},
		{name: "negative", capacity: "-1", wantError: "must be greater than zero"},
		{name: "fractional", capacity: "1.5", wantError: "error parsing"},
		{name: "overflow", capacity: "9223372036854775808", wantError: "error parsing"},
		{name: "invalid HA", ha: "sometimes", wantError: "error parsing"},
		{name: "HA without capacity", ha: "true", wantError: "requires maxActiveLeasesPerReplica"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := cosmosDBCapacityConfig("https://test.documents.azure.com")
			config.MetricType = tt.metricType
			if tt.capacity != "" {
				config.TriggerMetadata["maxActiveLeasesPerReplica"] = tt.capacity
			}
			if tt.ha != "" {
				config.TriggerMetadata["enableHighAvailability"] = tt.ha
			}
			scaler, err := NewAzureCosmosDBScaler(config)
			if tt.wantError != "" {
				require.ErrorContains(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, scaler.Close(context.Background())) })
			spec := scaler.GetMetricSpecForScaling(context.Background())
			require.Len(t, spec, 1)
			wantTarget := int64(100)
			if tt.capacity != "" {
				wantTarget = 1
			}
			if tt.metricType == v2.ValueMetricType {
				assert.Equal(t, wantTarget, spec[0].External.Target.Value.Value())
			} else {
				assert.Equal(t, wantTarget, spec[0].External.Target.AverageValue.Value())
			}
		})
	}
	config := cosmosDBCapacityConfig("https://test.documents.azure.com")
	config.TriggerMetadata["maxActiveLeasesPerReplica"] = ""
	_, err := NewAzureCosmosDBScaler(config)
	require.Error(t, err)
}

type cosmosDBCapacityScenario struct {
	activeLeases int
	totalLeases  int
	lagPerLease  int64
	checkpointed bool
}

func newCosmosDBCapacityServer(t *testing.T, scenario cosmosDBCapacityScenario) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dbs/testdb/colls/leases/docs":
			docs := make([]map[string]interface{}, scenario.totalLeases)
			for i := range docs {
				docs[i] = map[string]interface{}{"id": fmt.Sprintf("lease%d", i), "LeaseToken": strconv.Itoa(i)}
				if scenario.checkpointed {
					docs[i]["ContinuationToken"] = `"0"`
				}
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"Documents": docs})
		case "/dbs/testdb/colls/data/docs":
			id, err := strconv.Atoi(r.Header.Get("x-ms-documentdb-partitionkeyrangeid"))
			assert.NoError(t, err)
			if id >= scenario.activeLeases {
				w.WriteHeader(http.StatusNotModified)
				return
			}
			w.Header().Set("x-ms-session-token", fmt.Sprintf("%d:0#%d", id, scenario.lagPerLease))
			_, _ = w.Write([]byte(`{"Documents":[{"id":"item","_lsn":1}]}`))
		default:
			t.Errorf("unexpected request %s", r.URL)
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
}

func TestCosmosDBLeaseCapacityScaling(t *testing.T) {
	tests := []struct {
		name       string
		scenario   cosmosDBCapacityScenario
		capacity   int64
		ha         bool
		activation int64
		wantMetric int64
		wantActive bool
	}{
		{name: "many leases little work", scenario: cosmosDBCapacityScenario{20, 20, 1, true}, capacity: 4, wantMetric: 5, wantActive: true},
		{name: "round up lease count", scenario: cosmosDBCapacityScenario{5, 5, 1, true}, capacity: 4, wantMetric: 2, wantActive: true},
		{name: "lag dominates", scenario: cosmosDBCapacityScenario{10, 10, 70, true}, capacity: 4, wantMetric: 7, wantActive: true},
		{name: "lag rounds up", scenario: cosmosDBCapacityScenario{3, 3, 34, true}, capacity: 4, wantMetric: 2, wantActive: true},
		{name: "nonHA active cap", scenario: cosmosDBCapacityScenario{2, 20, 1000, true}, capacity: 4, wantMetric: 2, wantActive: true},
		{name: "HA total cap", scenario: cosmosDBCapacityScenario{2, 20, 2000, true}, capacity: 4, ha: true, wantMetric: 20, wantActive: true},
		{name: "HA caught up", scenario: cosmosDBCapacityScenario{0, 9, 0, true}, capacity: 4, ha: true, wantMetric: 3, wantActive: true},
		{name: "nonHA caught up", scenario: cosmosDBCapacityScenario{0, 9, 0, true}, capacity: 4},
		{name: "HA empty", capacity: 4, ha: true},
		{name: "nonHA empty", capacity: 4},
		{name: "capacity never checkpointed", scenario: cosmosDBCapacityScenario{20, 20, 1, false}, capacity: 4, wantMetric: 5, wantActive: true},
		{name: "capacity one per replica", scenario: cosmosDBCapacityScenario{6, 6, 1, true}, capacity: 1, wantMetric: 6, wantActive: true},
		{name: "activation threshold", scenario: cosmosDBCapacityScenario{5, 5, 1, true}, capacity: 4, activation: 5, wantMetric: 2},
		{name: "HA bypasses activation", scenario: cosmosDBCapacityScenario{0, 5, 0, true}, capacity: 4, ha: true, activation: 5, wantMetric: 2, wantActive: true},
		{name: "legacy bootstrap cap", scenario: cosmosDBCapacityScenario{20, 20, 100, false}, wantMetric: 100, wantActive: true},
		{name: "legacy checkpointed cap", scenario: cosmosDBCapacityScenario{2, 20, 1000, true}, wantMetric: 200, wantActive: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newCosmosDBCapacityServer(t, tt.scenario)
			defer server.Close()
			client := newCosmosDBEPKTestClient(server)
			state, err := client.estimateLag(context.Background())
			require.NoError(t, err)
			assert.EqualValues(t, tt.scenario.activeLeases, state.activeLeases)
			assert.EqualValues(t, tt.scenario.totalLeases, state.totalLeases)
			scaler := &azureCosmosDBScaler{
				cosmosClient: client,
				metricType:   v2.AverageValueMetricType,
				metadata: &azureCosmosDBMetadata{
					Threshold: 100, ActivationThreshold: tt.activation,
					MaxActiveLeasesPerReplica: tt.capacity, EnableHighAvailability: tt.ha,
				},
				logger: logr.Discard(),
			}
			metrics, active, err := scaler.GetMetricsAndActivity(context.Background(), "test")
			require.NoError(t, err)
			require.Len(t, metrics, 1)
			assert.Equal(t, tt.wantMetric, metrics[0].Value.Value())
			assert.Equal(t, tt.wantActive, active)
		})
	}
}

func TestCosmosDBLeaseCountsAcrossPages(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dbs/testdb/colls/leases/docs":
			requests++
			switch requests {
			case 1:
				assert.Empty(t, r.Header.Get("x-ms-continuation"))
				w.Header().Set("x-ms-continuation", "page2")
				_, _ = w.Write([]byte(`{"Documents":[{"id":"a","LeaseToken":"0"},{"id":"metadata"}]}`))
			case 2:
				assert.Equal(t, "page2", r.Header.Get("x-ms-continuation"))
				w.Header().Set("x-ms-continuation", "page3")
				_, _ = w.Write([]byte(`{"Documents":[]}`))
			case 3:
				assert.Equal(t, "page3", r.Header.Get("x-ms-continuation"))
				_, _ = w.Write([]byte(`{"Documents":[{"id":"b","LeaseToken":"1","ContinuationToken":"\"5\""},{"id":"c","LeaseToken":"2"}]}`))
			default:
				t.Errorf("unexpected page %d", requests)
				w.WriteHeader(http.StatusBadRequest)
			}
		case "/dbs/testdb/colls/data/docs":
			if r.Header.Get("x-ms-documentdb-partitionkeyrangeid") == "1" {
				w.WriteHeader(http.StatusNotModified)
			} else {
				w.Header().Set("x-ms-session-token", "0:0#1")
				_, _ = w.Write([]byte(`{"Documents":[{"_lsn":1}]}`))
			}
		}
	}))
	defer server.Close()
	state, err := newCosmosDBEPKTestClient(server).estimateLag(context.Background())
	require.NoError(t, err)
	assert.Equal(t, cosmosDBLeaseState{totalLag: 2, activeLeases: 2, totalLeases: 3, legacyActivePartitions: 1}, state)
	assert.Equal(t, 3, requests)
}

func TestCosmosDBCapacityArithmeticLimits(t *testing.T) {
	for _, tt := range []struct{ value, divisor, want int64 }{
		{0, 4, 0}, {4, 4, 1}, {5, 4, 2},
		{math.MaxInt64, 1, math.MaxInt64},
		{math.MaxInt64, 2, math.MaxInt64/2 + 1},
		{math.MaxInt64, math.MaxInt64, 1},
	} {
		assert.Equal(t, tt.want, cosmosDBCeilDivide(tt.value, tt.divisor))
	}
	server := newCosmosDBCapacityServer(t, cosmosDBCapacityScenario{3, 3, math.MaxInt64, true})
	defer server.Close()
	state, err := newCosmosDBEPKTestClient(server).estimateLag(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(math.MaxInt64), state.totalLag)
	assert.Equal(t, int64(3), state.activeLeases)
}

var cosmosDBMetricsOnce sync.Once

func cosmosDBDiagnosticValues(t *testing.T, config *scalersconfig.ScalerConfig) map[string]float64 {
	t.Helper()
	families, err := ctrlmetrics.Registry.Gather()
	require.NoError(t, err)
	values := map[string]float64{}
	for _, family := range families {
		if family.GetName() != "keda_scaler_metrics_value" {
			continue
		}
		for _, metric := range family.Metric {
			labels := map[string]string{}
			for _, label := range metric.Label {
				labels[label.GetName()] = label.GetValue()
			}
			if labels["namespace"] == config.ScalableObjectNamespace && labels["scaledObject"] == config.ScalableObjectName && labels["triggerIndex"] == strconv.Itoa(config.TriggerIndex) {
				assert.Equal(t, "azure-cosmosdb", labels["scaler"])
				if config.ScalableObjectType == "ScaledObject" {
					assert.Equal(t, "scaledobject", labels["type"])
				} else {
					assert.Equal(t, "scaledjob", labels["type"])
				}
				values[labels["metric"]] = metric.GetGauge().GetValue()
			}
		}
	}
	return values
}

func TestCosmosDBLeaseDiagnosticsAndFallback(t *testing.T) {
	cosmosDBMetricsOnce.Do(func() {
		metricscollector.NewMetricsCollectors(metricscollector.Options{EnablePrometheusMetrics: true})
	})
	var fail bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if r.URL.Path == "/dbs/testdb/colls/leases/docs" {
			_, _ = w.Write([]byte(`{"Documents":[{"id":"a","LeaseToken":"0"},{"id":"b","LeaseToken":"1"}]}`))
			return
		}
		if r.Header.Get("x-ms-documentdb-partitionkeyrangeid") == "1" {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("x-ms-session-token", "0:0#250")
		_, _ = w.Write([]byte(`{"Documents":[{"_lsn":1}]}`))
	}))
	defer server.Close()
	config := cosmosDBCapacityConfig(server.URL)
	config.TriggerMetadata["maxActiveLeasesPerReplica"] = "1"
	config.TriggerMetadata["enableHighAvailability"] = "true"
	scaler, err := NewAzureCosmosDBScaler(config)
	require.NoError(t, err)
	scaler.(DiagnosticsLifecycle).ActivateDiagnostics()
	t.Cleanup(func() {
		require.NoError(t, scaler.Close(context.Background()))
		metricscollector.DeleteScalerMetrics("default", "cosmos-capacity", true)
	})
	specs := scaler.GetMetricSpecForScaling(context.Background())
	require.Len(t, specs, 1)
	metricName := specs[0].External.Metric.Name
	metrics, active, err := scaler.GetMetricsAndActivity(context.Background(), metricName)
	require.NoError(t, err)
	assert.True(t, active)
	require.Len(t, metrics, 1)
	assert.Equal(t, int64(2), metrics[0].Value.Value())
	want := map[string]float64{
		metricName + "_total_lag":                       250,
		metricName + "_active_leases":                   1,
		metricName + "_total_leases":                    2,
		metricName + "_lag_desired_replicas":            3,
		metricName + "_lease_capacity_desired_replicas": 2,
	}
	assert.Equal(t, want, cosmosDBDiagnosticValues(t, config))

	fail = true
	metrics, active, metricErr := scaler.GetMetricsAndActivity(context.Background(), metricName)
	require.Error(t, metricErr)
	assert.Empty(t, metrics)
	assert.False(t, active)
	values := cosmosDBDiagnosticValues(t, config)
	require.Len(t, values, 5)
	for name, value := range values {
		assert.True(t, math.IsNaN(value), name)
	}

	scheme := runtime.NewScheme()
	require.NoError(t, kedav1alpha1.AddToScheme(scheme))
	object := &kedav1alpha1.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{Name: "cosmos-capacity", Namespace: "default"},
		Spec: kedav1alpha1.ScaledObjectSpec{Fallback: &kedav1alpha1.Fallback{
			FailureThreshold: 1, Replicas: 3, Behavior: kedav1alpha1.FallbackBehaviorStatic,
		}},
	}
	kubeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(object).WithStatusSubresource(object).Build()
	handler := fallback.ScaledObjectHandler{Ctx: context.Background(), KubeClient: kubeClient, UpdateLock: &sync.RWMutex{}, ScaledObject: object}
	_, fallingBack, err := fallback.GetMetricsWithFallback(handler, metrics, metricErr, metricName, specs[0])
	require.Error(t, err)
	assert.False(t, fallingBack)
	fallbackMetrics, fallingBack, err := fallback.GetMetricsWithFallback(handler, metrics, metricErr, metricName, specs[0])
	require.NoError(t, err)
	assert.True(t, fallingBack)
	require.Len(t, fallbackMetrics, 1)
	assert.Equal(t, int64(3), fallbackMetrics[0].Value.Value())

	fail = false
	_, _, err = scaler.GetMetricsAndActivity(context.Background(), metricName)
	require.NoError(t, err)
	assert.Equal(t, want, cosmosDBDiagnosticValues(t, config))
	require.NoError(t, scaler.Close(context.Background()))
	assert.Empty(t, cosmosDBDiagnosticValues(t, config))
}

func TestCosmosDBDiagnosticsAcrossScalerGenerations(t *testing.T) {
	cosmosDBMetricsOnce.Do(func() {
		metricscollector.NewMetricsCollectors(metricscollector.Options{EnablePrometheusMetrics: true})
	})
	for _, resourceType := range []string{"ScaledObject", "ScaledJob"} {
		t.Run(resourceType, func(t *testing.T) {
			block := atomic.NewBool(false)
			entered := make(chan struct{})
			release := make(chan struct{})
			oldServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/dbs/testdb/colls/leases/docs" {
					if block.Load() {
						close(entered)
						<-release
					}
					_, _ = w.Write([]byte(`{"Documents":[{"id":"a","LeaseToken":"0","ContinuationToken":"\"0\""}]}`))
				} else {
					w.Header().Set("x-ms-session-token", "0:0#10")
					_, _ = w.Write([]byte(`{"Documents":[{"_lsn":1}]}`))
				}
			}))
			defer oldServer.Close()
			var releaseOnce sync.Once
			defer releaseOnce.Do(func() { close(release) })
			newServer := newCosmosDBCapacityServer(t, cosmosDBCapacityScenario{2, 3, 100, true})
			defer newServer.Close()
			oldConfig := cosmosDBCapacityConfig(oldServer.URL)
			oldConfig.ScalableObjectType = resourceType
			oldScaler, err := NewAzureCosmosDBScaler(oldConfig)
			require.NoError(t, err)
			oldScaler.(DiagnosticsLifecycle).ActivateDiagnostics()
			defer oldScaler.Close(context.Background())
			metricName := oldScaler.GetMetricSpecForScaling(context.Background())[0].External.Metric.Name
			_, _, err = oldScaler.GetMetricsAndActivity(context.Background(), metricName)
			require.NoError(t, err)
			assert.Equal(t, float64(10), cosmosDBDiagnosticValues(t, oldConfig)[metricName+"_total_lag"])
			block.Store(true)
			done := make(chan error, 1)
			go func() {
				_, _, err := oldScaler.GetMetricsAndActivity(context.Background(), metricName)
				done <- err
			}()
			<-entered

			config := cosmosDBCapacityConfig(newServer.URL)
			config.ScalableObjectType = resourceType
			replacement, err := NewAzureCosmosDBScaler(config)
			require.NoError(t, err)
			oldScaler.(DiagnosticsLifecycle).DeactivateDiagnostics()
			replacement.(DiagnosticsLifecycle).ActivateDiagnostics()
			defer replacement.Close(context.Background())
			_, _, err = replacement.GetMetricsAndActivity(context.Background(), metricName)
			require.NoError(t, err)
			want := cosmosDBDiagnosticValues(t, config)
			require.Len(t, want, 5)
			assert.Equal(t, float64(200), want[metricName+"_total_lag"])

			releaseOnce.Do(func() { close(release) })
			require.NoError(t, <-done)
			assert.Equal(t, want, cosmosDBDiagnosticValues(t, config), "retiring polls must not overwrite replacement data")
			require.NoError(t, oldScaler.Close(context.Background()))
			assert.Equal(t, want, cosmosDBDiagnosticValues(t, config), "retiring Close must not delete replacement series")

			next, err := NewAzureCosmosDBScaler(config)
			require.NoError(t, err)
			replacement.(DiagnosticsLifecycle).DeactivateDiagnostics()
			next.(DiagnosticsLifecycle).ActivateDiagnostics()
			defer next.Close(context.Background())
			require.NoError(t, next.Close(context.Background()))
			assert.Empty(t, cosmosDBDiagnosticValues(t, config))
			_, _, err = replacement.GetMetricsAndActivity(context.Background(), metricName)
			require.NoError(t, err)
			assert.Empty(t, cosmosDBDiagnosticValues(t, config), "a superseded generation must not resurrect deleted diagnostics")
			require.NoError(t, replacement.Close(context.Background()))
		})
	}
}

func TestCosmosDBCloseClearsOnlyOwnedDiagnostics(t *testing.T) {
	cosmosDBMetricsOnce.Do(func() {
		metricscollector.NewMetricsCollectors(metricscollector.Options{EnablePrometheusMetrics: true})
	})
	for _, resourceType := range []string{"ScaledObject", "ScaledJob"} {
		t.Run(resourceType, func(t *testing.T) {
			block := atomic.NewBool(false)
			entered := make(chan struct{}, 1)
			release := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if block.Load() {
					select {
					case entered <- struct{}{}:
					default:
					}
					<-release
				}
				if r.URL.Path == "/dbs/testdb/colls/leases/docs" {
					_, _ = w.Write([]byte(`{"Documents":[{"id":"a","LeaseToken":"0"}]}`))
				} else {
					w.WriteHeader(http.StatusNotModified)
				}
			}))
			defer server.Close()
			// Ensure a failed assertion cannot leave the HTTP handler blocked.
			var releaseOnce sync.Once
			defer releaseOnce.Do(func() { close(release) })
			config := cosmosDBCapacityConfig(server.URL)
			config.ScalableObjectType = resourceType
			scaler, err := NewAzureCosmosDBScaler(config)
			require.NoError(t, err)
			scaler.(DiagnosticsLifecycle).ActivateDiagnostics()
			defer scaler.Close(context.Background())
			otherConfig := cosmosDBCapacityConfig(server.URL)
			otherConfig.ScalableObjectType = resourceType
			otherConfig.TriggerIndex = 1
			other, err := NewAzureCosmosDBScaler(otherConfig)
			require.NoError(t, err)
			other.(DiagnosticsLifecycle).ActivateDiagnostics()
			defer other.Close(context.Background())
			metricName := scaler.GetMetricSpecForScaling(context.Background())[0].External.Metric.Name
			otherName := other.GetMetricSpecForScaling(context.Background())[0].External.Metric.Name
			_, _, err = scaler.GetMetricsAndActivity(context.Background(), metricName)
			require.NoError(t, err)
			_, _, err = other.GetMetricsAndActivity(context.Background(), otherName)
			require.NoError(t, err)
			require.Len(t, cosmosDBDiagnosticValues(t, config), 5)
			require.Len(t, cosmosDBDiagnosticValues(t, otherConfig), 5)

			block.Store(true)
			done := make(chan error, 1)
			go func() {
				_, _, err := scaler.GetMetricsAndActivity(context.Background(), metricName)
				done <- err
			}()
			<-entered
			require.NoError(t, scaler.Close(context.Background()))
			assert.Empty(t, cosmosDBDiagnosticValues(t, config))
			require.Len(t, cosmosDBDiagnosticValues(t, otherConfig), 5)
			releaseOnce.Do(func() { close(release) })
			require.NoError(t, <-done)
			assert.Empty(t, cosmosDBDiagnosticValues(t, config), "late polls must not recreate closed diagnostics")
			require.Len(t, cosmosDBDiagnosticValues(t, otherConfig), 5)
			require.NoError(t, other.Close(context.Background()))
			assert.Empty(t, cosmosDBDiagnosticValues(t, otherConfig))
		})
	}
}
