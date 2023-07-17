package scaledjob

import (
	"math"

	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

// GetTargetAverageValue returns the average of all the metrics' average value.
func getTargetAverageValue(metricSpecs []v2.MetricSpec) float64 {
	var totalAverageValue float64
	var metricValue float64
	for _, metric := range metricSpecs {
		if metric.External.Target.AverageValue == nil {
			metricValue = 0
		} else {
			metricValue = metric.External.Target.AverageValue.AsApproximateFloat64()
		}

		totalAverageValue += metricValue
	}
	count := float64(len(metricSpecs))
	if count != 0 {
		return totalAverageValue / count
	}
	return 0
}

// CalculateQueueLengthAndMaxValue returns queueLength, maxValue, and targetAverageValue for the given metrics
func CalculateQueueLengthAndMaxValue(metrics []external_metrics.ExternalMetricValue, metricSpecs []v2.MetricSpec, maxReplicaCount int64) (queueLength, maxValue, targetAverageValue float64) {
	var metricValue float64
	for _, m := range metrics {
		if m.MetricName == metricSpecs[0].External.Metric.Name {
			metricValue = m.Value.AsApproximateFloat64()
			queueLength += metricValue
		}
	}
	targetAverageValue = getTargetAverageValue(metricSpecs)
	if targetAverageValue != 0 {
		averageLength := queueLength / targetAverageValue
		maxValue = getMaxValue(averageLength, maxReplicaCount)
	}
	return queueLength, maxValue, targetAverageValue
}

type ScalerMetrics struct {
	QueueLength float64
	MaxValue    float64
	IsActive    bool
}

// IsScaledJobActive returns whether the input ScaledJob is active and queueLength and maxValue for scale
func IsScaledJobActive(scalersMetrics []ScalerMetrics, multipleScalersCalculation string, minReplicaCount, maxReplicaCount int64) (bool, int64, int64, float64) {
	var queueLength float64
	var maxValue float64
	isActive := false

	switch multipleScalersCalculation {
	case "min":
		for _, metrics := range scalersMetrics {
			if (queueLength == 0 || metrics.QueueLength < queueLength) && metrics.IsActive {
				queueLength = metrics.QueueLength
				maxValue = metrics.MaxValue
				isActive = metrics.IsActive
			}
		}
	case "avg":
		queueLengthSum := float64(0)
		maxValueSum := float64(0)
		length := 0
		for _, metrics := range scalersMetrics {
			if metrics.IsActive {
				queueLengthSum += metrics.QueueLength
				maxValueSum += metrics.MaxValue
				isActive = metrics.IsActive
				length++
			}
		}
		if length != 0 {
			queueLength = queueLengthSum / float64(length)
			maxValue = maxValueSum / float64(length)
		}
	case "sum":
		for _, metrics := range scalersMetrics {
			if metrics.IsActive {
				queueLength += metrics.QueueLength
				maxValue += metrics.MaxValue
				isActive = metrics.IsActive
			}
		}
	default: // max
		for _, metrics := range scalersMetrics {
			if metrics.QueueLength > queueLength && metrics.IsActive {
				queueLength = metrics.QueueLength
				maxValue = metrics.MaxValue
				isActive = metrics.IsActive
			}
		}
	}

	if minReplicaCount > 0 {
		isActive = true
	}

	maxValue = getMaxValue(maxValue, maxReplicaCount)
	return isActive, ceilToInt64(queueLength), ceilToInt64(maxValue), maxValue
}

// ceilToInt64 returns the int64 ceil value for the float64 input
func ceilToInt64(x float64) int64 {
	return int64(math.Ceil(x))
}

// min returns the minimum for input float64 values
func min(x, y float64) float64 {
	if x > y {
		return y
	}
	return x
}

// getMaxValue returns maxValue, unless it is exceeding the MaxReplicaCount.
func getMaxValue(maxValue float64, maxReplicaCount int64) float64 {
	return min(maxValue, float64(maxReplicaCount))
}
