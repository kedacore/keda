package scaling

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestTargetAverageValue(t *testing.T) {
	// count = 0
	specs := []v2beta2.MetricSpec{}
	targetAverageValue := getTargetAverageValue(specs)
	assert.Equal(t, int64(0), targetAverageValue)
	// 1 1
	specs = []v2beta2.MetricSpec{
		createMetricSpec(1),
		createMetricSpec(1),
	}
	targetAverageValue = getTargetAverageValue(specs)
	assert.Equal(t, int64(1), targetAverageValue)
	// 5 5 3
	specs = []v2beta2.MetricSpec{
		createMetricSpec(5),
		createMetricSpec(5),
		createMetricSpec(3),
	}
	targetAverageValue = getTargetAverageValue(specs)
	assert.Equal(t, int64(4), targetAverageValue)

	// 5 5 4
	specs = []v2beta2.MetricSpec{
		createMetricSpec(5),
		createMetricSpec(5),
		createMetricSpec(3),
	}
	targetAverageValue = getTargetAverageValue(specs)
	assert.Equal(t, int64(4), targetAverageValue)
}

func createMetricSpec(averageValue int) v2beta2.MetricSpec {
	qty := resource.NewQuantity(int64(averageValue), resource.DecimalSI)
	return v2beta2.MetricSpec{
		External: &v2beta2.ExternalMetricSource{
			Target: v2beta2.MetricTarget{
				AverageValue: qty,
			},
		},
	}
}
