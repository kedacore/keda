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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	corev1listers "k8s.io/client-go/listers/core/v1"
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
	"github.com/kedacore/keda/v2/pkg/scaling/cache/metricscache"
	"github.com/kedacore/keda/v2/pkg/scaling/executor"
	"github.com/kedacore/keda/v2/pkg/scaling/resolver"
)

var log = logf.Log.WithName("scale_handler")

// ScaleHandler encapsulates the logic of calling the right scalers for
// each ScaledObject and making the final scale decision and operation
type ScaleHandler interface {
	HandleScalableObject(ctx context.Context, scalableObject interface{}) error
	DeleteScalableObject(ctx context.Context, scalableObject interface{}) error
	GetScalersCache(ctx context.Context, scalableObject interface{}) (*cache.ScalersCache, error)
	ClearScalersCache(ctx context.Context, scalableObject interface{}) error

	GetScaledObjectMetrics(ctx context.Context, scaledObjectName, scaledObjectNamespace, metricName string) (*external_metrics.ExternalMetricValueList, *metricsserviceapi.PromMetricsMsg, error)
}

type scaleHandler struct {
	client                   client.Client
	scaleLoopContexts        *sync.Map
	scaleExecutor            executor.ScaleExecutor
	globalHTTPTimeout        time.Duration
	recorder                 record.EventRecorder
	scalerCaches             map[string]*cache.ScalersCache
	scalerCachesLock         *sync.RWMutex
	scaledObjectsMetricCache metricscache.MetricsCache
	secretsLister            corev1listers.SecretLister
}

// NewScaleHandler creates a ScaleHandler object
func NewScaleHandler(client client.Client, scaleClient scale.ScalesGetter, reconcilerScheme *runtime.Scheme, globalHTTPTimeout time.Duration, recorder record.EventRecorder, secretsLister corev1listers.SecretLister) ScaleHandler {
	return &scaleHandler{
		client:                   client,
		scaleLoopContexts:        &sync.Map{},
		scaleExecutor:            executor.NewScaleExecutor(client, scaleClient, reconcilerScheme, recorder),
		globalHTTPTimeout:        globalHTTPTimeout,
		recorder:                 recorder,
		scalerCaches:             map[string]*cache.ScalersCache{},
		scalerCachesLock:         &sync.RWMutex{},
		scaledObjectsMetricCache: metricscache.NewMetricsCache(),
		secretsLister:            secretsLister,
	}
}

/// --------------------------------------------------------------------------- ///
/// ----------            Scaling logic related methods               --------- ///
/// --------------------------------------------------------------------------- ///

