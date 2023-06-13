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

package cache

import (
	"context"
	"fmt"
	"math"
	"time"

	v2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/eventreason"
	"github.com/kedacore/keda/v2/pkg/scalers"
)

var log = logf.Log.WithName("scalers_cache")

type ScalersCache struct {
	ScaledObject             *kedav1alpha1.ScaledObject
	Scalers                  []ScalerBuilder
	ScalableObjectGeneration int64
	Recorder                 record.EventRecorder
}

type ScalerBuilder struct {
	Scaler       scalers.Scaler
	ScalerConfig scalers.ScalerConfig
	Factory      func() (scalers.Scaler, *scalers.ScalerConfig, error)
}

// GetScalers returns array of scalers and scaler config stored in the cache
func (c *ScalersCache) GetScalers() ([]scalers.Scaler, []scalers.ScalerConfig) {
	scalersList := make([]scalers.Scaler, 0, len(c.Scalers))
	configsList := make([]scalers.ScalerConfig, 0, len(c.Scalers))
	for _, s := range c.Scalers {
		scalersList = append(scalersList, s.Scaler)
		configsList = append(configsList, s.ScalerConfig)
	}

	return scalersList, configsList
}

// GetPushScaler returns array of push scalers stored in the cache
func (c *ScalersCache) GetPushScalers() []scalers.PushScaler {
	var result []scalers.PushScaler
	for _, s := range c.Scalers {
		if ps, ok := s.Scaler.(scalers.PushScaler); ok {
			result = append(result, ps)
		}
	}
	return result
}

// Close closes all scalers in the cache
func (c *ScalersCache) Close(ctx context.Context) {
	scalers := c.Scalers
	c.Scalers = nil
	for _, s := range scalers {
		err := s.Scaler.Close(ctx)
		if err != nil {
			log.Error(err, "error closing scaler", "scaler", s)
		}
	}
}

// GetMetricSpecForScaling returns metrics specs for all scalers in the cache
func (c *ScalersCache) GetMetricSpecForScaling(ctx context.Context) []v2.MetricSpec {
	var spec []v2.MetricSpec
	for _, s := range c.Scalers {
		spec = append(spec, s.Scaler.GetMetricSpecForScaling(ctx)...)
	}
	return spec
}

// GetMetricSpecForScalingForScaler returns metrics spec for a scaler identified by the metric name
func (c *ScalersCache) GetMetricSpecForScalingForScaler(ctx context.Context, index int) ([]v2.MetricSpec, error) {
	var err error

	scalersList, _ := c.GetScalers()
	if index < 0 || index >= len(scalersList) {
		return nil, fmt.Errorf("scaler with id %d not found. Len = %d", index, len(c.Scalers))
	}

	metricSpecs := scalersList[index].GetMetricSpecForScaling(ctx)

	// no metric spec returned for a scaler -> this could signal error during connection to the scaler
	// usually in case this is an external scaler
	// let's try to refresh the scaler and query metrics spec again
	if len(metricSpecs) < 1 {
		var ns scalers.Scaler
		ns, err = c.refreshScaler(ctx, index)
		if err == nil {
			metricSpecs = ns.GetMetricSpecForScaling(ctx)
			if len(metricSpecs) < 1 {
				err = fmt.Errorf("got empty metric spec")
			}
		}
	}

	return metricSpecs, err
}

// GetMetricsAndActivityForScaler returns metric value, activity and latency for a scaler identified by the metric name
// and by the input index (from the list of scalers in this ScaledObject)
func (c *ScalersCache) GetMetricsAndActivityForScaler(ctx context.Context, index int, metricName string) ([]external_metrics.ExternalMetricValue, bool, int64, error) {
	if index < 0 || index >= len(c.Scalers) {
		return nil, false, -1, fmt.Errorf("scaler with id %d not found. Len = %d", index, len(c.Scalers))
	}
	startTime := time.Now()
	metric, activity, err := c.Scalers[index].Scaler.GetMetricsAndActivity(ctx, metricName)
	if err == nil {
		return metric, activity, time.Since(startTime).Milliseconds(), nil
	}

	ns, err := c.refreshScaler(ctx, index)
	if err != nil {
		return nil, false, -1, err
	}
	startTime = time.Now()
	metric, activity, err = ns.GetMetricsAndActivity(ctx, metricName)
	return metric, activity, time.Since(startTime).Milliseconds(), err
}

