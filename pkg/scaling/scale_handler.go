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
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
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
	"github.com/kedacore/keda/v2/pkg/common/message"
	"github.com/kedacore/keda/v2/pkg/eventreason"
	"github.com/kedacore/keda/v2/pkg/fallback"
	"github.com/kedacore/keda/v2/pkg/metricscollector"
	"github.com/kedacore/keda/v2/pkg/scalers"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	"github.com/kedacore/keda/v2/pkg/scaling/cache"
	"github.com/kedacore/keda/v2/pkg/scaling/cache/metricscache"
	"github.com/kedacore/keda/v2/pkg/scaling/executor"
	"github.com/kedacore/keda/v2/pkg/scaling/modifiers"
	"github.com/kedacore/keda/v2/pkg/scaling/resolver"
	"github.com/kedacore/keda/v2/pkg/scaling/scaledjob"
)

var log = logf.Log.WithName("scale_handler")

// ScaleHandler encapsulates the logic of calling the right scalers for
// each ScaledObject and making the final scale decision and operation
type ScaleHandler interface {
	HandleScalableObject(ctx context.Context, scalableObject interface{}) error
	DeleteScalableObject(ctx context.Context, scalableObject interface{}) error
	GetScalersCache(ctx context.Context, scalableObject interface{}) (*cache.ScalersCache, error)
	ClearScalersCache(ctx context.Context, scalableObject interface{}) error

	GetScaledObjectMetrics(ctx context.Context, scaledObjectName, scaledObjectNamespace, metricName string) (*external_metrics.ExternalMetricValueList, error)
}