// HandleScalableObject is the initial method when Scalable is created and it handles the main scaling logic
func (h *scaleHandler) HandleScalableObject(ctx context.Context, scalableObject interface{}) error {
	withTriggers, err := kedav1alpha1.AsDuckWithTriggers(scalableObject)
	if err != nil {
		log.Error(err, "error duck typing object into withTrigger", "scalableObject", scalableObject)
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

// DeleteScalableObject stops handling logic for input ScalableObject
func (h *scaleHandler) DeleteScalableObject(ctx context.Context, scalableObject interface{}) error {
	withTriggers, err := kedav1alpha1.AsDuckWithTriggers(scalableObject)
	if err != nil {
		log.Error(err, "error duck typing object into withTrigger", "scalableObject", scalableObject)
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
			log.Error(err, "error clearing scalers cache", "scalableObject", scalableObject, "key", key)
		}
		h.recorder.Event(withTriggers, corev1.EventTypeNormal, eventreason.KEDAScalersStopped, "Stopped scalers watch")
	} else {
		log.V(1).Info("ScalableObject was not found in controller cache", "key", key)
	}

	return nil
}

// startScaleLoop blocks forever and checks the scalableObject based on its pollingInterval
func (h *scaleHandler) startScaleLoop(ctx context.Context, withTriggers *kedav1alpha1.WithTriggers, scalableObject interface{}, scalingMutex sync.Locker) {
	logger := log.WithValues("type", withTriggers.Kind, "namespace", withTriggers.Namespace, "name", withTriggers.Name)

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

// startPushScalers starts all push scalers defined in the input scalableOjbect
func (h *scaleHandler) startPushScalers(ctx context.Context, withTriggers *kedav1alpha1.WithTriggers, scalableObject interface{}, scalingMutex sync.Locker) {
	logger := log.WithValues("type", withTriggers.Kind, "namespace", withTriggers.Namespace, "name", withTriggers.Name)
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
						logger.Info("Warning: External Push Scaler does not support ScaledJob", "object", scalableObject)
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
	scalingMutex.Lock()
	defer scalingMutex.Unlock()
	switch obj := scalableObject.(type) {
	case *kedav1alpha1.ScaledObject:
		err := h.client.Get(ctx, types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, obj)
		if err != nil {
			log.Error(err, "error getting scaledObject", "object", scalableObject)
			return
		}
		isActive, isError, metricsRecords, err := h.getScaledObjectState(ctx, obj)
		if err != nil {
			log.Error(err, "error getting state of scaledObject", "scaledObject.Namespace", obj.Namespace, "scaledObject.Name", obj.Name)
			return
		}

		h.scaleExecutor.RequestScale(ctx, obj, isActive, isError)

		if len(metricsRecords) > 0 {
			log.V(1).Info("Storing metrics to cache", "scaledObject.Namespace", obj.Namespace, "scaledObject.Name", obj.Name, "metricsRecords", metricsRecords)
			h.scaledObjectsMetricCache.StoreRecords(obj.GenerateIdentifier(), metricsRecords)
		}
	case *kedav1alpha1.ScaledJob:
		cache, err := h.GetScalersCache(ctx, scalableObject)
		if err != nil {
			log.Error(err, "error getting scalers cache", "scaledJob.Namespace", obj.Namespace, "scaledJob.Name", obj.Name)
			return
		}

		err = h.client.Get(ctx, types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, obj)
		if err != nil {
			log.Error(err, "error getting scaledJob", "scaledJob.Namespace", obj.Namespace, "scaledJob.Name", obj.Name)
			return
		}

		isActive, scaleTo, maxScale := cache.IsScaledJobActive(ctx, obj)
		h.scaleExecutor.RequestJobScale(ctx, obj, isActive, scaleTo, maxScale)
	}
}

/// --------------------------------------------------------------------------- ///
/// ----------              ScalersCache related methods              --------- ///
/// --------------------------------------------------------------------------- ///

// GetScalersCache returns cache for input scalableObject, if the object is not found in the cache, it returns a new one
// if the input object is ScaledObject, it also compares the Generation of the input of object with the one stored in the cache,
// this is needed for out of scalerLoop invocations of this method (in package `controllers/keda`).
func (h *scaleHandler) GetScalersCache(ctx context.Context, scalableObject interface{}) (*cache.ScalersCache, error) {
	withTriggers, err := kedav1alpha1.AsDuckWithTriggers(scalableObject)
	if err != nil {
		return nil, err
	}
	key := withTriggers.GenerateIdentifier()
	generation := withTriggers.Generation

	return h.performGetScalersCache(ctx, key, scalableObject, &generation, "", "", "")
}

// getScalersCacheForScaledObject returns cache for input ScaledObject, referenced by name and namespace
// we don't need to compare the Generation, because this method should be called only inside scale loop, where we have up to date object.
func (h *scaleHandler) getScalersCacheForScaledObject(ctx context.Context, scaledObjectName, scaledObjectNamespace string) (*cache.ScalersCache, error) {
	key := kedav1alpha1.GenerateIdentifier("ScaledObject", scaledObjectNamespace, scaledObjectName)

	return h.performGetScalersCache(ctx, key, nil, nil, "ScaledObject", scaledObjectNamespace, scaledObjectName)
}

// performGetScalersCache returns cache for input scalableObject, it is common code used by GetScalersCache() and getScalersCacheForScaledObject() methods
func (h *scaleHandler) performGetScalersCache(ctx context.Context, key string, scalableObject interface{}, scalableObjectGeneration *int64, scalableObjectKind, scalableObjectNamespace, scalableObjectName string) (*cache.ScalersCache, error) {
	h.scalerCachesLock.RLock()
	if cache, ok := h.scalerCaches[key]; ok {
		// generation was specified -> let's include it in the check as well
		if scalableObjectGeneration != nil {
			if cache.ScalableObjectGeneration == *scalableObjectGeneration {
				h.scalerCachesLock.RUnlock()
				return cache, nil
			}
		} else {
			h.scalerCachesLock.RUnlock()
			return cache, nil
		}
	}
	h.scalerCachesLock.RUnlock()

	h.scalerCachesLock.Lock()
	defer h.scalerCachesLock.Unlock()
	if cache, ok := h.scalerCaches[key]; ok {
		// generation was specified -> let's include it in the check as well
		if scalableObjectGeneration != nil {
			if cache.ScalableObjectGeneration == *scalableObjectGeneration {
				return cache, nil
			}
			// object was found in cache, but the generation is not correct,
			// let's close scalers in the cache and proceed further to recreate the cache
			cache.Close(ctx)
		} else {
			return cache, nil
		}
	}

	if scalableObject == nil {
		switch scalableObjectKind {
		case "ScaledObject":
			scaledObject := &kedav1alpha1.ScaledObject{}
			err := h.client.Get(ctx, types.NamespacedName{Name: scalableObjectName, Namespace: scalableObjectNamespace}, scaledObject)
			if err != nil {
				log.Error(err, "failed to get ScaledObject", "name", scalableObjectName, "namespace", scalableObjectNamespace)
				return nil, err
			}
			scalableObject = scaledObject
		case "ScaledJob":
			scaledJob := &kedav1alpha1.ScaledJob{}
			err := h.client.Get(ctx, types.NamespacedName{Name: scalableObjectName, Namespace: scalableObjectNamespace}, scaledJob)
			if err != nil {
				log.Error(err, "failed to get ScaledJob", "name", scalableObjectName, "namespace", scalableObjectNamespace)
				return nil, err
			}
			scalableObject = scaledJob
		default:
			err := fmt.Errorf("unknown ScalableObjectKind, got=%q", scalableObjectKind)
			log.Error(err, "unknown kind", "name", scalableObjectName, "namespace", scalableObjectNamespace)
			return nil, err
		}
	}

	withTriggers, err := kedav1alpha1.AsDuckWithTriggers(scalableObject)
	if err != nil {
		return nil, err
	}

	podTemplateSpec, containerName, err := resolver.ResolveScaleTargetPodSpec(ctx, h.client, scalableObject)
	if err != nil {
		return nil, err
	}

	scalers, err := h.buildScalers(ctx, withTriggers, podTemplateSpec, containerName)
	if err != nil {
		return nil, err
	}

	newCache := &cache.ScalersCache{
		Scalers:                  scalers,
		ScalableObjectGeneration: withTriggers.Generation,
		Recorder:                 h.recorder,
	}
	switch obj := scalableObject.(type) {
	case *kedav1alpha1.ScaledObject:
		newCache.ScaledObject = obj
	default:
	}

	h.scalerCaches[key] = newCache

	return h.scalerCaches[key], nil
}

// ClearScalersCache invalidates chache for the input scalableObject
func (h *scaleHandler) ClearScalersCache(ctx context.Context, scalableObject interface{}) error {
	withTriggers, err := kedav1alpha1.AsDuckWithTriggers(scalableObject)
	if err != nil {
		return err
	}

	key := withTriggers.GenerateIdentifier()

	go h.scaledObjectsMetricCache.Delete(key)

	h.scalerCachesLock.Lock()
	defer h.scalerCachesLock.Unlock()
	if cache, ok := h.scalerCaches[key]; ok {
		log.V(1).WithValues("key", key).Info("Removing entry from ScalersCache")
		cache.Close(ctx)
		delete(h.scalerCaches, key)
	}

	return nil
}

/// --------------------------------------------------------------------------- ///
/// ----------             ScaledObject related methods               --------- ///
/// --------------------------------------------------------------------------- ///

// GetScaledObjectMetrics returns metrics for specified metric name for a ScaledObject identified by it's name and namespace.
// The second return value are Prometheus metrics that needed to be exposed (used by DEPRECATED Prometheus Server on KEDA Metrics Server)
// It could either query the metric value directly from the scaler or from a cache, that's being stored for the scaler.
func (h *scaleHandler) GetScaledObjectMetrics(ctx context.Context, scaledObjectName, scaledObjectNamespace, metricName string) (*external_metrics.ExternalMetricValueList, *metricsserviceapi.PromMetricsMsg, error) {
	logger := log.WithValues("scaledObject.Namespace", scaledObjectNamespace, "scaledObject.Name", scaledObjectName)

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
		return nil, &exportedPromMetrics, fmt.Errorf("error getting scalers %w", err)
	}

	var scaledObject *kedav1alpha1.ScaledObject
	if cache.ScaledObject != nil {
		scaledObject = cache.ScaledObject
	} else {
		err := fmt.Errorf("scaledObject not found in the cache")
		logger.Error(err, "scaledObject not found in the cache")
		return nil, &exportedPromMetrics, err
	}

	isScalerError := false
	scaledObjectIdentifier := scaledObject.GenerateIdentifier()

	// let's check metrics for all scalers in a ScaledObject
	scalers, scalerConfigs := cache.GetScalers()
	for scalerIndex := 0; scalerIndex < len(scalers); scalerIndex++ {
		scalerName := strings.Replace(fmt.Sprintf("%T", scalers[scalerIndex]), "*scalers.", "", 1)
		if scalerConfigs[scalerIndex].TriggerName != "" {
			scalerName = scalerConfigs[scalerIndex].TriggerName
		}

		metricSpecs, err := cache.GetMetricSpecForScalingForScaler(ctx, scalerIndex)
		if err != nil {
			isScalerError = true
			logger.Error(err, "error getting metric spec for the scaler", "scaler", scalerName)
			cache.Recorder.Event(scaledObject, corev1.EventTypeWarning, eventreason.KEDAScalerFailed, err.Error())
		}

		for _, spec := range metricSpecs {
			// skip cpu/memory resource scaler
			if spec.External == nil {
				continue
			}

			// Filter only the desired metric
			if strings.EqualFold(spec.External.Metric.Name, metricName) {
				var metrics []external_metrics.ExternalMetricValue

				// if cache is defined for this scaler/metric, let's try to hit it first
				metricsFoundInCache := false
				if scalerConfigs[scalerIndex].TriggerUseCachedMetrics {
					var metricsRecord metricscache.MetricsRecord
					if metricsRecord, metricsFoundInCache = h.scaledObjectsMetricCache.ReadRecord(scaledObjectIdentifier, spec.External.Metric.Name); metricsFoundInCache {
						logger.V(1).Info("Reading metrics from cache", "scaler", scalerName, "metricName", spec.External.Metric.Name, "metricsRecord", metricsRecord)
						metrics = metricsRecord.Metric
						err = metricsRecord.ScalerError
					}
				}

				if !metricsFoundInCache {
					var latency int64
					metrics, _, latency, err = cache.GetMetricsAndActivityForScaler(ctx, scalerIndex, metricName)
					if latency != -1 {
						prommetrics.RecordScalerLatency(scaledObjectNamespace, scaledObject.Name, scalerName, scalerIndex, metricName, float64(latency))
					}
					logger.V(1).Info("Getting metrics from scaler", "scaler", scalerName, "metricName", spec.External.Metric.Name, "metrics", metrics, "scalerError", err)
				}

				// check if we need to set a fallback
				metrics, err = fallback.GetMetricsWithFallback(ctx, h.client, metrics, err, metricName, scaledObject, spec)

				if err != nil {
					isScalerError = true
					logger.Error(err, "error getting metric for scaler", "scaler", scalerName)
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
	if isScalerError {
		err := h.ClearScalersCache(ctx, scaledObject)
		if err != nil {
			logger.Error(err, "error clearing scalers cache")
		}
		logger.V(1).Info("scaler error encountered, clearing scaler cache")
	}

	if len(matchingMetrics) == 0 {
		return nil, &exportedPromMetrics, fmt.Errorf("no matching metrics found for " + metricName)
	}

	return &external_metrics.ExternalMetricValueList{
		Items: matchingMetrics,
	}, &exportedPromMetrics, nil
}

// getScaledObjectState returns whether the input ScaledObject:
// is active as the first return value,
// the second return value indicates whether there was any error during quering scalers,
// the third return value is a map of metrics record - a metric value for each scaler and it's metric
// the fourth return value contains error if is not able access scalers cache
func (h *scaleHandler) getScaledObjectState(ctx context.Context, scaledObject *kedav1alpha1.ScaledObject) (bool, bool, map[string]metricscache.MetricsRecord, error) {
	logger := log.WithValues("scaledObject.Namespace", scaledObject.Namespace, "scaledObject.Name", scaledObject.Name)

	isScaledObjectActive := false
	isScalerError := false
	metricsRecord := map[string]metricscache.MetricsRecord{}

	cache, err := h.GetScalersCache(ctx, scaledObject)
	prommetrics.RecordScaledObjectError(scaledObject.Namespace, scaledObject.Name, err)
	if err != nil {
		return false, true, map[string]metricscache.MetricsRecord{}, fmt.Errorf("error getting scalers cache %w", err)
	}

	// Let's collect status of all scalers, no matter if any scaler raises error or is active
	scalers, scalerConfigs := cache.GetScalers()
	for scalerIndex := 0; scalerIndex < len(scalers); scalerIndex++ {
		scalerName := strings.Replace(fmt.Sprintf("%T", scalers[scalerIndex]), "*scalers.", "", 1)
		if scalerConfigs[scalerIndex].TriggerName != "" {
			scalerName = scalerConfigs[scalerIndex].TriggerName
		}

		metricSpecs, err := cache.GetMetricSpecForScalingForScaler(ctx, scalerIndex)
		if err != nil {
			isScalerError = true
			logger.Error(err, "error getting metric spec for the scaler", "scaler", scalerName)
			cache.Recorder.Event(scaledObject, corev1.EventTypeWarning, eventreason.KEDAScalerFailed, err.Error())
		}

		for _, spec := range metricSpecs {
			// skip cpu/memory resource scaler, these scalers are also always Active
			if spec.External == nil {
				isScaledObjectActive = true
				continue
			}

			metricName := spec.External.Metric.Name

			var latency int64
			metrics, isMetricActive, latency, err := cache.GetMetricsAndActivityForScaler(ctx, scalerIndex, metricName)
			if latency != -1 {
				prommetrics.RecordScalerLatency(scaledObject.Namespace, scaledObject.Name, scalerName, scalerIndex, metricName, float64(latency))
			}
			logger.V(1).Info("Getting metrics and activity from scaler", "scaler", scalerName, "metricName", metricName, "metrics", metrics, "activity", isMetricActive, "scalerError", err)

			if scalerConfigs[scalerIndex].TriggerUseCachedMetrics {
				metricsRecord[metricName] = metricscache.MetricsRecord{
					IsActive:    isMetricActive,
					Metric:      metrics,
					ScalerError: err,
				}
			}

			if err != nil {
				isScalerError = true
				logger.Error(err, "error getting scale decision", "scaler", scalerName)
				cache.Recorder.Event(scaledObject, corev1.EventTypeWarning, eventreason.KEDAScalerFailed, err.Error())
			} else {
				for _, metric := range metrics {
					metricValue := metric.Value.AsApproximateFloat64()
					prommetrics.RecordScalerMetric(scaledObject.Namespace, scaledObject.Name, scalerName, scalerIndex, metric.MetricName, metricValue)
				}

				if isMetricActive {
					isScaledObjectActive = true
					if spec.External != nil {
						logger.V(1).Info("Scaler for scaledObject is active", "scaler", scalerName, "metricName", metricName)
					}
					if spec.Resource != nil {
						logger.V(1).Info("Scaler for scaledObject is active", "scaler", scalerName, "metricName", spec.Resource.Name)
					}
				}
			}
			prommetrics.RecordScalerError(scaledObject.Namespace, scaledObject.Name, scalerName, scalerIndex, metricName, err)
			prommetrics.RecordScalerActive(scaledObject.Namespace, scaledObject.Name, scalerName, scalerIndex, metricName, isMetricActive)
		}
	}

	// invalidate the cache for the ScaledObject, if we hit an error in any scaler
	// in this case we try to build all scalers (and resolve all secrets/creds) again in the next call
	if isScalerError {
		err := h.ClearScalersCache(ctx, scaledObject)
		if err != nil {
			logger.Error(err, "error clearing scalers cache")
		}
		logger.V(1).Info("scaler error encountered, clearing scaler cache")
	}

	return isScaledObjectActive, isScalerError, metricsRecord, nil
}
