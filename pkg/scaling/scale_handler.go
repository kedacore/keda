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
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/scale"
	"k8s.io/client-go/tools/record"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/eventreason"
	"github.com/kedacore/keda/v2/pkg/fallback"
	metricsserviceapi "github.com/kedacore/keda/v2/pkg/metricsservice/api"
	"github.com/kedacore/keda/v2/pkg/prommetrics"
	"github.com/kedacore/keda/v2/pkg/scalers"
	"github.com/kedacore/keda/v2/pkg/scaling/cache"
	"github.com/kedacore/keda/v2/pkg/scaling/executor"
	"github.com/kedacore/keda/v2/pkg/scaling/resolver"
)

// ScaleHandler encapsulates the logic of calling the right scalers for
// each ScaledObject and making the final scale decision and operation
type ScaleHandler interface {
	HandleScalableObject(ctx context.Context, scalableObject interface{}) error
	DeleteScalableObject(ctx context.Context, scalableObject interface{}) error
	GetScalersCache(ctx context.Context, scalableObject interface{}) (*cache.ScalersCache, error)
	ClearScalersCache(ctx context.Context, scalableObject interface{}) error

	GetExternalMetrics(ctx context.Context, scaledObjectName, scaledObjectNamespace, metricName string) (*external_metrics.ExternalMetricValueList, *metricsserviceapi.PromMetricsMsg, error)
}

type scaleHandler struct {
	client            client.Client
	logger            logr.Logger
	scaleLoopContexts *sync.Map
	scaleExecutor     executor.ScaleExecutor
	globalHTTPTimeout time.Duration
	recorder          record.EventRecorder
	scalerCaches      map[string]*cache.ScalersCache
	lock              *sync.RWMutex
}

// NewScaleHandler creates a ScaleHandler object
func NewScaleHandler(client client.Client, scaleClient scale.ScalesGetter, reconcilerScheme *runtime.Scheme, globalHTTPTimeout time.Duration, recorder record.EventRecorder) ScaleHandler {
	return &scaleHandler{
		client:            client,
		logger:            logf.Log.WithName("scalehandler"),
		scaleLoopContexts: &sync.Map{},
		scaleExecutor:     executor.NewScaleExecutor(client, scaleClient, reconcilerScheme, recorder),
		globalHTTPTimeout: globalHTTPTimeout,
		recorder:          recorder,
		scalerCaches:      map[string]*cache.ScalersCache{},
		lock:              &sync.RWMutex{},
	}
}

func (h *scaleHandler) HandleScalableObject(ctx context.Context, scalableObject interface{}) error {
	withTriggers, err := asDuckWithTriggers(scalableObject)
	if err != nil {
		h.logger.Error(err, "error duck typing object into withTrigger")
		return err
	}

	key := withTriggers.GenerateIdentifier()
	ctx, cancel := context.WithCancel(ctx)

	// cancel the outdated ScaleLoop for the same ScaledObject (if exists)
	value, loaded := h.scaleLoopContexts.LoadOrStore(key, cancel)
	if loaded {
		cancelValue, ok := value.(context.CancelFunc)
		if ok {
			cancelValue()
		}
		h.scaleLoopContexts.Store(key, cancel)
	} else {
		h.recorder.Event(withTriggers, corev1.EventTypeNormal, eventreason.KEDAScalersStarted, "Started scalers watch")
	}

	// a mutex is used to synchronize scale requests per scalableObject
	scalingMutex := &sync.Mutex{}

	// passing deep copy of ScaledObject/ScaledJob to the scaleLoop go routines, it's a precaution to not have global objects shared between threads
	switch obj := scalableObject.(type) {
	case *kedav1alpha1.ScaledObject:
		go h.startPushScalers(ctx, withTriggers, obj.DeepCopy(), scalingMutex)
		go h.startScaleLoop(ctx, withTriggers, obj.DeepCopy(), scalingMutex)
	case *kedav1alpha1.ScaledJob:
		go h.startPushScalers(ctx, withTriggers, obj.DeepCopy(), scalingMutex)
		go h.startScaleLoop(ctx, withTriggers, obj.DeepCopy(), scalingMutex)
	}
	return nil
}

