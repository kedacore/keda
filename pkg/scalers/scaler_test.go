package scalers

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

func TestGetMetricTargetType(t *testing.T) {
	cases := []struct {
		name           string
		config         *scalersconfig.ScalerConfig
		wantmetricType v2.MetricTargetType
		wantErr        error
	}{
		{
			name:           "utilization metric type",
			config:         &scalersconfig.ScalerConfig{MetricType: v2.UtilizationMetricType},
			wantmetricType: "",
			wantErr:        ErrScalerUnsupportedUtilizationMetricType,
		},
		{
			name:           "average value metric type",
			config:         &scalersconfig.ScalerConfig{MetricType: v2.AverageValueMetricType},
			wantmetricType: v2.AverageValueMetricType,
			wantErr:        nil,
		},
		{
			name:           "value metric type",
			config:         &scalersconfig.ScalerConfig{MetricType: v2.ValueMetricType},
			wantmetricType: v2.ValueMetricType,
			wantErr:        nil,
		},
		{
			name:           "no metric type",
			config:         &scalersconfig.ScalerConfig{},
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

func TestGetMetricTargetMili(t *testing.T) {
	cases := []struct {
		name        string
		metricValue float64
	}{
		{
			name:        "small value",
			metricValue: 100.5,
		},
		{
			name:        "large value exceeding int64 milli threshold",
			metricValue: 1e18,
		},
		{
			name:        "value just above int64 milli overflow threshold",
			metricValue: 9.3e15,
		},
		{
			name:        "NaN treated as zero",
			metricValue: math.NaN(),
		},
		{
			name:        "positive Inf treated as zero",
			metricValue: math.Inf(1),
		},
	}

	for _, testCase := range cases {
		c := testCase
		t.Run(c.name, func(t *testing.T) {
			target := GetMetricTargetMili(v2.AverageValueMetricType, c.metricValue)
			assert.NotNil(t, target.AverageValue)

			if math.IsNaN(c.metricValue) || math.IsInf(c.metricValue, 0) {
				assert.True(t, target.AverageValue.IsZero(), "expected zero quantity for NaN/Inf, got %v", target.AverageValue)
			} else {
				assert.True(t, target.AverageValue.AsApproximateFloat64() > 0, "expected positive quantity, got %v", target.AverageValue)
			}
		})
	}
}

func TestGenerateMetricInMili(t *testing.T) {
	cases := []struct {
		name  string
		value float64
	}{
		{
			name:  "small value",
			value: 42.5,
		},
		{
			name:  "large value in quintillion range",
			value: 1e18,
		},
		{
			name:  "value just above int64 milli overflow threshold",
			value: 9.3e15,
		},
		{
			name:  "NaN treated as zero",
			value: math.NaN(),
		},
		{
			name:  "positive Inf treated as zero",
			value: math.Inf(1),
		},
	}

	for _, testCase := range cases {
		c := testCase
		t.Run(c.name, func(t *testing.T) {
			metric := GenerateMetricInMili("test-metric", c.value)
			assert.Equal(t, "test-metric", metric.MetricName)

			if math.IsNaN(c.value) || math.IsInf(c.value, 0) {
				assert.True(t, metric.Value.IsZero(), "expected zero quantity for NaN/Inf, got %v", &metric.Value)
			} else {
				assert.True(t, metric.Value.AsApproximateFloat64() > 0, "expected positive quantity, got %v", &metric.Value)
			}
		})
	}
}

func TestRemoveIndexFromMetricName(t *testing.T) {
	cases := []struct {
		triggerIndex                         int
		metricName                           string
		expectedMetricNameWithoutIndexPrefix string
		isError                              bool
	}{
		// Proper input
		{triggerIndex: 0, metricName: "s0-metricName", expectedMetricNameWithoutIndexPrefix: "metricName", isError: false},
		// Proper input with triggerIndex > 9
		{triggerIndex: 123, metricName: "s123-metricName", expectedMetricNameWithoutIndexPrefix: "metricName", isError: false},
		// Incorrect index prefix
		{triggerIndex: 1, metricName: "s0-metricName", expectedMetricNameWithoutIndexPrefix: "", isError: true},
		// Incorrect index prefix
		{triggerIndex: 0, metricName: "0-metricName", expectedMetricNameWithoutIndexPrefix: "", isError: true},
		// No index prefix
		{triggerIndex: 0, metricName: "metricName", expectedMetricNameWithoutIndexPrefix: "", isError: true},
	}

	for _, testCase := range cases {
		metricName, err := RemoveIndexFromMetricName(testCase.triggerIndex, testCase.metricName)
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
