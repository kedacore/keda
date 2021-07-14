package controllers

import (
	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	"github.com/kedacore/keda/v2/api/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/mock/mock_client"
	mock_scalers "github.com/kedacore/keda/v2/pkg/mock/mock_scaler"
	"github.com/kedacore/keda/v2/pkg/mock/mock_scaling"
	kedascalers "github.com/kedacore/keda/v2/pkg/scalers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/autoscaling/v2beta2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		logger = logr.DiscardLogger{}
		reconciler = ScaledObjectReconciler{
			Client:       client,
			scaleHandler: scaleHandler,
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

		_, err := reconciler.getScaledObjectMetricSpecs(logger, scaledObject)

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

		_, err := reconciler.getScaledObjectMetricSpecs(logger, scaledObject)

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

	scalers := []kedascalers.Scaler{scaler}
	metricSpec := v2beta2.MetricSpec{
		External: &v2beta2.ExternalMetricSource{
			Metric: v2beta2.MetricIdentifier{
				Name: "some metric name",
			},
		},
	}
	metricSpecs := []v2beta2.MetricSpec{metricSpec}
	scaler.EXPECT().GetMetricSpecForScaling().Return(metricSpecs)
	scaler.EXPECT().Close()
	scaleHandler.EXPECT().GetScalers(gomock.Eq(scaledObject)).Return(scalers, nil)

	return scaledObject
}