func (h *scaleHandler) DeleteScalableObject(ctx context.Context, scalableObject interface{}) error {
	withTriggers, err := asDuckWithTriggers(scalableObject)
	if err != nil {
		h.logger.Error(err, "error duck typing object into withTrigger")
		return err
	}

	key := withTriggers.GenerateIdentifier()
	result, ok := h.scaleLoopContexts.Load(key)
	if ok {
		cancel, ok := result.(context.CancelFunc)
		if ok {
			cancel()
		}
		h.scaleLoopContexts.Delete(key)
		err := h.ClearScalersCache(ctx, scalableObject)
		if err != nil {
			h.logger.Error(err, "error clearing scalers cache")
		}
		h.recorder.Event(withTriggers, corev1.EventTypeNormal, eventreason.KEDAScalersStopped, "Stopped scalers watch")
	} else {
		h.logger.V(1).Info("ScaledObject was not found in controller cache", "key", key)
	}

	return nil
}

// startScaleLoop blocks forever and checks the scaledObject based on its pollingInterval
func (h *scaleHandler) startScaleLoop(ctx context.Context, withTriggers *kedav1alpha1.WithTriggers, scalableObject interface{}, scalingMutex sync.Locker) {
	logger := h.logger.WithValues("type", withTriggers.Kind, "namespace", withTriggers.Namespace, "name", withTriggers.Name)

	pollingInterval := withTriggers.GetPollingInterval()
	logger.V(1).Info("Watching with pollingInterval", "PollingInterval", pollingInterval)

	for {
		tmr := time.NewTimer(pollingInterval)
		h.checkScalers(ctx, scalableObject, scalingMutex)

		select {
		case <-tmr.C:
			tmr.Stop()
		case <-ctx.Done():
			logger.V(1).Info("Context canceled")
			err := h.ClearScalersCache(ctx, scalableObject)
			if err != nil {
				logger.Error(err, "error clearing scalers cache")
			}
			tmr.Stop()
			return
		}
	}
}

func (h *scaleHandler) getScalersCacheForScaledObject(ctx context.Context, scaledObjectName, scaledObjectNamespace string) (*cache.ScalersCache, error) {
	key := kedav1alpha1.GenerateIdentifier("ScaledObject", scaledObjectNamespace, scaledObjectName)

	h.lock.RLock()
	if cache, ok := h.scalerCaches[key]; ok {
		h.lock.RUnlock()
		return cache, nil
	}
	h.lock.RUnlock()

	h.lock.Lock()
	defer h.lock.Unlock()
	if cache, ok := h.scalerCaches[key]; ok {
		return cache, nil
	}

	scaledObject := &kedav1alpha1.ScaledObject{}
	err := h.client.Get(ctx, types.NamespacedName{Name: scaledObjectName, Namespace: scaledObjectNamespace}, scaledObject)
	if err != nil {
		h.logger.Error(err, "failed to get ScaledObject", "name", scaledObjectName, "namespace", scaledObjectNamespace)
		return nil, err
	}

	podTemplateSpec, containerName, err := resolver.ResolveScaleTargetPodSpec(ctx, h.client, h.logger, scaledObject)
	if err != nil {
		return nil, err
	}

	withTriggers, err := asDuckWithTriggers(scaledObject)
	if err != nil {
		return nil, err
	}

	scalers, err := h.buildScalers(ctx, withTriggers, podTemplateSpec, containerName)
	if err != nil {
		return nil, err
	}

	h.scalerCaches[key] = &cache.ScalersCache{
		ScaledObject: scaledObject,
		Scalers:      scalers,
		Logger:       h.logger,
		Recorder:     h.recorder,
	}

	return h.scalerCaches[key], nil
}

// TODO refactor this together with GetScalersCacheForScaledObject()
func (h *scaleHandler) GetScalersCache(ctx context.Context, scalableObject interface{}) (*cache.ScalersCache, error) {
	withTriggers, err := asDuckWithTriggers(scalableObject)
	if err != nil {
		return nil, err
	}

	key := withTriggers.GenerateIdentifier()

	h.lock.RLock()
	if cache, ok := h.scalerCaches[key]; ok && cache.Generation == withTriggers.Generation {
		h.lock.RUnlock()
		return cache, nil
	}
	h.lock.RUnlock()

	h.lock.Lock()
	defer h.lock.Unlock()
	if cache, ok := h.scalerCaches[key]; ok && cache.Generation == withTriggers.Generation {
		return cache, nil
	} else if ok {
		cache.Close(ctx)
	}

	podTemplateSpec, containerName, err := resolver.ResolveScaleTargetPodSpec(ctx, h.client, h.logger, scalableObject)
	if err != nil {
		return nil, err
	}

	scalers, err := h.buildScalers(ctx, withTriggers, podTemplateSpec, containerName)
	if err != nil {
		return nil, err
	}

	newCache := &cache.ScalersCache{
		Generation: withTriggers.Generation,
		Scalers:    scalers,
		Logger:     h.logger,
		Recorder:   h.recorder,
	}
	switch obj := scalableObject.(type) {
	case *kedav1alpha1.ScaledObject:
		newCache.ScaledObject = obj
	default:
	}

	h.scalerCaches[key] = newCache

	return h.scalerCaches[key], nil
}

