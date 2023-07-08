package scaledjob

import (
	"math"

	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/resource"
)

// GetTargetAverageValue returns the average of all the metrics' average value.
func GetTargetAverageValue(metricSpecs []v2.MetricSpec) float64 {
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

// CreateMetricSpec creates MetricSpec for given metric name and target value.
func CreateMetricSpec(averageValue int64, metricName string) v2.MetricSpec {
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

type ScalerMetrics struct {
	QueueLength float64
	MaxValue    float64
	IsActive    bool
}

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

	maxValue = GetMaxValue(maxValue, maxReplicaCount)
	return isActive, ceilToInt64(queueLength), ceilToInt64(maxValue), maxValue
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

func GetMaxValue(maxValue float64, maxReplicaCount int64) float64 {
	return min(maxValue, float64(maxReplicaCount))
}
