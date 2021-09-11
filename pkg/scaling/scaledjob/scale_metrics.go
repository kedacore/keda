package scaledjob

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/eventreason"
	"github.com/kedacore/keda/v2/pkg/scalers"
	"k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type scalerMetrics struct {
	queueLength int64
	maxValue    int64
	isActive    bool
}

// GetScaleMetrics gets the metrics for decision making of scaling.
func GetScaleMetrics(ctx context.Context, scalers []scalers.Scaler, scaledJob *kedav1alpha1.ScaledJob, recorder record.EventRecorder) (bool, int64, int64) {
	var queueLength int64
	var maxValue int64
	isActive := false

	logger := logf.Log.WithName("scalemetrics")
	scalersMetrics := getScalersMetrics(ctx, scalers, scaledJob, logger, recorder)
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
		queueLengthSum := int64(0)
		maxValueSum := int64(0)
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
			queueLength = divideWithCeil(queueLengthSum, int64(length))
			maxValue = divideWithCeil(maxValueSum, int64(length))
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
	maxValue = min(scaledJob.MaxReplicaCount(), maxValue)
	logger.V(1).WithValues("ScaledJob", scaledJob.Name).Info("Checking if ScaleJob scalers are active", "isActive", isActive, "maxValue", maxValue, "MultipleScalersCalculation", scaledJob.Spec.ScalingStrategy.MultipleScalersCalculation)

	return isActive, queueLength, maxValue
}

func getScalersMetrics(ctx context.Context, scalers []scalers.Scaler, scaledJob *kedav1alpha1.ScaledJob, logger logr.Logger, recorder record.EventRecorder) []scalerMetrics {
	scalersMetrics := []scalerMetrics{}

	for _, scaler := range scalers {
		var queueLength int64
		var targetAverageValue int64
		isActive := false
		maxValue := int64(0)
		scalerType := fmt.Sprintf("%T:", scaler)

		scalerLogger := logger.WithValues("ScaledJob", scaledJob.Name, "Scaler", scalerType)

		metricSpecs := scaler.GetMetricSpecForScaling()

		// skip scaler that doesn't return any metric specs (usually External scaler with incorrect metadata)
		// or skip cpu/memory resource scaler
		if len(metricSpecs) < 1 || metricSpecs[0].External == nil {
			continue
		}

		isTriggerActive, err := scaler.IsActive(ctx)
		if err != nil {
			scalerLogger.V(1).Info("Error getting scaler.IsActive, but continue", "Error", err)
			recorder.Event(scaledJob, corev1.EventTypeWarning, eventreason.KEDAScalerFailed, err.Error())
			scaler.Close()
			continue
		}

		targetAverageValue = getTargetAverageValue(metricSpecs)

		metrics, err := scaler.GetMetrics(ctx, "queueLength", nil)
		if err != nil {
			scalerLogger.V(1).Info("Error getting scaler metrics, but continue", "Error", err)
			recorder.Event(scaledJob, corev1.EventTypeWarning, eventreason.KEDAScalerFailed, err.Error())
			scaler.Close()
			continue
		}

		var metricValue int64

		for _, m := range metrics {
			if m.MetricName == "queueLength" {
				metricValue, _ = m.Value.AsInt64()
				queueLength += metricValue
			}
		}
		scalerLogger.V(1).Info("Scaler Metric value", "isTriggerActive", isTriggerActive, "queueLength", queueLength, "targetAverageValue", targetAverageValue)

		scaler.Close()

		if isTriggerActive {
			isActive = true
		}

		if targetAverageValue != 0 {
			maxValue = min(scaledJob.MaxReplicaCount(), divideWithCeil(queueLength, targetAverageValue))
		}
		scalersMetrics = append(scalersMetrics, scalerMetrics{
			queueLength: queueLength,
			maxValue:    maxValue,
			isActive:    isActive,
		})
	}
	return scalersMetrics
}

func getTargetAverageValue(metricSpecs []v2beta2.MetricSpec) int64 {
	var targetAverageValue int64
	var metricValue int64
	var flag bool
	for _, metric := range metricSpecs {
		if metric.External.Target.AverageValue == nil {
			metricValue = 0
		} else {
			metricValue, flag = metric.External.Target.AverageValue.AsInt64()
			if !flag {
				metricValue = 0
			}
		}

		targetAverageValue += metricValue
	}
	count := int64(len(metricSpecs))
	if count != 0 {
		return targetAverageValue / count
	}
	return 0
}

func divideWithCeil(x, y int64) int64 {
	ans := x / y
	reminder := x % y
	if reminder != 0 {
		return ans + 1
	}
	return ans
}

// Min function for int64
func min(x, y int64) int64 {
	if x > y {
		return y
	}
	return x
}
