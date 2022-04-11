package cache

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/tools/record"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	mock_scalers "github.com/kedacore/keda/v2/pkg/mock/mock_scaler"
	"github.com/kedacore/keda/v2/pkg/scalers"
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

func createMetricSpec(averageValue int64) v2beta2.MetricSpec {
	qty := resource.NewQuantity(averageValue, resource.DecimalSI)
	return v2beta2.MetricSpec{
		External: &v2beta2.ExternalMetricSource{
			Target: v2beta2.MetricTarget{
				AverageValue: qty,
			},
		},
	}
}

func TestIsScaledJobActive(t *testing.T) {
	ctrl := gomock.NewController(t)
	recorder := record.NewFakeRecorder(1)

	// Keep the current behavior
	// Assme 1 trigger only
	scaledJobSingle := createScaledObject(100, "") // testing default = max
	scalerSingle := []ScalerBuilder{{
		Scaler: createScaler(ctrl, int64(20), int64(2), true),
		Factory: func() (scalers.Scaler, error) {
			return createScaler(ctrl, int64(20), int64(2), true), nil
		},
	}}

	cache := ScalersCache{
		Scalers:  scalerSingle,
		Logger:   logr.Discard(),
		Recorder: recorder,
	}

	isActive, queueLength, maxValue := cache.IsScaledJobActive(context.TODO(), scaledJobSingle)
	assert.Equal(t, true, isActive)
	assert.Equal(t, int64(20), queueLength)
	assert.Equal(t, int64(10), maxValue)
	cache.Close(context.Background())

	// Non-Active trigger only
	scalerSingle = []ScalerBuilder{{
		Scaler: createScaler(ctrl, int64(0), int64(2), false),
		Factory: func() (scalers.Scaler, error) {
			return createScaler(ctrl, int64(0), int64(2), false), nil
		},
	}}

	cache = ScalersCache{
		Scalers:  scalerSingle,
		Logger:   logr.Discard(),
		Recorder: recorder,
	}

	isActive, queueLength, maxValue = cache.IsScaledJobActive(context.TODO(), scaledJobSingle)
	assert.Equal(t, false, isActive)
	assert.Equal(t, int64(0), queueLength)
	assert.Equal(t, int64(0), maxValue)
	cache.Close(context.Background())

	// Test the valiation
	scalerTestDatam := []scalerTestData{
		newScalerTestData(100, "max", 20, 1, true, 10, 2, true, 5, 3, true, 7, 4, false, true, 20, 20),
		newScalerTestData(100, "min", 20, 1, true, 10, 2, true, 5, 3, true, 7, 4, false, true, 5, 2),
		newScalerTestData(100, "avg", 20, 1, true, 10, 2, true, 5, 3, true, 7, 4, false, true, 12, 9),
		newScalerTestData(100, "sum", 20, 1, true, 10, 2, true, 5, 3, true, 7, 4, false, true, 35, 27),
		newScalerTestData(25, "sum", 20, 1, true, 10, 2, true, 5, 3, true, 7, 4, false, true, 35, 25),
	}

	for index, scalerTestData := range scalerTestDatam {
		scaledJob := createScaledObject(scalerTestData.MaxReplicaCount, scalerTestData.MultipleScalersCalculation)
		scalersToTest := []ScalerBuilder{{
			Scaler: createScaler(ctrl, scalerTestData.Scaler1QueueLength, scalerTestData.Scaler1AverageValue, scalerTestData.Scaler1IsActive),
			Factory: func() (scalers.Scaler, error) {
				return createScaler(ctrl, scalerTestData.Scaler1QueueLength, scalerTestData.Scaler1AverageValue, scalerTestData.Scaler1IsActive), nil
			},
		}, {
			Scaler: createScaler(ctrl, scalerTestData.Scaler2QueueLength, scalerTestData.Scaler2AverageValue, scalerTestData.Scaler2IsActive),
			Factory: func() (scalers.Scaler, error) {
				return createScaler(ctrl, scalerTestData.Scaler2QueueLength, scalerTestData.Scaler2AverageValue, scalerTestData.Scaler2IsActive), nil
			},
		}, {
			Scaler: createScaler(ctrl, scalerTestData.Scaler3QueueLength, scalerTestData.Scaler3AverageValue, scalerTestData.Scaler3IsActive),
			Factory: func() (scalers.Scaler, error) {
				return createScaler(ctrl, scalerTestData.Scaler3QueueLength, scalerTestData.Scaler3AverageValue, scalerTestData.Scaler3IsActive), nil
			},
		}, {
			Scaler: createScaler(ctrl, scalerTestData.Scaler4QueueLength, scalerTestData.Scaler4AverageValue, scalerTestData.Scaler4IsActive),
			Factory: func() (scalers.Scaler, error) {
				return createScaler(ctrl, scalerTestData.Scaler4QueueLength, scalerTestData.Scaler4AverageValue, scalerTestData.Scaler4IsActive), nil
			},
		}}

		cache = ScalersCache{
			Scalers:  scalersToTest,
			Logger:   logr.Discard(),
			Recorder: recorder,
		}
		fmt.Printf("index: %d", index)
		isActive, queueLength, maxValue = cache.IsScaledJobActive(context.TODO(), scaledJob)
		//	assert.Equal(t, 5, index)
		assert.Equal(t, scalerTestData.ResultIsActive, isActive)
		assert.Equal(t, scalerTestData.ResultQueueLength, queueLength)
		assert.Equal(t, scalerTestData.ResultMaxValue, maxValue)
		cache.Close(context.Background())
	}
}

