/*
Copyright 2021 The KEDA Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package scaling

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/tools/record"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/mock/mock_client"
	mock_scalers "github.com/kedacore/keda/v2/pkg/mock/mock_scaler"
	"github.com/kedacore/keda/v2/pkg/scalers"
	"github.com/kedacore/keda/v2/pkg/scaling/executor"
)

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
	scaler.EXPECT().Close(gomock.Any())

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
	activeScaler.EXPECT().GetMetricSpecForScaling(gomock.Any()).Times(2).Return(metricsSpecs)
	activeScaler.EXPECT().Close(gomock.Any())
	failingScaler.EXPECT().Close(gomock.Any())

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
