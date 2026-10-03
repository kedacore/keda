package scaling

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/metricscollector"
	"github.com/kedacore/keda/v2/pkg/scalers"
	"github.com/kedacore/keda/v2/pkg/scalers/authentication"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	"github.com/kedacore/keda/v2/pkg/scaling/cache"
)

type cosmosDBBuildContextKey struct{}

type cosmosDBBlockingAuthClient struct {
	client.Client
	entered     chan struct{}
	release     chan struct{}
	enterOnce   sync.Once
	releaseOnce sync.Once
}

func (c *cosmosDBBlockingAuthClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	if _, ok := obj.(*kedav1alpha1.TriggerAuthentication); ok && ctx.Value(cosmosDBBuildContextKey{}) != nil {
		c.enterOnce.Do(func() { close(c.entered) })
		select {
		case <-c.release:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return c.Client.Get(ctx, key, obj, opts...)
}

func (c *cosmosDBBlockingAuthClient) unblock() {
	c.releaseOnce.Do(func() { close(c.release) })
}

func newCosmosDBDiagnosticHandler(t *testing.T, failNext *atomic.Bool) (*scaleHandler, *kedav1alpha1.ScaledObject, *cosmosDBBlockingAuthClient) {
	t.Helper()
	promMetricsCollectorOnce.Do(func() {
		metricscollector.NewMetricsCollectors(metricscollector.Options{EnablePrometheusMetrics: true})
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if failNext != nil && failNext.Swap(false) {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		_, _ = w.Write([]byte(`{"Documents":[]}`))
	}))
	t.Cleanup(server.Close)
	scheme := runtime.NewScheme()
	require.NoError(t, kedav1alpha1.AddToScheme(scheme))
	require.NoError(t, appsv1.AddToScheme(scheme))
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "worker", Namespace: "default"},
		Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "worker"}}},
		}},
	}
	auth := &kedav1alpha1.TriggerAuthentication{ObjectMeta: metav1.ObjectMeta{Name: "block", Namespace: "default"}}
	kubeClient := &cosmosDBBlockingAuthClient{
		Client:  fake.NewClientBuilder().WithScheme(scheme).WithObjects(deployment, auth).Build(),
		entered: make(chan struct{}), release: make(chan struct{}),
	}
	t.Cleanup(kubeClient.unblock)
	object := &kedav1alpha1.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{Name: t.Name(), Namespace: "default", Generation: 1},
		Spec: kedav1alpha1.ScaledObjectSpec{
			ScaleTargetRef: &kedav1alpha1.ScaleTarget{Name: "worker"},
			Triggers: []kedav1alpha1.ScaleTriggers{{Type: "azure-cosmosdb", Metadata: map[string]string{
				"connection": "AccountEndpoint=" + server.URL + ";AccountKey=dGVzdGtleQ==",
				"databaseId": "db", "containerId": "data", "leaseDatabaseId": "db",
				"leaseContainerId": "leases", "processorName": "processor",
			}}},
		},
		Status: kedav1alpha1.ScaledObjectStatus{
			ScaleTargetGVKR: &kedav1alpha1.GroupVersionKindResource{Group: "apps", Version: "v1", Kind: "Deployment"},
		},
	}
	handler := &scaleHandler{
		client: kubeClient, recorder: events.NewFakeRecorder(100),
		authClientSet: &authentication.AuthClientSet{}, scalerCaches: map[string]*cache.ScalersCache{},
		scalerCachesLock: &sync.RWMutex{},
	}
	return handler, object, kubeClient
}

func cosmosDBCacheDiagnosticCount(t *testing.T, name string) int {
	t.Helper()
	families, err := metrics.Registry.Gather()
	require.NoError(t, err)
	count := 0
	for _, family := range families {
		if family.GetName() != "keda_scaler_metrics_value" {
			continue
		}
		for _, metric := range family.Metric {
			labels := map[string]string{}
			for _, label := range metric.Label {
				labels[label.GetName()] = label.GetValue()
			}
			if labels["scaledObject"] == name && labels["triggerIndex"] == "0" {
				count++
			}
		}
	}
	return count
}

