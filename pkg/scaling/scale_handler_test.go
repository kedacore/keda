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
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	mock_scalers "github.com/kedacore/keda/v2/pkg/mock/mock_scaler"
	"github.com/kedacore/keda/v2/pkg/scalers"
	"github.com/kedacore/keda/v2/pkg/scaling/cache"
)

func TestCheckScaledObjectScalersWithError(t *testing.T) {
	ctrl := gomock.NewController(t)
	recorder := record.NewFakeRecorder(1)

	factory := func() (scalers.Scaler, error) {
		scaler := mock_scalers.NewMockScaler(ctrl)
		scaler.EXPECT().IsActive(gomock.Any()).Return(false, errors.New("some error"))
		scaler.EXPECT().Close(gomock.Any())
		return scaler, nil
	}
	scaler, err := factory()
	assert.Nil(t, err)

	scaledObject := kedav1alpha1.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
	}

	cache := cache.ScalersCache{
		Scalers: []cache.ScalerBuilder{{
			Scaler:  scaler,
			Factory: factory,
		}},
		Logger:   logf.Log.WithName("scalehandler"),
		Recorder: recorder,
	}

	isActive, isError, _ := cache.IsScaledObjectActive(context.TODO(), &scaledObject)
	cache.Close(context.Background())

	assert.Equal(t, false, isActive)
	assert.Equal(t, true, isError)
}

func TestCheckScaledObjectFindFirstActiveIgnoringOthers(t *testing.T) {
	ctrl := gomock.NewController(t)
	recorder := record.NewFakeRecorder(1)
	activeScaler := mock_scalers.NewMockScaler(ctrl)
	failingScaler := mock_scalers.NewMockScaler(ctrl)
	scaledObject := &kedav1alpha1.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
	}

	metricsSpecs := []v2beta2.MetricSpec{createMetricSpec(1)}

	activeScaler.EXPECT().IsActive(gomock.Any()).Return(true, nil)
	activeScaler.EXPECT().GetMetricSpecForScaling(gomock.Any()).Times(2).Return(metricsSpecs)
	activeScaler.EXPECT().Close(gomock.Any())
	failingScaler.EXPECT().Close(gomock.Any())

	factory := func() (scalers.Scaler, error) {
		return mock_scalers.NewMockScaler(ctrl), nil
	}
	scalers := []cache.ScalerBuilder{{
		Scaler:  activeScaler,
		Factory: factory,
	}, {
		Scaler:  failingScaler,
		Factory: factory,
	}}

	scalersCache := cache.ScalersCache{
		Scalers:  scalers,
		Logger:   logf.Log.WithName("scalercache"),
		Recorder: recorder,
	}

	isActive, isError, _ := scalersCache.IsScaledObjectActive(context.TODO(), scaledObject)
	scalersCache.Close(context.Background())

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
