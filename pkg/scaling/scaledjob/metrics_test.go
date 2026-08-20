package scaledjob

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestTargetAverageValue(t *testing.T) {
	// count = 0
	var specs []v2.MetricSpec
	metricName := "s0-messageCount"
	targetAverageValue := getTargetAverageValue(specs)
	assert.Equal(t, float64(0), targetAverageValue)
	// 1 1
	specs = []v2.MetricSpec{
		createMetricSpec(1, metricName),
		createMetricSpec(1, metricName),
	}

	metricName = "s1-messageCount"
	targetAverageValue = getTargetAverageValue(specs)
	assert.Equal(t, float64(1), targetAverageValue)
	// 5 5 3 -> 4.333333333333333
	specs = []v2.MetricSpec{
		createMetricSpec(5, metricName),
		createMetricSpec(5, metricName),
		createMetricSpec(3, metricName),
	}

	metricName = "s2-messageCount"
	targetAverageValue = getTargetAverageValue(specs)
	assert.Equal(t, 4.333333333333333, targetAverageValue)

	// 5 5 4 -> 4.666666666666667
	specs = []v2.MetricSpec{
		createMetricSpec(5, metricName),
		createMetricSpec(5, metricName),
		createMetricSpec(4, metricName),
	}

	targetAverageValue = getTargetAverageValue(specs)
	assert.Equal(t, 4.666666666666667, targetAverageValue)
}

func TestCeilToInt64(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected int64
	}{
		{name: "positive integer", input: 5.0, expected: 5},
		{name: "positive ceil", input: 4.2, expected: 5},
		{name: "zero", input: 0.0, expected: 0},
		{name: "negative value", input: -2.5, expected: -2},
		{name: "max int64 boundary", input: float64(math.MaxInt64), expected: math.MaxInt64},
		{name: "above max int64", input: 1e19, expected: math.MaxInt64},
		{name: "2^63 exactly", input: math.Float64frombits(0x43e0000000000000), expected: math.MaxInt64},
		{name: "below min int64", input: -1e19, expected: math.MinInt64},
		{name: "positive infinity", input: math.Inf(1), expected: math.MaxInt64},
		{name: "negative infinity", input: math.Inf(-1), expected: math.MinInt64},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, ceilToInt64(tt.input))
		})
	}
}

// createMetricSpec creates MetricSpec for given metric name and target value.
func createMetricSpec(averageValue int64, metricName string) v2.MetricSpec {
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