func pollCosmosDBDiagnosticCache(t *testing.T, c *cache.ScalersCache) {
	t.Helper()
	specs, err := c.GetMetricSpecForScalingForScaler(t.Context(), 0)
	require.NoError(t, err)
	require.NotEmpty(t, specs)
	values, active, _, err := c.GetMetricsAndActivityForScaler(t.Context(), 0, specs[0].External.Metric.Name)
	require.NoError(t, err)
	require.Len(t, values, 1)
	require.Zero(t, values[0].Value.Value())
	require.False(t, active)
}

func TestCosmosDBDiagnosticsSurviveAbortedCacheRebuild(t *testing.T) {
	handler, object, _ := newCosmosDBDiagnosticHandler(t, nil)
	serving, err := handler.GetScalersCache(t.Context(), object)
	require.NoError(t, err)
	defer serving.Close(context.Background())
	pollCosmosDBDiagnosticCache(t, serving)
	require.Equal(t, 5, cosmosDBCacheDiagnosticCount(t, object.Name))

	next := object.DeepCopy()
	next.Generation++
	next.Spec.Triggers = append(next.Spec.Triggers, kedav1alpha1.ScaleTriggers{Type: "azure-cosmosdb"})
	_, err = handler.GetScalersCache(t.Context(), next)
	require.Error(t, err)
	retained, err := handler.getScalersCacheForScaledObject(t.Context(), object.Name, object.Namespace)
	require.NoError(t, err)
	require.Same(t, serving, retained)
	require.Equal(t, 5, cosmosDBCacheDiagnosticCount(t, object.Name), "abandoned scaler cleanup must not delete serving diagnostics")
	for range 3 {
		pollCosmosDBDiagnosticCache(t, retained)
		require.Equal(t, 5, cosmosDBCacheDiagnosticCount(t, object.Name))
	}
}

func TestCosmosDBDiagnosticsFollowCacheInstallationOrder(t *testing.T) {
	handler, object, kubeClient := newCosmosDBDiagnosticHandler(t, nil)
	second := object.Spec.Triggers[0]
	second.AuthenticationRef = &kedav1alpha1.AuthenticationRef{Name: "block"}
	object.Spec.Triggers = append(object.Spec.Triggers, second)
	type buildResult struct {
		cache *cache.ScalersCache
		err   error
	}
	done := make(chan buildResult, 1)
	go func() {
		c, err := handler.GetScalersCache(context.WithValue(t.Context(), cosmosDBBuildContextKey{}, true), object)
		done <- buildResult{cache: c, err: err}
	}()
	defer kubeClient.unblock()
	select {
	case <-kubeClient.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("first cache build did not reach the second trigger")
	}

	early, err := handler.GetScalersCache(t.Context(), object)
	require.NoError(t, err)
	defer early.Close(context.Background())
	retiringScalers, _ := early.GetScalers()
	pollCosmosDBDiagnosticCache(t, early)
	require.Equal(t, 5, cosmosDBCacheDiagnosticCount(t, object.Name))

	kubeClient.unblock()
	late := <-done
	require.NoError(t, late.err)
	defer late.cache.Close(context.Background())
	require.NotSame(t, early, late.cache)
	installed, err := handler.getScalersCacheForScaledObject(t.Context(), object.Name, object.Namespace)
	require.NoError(t, err)
	require.Same(t, late.cache, installed)
	pollCosmosDBDiagnosticCache(t, installed)
	early.Close(context.Background())
	// Ensure the asynchronous retirement has reached each scaler's cleanup before asserting.
	for _, scaler := range retiringScalers {
		require.NoError(t, scaler.Close(context.Background()))
	}
	for range 3 {
		require.Equal(t, 5, cosmosDBCacheDiagnosticCount(t, object.Name))
		pollCosmosDBDiagnosticCache(t, installed)
	}
	require.Equal(t, 5, cosmosDBCacheDiagnosticCount(t, object.Name))
}

