package scalers

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestGetMetricTargetType(t *testing.T) {
	cases := []struct {
		name           string
		config         *ScalerConfig
		wantmetricType v2beta2.MetricTargetType
		wantErr        error
	}{
		{
			name:           "utilization metric type",
			config:         &ScalerConfig{MetricType: v2beta2.UtilizationMetricType},
			wantmetricType: "",
			wantErr:        fmt.Errorf("'Utilization' metric type is unsupported for external metrics, allowed values are 'Value' or 'AverageValue'"),
		},
		{
			name:           "average value metric type",
			config:         &ScalerConfig{MetricType: v2beta2.AverageValueMetricType},
			wantmetricType: v2beta2.AverageValueMetricType,
			wantErr:        nil,
		},
		{
			name:           "value metric type",
			config:         &ScalerConfig{MetricType: v2beta2.ValueMetricType},
			wantmetricType: v2beta2.ValueMetricType,
			wantErr:        nil,
		},
		{
			name:           "no metric type",
			config:         &ScalerConfig{},
			wantmetricType: v2beta2.AverageValueMetricType,
			wantErr:        nil,
		},
	}

	for _, testCase := range cases {
		c := testCase
		t.Run(c.name, func(t *testing.T) {
			metricType, err := GetMetricTargetType(c.config)
			if c.wantErr != nil {
				assert.Contains(t, err.Error(), c.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, c.wantmetricType, metricType)
		})
	}
}

func TestGetMetricTarget(t *testing.T) {
	cases := []struct {
		name             string
		metricType       v2beta2.MetricTargetType
		metricValue      int64
		wantmetricTarget v2beta2.MetricTarget
	}{
		{
			name:             "average value metric type",
			metricType:       v2beta2.AverageValueMetricType,
			metricValue:      10,
			wantmetricTarget: v2beta2.MetricTarget{Type: v2beta2.AverageValueMetricType, AverageValue: resource.NewQuantity(10, resource.DecimalSI)},
		},
		{
			name:             "value metric type",
			metricType:       v2beta2.ValueMetricType,
			metricValue:      20,
			wantmetricTarget: v2beta2.MetricTarget{Type: v2beta2.ValueMetricType, Value: resource.NewQuantity(20, resource.DecimalSI)},
		},
	}

	for _, testCase := range cases {
		c := testCase
		t.Run(c.name, func(t *testing.T) {
			metricTarget := GetMetricTarget(c.metricType, c.metricValue)
			assert.Equal(t, c.wantmetricTarget, metricTarget)
		})
	}
}
