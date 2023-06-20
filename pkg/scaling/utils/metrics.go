package utils

import v2 "k8s.io/api/autoscaling/v2"

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
