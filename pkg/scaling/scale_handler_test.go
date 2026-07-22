/*
Copyright 2021 The KEDA Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package scaling

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/expr-lang/expr"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	v2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/metricscollector"
	"github.com/kedacore/keda/v2/pkg/mock/mock_client"
	mock_scalers "github.com/kedacore/keda/v2/pkg/mock/mock_scaler"
	"github.com/kedacore/keda/v2/pkg/mock/mock_scaling/mock_executor"
	"github.com/kedacore/keda/v2/pkg/scalers"
	"github.com/kedacore/keda/v2/pkg/scalers/authentication"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	"github.com/kedacore/keda/v2/pkg/scaling/cache"
	"github.com/kedacore/keda/v2/pkg/scaling/cache/metricscache"
	"github.com/kedacore/keda/v2/pkg/scaling/executor"
)

const testNamespaceGlobal = "testNamespace"
const compositeMetricNameGlobal = "composite-metric"
const testNameGlobal = "testName"

var promMetricsCollectorOnce sync.Once

func TestGetScaledObjectMetrics_DirectCall(t *testing.T) {
	scaledObjectName := testNameGlobal
	scaledObjectNamespace := testNamespaceGlobal
	metricName := "test-metric-name"
	longPollingInterval := int32(300)

	ctrl := gomock.NewController(t)
	recorder := events.NewFakeRecorder(1)
	mockClient := mock_client.NewMockClient(ctrl)
	mockExecutor := mock_executor.NewMockScaleExecutor(ctrl)

	metricsSpecs := []v2.MetricSpec{createMetricSpec(10, metricName)}
	metricValue := scalers.GenerateMetricInMili(metricName, float64(10))

	metricsRecord := map[string]metricscache.MetricsRecord{}
	metricsRecord[metricName] = metricscache.MetricsRecord{
		IsActive:    true,
		Metric:      []external_metrics.ExternalMetricValue{metricValue},
		ScalerError: nil,
	}

	scaler := mock_scalers.NewMockScaler(ctrl)
	// we are going to query metrics directly
	scalerConfig := scalersconfig.ScalerConfig{TriggerUseCachedMetrics: false}
	factory := func() (scalers.Scaler, *scalersconfig.ScalerConfig, error) {
		return scaler, &scalerConfig, nil
	}

	scaledObject := kedav1alpha1.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      scaledObjectName,
			Namespace: scaledObjectNamespace,
		},
		Spec: kedav1alpha1.ScaledObjectSpec{
			ScaleTargetRef: &kedav1alpha1.ScaleTarget{
				Name: "test",
			},
			PollingInterval: &longPollingInterval,
		},
		Status: kedav1alpha1.ScaledObjectStatus{
			ScaleTargetGVKR: &kedav1alpha1.GroupVersionKindResource{
				Group: "apps",
				Kind:  "Deployment",
			},
		},
	}

	scalerCache := cache.ScalersCache{
		ScaledObject: &scaledObject,
		Scalers: []cache.ScalerBuilder{{
			Scaler:       scaler,
			ScalerConfig: scalerConfig,
			Factory:      factory,
		}},
		Recorder: recorder,
	}

	caches := map[string]*cache.ScalersCache{}
	caches[scaledObject.GenerateIdentifier()] = &scalerCache

	sh := scaleHandler{
		client:                   mockClient,
		scaleLoopContexts:        &sync.Map{},
		scaleExecutor:            mockExecutor,
		globalHTTPTimeout:        time.Duration(1000),
		recorder:                 recorder,
		scalerCaches:             caches,
		scalerCachesLock:         &sync.RWMutex{},
		scaledObjectsMetricCache: metricscache.NewMetricsCache(),
		rawMetricsSubscriptions:  map[string]*RawMetricSubscriptions{},
		metricToSubscriptions:    map[metricMeta][]*RawMetricSubscriptions{},
		subsLock:                 &sync.RWMutex{},
	}

	mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	scaler.EXPECT().GetMetricSpecForScaling(gomock.Any()).Return(metricsSpecs)
	scaler.EXPECT().GetMetricsAndActivity(gomock.Any(), gomock.Any()).Return([]external_metrics.ExternalMetricValue{metricValue}, true, nil)
	mockExecutor.EXPECT().RequestScale(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())
	expectHandleResult(mockClient)
	sh.checkScalers(context.TODO(), &scaledObject, &sync.RWMutex{})

	expectNoStatusPatch(ctrl)
	scaler.EXPECT().GetMetricSpecForScaling(gomock.Any()).Return(metricsSpecs)
	// hitting directly GetMetricsAndActivity()
	scaler.EXPECT().GetMetricsAndActivity(gomock.Any(), gomock.Any()).Return([]external_metrics.ExternalMetricValue{metricValue}, true, nil)
	metrics, err := sh.GetScaledObjectMetrics(context.TODO(), scaledObjectName, scaledObjectNamespace, metricName)
	assert.NotNil(t, metrics)
	assert.Nil(t, err)

	scaler.EXPECT().Close(gomock.Any())
	scalerCache.Close(context.Background())
}

func TestGetScaledObjectMetrics_FromCache(t *testing.T) {
	scaledObjectName := "testName2"
	scaledObjectNamespace := "testNamespace2"
	metricName := "test-metric-name2"
	longPollingInterval := int32(300)

	ctrl := gomock.NewController(t)
	recorder := events.NewFakeRecorder(1)
	mockClient := mock_client.NewMockClient(ctrl)
	mockExecutor := mock_executor.NewMockScaleExecutor(ctrl)

	metricsSpecs := []v2.MetricSpec{createMetricSpec(10, metricName)}
	metricValue := scalers.GenerateMetricInMili(metricName, float64(10))

	metricsRecord := map[string]metricscache.MetricsRecord{}
	metricsRecord[metricName] = metricscache.MetricsRecord{
		IsActive:    true,
		Metric:      []external_metrics.ExternalMetricValue{metricValue},
		ScalerError: nil,
	}

	scaler := mock_scalers.NewMockScaler(ctrl)
	// we are going to use cache for metrics values
	scalerConfig := scalersconfig.ScalerConfig{TriggerUseCachedMetrics: true}
	factory := func() (scalers.Scaler, *scalersconfig.ScalerConfig, error) {
		return scaler, &scalerConfig, nil
	}

	scaledObject := kedav1alpha1.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      scaledObjectName,
			Namespace: scaledObjectNamespace,
		},
		Spec: kedav1alpha1.ScaledObjectSpec{
			ScaleTargetRef: &kedav1alpha1.ScaleTarget{
				Name: "test",
			},
			PollingInterval: &longPollingInterval,
		},
		Status: kedav1alpha1.ScaledObjectStatus{
			ScaleTargetGVKR: &kedav1alpha1.GroupVersionKindResource{
				Group: "apps",
				Kind:  "Deployment",
			},
		},
	}

	scalerCache := cache.ScalersCache{
		ScaledObject: &scaledObject,
		Scalers: []cache.ScalerBuilder{{
			Scaler:       scaler,
			ScalerConfig: scalerConfig,
			Factory:      factory,
		}},
		Recorder: recorder,
	}

	caches := map[string]*cache.ScalersCache{}
	caches[scaledObject.GenerateIdentifier()] = &scalerCache

	sh := scaleHandler{
		client:                   mockClient,
		scaleLoopContexts:        &sync.Map{},
		scaleExecutor:            mockExecutor,
		globalHTTPTimeout:        time.Duration(1000),
		recorder:                 recorder,
		scalerCaches:             caches,
		scalerCachesLock:         &sync.RWMutex{},
		scaledObjectsMetricCache: metricscache.NewMetricsCache(),
		rawMetricsSubscriptions:  map[string]*RawMetricSubscriptions{},
		metricToSubscriptions:    map[metricMeta][]*RawMetricSubscriptions{},
		subsLock:                 &sync.RWMutex{},
	}

	mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	scaler.EXPECT().GetMetricSpecForScaling(gomock.Any()).Return(metricsSpecs)
	scaler.EXPECT().GetMetricsAndActivity(gomock.Any(), gomock.Any()).Return([]external_metrics.ExternalMetricValue{metricValue}, true, nil)
	mockExecutor.EXPECT().RequestScale(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())
	expectHandleResult(mockClient)
	sh.checkScalers(context.TODO(), &scaledObject, &sync.RWMutex{})

	expectNoStatusPatch(ctrl)
	scaler.EXPECT().GetMetricSpecForScaling(gomock.Any()).Return(metricsSpecs)
	// hitting cache here instead of calling GetMetricsAndActivity()
	metrics, err := sh.GetScaledObjectMetrics(context.TODO(), scaledObjectName, scaledObjectNamespace, metricName)
	assert.NotNil(t, metrics)
	assert.Nil(t, err)

	scaler.EXPECT().Close(gomock.Any())
	scalerCache.Close(context.Background())
}

// TestGetScaledObjectMetrics_InParallel executes
// a request to multiple scalers with a delay.
// The sum off all the scalers is more than the timeout
// but all of them in parallel are recovered in time
func TestGetScaledObjectMetrics_InParallel(t *testing.T) {
	scaledObjectName := testNameGlobal
	scaledObjectNamespace := testNamespaceGlobal
	metricNames := []string{
		"test-metric-name-1",
		"test-metric-name-2",
		"test-metric-name-3",
		"test-metric-name-4",
		"test-metric-name-5",
		"test-metric-name-6",
		"test-metric-name-7",
		"test-metric-name-8",
		"test-metric-name-9",
		"test-metric-name-10",
	}
	metricsName := strings.Join(metricNames, ";")
	longPollingInterval := int32(300)

	ctrl := gomock.NewController(t)
	recorder := events.NewFakeRecorder(1)
	mockClient := mock_client.NewMockClient(ctrl)
	mockExecutor := mock_executor.NewMockScaleExecutor(ctrl)

	scalerCollection := []*mock_scalers.MockScaler{}

	for i := 0; i < len(metricNames); i++ {
		scalerCollection = append(scalerCollection, mock_scalers.NewMockScaler(ctrl))
	}

	metricsSpecFn := func(index int) []v2.MetricSpec {
		return []v2.MetricSpec{createMetricSpec(10, metricNames[index])}
	}
	metricsValueFn := func(index int) []external_metrics.ExternalMetricValue {
		time.Sleep(200 * time.Millisecond)
		return []external_metrics.ExternalMetricValue{scalers.GenerateMetricInMili(metricNames[index], float64(10))}
	}
	scalerConfigFn := func(index int) *scalersconfig.ScalerConfig {
		return &scalersconfig.ScalerConfig{
			TriggerUseCachedMetrics: false,
			TriggerIndex:            index,
		}
	}

	scalerFactoryFn := func(index int) func() (scalers.Scaler, *scalersconfig.ScalerConfig, error) {
		return func() (scalers.Scaler, *scalersconfig.ScalerConfig, error) {
			return scalerCollection[index], scalerConfigFn(index), nil
		}
	}

	scaledObject := kedav1alpha1.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      scaledObjectName,
			Namespace: scaledObjectNamespace,
		},
		Spec: kedav1alpha1.ScaledObjectSpec{
			ScaleTargetRef: &kedav1alpha1.ScaleTarget{
				Name: "test",
			},
			PollingInterval: &longPollingInterval,
			Advanced: &kedav1alpha1.AdvancedConfig{
				ScalingModifiers: kedav1alpha1.ScalingModifiers{
					Target: "1",
				},
			},
		},
		Status: kedav1alpha1.ScaledObjectStatus{
			ScaleTargetGVKR: &kedav1alpha1.GroupVersionKindResource{
				Group: "apps",
				Kind:  "Deployment",
			},
			ExternalMetricNames: metricNames,
		},
	}

	scalerCache := cache.ScalersCache{
		ScaledObject: &scaledObject,
		Scalers:      []cache.ScalerBuilder{},
		Recorder:     recorder,
	}
	for i := 0; i < len(metricNames); i++ {
		scalerCache.Scalers = append(scalerCache.Scalers, cache.ScalerBuilder{
			Scaler:       scalerCollection[i],
			ScalerConfig: *scalerConfigFn(i),
			Factory:      scalerFactoryFn(i),
		})
	}

	caches := map[string]*cache.ScalersCache{}
	caches[scaledObject.GenerateIdentifier()] = &scalerCache

	sh := scaleHandler{
		client:                   mockClient,
		scaleLoopContexts:        &sync.Map{},
		scaleExecutor:            mockExecutor,
		globalHTTPTimeout:        time.Duration(1000),
		recorder:                 recorder,
		scalerCaches:             caches,
		scalerCachesLock:         &sync.RWMutex{},
		scaledObjectsMetricCache: metricscache.NewMetricsCache(),
		rawMetricsSubscriptions:  map[string]*RawMetricSubscriptions{},
		metricToSubscriptions:    map[metricMeta][]*RawMetricSubscriptions{},
		subsLock:                 &sync.RWMutex{},
	}

	mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	for i := 0; i < len(metricNames); i++ {
		scalerCollection[i].EXPECT().GetMetricSpecForScaling(gomock.Any()).Return(metricsSpecFn(i))
		scalerCollection[i].EXPECT().GetMetricsAndActivity(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
			return metricsValueFn(i), true, nil
		})
	}
	mockExecutor.EXPECT().RequestScale(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())
	expectHandleResult(mockClient)
	assert.Eventually(t, func() bool {
		sh.checkScalers(context.TODO(), &scaledObject, &sync.RWMutex{})
		return true
	}, 1*time.Second, 400*time.Millisecond, "timeout exceeded: scalers not processed in parallel during `checkScalers`")

	expectNoStatusPatch(ctrl)

	for i := 0; i < len(metricNames); i++ {
		scalerCollection[i].EXPECT().GetMetricSpecForScaling(gomock.Any()).Return(metricsSpecFn(i))
		scalerCollection[i].EXPECT().GetMetricsAndActivity(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
			return metricsValueFn(i), true, nil
		})
	}
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		metrics, err := sh.GetScaledObjectMetrics(context.TODO(), scaledObjectName, scaledObjectNamespace, metricsName)
		assert.NotNil(c, metrics)
		assert.Nil(c, err)
	}, 1*time.Second, 400*time.Millisecond, "timeout exceeded: scalers not processed in parallel during `GetScaledObjectMetrics`")

	for i := 0; i < len(metricNames); i++ {
		scalerCollection[i].EXPECT().Close(gomock.Any())
	}
	scalerCache.Close(context.Background())
}

func TestCheckScaledObjectScalersWithError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)
	mockExecutor := mock_executor.NewMockScaleExecutor(ctrl)
	recorder := events.NewFakeRecorder(1)

	metricsSpecs := []v2.MetricSpec{createMetricSpec(1, "metric-name")}

	scaler := mock_scalers.NewMockScaler(ctrl)
	scaler.EXPECT().GetMetricSpecForScaling(gomock.Any()).Return(metricsSpecs)
	scaler.EXPECT().GetMetricsAndActivity(gomock.Any(), gomock.Any()).Return([]external_metrics.ExternalMetricValue{}, false, errors.New("some error"))
	scaler.EXPECT().Close(gomock.Any())

	factory := func() (scalers.Scaler, *scalersconfig.ScalerConfig, error) {
		scaler := mock_scalers.NewMockScaler(ctrl)
		scaler.EXPECT().GetMetricsAndActivity(gomock.Any(), gomock.Any()).Return([]external_metrics.ExternalMetricValue{}, false, errors.New("some error"))
		scaler.EXPECT().Close(gomock.Any())
		return scaler, &scalersconfig.ScalerConfig{}, nil
	}

	scaledObject := kedav1alpha1.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: kedav1alpha1.ScaledObjectSpec{
			ScaleTargetRef: &kedav1alpha1.ScaleTarget{
				Name: "test",
			},
		},
	}

	scalerCache := cache.ScalersCache{
		Scalers: []cache.ScalerBuilder{{
			Scaler:  scaler,
			Factory: factory,
		}},
		Recorder: recorder,
	}

	caches := map[string]*cache.ScalersCache{}
	caches[scaledObject.GenerateIdentifier()] = &scalerCache

	sh := scaleHandler{
		client:                   mockClient,
		scaleLoopContexts:        &sync.Map{},
		scaleExecutor:            mockExecutor,
		globalHTTPTimeout:        time.Duration(1000),
		recorder:                 recorder,
		scalerCaches:             caches,
		scalerCachesLock:         &sync.RWMutex{},
		scaledObjectsMetricCache: metricscache.NewMetricsCache(),
		rawMetricsSubscriptions:  map[string]*RawMetricSubscriptions{},
		metricToSubscriptions:    map[metricMeta][]*RawMetricSubscriptions{},
		subsLock:                 &sync.RWMutex{},
	}

	isActive, isError, _, activeTriggers, _, _ := sh.getScaledObjectState(context.TODO(), &scaledObject)
	scalerCache.Close(context.Background())

	assert.Equal(t, false, isActive)
	assert.Equal(t, true, isError)
	assert.Empty(t, activeTriggers)
}

func TestCheckScaledObjectScalersWithTriggerAuthError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)
	mockExecutor := mock_executor.NewMockScaleExecutor(ctrl)
	recorder := events.NewFakeRecorder(1)

	scaler := mock_scalers.NewMockScaler(ctrl)
	scaler.EXPECT().Close(gomock.Any())

	factory := func() (scalers.Scaler, *scalersconfig.ScalerConfig, error) {
		scaler := mock_scalers.NewMockScaler(ctrl)
		scaler.EXPECT().GetMetricsAndActivity(gomock.Any(), gomock.Any()).Return([]external_metrics.ExternalMetricValue{}, false, errors.New("some error"))
		scaler.EXPECT().Close(gomock.Any())
		return scaler, &scalersconfig.ScalerConfig{}, nil
	}

	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-test",
			Namespace: "test",
		},
		Spec: appsv1.DeploymentSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name: "container",
						},
					},
				},
			},
		},
	}

	scaledObject := kedav1alpha1.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scaledobject-test",
			Namespace: "test",
		},
		Spec: kedav1alpha1.ScaledObjectSpec{
			ScaleTargetRef: &kedav1alpha1.ScaleTarget{
				Name: deployment.Name,
			},
			Triggers: []kedav1alpha1.ScaleTriggers{
				{
					Name: triggerName1,
					Type: "fake_trig1",
					AuthenticationRef: &kedav1alpha1.AuthenticationRef{
						Name: "triggerauth-test",
					},
				},
			},
		},
		Status: kedav1alpha1.ScaledObjectStatus{
			ScaleTargetGVKR: &kedav1alpha1.GroupVersionKindResource{
				Group: "apps",
				Kind:  "Deployment",
			},
			ExternalMetricNames: []string{metricName1, metricName2},
		},
	}

	triggerAuth := kedav1alpha1.TriggerAuthentication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "triggerauth-test",
			Namespace: "test",
		},
		Spec: kedav1alpha1.TriggerAuthenticationSpec{
			HashiCorpVault: &kedav1alpha1.HashiCorpVault{
				Address:        "invalid-vault-address",
				Authentication: "token",
				Credential: &kedav1alpha1.Credential{
					Token: "my-token",
				},
				Mount: "kubernetes",
				Role:  "my-role",
				Secrets: []kedav1alpha1.VaultSecret{
					{
						Parameter: "username",
						Key:       "username",
						Path:      "secret_v2/data/my-username-path",
					},
				},
			},
		},
	}

	scalerCache := cache.ScalersCache{
		Scalers: []cache.ScalerBuilder{{
			Scaler:  scaler,
			Factory: factory,
		}},
		Recorder: recorder,
	}

	mockClient.EXPECT().Get(gomock.Any(), types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, gomock.Any()).SetArg(2, deployment)
	mockClient.EXPECT().Get(gomock.Any(), types.NamespacedName{Name: triggerAuth.Name, Namespace: triggerAuth.Namespace}, gomock.Any()).SetArg(2, triggerAuth)

	sh := scaleHandler{
		client:                   mockClient,
		scaleLoopContexts:        &sync.Map{},
		scaleExecutor:            mockExecutor,
		globalHTTPTimeout:        time.Duration(1000),
		recorder:                 recorder,
		scalerCaches:             map[string]*cache.ScalersCache{},
		scalerCachesLock:         &sync.RWMutex{},
		scaledObjectsMetricCache: metricscache.NewMetricsCache(),
		authClientSet: &authentication.AuthClientSet{
			SecretLister: nil,
		},
		rawMetricsSubscriptions: map[string]*RawMetricSubscriptions{},
		metricToSubscriptions:   map[metricMeta][]*RawMetricSubscriptions{},
		subsLock:                &sync.RWMutex{},
	}

	isActive, isError, _, activeTriggers, _, _ := sh.getScaledObjectState(context.TODO(), &scaledObject)
	scalerCache.Close(context.Background())

	assert.Equal(t, false, isActive)
	assert.Equal(t, true, isError)
	assert.Empty(t, activeTriggers)

	failureEvent := <-recorder.Events
	assert.Contains(t, failureEvent, "KEDAScalerFailed")
	assert.Contains(t, failureEvent, "unsupported protocol scheme")
}

func TestCheckScaledObjectFindFirstActiveNotIgnoreOthers(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)
	mockExecutor := mock_executor.NewMockScaleExecutor(ctrl)
	recorder := events.NewFakeRecorder(1)

	metricsSpecs := []v2.MetricSpec{createMetricSpec(1, "metric-name")}
	metricValue := scalers.GenerateMetricInMili("metric-name", float64(10))

	activeFactory := func() (scalers.Scaler, *scalersconfig.ScalerConfig, error) {
		scaler := mock_scalers.NewMockScaler(ctrl)
		scaler.EXPECT().GetMetricSpecForScaling(gomock.Any()).Return(metricsSpecs)
		scaler.EXPECT().GetMetricsAndActivity(gomock.Any(), gomock.Any()).Return([]external_metrics.ExternalMetricValue{metricValue}, true, nil)
		scaler.EXPECT().Close(gomock.Any())
		return scaler, &scalersconfig.ScalerConfig{}, nil
	}
	activeScaler, _, err := activeFactory()
	assert.Nil(t, err)

	failingFactory := func() (scalers.Scaler, *scalersconfig.ScalerConfig, error) {
		scaler := mock_scalers.NewMockScaler(ctrl)
		scaler.EXPECT().GetMetricsAndActivity(gomock.Any(), gomock.Any()).Return([]external_metrics.ExternalMetricValue{}, false, errors.New("some error"))
		scaler.EXPECT().Close(gomock.Any())
		return scaler, &scalersconfig.ScalerConfig{}, nil
	}
	failingScaler := mock_scalers.NewMockScaler(ctrl)
	failingScaler.EXPECT().GetMetricSpecForScaling(gomock.Any()).Return(metricsSpecs)
	failingScaler.EXPECT().GetMetricsAndActivity(gomock.Any(), gomock.Any()).Return([]external_metrics.ExternalMetricValue{}, false, errors.New("some error"))
	failingScaler.EXPECT().Close(gomock.Any())

	scaledObject := kedav1alpha1.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: kedav1alpha1.ScaledObjectSpec{
			ScaleTargetRef: &kedav1alpha1.ScaleTarget{
				Name: "test",
			},
		},
		Status: kedav1alpha1.ScaledObjectStatus{
			ExternalMetricNames: []string{"metric-name"},
		},
	}

	scalers := []cache.ScalerBuilder{{
		Scaler:  activeScaler,
		Factory: activeFactory,
	}, {
		Scaler:  failingScaler,
		Factory: failingFactory,
	}}

	scalerCache := cache.ScalersCache{
		Scalers:  scalers,
		Recorder: recorder,
	}

	caches := map[string]*cache.ScalersCache{}
	caches[scaledObject.GenerateIdentifier()] = &scalerCache

	sh := scaleHandler{
		client:                   mockClient,
		scaleLoopContexts:        &sync.Map{},
		scaleExecutor:            mockExecutor,
		globalHTTPTimeout:        time.Duration(1000),
		recorder:                 recorder,
		scalerCaches:             caches,
		scalerCachesLock:         &sync.RWMutex{},
		scaledObjectsMetricCache: metricscache.NewMetricsCache(),
		rawMetricsSubscriptions:  map[string]*RawMetricSubscriptions{},
		metricToSubscriptions:    map[metricMeta][]*RawMetricSubscriptions{},
		subsLock:                 &sync.RWMutex{},
	}

	isActive, isError, _, activeTriggers, _, _ := sh.getScaledObjectState(context.TODO(), &scaledObject)
	scalerCache.Close(context.Background())

	assert.Equal(t, true, isActive)
	assert.Equal(t, true, isError)
	assert.Equal(t, []string{"metric-name"}, activeTriggers)
}

func TestGetScaledObjectStateRecordsResourceScalerActiveMetric(t *testing.T) {
	promMetricsCollectorOnce.Do(func() {
		metricscollector.NewMetricsCollectors(true, false)
	})

	ctrl := gomock.NewController(t)
	recorder := events.NewFakeRecorder(1)
	scaler := mock_scalers.NewMockScaler(ctrl)

	metricSpecs := []v2.MetricSpec{{
		Type: v2.ResourceMetricSourceType,
		Resource: &v2.ResourceMetricSource{
			Name: v1.ResourceCPU,
		},
	}}

	scaler.EXPECT().GetMetricSpecForScaling(gomock.Any()).Return(metricSpecs)
	scaler.EXPECT().Close(gomock.Any())

	scaledObject := kedav1alpha1.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "resource-metric-test",
			Namespace: "resource-metric-namespace",
		},
		Spec: kedav1alpha1.ScaledObjectSpec{
			ScaleTargetRef: &kedav1alpha1.ScaleTarget{
				Name: "test",
			},
			Triggers: []kedav1alpha1.ScaleTriggers{{
				Type: "cpu",
			}},
		},
	}

	scalerCache := cache.ScalersCache{
		Scalers: []cache.ScalerBuilder{{
			Scaler: scaler,
			ScalerConfig: scalersconfig.ScalerConfig{
				TriggerName: "cpu",
			},
		}},
		Recorder: recorder,
	}

	caches := map[string]*cache.ScalersCache{
		scaledObject.GenerateIdentifier(): &scalerCache,
	}

	sh := scaleHandler{
		scaleLoopContexts:        &sync.Map{},
		globalHTTPTimeout:        time.Duration(1000),
		recorder:                 recorder,
		scalerCaches:             caches,
		scalerCachesLock:         &sync.RWMutex{},
		scaledObjectsMetricCache: metricscache.NewMetricsCache(),
		rawMetricsSubscriptions:  map[string]*RawMetricSubscriptions{},
		metricToSubscriptions:    map[metricMeta][]*RawMetricSubscriptions{},
		subsLock:                 &sync.RWMutex{},
	}

	isActive, isError, _, activeTriggers, _, err := sh.getScaledObjectState(context.Background(), &scaledObject)
	scalerCache.Close(context.Background())

	assert.NoError(t, err)
	assert.True(t, isActive)
	assert.False(t, isError)
	assert.Empty(t, activeTriggers)
	assertPromMetricWithLabels(t, "keda_scaler_active", map[string]string{
		"namespace":    "resource-metric-namespace",
		"scaledObject": "resource-metric-test",
		"scaler":       "cpu",
		"triggerIndex": "0",
		"metric":       "cpu",
		"type":         "scaledobject",
	}, 1)
}

func TestGetScaledObjectStateSkipsResourceScalerActiveMetricWithModifiers(t *testing.T) {
	promMetricsCollectorOnce.Do(func() {
		metricscollector.NewMetricsCollectors(true, false)
	})

	ctrl := gomock.NewController(t)
	recorder := events.NewFakeRecorder(1)
	scaler := mock_scalers.NewMockScaler(ctrl)

	metricSpecs := []v2.MetricSpec{{
		Type: v2.ResourceMetricSourceType,
		Resource: &v2.ResourceMetricSource{
			Name: v1.ResourceCPU,
		},
	}}

	scaler.EXPECT().GetMetricSpecForScaling(gomock.Any()).Return(metricSpecs)
	scaler.EXPECT().Close(gomock.Any())

	scaledObject := kedav1alpha1.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "resource-modifier-metric-test",
			Namespace: "resource-modifier-metric-namespace",
		},
		Spec: kedav1alpha1.ScaledObjectSpec{
			ScaleTargetRef: &kedav1alpha1.ScaleTarget{
				Name: "test",
			},
			Advanced: &kedav1alpha1.AdvancedConfig{
				ScalingModifiers: kedav1alpha1.ScalingModifiers{
					Formula: "cpu",
					Target:  "1",
				},
			},
			Triggers: []kedav1alpha1.ScaleTriggers{{
				Type: "cpu",
				Name: "cpu",
			}},
		},
	}

	scalerCache := cache.ScalersCache{
		Scalers: []cache.ScalerBuilder{{
			Scaler: scaler,
			ScalerConfig: scalersconfig.ScalerConfig{
				TriggerName: "cpu",
			},
		}},
		Recorder: recorder,
	}

	caches := map[string]*cache.ScalersCache{
		scaledObject.GenerateIdentifier(): &scalerCache,
	}

	sh := scaleHandler{
		scaleLoopContexts:        &sync.Map{},
		globalHTTPTimeout:        time.Duration(1000),
		recorder:                 recorder,
		scalerCaches:             caches,
		scalerCachesLock:         &sync.RWMutex{},
		scaledObjectsMetricCache: metricscache.NewMetricsCache(),
		rawMetricsSubscriptions:  map[string]*RawMetricSubscriptions{},
		metricToSubscriptions:    map[metricMeta][]*RawMetricSubscriptions{},
		subsLock:                 &sync.RWMutex{},
	}

	isActive, isError, _, activeTriggers, _, err := sh.getScaledObjectState(context.Background(), &scaledObject)
	scalerCache.Close(context.Background())

	assert.NoError(t, err)
	assert.True(t, isActive)
	assert.False(t, isError)
	assert.Empty(t, activeTriggers)
	assertPromMetricWithLabelsNotFound(t, "keda_scaler_active", map[string]string{
		"namespace":    "resource-modifier-metric-namespace",
		"scaledObject": "resource-modifier-metric-test",
		"scaler":       "cpu",
		"triggerIndex": "0",
		"metric":       "cpu",
		"type":         "scaledobject",
	})
}

func assertPromMetricWithLabels(t *testing.T, name string, labels map[string]string, value float64) {
	t.Helper()

	metricFamilies, err := metrics.Registry.Gather()
	assert.NoError(t, err)

	for _, family := range metricFamilies {
		if family.GetName() != name {
			continue
		}

		for _, metric := range family.GetMetric() {
			if metric.GetGauge().GetValue() == value && hasLabels(metric.GetLabel(), labels) {
				return
			}
		}
	}

	t.Fatalf("metric %q with labels %#v and value %v not found", name, labels, value)
}

func assertPromMetricWithLabelsNotFound(t *testing.T, name string, labels map[string]string) {
	t.Helper()

	metricFamilies, err := metrics.Registry.Gather()
	assert.NoError(t, err)

	for _, family := range metricFamilies {
		if family.GetName() != name {
			continue
		}

		for _, metric := range family.GetMetric() {
			if hasLabels(metric.GetLabel(), labels) {
				t.Fatalf("metric %q with labels %#v found", name, labels)
			}
		}
	}
}

func hasLabels(pairs []*dto.LabelPair, expected map[string]string) bool {
	found := make(map[string]string, len(pairs))
	for _, pair := range pairs {
		found[pair.GetName()] = pair.GetValue()
	}

	for name, value := range expected {
		if found[name] != value {
			return false
		}
	}

	return true
}

func TestGetScaledJobState(t *testing.T) {
	metricName := "s0-queueLength"
	ctrl := gomock.NewController(t)
	recorder := events.NewFakeRecorder(1)
	// Keep the current behavior
	// Assme 1 trigger only
	scaledJobSingle := createScaledJob(1, 100, "") // testing default = max
	scalerCache := cache.ScalersCache{
		Scalers: []cache.ScalerBuilder{{
			Scaler: createScaler(ctrl, int64(20), int64(2), true, metricName),
			Factory: func() (scalers.Scaler, *scalersconfig.ScalerConfig, error) {
				return createScaler(ctrl, int64(20), int64(2), true, metricName), &scalersconfig.ScalerConfig{}, nil
			},
		}},
		Recorder: recorder,
	}

	caches := map[string]*cache.ScalersCache{}
	caches[scaledJobSingle.GenerateIdentifier()] = &scalerCache

	sh := scaleHandler{
		scaleLoopContexts:        &sync.Map{},
		globalHTTPTimeout:        time.Duration(1000),
		recorder:                 recorder,
		scalerCaches:             caches,
		scalerCachesLock:         &sync.RWMutex{},
		scaledObjectsMetricCache: metricscache.NewMetricsCache(),
		rawMetricsSubscriptions:  map[string]*RawMetricSubscriptions{},
		metricToSubscriptions:    map[metricMeta][]*RawMetricSubscriptions{},
		subsLock:                 &sync.RWMutex{},
	}
	// nosemgrep: context-todo
	isActive, isError, queueLength, maxValue, _ := sh.getScaledJobState(context.TODO(), scaledJobSingle)
	assert.Equal(t, true, isActive)
	assert.Equal(t, false, isError)
	assert.Equal(t, int64(20), queueLength)
	assert.Equal(t, int64(10), maxValue)
	scalerCache.Close(context.Background())

	// Test the valiation
	scalerTestDatam := []scalerTestData{
		newScalerTestData("s0-queueLength", 100, "max", 20, 1, true, 10, 2, true, 5, 3, true, 7, 4, false, true, false, 20, 20),
		newScalerTestData("queueLength", 100, "min", 20, 1, true, 10, 2, true, 5, 3, true, 7, 4, false, true, false, 5, 2),
		newScalerTestData("messageCount", 100, "avg", 20, 1, true, 10, 2, true, 5, 3, true, 7, 4, false, true, false, 12, 9),
		newScalerTestData("s3-messageCount", 100, "sum", 20, 1, true, 10, 2, true, 5, 3, true, 7, 4, false, true, false, 35, 27),
		newScalerTestData("s10-messageCount", 25, "sum", 20, 1, true, 10, 2, true, 5, 3, true, 7, 4, false, true, false, 35, 25),
	}

	for index, scalerTestData := range scalerTestDatam {
		scaledJob := createScaledJob(scalerTestData.MinReplicaCount, scalerTestData.MaxReplicaCount, scalerTestData.MultipleScalersCalculation)
		scalersToTest := []cache.ScalerBuilder{{
			Scaler: createScaler(ctrl, scalerTestData.Scaler1QueueLength, scalerTestData.Scaler1AverageValue, scalerTestData.Scaler1IsActive, scalerTestData.MetricName),
			Factory: func() (scalers.Scaler, *scalersconfig.ScalerConfig, error) {
				return createScaler(ctrl, scalerTestData.Scaler1QueueLength, scalerTestData.Scaler1AverageValue, scalerTestData.Scaler1IsActive, scalerTestData.MetricName), &scalersconfig.ScalerConfig{}, nil
			},
		}, {
			Scaler: createScaler(ctrl, scalerTestData.Scaler2QueueLength, scalerTestData.Scaler2AverageValue, scalerTestData.Scaler2IsActive, scalerTestData.MetricName),
			Factory: func() (scalers.Scaler, *scalersconfig.ScalerConfig, error) {
				return createScaler(ctrl, scalerTestData.Scaler2QueueLength, scalerTestData.Scaler2AverageValue, scalerTestData.Scaler2IsActive, scalerTestData.MetricName), &scalersconfig.ScalerConfig{}, nil
			},
		}, {
			Scaler: createScaler(ctrl, scalerTestData.Scaler3QueueLength, scalerTestData.Scaler3AverageValue, scalerTestData.Scaler3IsActive, scalerTestData.MetricName),
			Factory: func() (scalers.Scaler, *scalersconfig.ScalerConfig, error) {
				return createScaler(ctrl, scalerTestData.Scaler3QueueLength, scalerTestData.Scaler3AverageValue, scalerTestData.Scaler3IsActive, scalerTestData.MetricName), &scalersconfig.ScalerConfig{}, nil
			},
		}, {
			Scaler: createScaler(ctrl, scalerTestData.Scaler4QueueLength, scalerTestData.Scaler4AverageValue, scalerTestData.Scaler4IsActive, scalerTestData.MetricName),
			Factory: func() (scalers.Scaler, *scalersconfig.ScalerConfig, error) {
				return createScaler(ctrl, scalerTestData.Scaler4QueueLength, scalerTestData.Scaler4AverageValue, scalerTestData.Scaler4IsActive, scalerTestData.MetricName), &scalersconfig.ScalerConfig{}, nil
			},
		}}

		scalerCache = cache.ScalersCache{
			Scalers:  scalersToTest,
			Recorder: recorder,
		}

		caches = map[string]*cache.ScalersCache{}
		caches[scaledJobSingle.GenerateIdentifier()] = &scalerCache

		sh = scaleHandler{
			scaleLoopContexts:        &sync.Map{},
			globalHTTPTimeout:        time.Duration(1000),
			recorder:                 recorder,
			scalerCaches:             caches,
			scalerCachesLock:         &sync.RWMutex{},
			scaledObjectsMetricCache: metricscache.NewMetricsCache(),
			rawMetricsSubscriptions:  map[string]*RawMetricSubscriptions{},
			metricToSubscriptions:    map[metricMeta][]*RawMetricSubscriptions{},
			subsLock:                 &sync.RWMutex{},
		}
		fmt.Printf("index: %d", index)
		// nosemgrep: context-todo
		isActive, isError, queueLength, maxValue, _ := sh.getScaledJobState(context.TODO(), scaledJob)
		//	assert.Equal(t, 5, index)
		assert.Equal(t, scalerTestData.ResultIsActive, isActive)
		assert.Equal(t, scalerTestData.ResultIsError, isError)
		assert.Equal(t, scalerTestData.ResultQueueLength, queueLength)
		assert.Equal(t, scalerTestData.ResultMaxValue, maxValue)
		scalerCache.Close(context.Background())
	}
}

func TestGetScaledJobStateIfQueueEmptyButMinReplicaCountGreaterZero(t *testing.T) {
	metricName := "s0-queueLength"
	ctrl := gomock.NewController(t)
	recorder := events.NewFakeRecorder(1)
	// Keep the current behavior
	// Assme 1 trigger only
	scaledJobSingle := createScaledJob(1, 100, "") // testing default = max
	scalerSingle := []cache.ScalerBuilder{{
		Scaler: createScaler(ctrl, int64(0), int64(1), true, metricName),
		Factory: func() (scalers.Scaler, *scalersconfig.ScalerConfig, error) {
			return createScaler(ctrl, int64(0), int64(1), true, metricName), &scalersconfig.ScalerConfig{}, nil
		},
	}}

	scalerCache := cache.ScalersCache{
		Scalers:  scalerSingle,
		Recorder: recorder,
	}

	caches := map[string]*cache.ScalersCache{}
	caches[scaledJobSingle.GenerateIdentifier()] = &scalerCache

	sh := scaleHandler{
		scaleLoopContexts:        &sync.Map{},
		globalHTTPTimeout:        time.Duration(1000),
		recorder:                 recorder,
		scalerCaches:             caches,
		scalerCachesLock:         &sync.RWMutex{},
		scaledObjectsMetricCache: metricscache.NewMetricsCache(),
		rawMetricsSubscriptions:  map[string]*RawMetricSubscriptions{},
		metricToSubscriptions:    map[metricMeta][]*RawMetricSubscriptions{},
		subsLock:                 &sync.RWMutex{},
	}

	// nosemgrep: context-todo
	isActive, isError, queueLength, maxValue, _ := sh.getScaledJobState(context.TODO(), scaledJobSingle)
	assert.Equal(t, true, isActive)
	assert.Equal(t, false, isError)
	assert.Equal(t, int64(0), queueLength)
	assert.Equal(t, int64(0), maxValue)
	scalerCache.Close(context.Background())
}

func newScalerTestData(
	metricName string,
	maxReplicaCount int,
	multipleScalersCalculation string,
	scaler1QueueLength, //nolint:unparam
	scaler1AverageValue int, //nolint:unparam
	scaler1IsActive bool, //nolint:unparam
	scaler2QueueLength, //nolint:unparam
	scaler2AverageValue int, //nolint:unparam
	scaler2IsActive bool, //nolint:unparam
	scaler3QueueLength, //nolint:unparam
	scaler3AverageValue int, //nolint:unparam
	scaler3IsActive bool, //nolint:unparam
	scaler4QueueLength, //nolint:unparam
	scaler4AverageValue int, //nolint:unparam
	scaler4IsActive bool, //nolint:unparam
	resultIsActive bool, //nolint:unparam
	resultIsError bool, //nolint:unparam
	resultQueueLength,
	resultMaxLength int) scalerTestData {
	return scalerTestData{
		MetricName:                 metricName,
		MaxReplicaCount:            int32(maxReplicaCount),
		MultipleScalersCalculation: multipleScalersCalculation,
		Scaler1QueueLength:         int64(scaler1QueueLength),
		Scaler1AverageValue:        int64(scaler1AverageValue),
		Scaler1IsActive:            scaler1IsActive,
		Scaler2QueueLength:         int64(scaler2QueueLength),
		Scaler2AverageValue:        int64(scaler2AverageValue),
		Scaler2IsActive:            scaler2IsActive,
		Scaler3QueueLength:         int64(scaler3QueueLength),
		Scaler3AverageValue:        int64(scaler3AverageValue),
		Scaler3IsActive:            scaler3IsActive,
		Scaler4QueueLength:         int64(scaler4QueueLength),
		Scaler4AverageValue:        int64(scaler4AverageValue),
		Scaler4IsActive:            scaler4IsActive,
		ResultIsActive:             resultIsActive,
		ResultIsError:              resultIsError,
		ResultQueueLength:          int64(resultQueueLength),
		ResultMaxValue:             int64(resultMaxLength),
	}
}

type scalerTestData struct {
	MetricName                 string
	MaxReplicaCount            int32
	MultipleScalersCalculation string
	Scaler1QueueLength         int64
	Scaler1AverageValue        int64
	Scaler1IsActive            bool
	Scaler2QueueLength         int64
	Scaler2AverageValue        int64
	Scaler2IsActive            bool
	Scaler3QueueLength         int64
	Scaler3AverageValue        int64
	Scaler3IsActive            bool
	Scaler4QueueLength         int64
	Scaler4AverageValue        int64
	Scaler4IsActive            bool
	ResultIsActive             bool
	ResultIsError              bool
	ResultQueueLength          int64
	ResultMaxValue             int64
	MinReplicaCount            int32
}

func createScaledJob(minReplicaCount int32, maxReplicaCount int32, multipleScalersCalculation string) *kedav1alpha1.ScaledJob {
	if multipleScalersCalculation != "" {
		return &kedav1alpha1.ScaledJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "test",
			},
			Spec: kedav1alpha1.ScaledJobSpec{
				MinReplicaCount: &minReplicaCount,
				MaxReplicaCount: &maxReplicaCount,
				ScalingStrategy: kedav1alpha1.ScalingStrategy{
					MultipleScalersCalculation: multipleScalersCalculation,
				},
				JobTargetRef: &batchv1.JobSpec{
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test",
							Namespace: "test",
						},
					},
				},
				EnvSourceContainerName: "test",
			},
		}
	}
	return &kedav1alpha1.ScaledJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: kedav1alpha1.ScaledJobSpec{
			MinReplicaCount: &minReplicaCount,
			MaxReplicaCount: &maxReplicaCount,
			JobTargetRef: &batchv1.JobSpec{
				Template: v1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
				},
			},
			EnvSourceContainerName: "test",
		},
	}
}

func createScaler(ctrl *gomock.Controller, queueLength int64, averageValue int64, isActive bool, metricName string) *mock_scalers.MockScaler {
	scaler := mock_scalers.NewMockScaler(ctrl)
	metricsSpecs := []v2.MetricSpec{createMetricSpec(averageValue, metricName)}

	metrics := []external_metrics.ExternalMetricValue{
		{
			MetricName: metricName,
			Value:      *resource.NewQuantity(queueLength, resource.DecimalSI),
		},
	}
	scaler.EXPECT().GetMetricSpecForScaling(gomock.Any()).Return(metricsSpecs)
	scaler.EXPECT().GetMetricsAndActivity(gomock.Any(), gomock.Any()).Return(metrics, isActive, nil)
	scaler.EXPECT().Close(gomock.Any())
	return scaler
}

// -----------------------------------------------------------------------------
// test for scalingModifiers formula
// -----------------------------------------------------------------------------

const triggerName1 = "trigger_one"
const triggerName2 = "trigger_two"
const metricName1 = "metric_one"
const metricName2 = "metric_two"

func TestScalingModifiersFormula(t *testing.T) {
	scaledObjectName := testNameGlobal
	scaledObjectNamespace := testNamespaceGlobal
	compositeMetricName := compositeMetricNameGlobal

	ctrl := gomock.NewController(t)
	recorder := events.NewFakeRecorder(1)
	mockClient := mock_client.NewMockClient(ctrl)
	mockExecutor := mock_executor.NewMockScaleExecutor(ctrl)

	metricsSpecs1 := []v2.MetricSpec{createMetricSpec(2, metricName1)}
	metricsSpecs2 := []v2.MetricSpec{createMetricSpec(5, metricName2)}
	metricValue1 := scalers.GenerateMetricInMili(metricName1, float64(2))
	metricValue2 := scalers.GenerateMetricInMili(metricName2, float64(5))

	scaler1 := mock_scalers.NewMockScaler(ctrl)
	scaler2 := mock_scalers.NewMockScaler(ctrl)
	// dont use cached metrics
	scalerConfig1 := scalersconfig.ScalerConfig{TriggerUseCachedMetrics: false, TriggerName: triggerName1, TriggerIndex: 0}
	scalerConfig2 := scalersconfig.ScalerConfig{TriggerUseCachedMetrics: false, TriggerName: triggerName2, TriggerIndex: 1}
	factory1 := func() (scalers.Scaler, *scalersconfig.ScalerConfig, error) {
		return scaler1, &scalerConfig1, nil
	}
	factory2 := func() (scalers.Scaler, *scalersconfig.ScalerConfig, error) {
		return scaler2, &scalerConfig2, nil
	}

	scaledObject := kedav1alpha1.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      scaledObjectName,
			Namespace: scaledObjectNamespace,
		},
		Spec: kedav1alpha1.ScaledObjectSpec{
			ScaleTargetRef: &kedav1alpha1.ScaleTarget{
				Name: "test",
			},
			Advanced: &kedav1alpha1.AdvancedConfig{
				ScalingModifiers: kedav1alpha1.ScalingModifiers{
					Target:  "2",
					Formula: fmt.Sprintf("%s + %s", triggerName1, triggerName2),
				},
			},
			Triggers: []kedav1alpha1.ScaleTriggers{
				{Name: triggerName1, Type: "fake_trig1"},
				{Name: triggerName2, Type: "fake_trig2"},
			},
		},
		Status: kedav1alpha1.ScaledObjectStatus{
			ScaleTargetGVKR: &kedav1alpha1.GroupVersionKindResource{
				Group: "apps",
				Kind:  "Deployment",
			},
			ExternalMetricNames: []string{metricName1, metricName2},
		},
	}

	// formula is compiled and cached
	compiledFormula, err := expr.Compile(scaledObject.Spec.Advanced.ScalingModifiers.Formula)
	assert.Equal(t, err, nil)

	scalerCache := cache.ScalersCache{
		ScaledObject: &scaledObject,
		Scalers: []cache.ScalerBuilder{{
			Scaler:       scaler1,
			ScalerConfig: scalerConfig1,
			Factory:      factory1,
		},
			{
				Scaler:       scaler2,
				ScalerConfig: scalerConfig2,
				Factory:      factory2,
			},
		},
		Recorder:        recorder,
		CompiledFormula: compiledFormula,
	}

	caches := map[string]*cache.ScalersCache{}
	caches[scaledObject.GenerateIdentifier()] = &scalerCache

	sh := scaleHandler{
		client:                   mockClient,
		scaleLoopContexts:        &sync.Map{},
		scaleExecutor:            mockExecutor,
		globalHTTPTimeout:        time.Duration(1000),
		recorder:                 recorder,
		scalerCaches:             caches,
		scalerCachesLock:         &sync.RWMutex{},
		scaledObjectsMetricCache: metricscache.NewMetricsCache(),
		rawMetricsSubscriptions:  map[string]*RawMetricSubscriptions{},
		metricToSubscriptions:    map[metricMeta][]*RawMetricSubscriptions{},
		subsLock:                 &sync.RWMutex{},
	}

	mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	scaler1.EXPECT().GetMetricSpecForScaling(gomock.Any()).Return(metricsSpecs1)
	scaler2.EXPECT().GetMetricSpecForScaling(gomock.Any()).Return(metricsSpecs2)
	scaler1.EXPECT().GetMetricsAndActivity(gomock.Any(), gomock.Any()).Return([]external_metrics.ExternalMetricValue{metricValue1, metricValue2}, true, nil)
	scaler2.EXPECT().GetMetricsAndActivity(gomock.Any(), gomock.Any()).Return([]external_metrics.ExternalMetricValue{metricValue1, metricValue2}, true, nil)
	mockExecutor.EXPECT().RequestScale(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())
	expectHandleResult(mockClient)
	sh.checkScalers(context.TODO(), &scaledObject, &sync.RWMutex{})

	expectNoStatusPatch(ctrl)

	scaler1.EXPECT().GetMetricSpecForScaling(gomock.Any()).Return(metricsSpecs1)
	scaler2.EXPECT().GetMetricSpecForScaling(gomock.Any()).Return(metricsSpecs2)
	scaler1.EXPECT().GetMetricsAndActivity(gomock.Any(), gomock.Any()).Return([]external_metrics.ExternalMetricValue{metricValue1, metricValue2}, true, nil)
	scaler2.EXPECT().GetMetricsAndActivity(gomock.Any(), gomock.Any()).Return([]external_metrics.ExternalMetricValue{metricValue1, metricValue2}, true, nil)
	metrics, err := sh.GetScaledObjectMetrics(context.TODO(), scaledObjectName, scaledObjectNamespace, compositeMetricName)
	assert.Nil(t, err)
	assert.Equal(t, float64(7), metrics.Items[0].Value.AsApproximateFloat64())
}

// createMetricSpec creates MetricSpec for given metric name and target value.
func createMetricSpec(averageValue int64, metricName string) v2.MetricSpec {
	qty := resource.NewQuantity(averageValue, resource.DecimalSI)
	return v2.MetricSpec{
		External: &v2.ExternalMetricSource{
			Target: v2.MetricTarget{
				AverageValue: qty,
			},
			Metric: v2.MetricIdentifier{
				Name: metricName,
			},
		},
	}
}

func expectNoStatusPatch(ctrl *gomock.Controller) {
	statusWriter := mock_client.NewMockStatusWriter(ctrl)
	statusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
}

// expectHandleResult sets up mock expectations for handleResult:
// a Get call to fetch the latest object. The status patch is only issued when the
// computed status differs from the fetched object, so we don't expect it here
// (mock executor returns a zero ScaleResult which causes no status changes).
func expectHandleResult(mockClient *mock_client.MockClient) {
	mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
}

func TestHandleResult_PatchesWhenConditionsChange(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)
	statusWriter := mock_client.NewMockStatusWriter(ctrl)

	sh := scaleHandler{client: mockClient}

	existingSO := kedav1alpha1.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "ns"},
		Status: kedav1alpha1.ScaledObjectStatus{
			Conditions: kedav1alpha1.Conditions{
				{Type: kedav1alpha1.ConditionReady, Status: metav1.ConditionTrue, Reason: "old"},
			},
		},
	}

	mockClient.EXPECT().Get(gomock.Any(), types.NamespacedName{Name: "test", Namespace: "ns"}, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ types.NamespacedName, obj *kedav1alpha1.ScaledObject, _ ...interface{}) error {
			*obj = *existingSO.DeepCopy()
			return nil
		})
	mockClient.EXPECT().Status().Return(statusWriter)
	statusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	result := executor.ScaleResult{
		Conditions: kedav1alpha1.Conditions{},
	}
	result.Conditions.SetReadyCondition(metav1.ConditionFalse, "SomeError", "something went wrong")

	sh.handleResult(context.TODO(), &existingSO, result)
}

func TestHandleResult_SkipsPatchWhenUnchanged(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)

	sh := scaleHandler{client: mockClient}

	existingSO := kedav1alpha1.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "ns"},
	}

	mockClient.EXPECT().Get(gomock.Any(), types.NamespacedName{Name: "test", Namespace: "ns"}, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ types.NamespacedName, obj *kedav1alpha1.ScaledObject, _ ...interface{}) error {
			*obj = *existingSO.DeepCopy()
			return nil
		})
	// no Status().Patch expectation - if handleResult tries to patch, the test will fail

	result := executor.ScaleResult{}
	sh.handleResult(context.TODO(), &existingSO, result)
}

func TestHandleResult_SetsLastActiveTime(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)
	statusWriter := mock_client.NewMockStatusWriter(ctrl)

	sh := scaleHandler{client: mockClient}

	existingSO := kedav1alpha1.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "ns"},
	}

	now := metav1.Now()
	var patchedObj *kedav1alpha1.ScaledObject

	mockClient.EXPECT().Get(gomock.Any(), types.NamespacedName{Name: "test", Namespace: "ns"}, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ types.NamespacedName, obj *kedav1alpha1.ScaledObject, _ ...interface{}) error {
			*obj = *existingSO.DeepCopy()
			return nil
		})
	mockClient.EXPECT().Status().Return(statusWriter)
	statusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, obj *kedav1alpha1.ScaledObject, _ interface{}, _ ...interface{}) error {
			patchedObj = obj
			return nil
		})

	result := executor.ScaleResult{
		LastActiveTime: &now,
	}
	sh.handleResult(context.TODO(), &existingSO, result)

	assert.NotNil(t, patchedObj)
	assert.Equal(t, &now, patchedObj.Status.LastActiveTime)
}

func TestHandleResult_TriggersActivityUpdatesAndRemovals(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)
	statusWriter := mock_client.NewMockStatusWriter(ctrl)

	sh := scaleHandler{client: mockClient}

	// baseline object (obj) has triggers a, b, c - all active
	baselineSO := kedav1alpha1.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "ns"},
		Status: kedav1alpha1.ScaledObjectStatus{
			TriggersActivity: map[string]kedav1alpha1.TriggerActivityStatus{
				"trigger-a": {IsActive: true},
				"trigger-b": {IsActive: true},
				"trigger-c": {IsActive: true},
			},
		},
	}

	var patchedObj *kedav1alpha1.ScaledObject

	mockClient.EXPECT().Get(gomock.Any(), types.NamespacedName{Name: "test", Namespace: "ns"}, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ types.NamespacedName, obj *kedav1alpha1.ScaledObject, _ ...interface{}) error {
			*obj = *baselineSO.DeepCopy()
			return nil
		})
	mockClient.EXPECT().Status().Return(statusWriter)
	statusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, obj *kedav1alpha1.ScaledObject, _ interface{}, _ ...interface{}) error {
			patchedObj = obj
			return nil
		})

	// standard scaler result: trigger-c removed, trigger-b deactivated
	result := executor.ScaleResult{
		TriggersActivity: map[string]kedav1alpha1.TriggerActivityStatus{
			"trigger-a": {IsActive: true},
			"trigger-b": {IsActive: false},
		},
	}
	sh.handleResult(context.TODO(), &baselineSO, result)

	assert.NotNil(t, patchedObj)
	assert.Len(t, patchedObj.Status.TriggersActivity, 2)
	assert.True(t, patchedObj.Status.TriggersActivity["trigger-a"].IsActive)
	assert.False(t, patchedObj.Status.TriggersActivity["trigger-b"].IsActive)
	_, exists := patchedObj.Status.TriggersActivity["trigger-c"]
	assert.False(t, exists, "trigger-c should be removed")
}

func TestHandleResult_PushScalerDeltaMerge(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)
	statusWriter := mock_client.NewMockStatusWriter(ctrl)

	sh := scaleHandler{client: mockClient}

	// baseline object: trigger-a active, trigger-b inactive, trigger-c active
	baselineSO := kedav1alpha1.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "ns"},
		Status: kedav1alpha1.ScaledObjectStatus{
			TriggersActivity: map[string]kedav1alpha1.TriggerActivityStatus{
				"trigger-a": {IsActive: true},
				"trigger-b": {IsActive: false},
				"trigger-c": {IsActive: true},
			},
		},
	}

	var patchedObj *kedav1alpha1.ScaledObject

	mockClient.EXPECT().Get(gomock.Any(), types.NamespacedName{Name: "test", Namespace: "ns"}, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ types.NamespacedName, obj *kedav1alpha1.ScaledObject, _ ...interface{}) error {
			*obj = *baselineSO.DeepCopy()
			return nil
		})
	mockClient.EXPECT().Status().Return(statusWriter)
	statusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, obj *kedav1alpha1.ScaledObject, _ interface{}, _ ...interface{}) error {
			patchedObj = obj
			return nil
		})

	// push scaler result: full map copied from baseline + trigger-b activated
	// (this is what getTriggersActivity produces for push scalers)
	result := executor.ScaleResult{
		TriggersActivity: map[string]kedav1alpha1.TriggerActivityStatus{
			"trigger-a": {IsActive: true},
			"trigger-b": {IsActive: true}, // changed by push scaler
			"trigger-c": {IsActive: true},
		},
	}
	sh.handleResult(context.TODO(), &baselineSO, result)

	assert.NotNil(t, patchedObj)
	assert.Len(t, patchedObj.Status.TriggersActivity, 3)
	assert.True(t, patchedObj.Status.TriggersActivity["trigger-a"].IsActive, "trigger-a preserved")
	assert.True(t, patchedObj.Status.TriggersActivity["trigger-b"].IsActive, "trigger-b updated by push scaler")
	assert.True(t, patchedObj.Status.TriggersActivity["trigger-c"].IsActive, "trigger-c preserved")
}

func startInFlightCloseRace(t *testing.T, scalerCache *cache.ScalersCache, scaler *mock_scalers.MockScaler) {
	t.Helper()

	release := make(chan struct{})
	entered := make(chan struct{})
	var releaseOnce sync.Once
	releaseFn := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(releaseFn)

	scaler.EXPECT().GetMetricsAndActivity(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ string) ([]external_metrics.ExternalMetricValue, bool, error) {
			close(entered)
			<-release
			return nil, false, nil
		},
	)
	scaler.EXPECT().Close(gomock.Any())

	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		_, _, _, err := scalerCache.GetMetricsAndActivityForScaler(context.Background(), 0, "blocker")
		if err != nil {
			t.Errorf("blocking reader got unexpected error: %v", err)
		}
	}()
	t.Cleanup(func() {
		releaseFn()
		<-readerDone
	})

	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("blocking reader did not enter the scaler")
	}

	closeReturned := make(chan struct{})
	go func() {
		defer close(closeReturned)
		scalerCache.Close(context.Background())
	}()
	t.Cleanup(func() {
		releaseFn()
		<-closeReturned
	})

	// -1 polls without invoking the scaler.
	deadline := time.Now().Add(2 * time.Second)
	for {
		_, _, _, err := scalerCache.GetMetricsAndActivityForScaler(context.Background(), -1, "")
		if errors.Is(err, cache.ErrCacheClosed) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("cache did not transition to closed state in time")
		}
		time.Sleep(time.Millisecond)
	}
}

func TestGetScaledJobMetrics_CacheClosedIsBenign(t *testing.T) {
	ctrl := gomock.NewController(t)
	recorder := events.NewFakeRecorder(10)
	metricName := "s0-queueLength"
	metricsSpecs := []v2.MetricSpec{createMetricSpec(10, metricName)}

	scaledJob := createScaledJob(1, 100, "")

	scaler := mock_scalers.NewMockScaler(ctrl)
	scaler.EXPECT().GetMetricSpecForScaling(gomock.Any()).Return(metricsSpecs)
	factory := func() (scalers.Scaler, *scalersconfig.ScalerConfig, error) {
		return scaler, &scalersconfig.ScalerConfig{}, nil
	}

	scalerCache := &cache.ScalersCache{
		Scalers:  []cache.ScalerBuilder{{Scaler: scaler, Factory: factory}},
		Recorder: recorder,
	}

	startInFlightCloseRace(t, scalerCache, scaler)

	caches := map[string]*cache.ScalersCache{
		scaledJob.GenerateIdentifier(): scalerCache,
	}

	sh := scaleHandler{
		scaleLoopContexts:        &sync.Map{},
		globalHTTPTimeout:        time.Duration(1000),
		recorder:                 recorder,
		scalerCaches:             caches,
		scalerCachesLock:         &sync.RWMutex{},
		scaledObjectsMetricCache: metricscache.NewMetricsCache(),
		rawMetricsSubscriptions:  map[string]*RawMetricSubscriptions{},
		metricToSubscriptions:    map[metricMeta][]*RawMetricSubscriptions{},
		subsLock:                 &sync.RWMutex{},
	}

	_, isError, activeTriggers := sh.getScaledJobMetrics(context.Background(), scaledJob)

	assert.False(t, isError)
	assert.Empty(t, activeTriggers)
	select {
	case ev := <-recorder.Events:
		t.Fatalf("unexpected event: %s", ev)
	default:
	}
}

func TestGetScaledObjectState_CacheClosedIsBenign(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)
	mockExecutor := mock_executor.NewMockScaleExecutor(ctrl)
	recorder := events.NewFakeRecorder(10)

	scaler := mock_scalers.NewMockScaler(ctrl)
	factory := func() (scalers.Scaler, *scalersconfig.ScalerConfig, error) {
		return scaler, &scalersconfig.ScalerConfig{}, nil
	}

	scaledObject := kedav1alpha1.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"},
		Spec: kedav1alpha1.ScaledObjectSpec{
			ScaleTargetRef: &kedav1alpha1.ScaleTarget{Name: "test"},
		},
	}

	scalerCache := &cache.ScalersCache{
		Scalers:  []cache.ScalerBuilder{{Scaler: scaler, Factory: factory}},
		Recorder: recorder,
	}

	startInFlightCloseRace(t, scalerCache, scaler)

	key := scaledObject.GenerateIdentifier()
	caches := map[string]*cache.ScalersCache{key: scalerCache}

	sh := scaleHandler{
		client:                   mockClient,
		scaleLoopContexts:        &sync.Map{},
		scaleExecutor:            mockExecutor,
		globalHTTPTimeout:        time.Duration(1000),
		recorder:                 recorder,
		scalerCaches:             caches,
		scalerCachesLock:         &sync.RWMutex{},
		scaledObjectsMetricCache: metricscache.NewMetricsCache(),
		rawMetricsSubscriptions:  map[string]*RawMetricSubscriptions{},
		metricToSubscriptions:    map[metricMeta][]*RawMetricSubscriptions{},
		subsLock:                 &sync.RWMutex{},
	}

	_, isError, _, _, _, err := sh.getScaledObjectState(context.Background(), &scaledObject)
	assert.NoError(t, err)
	assert.False(t, isError)

	sh.scalerCachesLock.RLock()
	_, stillPresent := sh.scalerCaches[key]
	sh.scalerCachesLock.RUnlock()
	assert.True(t, stillPresent, "ClearScalersCache should not have run")
}

func TestHandleResult_DeltaDoesNotOverwriteConcurrentChanges(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)
	statusWriter := mock_client.NewMockStatusWriter(ctrl)

	sh := scaleHandler{client: mockClient}

	// baseline object (stale snapshot from before the lock, used by executor):
	// trigger-a=false, trigger-b=false
	baselineSO := kedav1alpha1.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "ns"},
		Status: kedav1alpha1.ScaledObjectStatus{
			TriggersActivity: map[string]kedav1alpha1.TriggerActivityStatus{
				"trigger-a": {IsActive: false},
				"trigger-b": {IsActive: false},
			},
		},
	}

	// the fresh object returned by Get (concurrent polling loop updated trigger-a to true)
	freshSO := kedav1alpha1.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "ns"},
		Status: kedav1alpha1.ScaledObjectStatus{
			TriggersActivity: map[string]kedav1alpha1.TriggerActivityStatus{
				"trigger-a": {IsActive: true},
				"trigger-b": {IsActive: false},
			},
		},
	}

	var patchedObj *kedav1alpha1.ScaledObject

	mockClient.EXPECT().Get(gomock.Any(), types.NamespacedName{Name: "test", Namespace: "ns"}, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ types.NamespacedName, obj *kedav1alpha1.ScaledObject, _ ...interface{}) error {
			*obj = *freshSO.DeepCopy()
			return nil
		})
	mockClient.EXPECT().Status().Return(statusWriter)
	statusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, obj *kedav1alpha1.ScaledObject, _ interface{}, _ ...interface{}) error {
			patchedObj = obj
			return nil
		})

	// push scaler result: full map copied from baseline + trigger-b activated
	// trigger-a is false (stale), trigger-b is true (changed by push scaler)
	result := executor.ScaleResult{
		TriggersActivity: map[string]kedav1alpha1.TriggerActivityStatus{
			"trigger-a": {IsActive: false}, // stale copy from baseline
			"trigger-b": {IsActive: true},  // changed by push scaler
		},
	}
	sh.handleResult(context.TODO(), &baselineSO, result)

	assert.NotNil(t, patchedObj)
	// delta = diff(result, baseline) = {trigger-b: true} (trigger-a unchanged from baseline)
	// applied to fresh current: trigger-a stays true (concurrent update preserved), trigger-b set to true
	assert.True(t, patchedObj.Status.TriggersActivity["trigger-a"].IsActive, "concurrent update to trigger-a must be preserved")
	assert.True(t, patchedObj.Status.TriggersActivity["trigger-b"].IsActive, "trigger-b updated by push scaler")
}

type metricSpecWatcherTestPushScaler struct {
	specCh chan []v2.MetricSpec
}

func (s *metricSpecWatcherTestPushScaler) GetMetricsAndActivity(context.Context, string) ([]external_metrics.ExternalMetricValue, bool, error) {
	return nil, false, nil
}

func (s *metricSpecWatcherTestPushScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	return nil
}

func (s *metricSpecWatcherTestPushScaler) Close(context.Context) error {
	return nil
}

func (s *metricSpecWatcherTestPushScaler) Run(context.Context, chan<- bool) {}

func (s *metricSpecWatcherTestPushScaler) MetricSpecChan() <-chan []v2.MetricSpec {
	return s.specCh
}

func TestWatchMetricSpecUpdates_UsesLatestCacheAfterInvalidation(t *testing.T) {
	const generation = int64(3)
	uid := types.UID("so-uid-1")
	scaledObject := &kedav1alpha1.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{
			Name:       testNameGlobal,
			Namespace:  testNamespaceGlobal,
			UID:        uid,
			Generation: generation,
		},
	}
	key := scaledObject.GenerateIdentifier()

	streamer := &metricSpecWatcherTestPushScaler{specCh: make(chan []v2.MetricSpec, 1)}
	oldCache := &cache.ScalersCache{
		ScaledObject:             scaledObject,
		ScalableObjectGeneration: generation,
		Scalers: []cache.ScalerBuilder{{
			Scaler:       streamer,
			ScalerConfig: scalersconfig.ScalerConfig{TriggerIndex: 0},
		}},
		Recorder: events.NewFakeRecorder(10),
	}
	// A scaler-error invalidation rebuilds the cache for the same object, so the
	// replacement keeps the same UID and generation.
	newCache := &cache.ScalersCache{
		ScaledObject:             scaledObject,
		ScalableObjectGeneration: generation,
		Scalers: []cache.ScalerBuilder{{
			Scaler:       &metricSpecWatcherTestPushScaler{specCh: make(chan []v2.MetricSpec)},
			ScalerConfig: scalersconfig.ScalerConfig{TriggerIndex: 0},
		}},
		Recorder: events.NewFakeRecorder(10),
	}

	h := &scaleHandler{
		scalerCaches:          map[string]*cache.ScalersCache{key: oldCache},
		scalerCachesLock:      &sync.RWMutex{},
		metricSpecReconcileCh: make(chan event.GenericEvent, 1),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go h.watchMetricSpecUpdates(ctx, scaledObject.Name, scaledObject.Namespace, 0, streamer, uid, generation)

	oldCache.Close(context.Background())
	h.scalerCachesLock.Lock()
	h.scalerCaches[key] = newCache
	h.scalerCachesLock.Unlock()

	expectedSpecs := []v2.MetricSpec{createMetricSpec(42, "s0-updated")}
	streamer.specCh <- expectedSpecs

	assert.Eventually(t, func() bool {
		specs := newCache.GetMetricSpecForScaling(context.Background())
		return len(specs) == 1 && specs[0].External != nil && specs[0].External.Metric.Name == "s0-updated"
	}, time.Second, 10*time.Millisecond)

	select {
	case evt := <-h.MetricSpecReconcileChan():
		assert.Equal(t, scaledObject.Name, evt.Object.GetName())
		assert.Equal(t, scaledObject.Namespace, evt.Object.GetNamespace())
	case <-time.After(time.Second):
		t.Fatal("expected a reconcile event for the replacement cache")
	}
}

// TestWatchMetricSpecUpdates_IgnoresStaleGeneration verifies that a watcher
// bound to an old ScaledObject generation cannot overwrite the metric spec of a
// cache installed for a newer generation, and does not enqueue a reconcile.
func TestWatchMetricSpecUpdates_IgnoresStaleGeneration(t *testing.T) {
	uid := types.UID("so-uid-1")
	scaledObject := &kedav1alpha1.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testNameGlobal,
			Namespace: testNamespaceGlobal,
			UID:       uid,
		},
	}
	key := scaledObject.GenerateIdentifier()

	streamer := &metricSpecWatcherTestPushScaler{specCh: make(chan []v2.MetricSpec, 1)}
	// The cache in the map belongs to generation 2, but the watcher was created
	// for generation 1.
	newGenCache := &cache.ScalersCache{
		ScaledObject:             scaledObject,
		ScalableObjectGeneration: 2,
		Scalers: []cache.ScalerBuilder{{
			Scaler:       &metricSpecWatcherTestPushScaler{specCh: make(chan []v2.MetricSpec)},
			ScalerConfig: scalersconfig.ScalerConfig{TriggerIndex: 0},
		}},
		Recorder: events.NewFakeRecorder(10),
	}

	h := &scaleHandler{
		scalerCaches:          map[string]*cache.ScalersCache{key: newGenCache},
		scalerCachesLock:      &sync.RWMutex{},
		metricSpecReconcileCh: make(chan event.GenericEvent, 1),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go h.watchMetricSpecUpdates(ctx, scaledObject.Name, scaledObject.Namespace, 0, streamer, uid, 1)

	streamer.specCh <- []v2.MetricSpec{createMetricSpec(42, "s0-stale")}

	// The generation-2 cache must never receive the generation-1 update, and no
	// reconcile must be enqueued.
	assert.Never(t, func() bool {
		specs := newGenCache.GetMetricSpecForScaling(context.Background())
		return len(specs) == 1 && specs[0].External != nil && specs[0].External.Metric.Name == "s0-stale"
	}, 200*time.Millisecond, 10*time.Millisecond)

	select {
	case evt := <-h.MetricSpecReconcileChan():
		t.Fatalf("unexpected reconcile event for stale generation update: %s/%s", evt.Object.GetNamespace(), evt.Object.GetName())
	default:
	}
}

// TestWatchMetricSpecUpdates_IgnoresRecreatedObject verifies that a watcher for
// a deleted object cannot update the cache of a new object created under the
// same namespace and name but with a different UID.
func TestWatchMetricSpecUpdates_IgnoresRecreatedObject(t *testing.T) {
	oldUID := types.UID("so-uid-old")
	newUID := types.UID("so-uid-new")
	scaledObject := &kedav1alpha1.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testNameGlobal,
			Namespace: testNamespaceGlobal,
			UID:       newUID,
		},
	}
	key := scaledObject.GenerateIdentifier()

	streamer := &metricSpecWatcherTestPushScaler{specCh: make(chan []v2.MetricSpec, 1)}
	recreatedCache := &cache.ScalersCache{
		ScaledObject: scaledObject,
		Scalers: []cache.ScalerBuilder{{
			Scaler:       &metricSpecWatcherTestPushScaler{specCh: make(chan []v2.MetricSpec)},
			ScalerConfig: scalersconfig.ScalerConfig{TriggerIndex: 0},
		}},
		Recorder: events.NewFakeRecorder(10),
	}

	h := &scaleHandler{
		scalerCaches:          map[string]*cache.ScalersCache{key: recreatedCache},
		scalerCachesLock:      &sync.RWMutex{},
		metricSpecReconcileCh: make(chan event.GenericEvent, 1),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Watcher was created for the deleted object (oldUID).
	go h.watchMetricSpecUpdates(ctx, scaledObject.Name, scaledObject.Namespace, 0, streamer, oldUID, 0)

	streamer.specCh <- []v2.MetricSpec{createMetricSpec(42, "s0-stale")}

	assert.Never(t, func() bool {
		specs := recreatedCache.GetMetricSpecForScaling(context.Background())
		return len(specs) == 1 && specs[0].External != nil && specs[0].External.Metric.Name == "s0-stale"
	}, 200*time.Millisecond, 10*time.Millisecond)

	select {
	case evt := <-h.MetricSpecReconcileChan():
		t.Fatalf("unexpected reconcile event for recreated object update: %s/%s", evt.Object.GetNamespace(), evt.Object.GetName())
	default:
	}
}

func TestEnqueueMetricSpecReconcile(t *testing.T) {
	h := &scaleHandler{metricSpecReconcileCh: make(chan event.GenericEvent, 1)}

	h.enqueueMetricSpecReconcile(context.Background(), "test-so", "test-ns")

	select {
	case evt := <-h.MetricSpecReconcileChan():
		assert.Equal(t, "test-so", evt.Object.GetName())
		assert.Equal(t, "test-ns", evt.Object.GetNamespace())
	default:
		t.Fatal("expected a reconcile event to be enqueued")
	}
}

func TestEnqueueMetricSpecReconcile_WaitsForRoomWhenFull(t *testing.T) {
	h := &scaleHandler{metricSpecReconcileCh: make(chan event.GenericEvent, 1)}
	h.metricSpecReconcileCh <- event.GenericEvent{Object: &kedav1alpha1.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{Name: "other-so", Namespace: "other-ns"},
	}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		h.enqueueMetricSpecReconcile(ctx, "test-so", "test-ns")
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("enqueueMetricSpecReconcile returned before channel space was available")
	case <-time.After(100 * time.Millisecond):
	}

	select {
	case evt := <-h.MetricSpecReconcileChan():
		assert.Equal(t, "other-so", evt.Object.GetName())
		assert.Equal(t, "other-ns", evt.Object.GetNamespace())
	case <-time.After(time.Second):
		t.Fatal("expected the preloaded event to be drained")
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("enqueueMetricSpecReconcile did not complete after channel space became available")
	}

	select {
	case evt := <-h.MetricSpecReconcileChan():
		assert.Equal(t, "test-so", evt.Object.GetName())
		assert.Equal(t, "test-ns", evt.Object.GetNamespace())
	case <-time.After(time.Second):
		t.Fatal("expected the waiting reconcile event to be enqueued")
	}
}

func TestEnqueueMetricSpecReconcile_RespectsContextCancellation(t *testing.T) {
	h := &scaleHandler{metricSpecReconcileCh: make(chan event.GenericEvent, 1)}
	h.metricSpecReconcileCh <- event.GenericEvent{Object: &kedav1alpha1.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{Name: "other-so", Namespace: "other-ns"},
	}}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		h.enqueueMetricSpecReconcile(ctx, "test-so", "test-ns")
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("enqueueMetricSpecReconcile did not return after context cancellation")
	}

	select {
	case evt := <-h.MetricSpecReconcileChan():
		assert.Equal(t, "other-so", evt.Object.GetName())
	case <-time.After(time.Second):
		t.Fatal("expected only the preloaded event to remain in the channel")
	}

	select {
	case evt := <-h.MetricSpecReconcileChan():
		t.Fatalf("unexpected reconcile event after cancellation: %s/%s", evt.Object.GetNamespace(), evt.Object.GetName())
	default:
	}
}