func (h *scaleHandler) ClearScalersCache(ctx context.Context, scalableObject interface{}) error {
	withTriggers, err := asDuckWithTriggers(scalableObject)
	if err != nil {
		return err
	}

	key := withTriggers.GenerateIdentifier()

	h.lock.Lock()
	defer h.lock.Unlock()
	if cache, ok := h.scalerCaches[key]; ok {
		h.logger.V(1).WithValues("key", key).Info("Removing entry from ScalersCache")
		cache.Close(ctx)
		delete(h.scalerCaches, key)
	}

	return nil
}

func (h *scaleHandler) startPushScalers(ctx context.Context, withTriggers *kedav1alpha1.WithTriggers, scalableObject interface{}, scalingMutex sync.Locker) {
	logger := h.logger.WithValues("type", withTriggers.Kind, "namespace", withTriggers.Namespace, "name", withTriggers.Name)
	cache, err := h.GetScalersCache(ctx, scalableObject)
	if err != nil {
		logger.Error(err, "Error getting scalers", "object", scalableObject)
		return
	}

	for _, ps := range cache.GetPushScalers() {
		go func(s scalers.PushScaler) {
			activeCh := make(chan bool)
			go s.Run(ctx, activeCh)
			for {
				select {
				case <-ctx.Done():
					return
				case active := <-activeCh:
					scalingMutex.Lock()
					switch obj := scalableObject.(type) {
					case *kedav1alpha1.ScaledObject:
						h.scaleExecutor.RequestScale(ctx, obj, active, false)
					case *kedav1alpha1.ScaledJob:
						h.logger.Info("Warning: External Push Scaler does not support ScaledJob", "object", scalableObject)
					}
					scalingMutex.Unlock()
				}
			}
		}(ps)
	}
}

// checkScalers contains the main logic for the ScaleHandler scaling logic.
// It'll check each trigger active status then call RequestScale
func (h *scaleHandler) checkScalers(ctx context.Context, scalableObject interface{}, scalingMutex sync.Locker) {
	cache, err := h.GetScalersCache(ctx, scalableObject)
	if err != nil {
		h.logger.Error(err, "Error getting scalers", "object", scalableObject)
		return
	}

	scalingMutex.Lock()
	defer scalingMutex.Unlock()
	switch obj := scalableObject.(type) {
	case *kedav1alpha1.ScaledObject:
		err = h.client.Get(ctx, types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, obj)
		if err != nil {
			h.logger.Error(err, "Error getting scaledObject", "object", scalableObject)
			return
		}
		isActive, isError, _ := cache.IsScaledObjectActive(ctx, obj)
		h.scaleExecutor.RequestScale(ctx, obj, isActive, isError)
	case *kedav1alpha1.ScaledJob:
		err = h.client.Get(ctx, types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, obj)
		if err != nil {
			h.logger.Error(err, "Error getting scaledJob", "object", scalableObject)
			return
		}
		isActive, scaleTo, maxScale := cache.IsScaledJobActive(ctx, obj)
		h.scaleExecutor.RequestJobScale(ctx, obj, isActive, scaleTo, maxScale)
	}
}