// TODO needs refactor - move ScaledJob related methods to scale_handler, the similar way ScaledObject methods are
// refactor logic
func (c *ScalersCache) IsScaledJobActive(ctx context.Context, scaledJob *kedav1alpha1.ScaledJob) (bool, int64, int64) {
	var queueLength float64
	var maxValue float64
	isActive := false

	logger := logf.Log.WithName("scalemetrics")
	scalersMetrics := c.getScaledJobMetrics(ctx, scaledJob)
	switch scaledJob.Spec.ScalingStrategy.MultipleScalersCalculation {
	case "min":
		for _, metrics := range scalersMetrics {
			if (queueLength == 0 || metrics.queueLength < queueLength) && metrics.isActive {
				queueLength = metrics.queueLength
				maxValue = metrics.maxValue
				isActive = metrics.isActive
			}
		}
	case "avg":
		queueLengthSum := float64(0)
		maxValueSum := float64(0)
		length := 0
		for _, metrics := range scalersMetrics {
			if metrics.isActive {
				queueLengthSum += metrics.queueLength
				maxValueSum += metrics.maxValue
				isActive = metrics.isActive
				length++
			}
		}
		if length != 0 {
			queueLength = queueLengthSum / float64(length)
			maxValue = maxValueSum / float64(length)
		}
	case "sum":
		for _, metrics := range scalersMetrics {
			if metrics.isActive {
				queueLength += metrics.queueLength
				maxValue += metrics.maxValue
				isActive = metrics.isActive
			}
		}
	default: // max
		for _, metrics := range scalersMetrics {
			if metrics.queueLength > queueLength && metrics.isActive {
				queueLength = metrics.queueLength
				maxValue = metrics.maxValue
				isActive = metrics.isActive
			}
		}
	}

	if scaledJob.MinReplicaCount() > 0 {
		isActive = true
	}

	maxValue = min(float64(scaledJob.MaxReplicaCount()), maxValue)
	logger.V(1).WithValues("ScaledJob", scaledJob.Name).Info("Checking if ScaleJob Scalers are active", "isActive", isActive, "maxValue", maxValue, "MultipleScalersCalculation", scaledJob.Spec.ScalingStrategy.MultipleScalersCalculation)

	return isActive, ceilToInt64(queueLength), ceilToInt64(maxValue)
}

func (c *ScalersCache) refreshScaler(ctx context.Context, id int) (scalers.Scaler, error) {
	if id < 0 || id >= len(c.Scalers) {
		return nil, fmt.Errorf("scaler with id %d not found, len = %d, cache has been probably already invalidated", id, len(c.Scalers))
	}

	sb := c.Scalers[id]
	defer sb.Scaler.Close(ctx)
	ns, sConfig, err := sb.Factory()
	if err != nil {
		return nil, err
	}

	if id < 0 || id >= len(c.Scalers) {
		return nil, fmt.Errorf("scaler with id %d not found, len = %d, cache has been probably already invalidated", id, len(c.Scalers))
	}
	c.Scalers[id] = ScalerBuilder{
		Scaler:       ns,
		ScalerConfig: *sConfig,
		Factory:      sb.Factory,
	}

	return ns, nil
}

type scalerMetrics struct {
	queueLength float64
	maxValue    float64
	isActive    bool
}

func (c *ScalersCache) getScaledJobMetrics(ctx context.Context, scaledJob *kedav1alpha1.ScaledJob) []scalerMetrics {
	// TODO this loop should be probably done similar way the ScaledObject loop is done
	var scalersMetrics []scalerMetrics
	for i, s := range c.Scalers {
		var queueLength float64
		var targetAverageValue float64
		isActive := false
		maxValue := float64(0)
		scalerType := fmt.Sprintf("%T:", s)

		scalerLogger := log.WithValues("ScaledJob", scaledJob.Name, "Scaler", scalerType)

		metricSpecs := s.Scaler.GetMetricSpecForScaling(ctx)

		// skip scaler that doesn't return any metric specs (usually External scaler with incorrect metadata)
		// or skip cpu/memory resource scaler
		if len(metricSpecs) < 1 || metricSpecs[0].External == nil {
			continue
		}

		// TODO here we should probably loop through all metrics in a Scaler
		metrics, isTriggerActive, _, err := c.GetMetricsAndActivityForScaler(ctx, i, metricSpecs[0].External.Metric.Name)
		if err != nil {
			scalerLogger.V(1).Info("Error getting scaler metrics and activity, but continue", "error", err)
			c.Recorder.Event(scaledJob, corev1.EventTypeWarning, eventreason.KEDAScalerFailed, err.Error())
			continue
		}

		targetAverageValue = getTargetAverageValue(metricSpecs)

		var metricValue float64
		for _, m := range metrics {
			if m.MetricName == metricSpecs[0].External.Metric.Name {
				metricValue = m.Value.AsApproximateFloat64()
				queueLength += metricValue
			}
		}
		scalerLogger.V(1).Info("Scaler Metric value", "isTriggerActive", isTriggerActive, metricSpecs[0].External.Metric.Name, queueLength, "targetAverageValue", targetAverageValue)

		if isTriggerActive {
			isActive = true
		}

		if targetAverageValue != 0 {
			averageLength := queueLength / targetAverageValue
			maxValue = min(float64(scaledJob.MaxReplicaCount()), averageLength)
		}
		scalersMetrics = append(scalersMetrics, scalerMetrics{
			queueLength: queueLength,
			maxValue:    maxValue,
			isActive:    isActive,
		})
	}
	return scalersMetrics
}

func getTargetAverageValue(metricSpecs []v2.MetricSpec) float64 {
	var targetAverageValue float64
	var metricValue float64
	for _, metric := range metricSpecs {
		if metric.External.Target.AverageValue == nil {
			metricValue = 0
		} else {
			metricValue = metric.External.Target.AverageValue.AsApproximateFloat64()
		}

		targetAverageValue += metricValue
	}
	count := float64(len(metricSpecs))
	if count != 0 {
		return targetAverageValue / count
	}
	return 0
}

func ceilToInt64(x float64) int64 {
	return int64(math.Ceil(x))
}

// Min function for float64
func min(x, y float64) float64 {
	if x > y {
		return y
	}
	return x
}