type scaleHandler struct {
	client                   client.Client
	scaleClient              scale.ScalesGetter
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
		h.recorder.Event(withTriggers, corev1.EventTypeNormal, eventreason.KEDAScalersStarted, message.ScalerStartMsg)
	}

	// a mutex is used to synchronize scale requests per scalableObject
	scalingMutex := &sync.Mutex{}

	// passing deep copy of ScaledObject/ScaledJob to the scaleLoop go routines, it's a precaution to not have global objects shared between threads
	switch obj := scalableObject.(type) {
	case *kedav1alpha1.ScaledObject:
		go h.startPushScalers(ctx, withTriggers, obj.DeepCopy(), scalingMutex)
		go h.startScaleLoop(ctx, withTriggers, obj.DeepCopy(), scalingMutex, true)
	case *kedav1alpha1.ScaledJob:
		go h.startPushScalers(ctx, withTriggers, obj.DeepCopy(), scalingMutex)
		go h.startScaleLoop(ctx, withTriggers, obj.DeepCopy(), scalingMutex, false)
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
func (h *scaleHandler) startScaleLoop(ctx context.Context, withTriggers *kedav1alpha1.WithTriggers, scalableObject interface{}, scalingMutex sync.Locker, isScaledObject bool) {
	logger := log.WithValues("type", withTriggers.Kind, "namespace", withTriggers.Namespace, "name", withTriggers.Name)

	pollingInterval := withTriggers.GetPollingInterval()
	logger.V(1).Info("Watching with pollingInterval", "PollingInterval", pollingInterval)

	next := time.Now()

	for {
		// we calculate the next execution time based on the pollingInterval and record the difference
		// between the expected execution time and the real execution time
		delay := time.Since(next)
		metricscollector.RecordScalableObjectLatency(withTriggers.Namespace, withTriggers.Name, isScaledObject, delay)

		tmr := time.NewTimer(pollingInterval)
		next = time.Now().Add(pollingInterval)

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
						h.scaleExecutor.RequestScale(ctx, obj, active, false, &executor.ScaleExecutorOptions{})
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
		isActive, isError, metricsRecords, activeTriggers, err := h.getScaledObjectState(ctx, obj)
		if err != nil {
			log.Error(err, "error getting state of scaledObject", "scaledObject.Namespace", obj.Namespace, "scaledObject.Name", obj.Name)
			return
		}

		h.scaleExecutor.RequestScale(ctx, obj, isActive, isError, &executor.ScaleExecutorOptions{ActiveTriggers: activeTriggers})

		if len(metricsRecords) > 0 {
			log.V(1).Info("Storing metrics to cache", "scaledObject.Namespace", obj.Namespace, "scaledObject.Name", obj.Name, "metricsRecords", metricsRecords)
			h.scaledObjectsMetricCache.StoreRecords(obj.GenerateIdentifier(), metricsRecords)
		}
	case *kedav1alpha1.ScaledJob:
		err := h.client.Get(ctx, types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, obj)
		if err != nil {
			log.Error(err, "error getting scaledJob", "scaledJob.Namespace", obj.Namespace, "scaledJob.Name", obj.Name)
			return
		}

		isActive, isError, scaleTo, maxScale := h.isScaledJobActive(ctx, obj)
		h.scaleExecutor.RequestJobScale(ctx, obj, isActive, isError, scaleTo, maxScale)
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

	asMetricSource := false
	switch obj := scalableObject.(type) {
	case *kedav1alpha1.ScaledObject:
		asMetricSource = obj.IsUsingModifiers()
	default:
	}

	scalers, err := h.buildScalers(ctx, withTriggers, podTemplateSpec, containerName, asMetricSource)
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
		if obj.Spec.Advanced != nil && obj.Spec.Advanced.ScalingModifiers.Formula != "" {
			// validate scalingModifiers struct and compile formula
			program, err := kedav1alpha1.ValidateAndCompileScalingModifiers(obj)
			if err != nil {
				log.Error(err, "error validating-compiling scalingModifiers")
				return nil, err
			}
			newCache.CompiledFormula = program
		}
		newCache.ScaledObject = obj
	default:
	}

	h.scalerCachesLock.Lock()
	defer h.scalerCachesLock.Unlock()

	if oldCache, ok := h.scalerCaches[key]; ok {
		// Scalers Close() could be impacted by timeouts, blocking the mutex
		// until the timeout happens. Instead of locking the mutex, we take
		// the old cache item and we close it in another goroutine, not locking
		// the cache: https://github.com/kedacore/keda/issues/5083
		go oldCache.Close(ctx)
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

// GetScaledObjectMetrics returns metrics for specified metric name for a ScaledObject identified by its name and namespace.
// It could either query the metric value directly from the scaler or from a cache, that's being stored for the scaler.
func (h *scaleHandler) GetScaledObjectMetrics(ctx context.Context, scaledObjectName, scaledObjectNamespace, metricsName string) (*external_metrics.ExternalMetricValueList, error) {
	logger := log.WithValues("scaledObject.Namespace", scaledObjectNamespace, "scaledObject.Name", scaledObjectName)
	var matchingMetrics []external_metrics.ExternalMetricValue
	var fallbackMetrics []external_metrics.ExternalMetricValue

	cache, err := h.getScalersCacheForScaledObject(ctx, scaledObjectName, scaledObjectNamespace)
	metricscollector.RecordScaledObjectError(scaledObjectNamespace, scaledObjectName, err)

	if err != nil {
		return nil, fmt.Errorf("error getting scalers %w", err)
	}

	var scaledObject *kedav1alpha1.ScaledObject
	if cache.ScaledObject != nil {
		scaledObject = cache.ScaledObject
	} else {
		err := fmt.Errorf("scaledObject not found in the cache")
		logger.Error(err, "scaledObject not found in the cache")
		return nil, err
	}
	isScalerError := false
	scaledObjectIdentifier := scaledObject.GenerateIdentifier()

	// returns all relevant metrics for current scaler (standard is one metric,
	// composite scaler gets all external metrics for further computation)
	metricsArray, err := h.getTrueMetricArray(ctx, metricsName, scaledObject)
	if err != nil {
		logger.Error(err, "error getting true metrics array, probably because of invalid cache")
	}
	metricTriggerPairList := make(map[string]string)
	isFallbackActive := false

	// let's check metrics for all scalers in a ScaledObject
	// as we can have multiple metrics in parallel for scaling modifiers
	// we parallelize the scalers process to speed up the
	// querying of the metric sources
	type metricResult struct {
		metrics           []external_metrics.ExternalMetricValue
		metricTriggerPair map[string]string
		metricName        string
		triggerName       string
		triggerIndex      int
		metricSpec        v2.MetricSpec
		err               error
	}
	allScalers, scalerConfigs := cache.GetScalers()
	// the matching metrics length has to be the same as required metrics length
	matchingMetricsChan := make(chan metricResult, len(metricsArray))
	wg := sync.WaitGroup{}
	for triggerIndex := 0; triggerIndex < len(allScalers); triggerIndex++ {
		triggerName := strings.Replace(fmt.Sprintf("%T", allScalers[triggerIndex]), "*scalers.", "", 1)
		if scalerConfigs[triggerIndex].TriggerName != "" {
			triggerName = scalerConfigs[triggerIndex].TriggerName
		}

		metricSpecs, err := cache.GetMetricSpecForScalingForScaler(ctx, triggerIndex)
		if err != nil {
			isScalerError = true
			logger.Error(err, "error getting metric spec for the scaler", "scaler", triggerName)
			cache.Recorder.Event(scaledObject, corev1.EventTypeWarning, eventreason.KEDAScalerFailed, err.Error())
		}

		if len(metricsArray) == 0 {
			err = fmt.Errorf("no metrics found getting metricsArray array %s", metricsName)
			logger.Error(err, "error metricsArray is empty")
			cache.Recorder.Event(scaledObject, corev1.EventTypeWarning, eventreason.KEDAScalerFailed, err.Error())
		}
		for _, spec := range metricSpecs {
			// skip cpu/memory resource scaler
			if spec.External == nil {
				continue
			}
			// Filter only the desired metric or if composite scaler is active,
			// metricsArray contains all external metrics
			if modifiers.ArrayContainsElement(spec.External.Metric.Name, metricsArray) {
				// if compositeScaler is used, override with current metric, otherwise do nothing
				metricName := spec.External.Metric.Name
				wg.Add(1)
				go func(results chan metricResult, wg *sync.WaitGroup, metricName string, triggerIndex int, scalerConfig scalersconfig.ScalerConfig, spec v2.MetricSpec) {
					result := metricResult{}

					// Pair metric values with their trigger names. This is applied only when
					// ScalingModifiers.Formula is defined in SO.
					result.metricTriggerPair, err = modifiers.GetPairTriggerAndMetric(scaledObject, metricName, scalerConfig.TriggerName)
					if err != nil {
						logger.Error(err, "error pairing triggers & metrics for compositeScaler")
					}
					var metrics []external_metrics.ExternalMetricValue

					// if cache is defined for this scaler/metric, let's try to hit it first
					metricsFoundInCache := false
					if scalerConfig.TriggerUseCachedMetrics {
						var metricsRecord metricscache.MetricsRecord
						if metricsRecord, metricsFoundInCache = h.scaledObjectsMetricCache.ReadRecord(scaledObjectIdentifier, metricName); metricsFoundInCache {
							logger.V(1).Info("Reading metrics from cache", "scaler", triggerName, "metricName", metricName, "metricsRecord", metricsRecord)
							metrics = metricsRecord.Metric
							err = metricsRecord.ScalerError
						}
					}

					if !metricsFoundInCache {
						var latency time.Duration
						metrics, _, latency, err = cache.GetMetricsAndActivityForScaler(ctx, triggerIndex, metricName)
						if latency != -1 {
							metricscollector.RecordScalerLatency(scaledObjectNamespace, scaledObject.Name, triggerName, triggerIndex, metricName, true, latency)
						}
						logger.V(1).Info("Getting metrics from trigger", "trigger", triggerName, "metricName", metricName, "metrics", metrics, "scalerError", err)
					}
					result.metricName = metricName
					result.triggerName = triggerName
					result.triggerIndex = triggerIndex
					result.metricSpec = spec
					result.metrics = metrics
					result.err = err
					results <- result
					wg.Done()
				}(matchingMetricsChan, &wg, metricName, triggerIndex, scalerConfigs[triggerIndex], spec)
			}
		}
	}

	wg.Wait()
	close(matchingMetricsChan)
	for result := range matchingMetricsChan {
		for key, value := range result.metricTriggerPair {
			metricTriggerPairList[key] = value
		}
		// check if we need to set a fallback
		metrics, fallbackActive, err := fallback.GetMetricsWithFallback(ctx, h.client, h.scaleClient, result.metrics, result.err, result.metricName, scaledObject, result.metricSpec)
		if err != nil {
			isScalerError = true
			logger.Error(err, "error getting metric for trigger", "trigger", result.triggerName)
		} else {
			for _, metric := range metrics {
				metricValue := metric.Value.AsApproximateFloat64()
				metricscollector.RecordScalerMetric(scaledObjectNamespace, scaledObjectName, result.triggerName, result.triggerIndex, metric.MetricName, true, metricValue)
			}
		}
		if fallbackActive {
			isFallbackActive = true
			fallbackMetrics = append(fallbackMetrics, metrics...)
		}
		metricscollector.RecordScalerError(scaledObjectNamespace, scaledObjectName, result.triggerName, result.triggerIndex, result.metricName, true, err)
		matchingMetrics = append(matchingMetrics, metrics...)
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

	// This case happens in failed times under failureThreshold. Report error to HPA directly.
	if !isFallbackActive && isScalerError {
		return nil, fmt.Errorf("metric:%s encountered error", metricsName)
	}

	if len(matchingMetrics) == 0 {
		return nil, fmt.Errorf("no matching metrics found for %s", metricsName)
	}

	// handle scalingModifiers here and simply return the matchingMetrics
	matchingMetrics = modifiers.HandleScalingModifiers(scaledObject, matchingMetrics, metricTriggerPairList, isFallbackActive, fallbackMetrics, cache, logger)
	return &external_metrics.ExternalMetricValueList{
		Items: matchingMetrics,
	}, nil
}

// getScaledObjectState returns whether the input ScaledObject:
// is active as the first return value,
// the second return value indicates whether there was any error during querying scalers,
// the third return value is a map of metrics record - a metric value for each scaler and its metric
// the fourth return value contains error if is not able to access scalers cache
func (h *scaleHandler) getScaledObjectState(ctx context.Context, scaledObject *kedav1alpha1.ScaledObject) (bool, bool, map[string]metricscache.MetricsRecord, []string, error) {
	logger := log.WithValues("scaledObject.Namespace", scaledObject.Namespace, "scaledObject.Name", scaledObject.Name)

	isScaledObjectActive := false
	isScaledObjectError := false
	metricsRecord := map[string]metricscache.MetricsRecord{}
	metricTriggerPairList := make(map[string]string)
	var matchingMetrics []external_metrics.ExternalMetricValue
	var activeTriggers []string

	cache, err := h.GetScalersCache(ctx, scaledObject)
	metricscollector.RecordScaledObjectError(scaledObject.Namespace, scaledObject.Name, err)
	if err != nil {
		return false, true, map[string]metricscache.MetricsRecord{}, []string{}, fmt.Errorf("error getting scalers cache %w", err)
	}

	// count the number of non-external triggers (cpu/mem) in order to check for
	// scale to zero requirements if atleast one cpu/mem trigger is given.
	// This is calculated here because of algorithm complexity but
	// evaluated in the loop below.
	cpuMemCount := 0
	for _, trigger := range scaledObject.Spec.Triggers {
		if trigger.Type == "cpu" || trigger.Type == "memory" {
			cpuMemCount++
		}
	}

	// Let's collect status of all allScalers in parallel,
	// no matter if any scaler raises error or is active
	allScalers, scalerConfigs := cache.GetScalers()
	results := make(chan scalerState, len(allScalers))
	wg := sync.WaitGroup{}
	for scalerIndex := 0; scalerIndex < len(allScalers); scalerIndex++ {
		wg.Add(1)
		go func(scaler scalers.Scaler, index int, scalerConfig scalersconfig.ScalerConfig, results chan scalerState, wg *sync.WaitGroup) {
			results <- h.getScalerState(ctx, scaler, index, scalerConfig, cache, logger, scaledObject)
			wg.Done()
		}(allScalers[scalerIndex], scalerIndex, scalerConfigs[scalerIndex], results, &wg)
	}
	wg.Wait()
	close(results)
	for result := range results {
		if result.IsActive {
			isScaledObjectActive = true
			activeTriggers = append(activeTriggers, result.TriggerName)
		}
		if result.Err != nil {
			isScaledObjectError = true
		}
		matchingMetrics = append(matchingMetrics, result.Metrics...)
		for k, v := range result.Pairs {
			metricTriggerPairList[k] = v
		}
		for k, v := range result.Records {
			metricsRecord[k] = v
		}

		metricscollector.RecordScaledObjectError(scaledObject.Namespace, scaledObject.Name, result.Err)
	}

	// invalidate the cache for the ScaledObject, if we hit an error in any scaler
	// in this case we try to build all scalers (and resolve all secrets/creds) again in the next call
	if isScaledObjectError {
		err := h.ClearScalersCache(ctx, scaledObject)
		if err != nil {
			logger.Error(err, "error clearing scalers cache")
		}
		logger.V(1).Info("scaler error encountered, clearing scaler cache")
	}

	// apply scaling modifiers
	matchingMetrics = modifiers.HandleScalingModifiers(scaledObject, matchingMetrics, metricTriggerPairList, false, nil, cache, logger)

	// when we are using formula, we need to reevaluate if it's active here
	if scaledObject.IsUsingModifiers() {
		// we need to reset the activity even if there is an error
		isScaledObjectActive = false
		activeTriggers = []string{}
		if !isScaledObjectError {
			activationValue := float64(0)
			if scaledObject.Spec.Advanced.ScalingModifiers.ActivationTarget != "" {
				targetValue, err := strconv.ParseFloat(scaledObject.Spec.Advanced.ScalingModifiers.ActivationTarget, 64)
				if err != nil {
					return false, true, metricsRecord, []string{}, fmt.Errorf("scalingModifiers.ActivationTarget parsing error %w", err)
				}
				activationValue = targetValue
			}

			for _, metric := range matchingMetrics {
				value := metric.Value.AsApproximateFloat64()
				metricscollector.RecordScalerMetric(scaledObject.Namespace, scaledObject.Name, kedav1alpha1.CompositeMetricName, 0, metric.MetricName, true, value)
				metricscollector.RecordScalerActive(scaledObject.Namespace, scaledObject.Name, kedav1alpha1.CompositeMetricName, 0, metric.MetricName, true, value > activationValue)
				if !isScaledObjectActive {
					isScaledObjectActive = value > activationValue

					if isScaledObjectActive {
						activeTriggers = append(activeTriggers, "ModifiersTrigger")
					}
				}
			}
		}
	}

	// cpu/memory scaler only can scale to zero if there is any other external metric because otherwise
	// it'll never scale from 0. If all the triggers are only cpu/memory, we enforce the IsActive
	if len(scaledObject.Spec.Triggers) <= cpuMemCount && !isScaledObjectError {
		isScaledObjectActive = true
	}
	return isScaledObjectActive, isScaledObjectError, metricsRecord, activeTriggers, err
}

// scalerState is used as return
// for the function getScalerState. It contains
// the state of the scaler and all the required
// info for calculating the ScaledObjectState
type scalerState struct {
	// IsActive will be overrided by formula calculation
	IsActive    bool
	TriggerName string
	Metrics     []external_metrics.ExternalMetricValue
	Pairs       map[string]string
	Records     map[string]metricscache.MetricsRecord
	Err         error
}

// getScalerState returns getStateScalerResult with the state
// for an specific scaler. The state contains if it's active or
// with erros, but also the records for the cache and he metrics
// for the custom formulas
func (*scaleHandler) getScalerState(ctx context.Context, scaler scalers.Scaler, triggerIndex int, scalerConfig scalersconfig.ScalerConfig,
	cache *cache.ScalersCache, logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject) scalerState {
	result := scalerState{
		IsActive:    false,
		Err:         nil,
		TriggerName: "",
		Metrics:     []external_metrics.ExternalMetricValue{},
		Pairs:       map[string]string{},
		Records:     map[string]metricscache.MetricsRecord{},
	}

	result.TriggerName = strings.Replace(fmt.Sprintf("%T", scaler), "*scalers.", "", 1)
	if scalerConfig.TriggerName != "" {
		result.TriggerName = scalerConfig.TriggerName
	}

	metricSpecs, err := cache.GetMetricSpecForScalingForScaler(ctx, triggerIndex)
	if err != nil {
		result.Err = err
		logger.Error(err, "error getting metric spec for the scaler", "scaler", result.TriggerName)
		cache.Recorder.Event(scaledObject, corev1.EventTypeWarning, eventreason.KEDAScalerFailed, err.Error())
	}

	for _, spec := range metricSpecs {
		if spec.External == nil {
			continue
		}

		metricName := spec.External.Metric.Name

		var latency time.Duration
		metrics, isMetricActive, latency, err := cache.GetMetricsAndActivityForScaler(ctx, triggerIndex, metricName)
		metricscollector.RecordScalerError(scaledObject.Namespace, scaledObject.Name, result.TriggerName, triggerIndex, metricName, true, err)
		if latency != -1 {
			metricscollector.RecordScalerLatency(scaledObject.Namespace, scaledObject.Name, result.TriggerName, triggerIndex, metricName, true, latency)
		}
		result.Metrics = append(result.Metrics, metrics...)
		logger.V(1).Info("Getting metrics and activity from scaler", "scaler", result.TriggerName, "metricName", metricName, "metrics", metrics, "activity", isMetricActive, "scalerError", err)

		if scalerConfig.TriggerUseCachedMetrics {
			result.Records[metricName] = metricscache.MetricsRecord{
				IsActive:    isMetricActive,
				Metric:      metrics,
				ScalerError: err,
			}
		}

		if err != nil {
			result.Err = err
			if scaledObject.IsUsingModifiers() {
				logger.Error(err, "error getting metric source", "source", result.TriggerName)
				cache.Recorder.Event(scaledObject, corev1.EventTypeWarning, eventreason.KEDAMetricSourceFailed, err.Error())
			} else {
				logger.Error(err, "error getting scale decision", "scaler", result.TriggerName)
				cache.Recorder.Event(scaledObject, corev1.EventTypeWarning, eventreason.KEDAScalerFailed, err.Error())
			}
		} else {
			result.IsActive = isMetricActive
			for _, metric := range metrics {
				metricValue := metric.Value.AsApproximateFloat64()
				metricscollector.RecordScalerMetric(scaledObject.Namespace, scaledObject.Name, result.TriggerName, triggerIndex, metric.MetricName, true, metricValue)
			}
			if !scaledObject.IsUsingModifiers() {
				if isMetricActive {
					if spec.External != nil {
						logger.V(1).Info("Scaler for scaledObject is active", "scaler", result.TriggerName, "metricName", metricName)
					}
					if spec.Resource != nil {
						logger.V(1).Info("Scaler for scaledObject is active", "scaler", result.TriggerName, "metricName", spec.Resource.Name)
					}
				}
				metricscollector.RecordScalerActive(scaledObject.Namespace, scaledObject.Name, result.TriggerName, triggerIndex, metricName, true, isMetricActive)
			}
		}

		result.Pairs, err = modifiers.GetPairTriggerAndMetric(scaledObject, metricName, scalerConfig.TriggerName)
		if err != nil {
			logger.Error(err, "error pairing triggers & metrics for compositeScaler")
		}
	}
	return result
}

// / --------------------------------------------------------------------------- ///
// / ----------             ScaledJob related methods               --------- ///
// / --------------------------------------------------------------------------- ///

// getScaledJobMetrics returns metrics for specified metric name for a ScaledJob identified by its name and namespace.
// It could either query the metric value directly from the scaler or from a cache, that's being stored for the scaler.
func (h *scaleHandler) getScaledJobMetrics(ctx context.Context, scaledJob *kedav1alpha1.ScaledJob) ([]scaledjob.ScalerMetrics, bool) {
	logger := log.WithValues("scaledJob.Namespace", scaledJob.Namespace, "scaledJob.Name", scaledJob.Name)

	cache, err := h.GetScalersCache(ctx, scaledJob)
	metricscollector.RecordScaledJobError(scaledJob.Namespace, scaledJob.Name, err)
	if err != nil {
		log.Error(err, "error getting scalers cache", "scaledJob.Namespace", scaledJob.Namespace, "scaledJob.Name", scaledJob.Name)
		return nil, true
	}
	var isError bool
	var scalersMetrics []scaledjob.ScalerMetrics
	scalers, scalerConfigs := cache.GetScalers()
	for scalerIndex, scaler := range scalers {
		scalerName := strings.Replace(fmt.Sprintf("%T", scalers[scalerIndex]), "*scalers.", "", 1)
		if scalerConfigs[scalerIndex].TriggerName != "" {
			scalerName = scalerConfigs[scalerIndex].TriggerName
		}
		isActive := false
		scalerType := fmt.Sprintf("%T:", scaler)

		scalerLogger := log.WithValues("scaledJob.Name", scaledJob.Name, "Scaler", scalerType)

		metricSpecs := scaler.GetMetricSpecForScaling(ctx)

		for _, spec := range metricSpecs {
			// skip scaler that doesn't return any metric specs (usually External scaler with incorrect metadata)
			// or skip cpu/memory resource scaler
			if len(metricSpecs) < 1 || spec.External == nil {
				continue
			}
			metricName := spec.External.Metric.Name
			metrics, isTriggerActive, latency, err := cache.GetMetricsAndActivityForScaler(ctx, scalerIndex, metricName)
			metricscollector.RecordScaledJobError(scaledJob.Namespace, scaledJob.Name, err)
			if latency != -1 {
				metricscollector.RecordScalerLatency(scaledJob.Namespace, scaledJob.Name, scalerName, scalerIndex, metricName, false, latency)
			}
			if err != nil {
				scalerLogger.Error(err, "Error getting scaler metrics and activity, but continue")
				cache.Recorder.Event(scaledJob, corev1.EventTypeWarning, eventreason.KEDAScalerFailed, err.Error())
				isError = true
				continue
			}
			if isTriggerActive {
				isActive = true
			}
			queueLength, maxValue, targetAverageValue := scaledjob.CalculateQueueLengthAndMaxValue(metrics, metricSpecs, scaledJob.MaxReplicaCount())

			scalerLogger.V(1).Info("Scaler Metric value", "isTriggerActive", isTriggerActive, metricSpecs[0].External.Metric.Name, queueLength, "targetAverageValue", targetAverageValue)

			scalersMetrics = append(scalersMetrics, scaledjob.ScalerMetrics{
				QueueLength: queueLength,
				MaxValue:    maxValue,
				IsActive:    isActive,
			})
			for _, metric := range metrics {
				metricValue := metric.Value.AsApproximateFloat64()
				metricscollector.RecordScalerMetric(scaledJob.Namespace, scaledJob.Name, scalerName, scalerIndex, metric.MetricName, false, metricValue)
			}

			if isTriggerActive {
				if spec.External != nil {
					logger.V(1).Info("Scaler for scaledJob is active", "scaler", scalerName, "metricName", metricName)
				}
				if spec.Resource != nil {
					logger.V(1).Info("Scaler for scaledJob is active", "scaler", scalerName, "metricName", spec.Resource.Name)
				}
			}

			metricscollector.RecordScalerError(scaledJob.Namespace, scaledJob.Name, scalerName, scalerIndex, metricName, false, err)
			metricscollector.RecordScalerActive(scaledJob.Namespace, scaledJob.Name, scalerName, scalerIndex, metricName, false, isTriggerActive)
		}
	}
	return scalersMetrics, isError
}

// isScaledJobActive returns whether the input ScaledJob:
// is active as the first return value,
// the second and the third return values indicate queueLength and maxValue for scale
func (h *scaleHandler) isScaledJobActive(ctx context.Context, scaledJob *kedav1alpha1.ScaledJob) (bool, bool, int64, int64) {
	logger := logf.Log.WithName("scalemetrics")

	scalersMetrics, isError := h.getScaledJobMetrics(ctx, scaledJob)
	isActive, queueLength, maxValue, maxFloatValue :=
		scaledjob.IsScaledJobActive(scalersMetrics, scaledJob.Spec.ScalingStrategy.MultipleScalersCalculation, scaledJob.MinReplicaCount(), scaledJob.MaxReplicaCount())

	logger.V(1).WithValues("scaledJob.Name", scaledJob.Name).Info("Checking if ScaleJob Scalers are active", "isActive", isActive, "maxValue", maxFloatValue, "MultipleScalersCalculation", scaledJob.Spec.ScalingStrategy.MultipleScalersCalculation)
	return isActive, isError, queueLength, maxValue
}

// getTrueMetricArray is a help function made for composite scaler to determine
// what metrics should be used. In case of composite scaler (ScalingModifiers struct),
// all external metrics will be used. Returns all external metrics otherwise it
// returns the same metric given.
// TODO: if the bug (mentioned in function below) is fixed this can be moved to
// 'modifiers/' directory with the rest of the functions
func (h *scaleHandler) getTrueMetricArray(ctx context.Context, metricName string, so *kedav1alpha1.ScaledObject) ([]string, error) {
	// if composite scaler is given return all external metrics

	// bug fix for the invalid cache (not loaded properly) and needs to be fetched again
	// Tracking issue: https://github.com/kedacore/keda/issues/4955
	if so != nil && so.Spec.Advanced != nil && so.Spec.Advanced.ScalingModifiers.Target != "" {
		if len(so.Status.ExternalMetricNames) == 0 {
			scaledObject := &kedav1alpha1.ScaledObject{}
			err := h.client.Get(ctx, types.NamespacedName{Name: so.Name, Namespace: so.Namespace}, scaledObject)
			if err != nil {
				log.Error(err, "failed to get ScaledObject", "name", so.Name, "namespace", so.Namespace)
				return nil, err
			}
			if len(scaledObject.Status.ExternalMetricNames) == 0 {
				err := fmt.Errorf("failed to get ScaledObject.Status.ExternalMetricNames, probably invalid ScaledObject cache")
				log.Error(err, "failed to get ScaledObject.Status.ExternalMetricNames, probably invalid ScaledObject cache", "scaledObject.Name", scaledObject.Name, "scaledObject.Namespace", scaledObject.Namespace)
				return nil, err
			}

			so = scaledObject
		}
		return so.Status.ExternalMetricNames, nil
	}
	return []string{metricName}, nil
}