func (h *scaleHandler) GetExternalMetrics(ctx context.Context, scaledObjectName, scaledObjectNamespace, metricName string) (*external_metrics.ExternalMetricValueList, *metricsserviceapi.PromMetricsMsg, error) {
	var matchingMetrics []external_metrics.ExternalMetricValue

	exportedPromMetrics := metricsserviceapi.PromMetricsMsg{
		ScaledObjectErr: false,
		ScalerMetric:    []*metricsserviceapi.ScalerMetricMsg{},
		ScalerError:     []*metricsserviceapi.ScalerErrorMsg{},
	}

	cache, err := h.getScalersCacheForScaledObject(ctx, scaledObjectName, scaledObjectNamespace)
	prommetrics.RecordScaledObjectError(scaledObjectNamespace, scaledObjectName, err)

	// [DEPRECATED] handle exporting Prometheus metrics from Operator to Metrics Server
	exportedPromMetrics.ScaledObjectErr = (err != nil)

	if err != nil {
		return nil, &exportedPromMetrics, fmt.Errorf("error when getting scalers %s", err)
	}

	var scaledObject *kedav1alpha1.ScaledObject
	if cache.ScaledObject != nil {
		scaledObject = cache.ScaledObject
	} else {
		err := fmt.Errorf("scaledObject not found in the cache")
		h.logger.Error(err, "scaledObject not found in the cache", "name", scaledObjectName, "namespace", scaledObjectNamespace)
		return nil, &exportedPromMetrics, err
	}

	// let's check metrics for all scalers in a ScaledObject
	scalerError := false
	scalers, scalerConfigs := cache.GetScalers()

	h.logger.V(1).WithValues("name", scaledObjectName, "namespace", scaledObjectNamespace, "metricName", metricName, "scalers", scalers).Info("Getting metric value")

	for scalerIndex := 0; scalerIndex < len(scalers); scalerIndex++ {
		metricSpecs := scalers[scalerIndex].GetMetricSpecForScaling(ctx)
		scalerName := strings.Replace(fmt.Sprintf("%T", scalers[scalerIndex]), "*scalers.", "", 1)
		if scalerConfigs[scalerIndex].TriggerName != "" {
			scalerName = scalerConfigs[scalerIndex].TriggerName
		}

		for _, metricSpec := range metricSpecs {
			// skip cpu/memory resource scaler
			if metricSpec.External == nil {
				continue
			}

			// Filter only the desired metric
			if strings.EqualFold(metricSpec.External.Metric.Name, metricName) {
				metrics, err := cache.GetMetricsForScaler(ctx, scalerIndex, metricName)
				metrics, err = fallback.GetMetricsWithFallback(ctx, h.client, h.logger, metrics, err, metricName, scaledObject, metricSpec)
				if err != nil {
					scalerError = true
					h.logger.Error(err, "error getting metric for scaler", "scaledObject.Namespace", scaledObjectNamespace, "scaledObject.Name", scaledObjectName, "scaler", scalerName)
				} else {
					for _, metric := range metrics {
						metricValue := metric.Value.AsApproximateFloat64()
						prommetrics.RecordScalerMetric(scaledObjectNamespace, scaledObjectName, scalerName, scalerIndex, metric.MetricName, metricValue)

						// [DEPRECATED] handle exporting Prometheus metrics from Operator to Metrics Server
						scalerMetricMsg := metricsserviceapi.ScalerMetricMsg{
							ScalerName:  scalerName,
							ScalerIndex: int32(scalerIndex),
							MetricName:  metricName,
							MetricValue: float32(metricValue),
						}
						exportedPromMetrics.ScalerMetric = append(exportedPromMetrics.ScalerMetric, &scalerMetricMsg)
					}
					matchingMetrics = append(matchingMetrics, metrics...)
				}
				prommetrics.RecordScalerError(scaledObjectNamespace, scaledObjectName, scalerName, scalerIndex, metricName, err)

				// [DEPRECATED] handle exporting Prometheus metrics from Operator to Metrics Server
				scalerErrMsg := metricsserviceapi.ScalerErrorMsg{
					ScalerName:  scalerName,
					ScalerIndex: int32(scalerIndex),
					MetricName:  metricName,
					Error:       (err != nil),
				}
				exportedPromMetrics.ScalerError = append(exportedPromMetrics.ScalerError, &scalerErrMsg)
			}
		}
	}

	// invalidate the cache for the ScaledObject, if we hit an error in any scaler
	// in this case we try to build all scalers (and resolve all secrets/creds) again in the next call
	if scalerError {
		err := h.ClearScalersCache(ctx, scaledObject)
		if err != nil {
			h.logger.Error(err, "error clearing scalers cache")
		}
		h.logger.V(1).Info("scaler error encountered, clearing scaler cache")
	}

	if len(matchingMetrics) == 0 {
		return nil, &exportedPromMetrics, fmt.Errorf("no matching metrics found for " + metricName)
	}

	return &external_metrics.ExternalMetricValueList{
		Items: matchingMetrics,
	}, &exportedPromMetrics, nil
}