func newScalerTestData(
	maxReplicaCount int,
	multipleScalersCalculation string,
	scaler1QueueLength, //nolint:golint,unparam
	scaler1AverageValue int, //nolint:golint,unparam
	scaler1IsActive bool, //nolint:golint,unparam
	scaler2QueueLength, //nolint:golint,unparam
	scaler2AverageValue int, //nolint:golint,unparam
	scaler2IsActive bool, //nolint:golint,unparam
	scaler3QueueLength, //nolint:golint,unparam
	scaler3AverageValue int, //nolint:golint,unparam
	scaler3IsActive bool, //nolint:golint,unparam
	scaler4QueueLength, //nolint:golint,unparam
	scaler4AverageValue int, //nolint:golint,unparam
	scaler4IsActive bool, //nolint:golint,unparam
	resultIsActive bool, //nolint:golint,unparam
	resultQueueLength,
	resultMaxLength int) scalerTestData {
	return scalerTestData{
		MaxReplicaCount:            int32(maxReplicaCount),
		MultipleScalersCalculation: multipleScalersCalculation,
		Scaler1QueueLength:         int64(scaler1QueueLength),
		Scaler1AverageValue:        int64(scaler1AverageValue),
		Scaler1IsActive:            scaler1IsActive,
		Scaler2QueueLength:         int64(scaler2QueueLength),
		Scaler2AverageValue:        int64(scaler2AverageValue),
		Scaler2IsActive:            scaler2IsActive,
		Scaler3QueueLength:         int64(scaler3QueueLength),
		Scaler3AverageValue:        int64(scaler3AverageValue),
		Scaler3IsActive:            scaler3IsActive,
		Scaler4QueueLength:         int64(scaler4QueueLength),
		Scaler4AverageValue:        int64(scaler4AverageValue),
		Scaler4IsActive:            scaler4IsActive,
		ResultIsActive:             resultIsActive,
		ResultQueueLength:          int64(resultQueueLength),
		ResultMaxValue:             int64(resultMaxLength),
	}
}

type scalerTestData struct {
	MaxReplicaCount            int32
	MultipleScalersCalculation string
	Scaler1QueueLength         int64
	Scaler1AverageValue        int64
	Scaler1IsActive            bool
	Scaler2QueueLength         int64
	Scaler2AverageValue        int64
	Scaler2IsActive            bool
	Scaler3QueueLength         int64
	Scaler3AverageValue        int64
	Scaler3IsActive            bool
	Scaler4QueueLength         int64
	Scaler4AverageValue        int64
	Scaler4IsActive            bool
	ResultIsActive             bool
	ResultQueueLength          int64
	ResultMaxValue             int64
}

func createScaledObject(maxReplicaCount int32, multipleScalersCalculation string) *kedav1alpha1.ScaledJob {
	if multipleScalersCalculation != "" {
		return &kedav1alpha1.ScaledJob{
			Spec: kedav1alpha1.ScaledJobSpec{
				MaxReplicaCount: &maxReplicaCount,
				ScalingStrategy: kedav1alpha1.ScalingStrategy{
					MultipleScalersCalculation: multipleScalersCalculation,
				},
			},
		}
	}
	return &kedav1alpha1.ScaledJob{
		Spec: kedav1alpha1.ScaledJobSpec{
			MaxReplicaCount: &maxReplicaCount,
		},
	}
}

func createScaler(ctrl *gomock.Controller, queueLength int64, averageValue int64, isActive bool) *mock_scalers.MockScaler {
	metricName := "queueLength"
	scaler := mock_scalers.NewMockScaler(ctrl)
	metricsSpecs := []v2beta2.MetricSpec{createMetricSpec(averageValue)}
	metrics := []external_metrics.ExternalMetricValue{
		{
			MetricName: metricName,
			Value:      *resource.NewQuantity(queueLength, resource.DecimalSI),
		},
	}
	scaler.EXPECT().IsActive(gomock.Any()).Return(isActive, nil)
	scaler.EXPECT().GetMetricSpecForScaling(gomock.Any()).Return(metricsSpecs)
	scaler.EXPECT().GetMetrics(gomock.Any(), metricName, nil).Return(metrics, nil)
	scaler.EXPECT().Close(gomock.Any())
	return scaler
}
