package scaledjob

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v2 "k8s.io/api/autoscaling/v2"
)

func TestTargetAverageValue(t *testing.T) {
	// count = 0
	specs := []v2.MetricSpec{}
	metricName := "s0-messageCount"
	targetAverageValue := GetTargetAverageValue(specs)
	assert.Equal(t, float64(0), targetAverageValue)
	// 1 1
	specs = []v2.MetricSpec{
		CreateMetricSpec(1, metricName),
		CreateMetricSpec(1, metricName),
	}
	targetAverageValue = GetTargetAverageValue(specs)
	assert.Equal(t, float64(1), targetAverageValue)
	// 5 5 3 -> 4.333333333333333
	specs = []v2.MetricSpec{
		CreateMetricSpec(5, metricName),
		CreateMetricSpec(5, metricName),
		CreateMetricSpec(3, metricName),
	}
	targetAverageValue = GetTargetAverageValue(specs)
	assert.Equal(t, 4.333333333333333, targetAverageValue)

	// 5 5 4 -> 4.666666666666667
	specs = []v2.MetricSpec{
		CreateMetricSpec(5, metricName),
		CreateMetricSpec(5, metricName),
		CreateMetricSpec(4, metricName),
	}
	targetAverageValue = GetTargetAverageValue(specs)
	assert.Equal(t, 4.666666666666667, targetAverageValue)
}
