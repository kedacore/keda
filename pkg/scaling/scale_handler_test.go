package scaling

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/mock/mock_client"
	mock_scalers "github.com/kedacore/keda/v2/pkg/mock/mock_scaler"
	"github.com/kedacore/keda/v2/pkg/scalers"
	"github.com/kedacore/keda/v2/pkg/scaling/executor"
	"k8s.io/client-go/tools/record"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

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

func TestCheckScaledObjectScalersWithError(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := record.NewFakeRecorder(1)

	scaleHandler := &scaleHandler{
		client:            client,
		logger:            logf.Log.WithName("scalehandler"),
		scaleLoopContexts: &sync.Map{},
		scaleExecutor:     executor.NewScaleExecutor(client, nil, nil, recorder),
		globalHTTPTimeout: 5 * time.Second,
		recorder:          recorder,
	}
	scaler := mock_scalers.NewMockScaler(ctrl)
	scalers := []scalers.Scaler{scaler}
	scaledObject := &kedav1alpha1.ScaledObject{}

	scaler.EXPECT().IsActive(gomock.Any()).Return(false, errors.New("Some error"))
	scaler.EXPECT().Close()

	isActive, isError := scaleHandler.isScaledObjectActive(context.TODO(), scalers, scaledObject)

	assert.Equal(t, false, isActive)
	assert.Equal(t, true, isError)
}

func TestCheckScaledObjectFindFirstActiveIgnoringOthers(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := record.NewFakeRecorder(1)

	scaleHandler := &scaleHandler{
		client:            client,
		logger:            logf.Log.WithName("scalehandler"),
		scaleLoopContexts: &sync.Map{},
		scaleExecutor:     executor.NewScaleExecutor(client, nil, nil, recorder),
		globalHTTPTimeout: 5 * time.Second,
		recorder:          recorder,
	}

	activeScaler := mock_scalers.NewMockScaler(ctrl)
	failingScaler := mock_scalers.NewMockScaler(ctrl)
	scalers := []scalers.Scaler{activeScaler, failingScaler}
	scaledObject := &kedav1alpha1.ScaledObject{}

	metricsSpecs := []v2beta2.MetricSpec{createMetricSpec(1)}

	activeScaler.EXPECT().IsActive(gomock.Any()).Return(true, nil)
	activeScaler.EXPECT().GetMetricSpecForScaling().Times(2).Return(metricsSpecs)
	activeScaler.EXPECT().Close()
	failingScaler.EXPECT().Close()

	isActive, isError := scaleHandler.isScaledObjectActive(context.TODO(), scalers, scaledObject)

	assert.Equal(t, true, isActive)
	assert.Equal(t, false, isError)
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
