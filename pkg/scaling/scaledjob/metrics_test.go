package scaledjob

import (
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
