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

package keda

import (
	"context"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	v2 "k8s.io/api/autoscaling/v2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/keda-scalers"
	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
	"github.com/kedacore/keda/v2/pkg/mock/mock_client"
	mock_scalers "github.com/kedacore/keda/v2/pkg/mock/mock_scaler"
	"github.com/kedacore/keda/v2/pkg/mock/mock_scaling"
	"github.com/kedacore/keda/v2/pkg/scaling/cache"
)

var _ = Describe("hpa", func() {
	var (
		reconciler   ScaledObjectReconciler
		scaleHandler *mock_scaling.MockScaleHandler
		client       *mock_client.MockClient
		statusWriter *mock_client.MockStatusWriter
		scaler       *mock_scalers.MockScaler
		logger       logr.Logger
		ctrl         *gomock.Controller
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		client = mock_client.NewMockClient(ctrl)
		scaleHandler = mock_scaling.NewMockScaleHandler(ctrl)
		scaler = mock_scalers.NewMockScaler(ctrl)
		statusWriter = mock_client.NewMockStatusWriter(ctrl)
		logger = logr.Discard()
		reconciler = ScaledObjectReconciler{
			Client:       client,
			ScaleHandler: scaleHandler,
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("should remove deleted metric from health status", func() {
		numberOfFailures := int32(87)
		health := make(map[string]v1alpha1.HealthStatus)
		health["another metric name"] = v1alpha1.HealthStatus{
			NumberOfFailures: &numberOfFailures,
			Status:           v1alpha1.HealthStatusFailing,
		}

		scaledObject := setupTest(health, scaler, scaleHandler)

		var capturedScaledObject v1alpha1.ScaledObject
		client.EXPECT().Status().Return(statusWriter)
		statusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(arg interface{}, scaledObject *v1alpha1.ScaledObject, anotherArg interface{}, opts ...interface{}) {
			capturedScaledObject = *scaledObject
		})

		_, err := reconciler.getScaledObjectMetricSpecs(context.Background(), logger, scaledObject)

		Expect(err).ToNot(HaveOccurred())
		Expect(capturedScaledObject.Status.Health).To(BeEmpty())
	})

	It("should not remove existing metric from health status", func() {
		numberOfFailures := int32(87)
		health := make(map[string]v1alpha1.HealthStatus)
		health["another metric name"] = v1alpha1.HealthStatus{
			NumberOfFailures: &numberOfFailures,
			Status:           v1alpha1.HealthStatusFailing,
		}

		health["some metric name"] = v1alpha1.HealthStatus{
			NumberOfFailures: &numberOfFailures,
			Status:           v1alpha1.HealthStatusFailing,
		}

		scaledObject := setupTest(health, scaler, scaleHandler)

		var capturedScaledObject v1alpha1.ScaledObject
		client.EXPECT().Status().Return(statusWriter)
		statusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(arg interface{}, scaledObject *v1alpha1.ScaledObject, anotherArg interface{}, opts ...interface{}) {
			capturedScaledObject = *scaledObject
		})

		_, err := reconciler.getScaledObjectMetricSpecs(context.Background(), logger, scaledObject)

		expectedHealth := make(map[string]v1alpha1.HealthStatus)
		expectedHealth["some metric name"] = v1alpha1.HealthStatus{
			NumberOfFailures: &numberOfFailures,
			Status:           v1alpha1.HealthStatusFailing,
		}

		Expect(err).ToNot(HaveOccurred())
		Expect(capturedScaledObject.Status.Health).To(HaveLen(1))
		Expect(capturedScaledObject.Status.Health).To(Equal(expectedHealth))
	})

})

func setupTest(health map[string]v1alpha1.HealthStatus, scaler *mock_scalers.MockScaler, scaleHandler *mock_scaling.MockScaleHandler) *v1alpha1.ScaledObject {
	scaledObject := &v1alpha1.ScaledObject{
		ObjectMeta: v1.ObjectMeta{
			Name: "some scaled object name",
		},
		Status: v1alpha1.ScaledObjectStatus{
			Health: health,
		},
	}

	scalersCache := cache.ScalersCache{
		Scalers: []cache.ScalerBuilder{{
			Scaler: scaler,
			Factory: func() (scalers.Scaler, *scalersconfig.ScalerConfig, error) {
				return scaler, &scalersconfig.ScalerConfig{}, nil
			},
		}},
		Recorder: nil,
	}
	metricSpec := v2.MetricSpec{
		External: &v2.ExternalMetricSource{
			Metric: v2.MetricIdentifier{
				Name: "some metric name",
			},
		},
	}
	metricSpecs := []v2.MetricSpec{metricSpec}
	ctx := context.Background()
	scaler.EXPECT().GetMetricSpecForScaling(ctx).Return(metricSpecs)
	scaleHandler.EXPECT().GetScalersCache(context.Background(), gomock.Eq(scaledObject)).Return(&scalersCache, nil)

	return scaledObject
}
