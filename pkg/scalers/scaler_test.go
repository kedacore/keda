package scalers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestGetMetricTargetType(t *testing.T) {
	cases := []struct {
		name           string
		config         *ScalerConfig
		wantmetricType v2.MetricTargetType
		wantErr        error
	}{
		{
			name:           "utilization metric type",
			config:         &ScalerConfig{MetricType: v2.UtilizationMetricType},
			wantmetricType: "",
			wantErr:        ErrScalerUnsupportedUtilizationMetricType,
		},
		{
			name:           "average value metric type",
			config:         &ScalerConfig{MetricType: v2.AverageValueMetricType},
			wantmetricType: v2.AverageValueMetricType,
			wantErr:        nil,
		},
		{
			name:           "value metric type",
			config:         &ScalerConfig{MetricType: v2.ValueMetricType},
			wantmetricType: v2.ValueMetricType,
			wantErr:        nil,
		},
		{
			name:           "no metric type",
			config:         &ScalerConfig{},
			wantmetricType: v2.AverageValueMetricType,
			wantErr:        nil,
		},
	}

	for _, testCase := range cases {
		c := testCase
		t.Run(c.name, func(t *testing.T) {
			metricType, err := GetMetricTargetType(c.config)
			if c.wantErr != nil {
				assert.ErrorIs(t, err, c.wantErr)
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
		metricType       v2.MetricTargetType
		metricValue      int64
		wantmetricTarget v2.MetricTarget
	}{
		{
			name:             "average value metric type",
			metricType:       v2.AverageValueMetricType,
			metricValue:      10,
			wantmetricTarget: v2.MetricTarget{Type: v2.AverageValueMetricType, AverageValue: resource.NewQuantity(10, resource.DecimalSI)},
		},
		{
			name:             "value metric type",
			metricType:       v2.ValueMetricType,
			metricValue:      20,
			wantmetricTarget: v2.MetricTarget{Type: v2.ValueMetricType, Value: resource.NewQuantity(20, resource.DecimalSI)},
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

func TestRemoveIndexFromMetricName(t *testing.T) {
	cases := []struct {
		scalerIndex                          int
		metricName                           string
		expectedMetricNameWithoutIndexPrefix string
		isError                              bool
	}{
		// Proper input
		{scalerIndex: 0, metricName: "s0-metricName", expectedMetricNameWithoutIndexPrefix: "metricName", isError: false},
		// Proper input with scalerIndex > 9
		{scalerIndex: 123, metricName: "s123-metricName", expectedMetricNameWithoutIndexPrefix: "metricName", isError: false},
		// Incorrect index prefix
		{scalerIndex: 1, metricName: "s0-metricName", expectedMetricNameWithoutIndexPrefix: "", isError: true},
		// Incorrect index prefix
		{scalerIndex: 0, metricName: "0-metricName", expectedMetricNameWithoutIndexPrefix: "", isError: true},
		// No index prefix
		{scalerIndex: 0, metricName: "metricName", expectedMetricNameWithoutIndexPrefix: "", isError: true},
	}

	for _, testCase := range cases {
		metricName, err := RemoveIndexFromMetricName(testCase.scalerIndex, testCase.metricName)
		if err != nil && !testCase.isError {
			t.Error("Expected success but got error", err)
		}

		if testCase.isError && err == nil {
			t.Error("Expected error but got success")
		}

		if err == nil {
			if metricName != testCase.expectedMetricNameWithoutIndexPrefix {
				t.Errorf("Expected - %s, Got - %s", testCase.expectedMetricNameWithoutIndexPrefix, metricName)
			}
		}
	}
}
