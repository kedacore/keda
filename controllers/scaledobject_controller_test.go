package controllers

import (
	"fmt"

	"github.com/golang/mock/gomock"
	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/mock/mock_scaling"
	"github.com/kedacore/keda/v2/pkg/scalers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type GinkgoTestReporter struct{}

func (g GinkgoTestReporter) Errorf(format string, args ...interface{}) {
	Fail(fmt.Sprintf(format, args...))
}

func (g GinkgoTestReporter) Fatalf(format string, args ...interface{}) {
	Fail(fmt.Sprintf(format, args...))
}

var _ = Describe("ScaledObjectController", func() {
	var (
		testLogger = zap.LoggerTo(GinkgoWriter, true)
	)

	Describe("Metric Names", func() {
		var (
			metricNameTestReconciler ScaledObjectReconciler
			mockScaleHandler         *mock_scaling.MockScaleHandler
		)

		var triggerMeta []map[string]string = []map[string]string{
			{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "disableScaleToZero": "true"},
			{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total2", "threshold": "100", "query": "up"},
		}

		BeforeEach(func() {
			mockScaleHandler = mock_scaling.NewMockScaleHandler(gomock.NewController(GinkgoTestReporter{}))

			metricNameTestReconciler = ScaledObjectReconciler{
				scaleHandler: mockScaleHandler,
			}
		})

		Context("With Unique Values", func() {
			var uniqueNamedScaledObjectTrigger = &kedav1alpha1.ScaledObject{}

			It("should pass metric name validation", func() {
				testScalers := make([]scalers.Scaler, 0)
				for i, tm := range triggerMeta {
					config := &scalers.ScalerConfig{
						Name:            fmt.Sprintf("test.%d", i),
						Namespace:       "test",
						TriggerMetadata: tm,
						ResolvedEnv:     nil,
						AuthParams:      nil,
					}

					s, err := scalers.NewPrometheusScaler(config)
					if err != nil {
						Fail(err.Error())
					}

					testScalers = append(testScalers, s)
				}

				mockScaleHandler.EXPECT().GetScalers(uniqueNamedScaledObjectTrigger).Return(testScalers, nil)

				Ω(metricNameTestReconciler.validateMetricNameUniqueness(testLogger, uniqueNamedScaledObjectTrigger)).Should(BeNil())
			})

			It("should pass metric name validation with single value", func() {
				config := &scalers.ScalerConfig{
					Name:            "test",
					Namespace:       "test",
					TriggerMetadata: triggerMeta[0],
					ResolvedEnv:     nil,
					AuthParams:      nil,
				}

				s, err := scalers.NewPrometheusScaler(config)
				if err != nil {
					Fail(err.Error())
				}

				mockScaleHandler.EXPECT().GetScalers(uniqueNamedScaledObjectTrigger).Return([]scalers.Scaler{s}, nil)

				Ω(metricNameTestReconciler.validateMetricNameUniqueness(testLogger, uniqueNamedScaledObjectTrigger)).Should(BeNil())
			})
		})

		Context("With Duplicate Values", func() {
			var duplicateNamedScaledObjectTrigger = &kedav1alpha1.ScaledObject{}

			It("should pass metric name validation", func() {
				testScalers := make([]scalers.Scaler, 0)
				for i := 0; i < 4; i++ {
					config := &scalers.ScalerConfig{
						Name:            fmt.Sprintf("test.%d", i),
						Namespace:       "test",
						TriggerMetadata: triggerMeta[0],
						ResolvedEnv:     nil,
						AuthParams:      nil,
					}

					s, err := scalers.NewPrometheusScaler(config)
					if err != nil {
						Fail(err.Error())
					}

					testScalers = append(testScalers, s)
				}

				mockScaleHandler.EXPECT().GetScalers(duplicateNamedScaledObjectTrigger).Return(testScalers, nil)

				Ω(metricNameTestReconciler.validateMetricNameUniqueness(testLogger, duplicateNamedScaledObjectTrigger)).ShouldNot(BeNil())
			})
		})
	})
})
