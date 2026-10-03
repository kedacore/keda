package scaling

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/metricscollector"
	"github.com/kedacore/keda/v2/pkg/scalers"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	"github.com/kedacore/keda/v2/pkg/scaling/cache"
	"github.com/kedacore/keda/v2/pkg/scaling/cache/metricscache"
)

func TestDeleteScalableObjectRemovesCosmosDBDiagnostics(t *testing.T) {
	promMetricsCollectorOnce.Do(func() {
		metricscollector.NewMetricsCollectors(metricscollector.Options{EnablePrometheusMetrics: true})
	})
	for _, kind := range []string{"ScaledObject", "ScaledJob"} {
		t.Run(kind, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{"Documents":[]}`))
			}))
			defer server.Close()
			config := &scalersconfig.ScalerConfig{
				ScalableObjectName: "cosmos-diagnostic-delete-" + kind, ScalableObjectNamespace: "default", ScalableObjectType: kind,
				TriggerMetadata: map[string]string{
					"endpoint": server.URL, "databaseId": "db", "containerId": "data",
					"leaseDatabaseId": "db", "leaseContainerId": "leases", "processorName": "test",
				},
				AuthParams: map[string]string{"cosmosDBKey": "dGVzdGtleQ=="},
			}
			scaler, err := scalers.NewAzureCosmosDBScaler(config)
			require.NoError(t, err)
			scalerCache := &cache.ScalersCache{Scalers: []cache.ScalerBuilder{{Scaler: scaler}}}
			scalerCache.ActivateDiagnostics()
			t.Cleanup(func() { scalerCache.Close(context.Background()) })
			name := scaler.GetMetricSpecForScaling(t.Context())[0].External.Metric.Name
			_, _, err = scaler.GetMetricsAndActivity(t.Context(), name)
			require.NoError(t, err)
			countDiagnostics := func() int {
				families, err := metrics.Registry.Gather()
				require.NoError(t, err)
				count := 0
				for _, family := range families {
					if family.GetName() == "keda_scaler_metrics_value" {
						for _, metric := range family.Metric {
							for _, label := range metric.Label {
								if label.GetName() == "scaledObject" && label.GetValue() == config.ScalableObjectName {
									count++
								}
							}
						}
					}
				}
				return count
			}
			require.Equal(t, 5, countDiagnostics())

			meta := metav1.ObjectMeta{Name: config.ScalableObjectName, Namespace: config.ScalableObjectNamespace}
			var object kedav1alpha1.ScalableObject = &kedav1alpha1.ScaledObject{ObjectMeta: meta}
			if kind == "ScaledJob" {
				object = &kedav1alpha1.ScaledJob{ObjectMeta: meta}
			}
			withTriggers, err := kedav1alpha1.AsDuckWithTriggers(object)
			require.NoError(t, err)
			handler := &scaleHandler{
				scaleLoopContexts: &sync.Map{},
				scalerCachesLock:  &sync.RWMutex{},
				scalerCaches: map[string]*cache.ScalersCache{
					withTriggers.GenerateIdentifier(): scalerCache,
				},
				scaledObjectsMetricCache: metricscache.NewMetricsCache(),
				recorder:                 events.NewFakeRecorder(1),
			}
			require.NoError(t, handler.DeleteScalableObject(t.Context(), object))
			assert.Eventually(t, func() bool { return countDiagnostics() == 0 }, 5*time.Second, 10*time.Millisecond)
			require.NoError(t, handler.DeleteScalableObject(t.Context(), object))
		})
	}
}