func TestCosmosDBDiagnosticsDuringCacheRefresh(t *testing.T) {
	for _, scenario := range []string{"serving refresh", "failed refresh", "retired refresh"} {
		t.Run(scenario, func(t *testing.T) {
			failNext := atomic.NewBool(false)
			handler, object, _ := newCosmosDBDiagnosticHandler(t, failNext)
			serving, err := handler.GetScalersCache(t.Context(), object)
			require.NoError(t, err)
			defer serving.Close(context.Background())
			pollCosmosDBDiagnosticCache(t, serving)
			require.Equal(t, 5, cosmosDBCacheDiagnosticCount(t, object.Name))

			entered := make(chan struct{})
			release := make(chan struct{})
			var releaseOnce sync.Once
			unblock := func() { releaseOnce.Do(func() { close(release) }) }
			defer unblock()
			constructed := make(chan scalers.Scaler, 1)
			factoryErr := errors.New("refresh factory failed")
			factory := serving.Scalers[0].Factory
			serving.Scalers[0].Factory = func(ctx context.Context) (scalers.Scaler, *scalersconfig.ScalerConfig, error) {
				close(entered)
				select {
				case <-release:
				case <-ctx.Done():
					return nil, nil, ctx.Err()
				}
				if scenario == "failed refresh" {
					return nil, nil, factoryErr
				}
				scaler, config, err := factory(ctx)
				if err == nil {
					constructed <- scaler
				}
				return scaler, config, err
			}
			metricName := serving.GetMetricSpecForScaling(t.Context())[0].External.Metric.Name
			failNext.Store(true)
			done := make(chan error, 1)
			go func() {
				_, _, _, err := serving.GetMetricsAndActivityForScaler(t.Context(), 0, metricName)
				done <- err
			}()
			select {
			case <-entered:
			case <-time.After(5 * time.Second):
				t.Fatal("poll did not start its refresh factory")
			}

			installed := serving
			if scenario == "retired refresh" {
				next := object.DeepCopy()
				next.Generation++
				// Retirement must not wait for the old cache's blocked refresh factory.
				ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
				defer cancel()
				type installResult struct {
					cache *cache.ScalersCache
					err   error
				}
				installDone := make(chan installResult, 1)
				go func() {
					c, err := handler.GetScalersCache(ctx, next)
					installDone <- installResult{cache: c, err: err}
				}()
				select {
				case result := <-installDone:
					require.NoError(t, result.err)
					installed = result.cache
				case <-ctx.Done():
					t.Fatal("installation waited for the retired cache's refresh factory")
				}
				defer installed.Close(context.Background())
				pollCosmosDBDiagnosticCache(t, installed)
				require.Equal(t, 5, cosmosDBCacheDiagnosticCount(t, object.Name))
			}

			unblock()
			err = <-done
			switch scenario {
			case "failed refresh":
				require.ErrorIs(t, err, factoryErr)
			case "retired refresh":
				require.True(t, err == nil || errors.Is(err, cache.ErrCacheClosed), "unexpected retired poll error: %v", err)
				stale := <-constructed
				require.NoError(t, stale.Close(context.Background()))
				serving.Close(context.Background())
			default:
				require.NoError(t, err)
				refreshed := <-constructed
				current, _ := installed.GetScalers()
				require.Same(t, refreshed, current[0])
			}
			for range 3 {
				require.Equal(t, 5, cosmosDBCacheDiagnosticCount(t, object.Name))
				pollCosmosDBDiagnosticCache(t, installed)
			}
			require.Equal(t, 5, cosmosDBCacheDiagnosticCount(t, object.Name))
		})
	}
}