// buildScalers returns list of Scalers for the specified triggers
func (h *scaleHandler) buildScalers(ctx context.Context, withTriggers *kedav1alpha1.WithTriggers, podTemplateSpec *corev1.PodTemplateSpec, containerName string) ([]cache.ScalerBuilder, error) {
	logger := h.logger.WithValues("type", withTriggers.Kind, "namespace", withTriggers.Namespace, "name", withTriggers.Name)
	var err error
	resolvedEnv := make(map[string]string)
	result := make([]cache.ScalerBuilder, 0, len(withTriggers.Spec.Triggers))

	for i, t := range withTriggers.Spec.Triggers {
		triggerIndex, trigger := i, t

		factory := func() (scalers.Scaler, *scalers.ScalerConfig, error) {
			if podTemplateSpec != nil {
				resolvedEnv, err = resolver.ResolveContainerEnv(ctx, h.client, logger, &podTemplateSpec.Spec, containerName, withTriggers.Namespace)
				if err != nil {
					return nil, nil, fmt.Errorf("error resolving secrets for ScaleTarget: %s", err)
				}
			}
			config := &scalers.ScalerConfig{
				ScalableObjectName:      withTriggers.Name,
				ScalableObjectNamespace: withTriggers.Namespace,
				ScalableObjectType:      withTriggers.Kind,
				TriggerName:             trigger.Name,
				TriggerMetadata:         trigger.Metadata,
				ResolvedEnv:             resolvedEnv,
				AuthParams:              make(map[string]string),
				GlobalHTTPTimeout:       h.globalHTTPTimeout,
				ScalerIndex:             triggerIndex,
				MetricType:              trigger.MetricType,
			}

			config.AuthParams, config.PodIdentity, err = resolver.ResolveAuthRefAndPodIdentity(ctx, h.client, logger, trigger.AuthenticationRef, podTemplateSpec, withTriggers.Namespace)
			if err != nil {
				return nil, nil, err
			}

			scaler, err := buildScaler(ctx, h.client, trigger.Type, config)
			return scaler, config, err
		}

		scaler, config, err := factory()
		if err != nil {
			h.recorder.Event(withTriggers, corev1.EventTypeWarning, eventreason.KEDAScalerFailed, err.Error())
			h.logger.Error(err, "error resolving auth params", "scalerIndex", triggerIndex, "object", withTriggers)
			if scaler != nil {
				scaler.Close(ctx)
			}
			for _, builder := range result {
				builder.Scaler.Close(ctx)
			}
			return nil, err
		}

		result = append(result, cache.ScalerBuilder{
			Scaler:       scaler,
			ScalerConfig: *config,
			Factory:      factory,
		})
	}

	return result, nil
}

