package utils

import (
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/resource"
)

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