func buildScaler(ctx context.Context, client client.Client, triggerType string, config *scalers.ScalerConfig) (scalers.Scaler, error) {
	// TRIGGERS-START
	switch triggerType {
	case "activemq":
		return scalers.NewActiveMQScaler(config)
	case "artemis-queue":
		return scalers.NewArtemisQueueScaler(config)
	case "aws-cloudwatch":
		return scalers.NewAwsCloudwatchScaler(config)
	case "aws-dynamodb":
		return scalers.NewAwsDynamoDBScaler(config)
	case "aws-dynamodb-streams":
		return scalers.NewAwsDynamoDBStreamsScaler(ctx, config)
	case "aws-kinesis-stream":
		return scalers.NewAwsKinesisStreamScaler(config)
	case "aws-sqs-queue":
		return scalers.NewAwsSqsQueueScaler(config)
	case "azure-app-insights":
		return scalers.NewAzureAppInsightsScaler(config)
	case "azure-blob":
		return scalers.NewAzureBlobScaler(config)
	case "azure-data-explorer":
		return scalers.NewAzureDataExplorerScaler(ctx, config)
	case "azure-eventhub":
		return scalers.NewAzureEventHubScaler(ctx, config)
	case "azure-log-analytics":
		return scalers.NewAzureLogAnalyticsScaler(config)
	case "azure-monitor":
		return scalers.NewAzureMonitorScaler(config)
	case "azure-pipelines":
		return scalers.NewAzurePipelinesScaler(ctx, config)
	case "azure-queue":
		return scalers.NewAzureQueueScaler(config)
	case "azure-servicebus":
		return scalers.NewAzureServiceBusScaler(ctx, config)
	case "cassandra":
		return scalers.NewCassandraScaler(config)
	case "cpu":
		return scalers.NewCPUMemoryScaler(corev1.ResourceCPU, config)
	case "cron":
		return scalers.NewCronScaler(config)
	case "datadog":
		return scalers.NewDatadogScaler(ctx, config)
	case "elasticsearch":
		return scalers.NewElasticsearchScaler(config)
	case "etcd":
		return scalers.NewEtcdScaler(config)
	case "external":
		return scalers.NewExternalScaler(config)
	// TODO: use other way for test.
	case "external-mock":
		return scalers.NewExternalMockScaler(config)
	case "external-push":
		return scalers.NewExternalPushScaler(config)
	case "gcp-pubsub":
		return scalers.NewPubSubScaler(config)
	case "gcp-stackdriver":
		return scalers.NewStackdriverScaler(ctx, config)
	case "gcp-storage":
		return scalers.NewGcsScaler(config)
	case "graphite":
		return scalers.NewGraphiteScaler(config)
	case "huawei-cloudeye":
		return scalers.NewHuaweiCloudeyeScaler(config)
	case "ibmmq":
		return scalers.NewIBMMQScaler(config)
	case "influxdb":
		return scalers.NewInfluxDBScaler(config)
	case "kafka":
		return scalers.NewKafkaScaler(config)
	case "kubernetes-workload":
		return scalers.NewKubernetesWorkloadScaler(client, config)
	case "liiklus":
		return scalers.NewLiiklusScaler(config)
	case "loki":
		return scalers.NewLokiScaler(config)
	case "memory":
		return scalers.NewCPUMemoryScaler(corev1.ResourceMemory, config)
	case "metrics-api":
		return scalers.NewMetricsAPIScaler(config)
	case "mongodb":
		return scalers.NewMongoDBScaler(ctx, config)
	case "mssql":
		return scalers.NewMSSQLScaler(config)
	case "mysql":
		return scalers.NewMySQLScaler(config)
	case "nats-jetstream":
		return scalers.NewNATSJetStreamScaler(config)
	case "new-relic":
		return scalers.NewNewRelicScaler(config)
	case "openstack-metric":
		return scalers.NewOpenstackMetricScaler(ctx, config)
	case "openstack-swift":
		return scalers.NewOpenstackSwiftScaler(ctx, config)
	case "postgresql":
		return scalers.NewPostgreSQLScaler(config)
	case "predictkube":
		return scalers.NewPredictKubeScaler(ctx, config)
	case "prometheus":
		return scalers.NewPrometheusScaler(config)
	case "pulsar":
		return scalers.NewPulsarScaler(config)
	case "rabbitmq":
		return scalers.NewRabbitMQScaler(config)
	case "redis":
		return scalers.NewRedisScaler(ctx, false, false, config)
	case "redis-cluster":
		return scalers.NewRedisScaler(ctx, true, false, config)
	case "redis-cluster-streams":
		return scalers.NewRedisStreamsScaler(ctx, true, false, config)
	case "redis-sentinel":
		return scalers.NewRedisScaler(ctx, false, true, config)
	case "redis-sentinel-streams":
		return scalers.NewRedisStreamsScaler(ctx, false, true, config)
	case "redis-streams":
		return scalers.NewRedisStreamsScaler(ctx, false, false, config)
	case "selenium-grid":
		return scalers.NewSeleniumGridScaler(config)
	case "solace-event-queue":
		return scalers.NewSolaceScaler(config)
	case "stan":
		return scalers.NewStanScaler(config)
	default:
		return nil, fmt.Errorf("no scaler found for type: %s", triggerType)
	}
	// TRIGGERS-END
}

func asDuckWithTriggers(scalableObject interface{}) (*kedav1alpha1.WithTriggers, error) {
	switch obj := scalableObject.(type) {
	case *kedav1alpha1.ScaledObject:
		return &kedav1alpha1.WithTriggers{
			TypeMeta:   obj.TypeMeta,
			ObjectMeta: obj.ObjectMeta,
			Spec: kedav1alpha1.WithTriggersSpec{
				PollingInterval: obj.Spec.PollingInterval,
				Triggers:        obj.Spec.Triggers,
			},
		}, nil
	case *kedav1alpha1.ScaledJob:
		return &kedav1alpha1.WithTriggers{
			TypeMeta:   obj.TypeMeta,
			ObjectMeta: obj.ObjectMeta,
			Spec: kedav1alpha1.WithTriggersSpec{
				PollingInterval: obj.Spec.PollingInterval,
				Triggers:        obj.Spec.Triggers,
			},
		}, nil
	default:
		// here could be the conversion from unknown Duck type potentially in the future
		return nil, fmt.Errorf("unknown scalable object type %v", scalableObject)
	}
}
