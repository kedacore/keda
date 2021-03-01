package controllers

import (
	"fmt"

	"github.com/golang/mock/gomock"
	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/mock/mock_client"
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
			mockClient               *mock_client.MockClient
			mockStatusWriter         *mock_client.MockStatusWriter
		)

		var triggerMeta []map[string]string = []map[string]string{
			{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "disableScaleToZero": "true"},
			{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total2", "threshold": "100", "query": "up"},
		}

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoTestReporter{})
			mockScaleHandler = mock_scaling.NewMockScaleHandler(ctrl)
			mockClient = mock_client.NewMockClient(ctrl)
			mockStatusWriter = mock_client.NewMockStatusWriter(ctrl)

			metricNameTestReconciler = ScaledObjectReconciler{
				scaleHandler: mockScaleHandler,
				Client:       mockClient,
			}
		})

		Context("With Unique Values", func() {
			var uniquelyNamedScaledObject = &kedav1alpha1.ScaledObject{}

			It("should pass metric name validation", func() {
				// Generate test data
				testScalers := make([]scalers.Scaler, 0)
				expectedExternalMetricNames := make([]string, 0)

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
					for _, metricSpec := range s.GetMetricSpecForScaling() {
						if metricSpec.External != nil {
							expectedExternalMetricNames = append(expectedExternalMetricNames, metricSpec.External.Metric.Name)
						}
					}
				}

				// Set up expectations
				mockScaleHandler.EXPECT().GetScalers(uniquelyNamedScaledObject).Return(testScalers, nil)
				mockClient.EXPECT().Status().Return(mockStatusWriter)
				mockStatusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any())

				// Call function to be tested
				metricSpecs, err := metricNameTestReconciler.getScaledObjectMetricSpecs(testLogger, uniquelyNamedScaledObject)

				// Test that the status was updated with metric names
				Ω(uniquelyNamedScaledObject.Status.ExternalMetricNames).Should(Equal(expectedExternalMetricNames))

				// Test returned values
				Ω(len(metricSpecs)).Should(Equal(len(testScalers)))
				Ω(err).Should(BeNil())
			})

			It("should pass metric name validation with single value", func() {
				// Generate test data
				expectedExternalMetricNames := make([]string, 0)

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
				for _, metricSpec := range s.GetMetricSpecForScaling() {
					if metricSpec.External != nil {
						expectedExternalMetricNames = append(expectedExternalMetricNames, metricSpec.External.Metric.Name)
					}
				}

				// Set up expectations
				mockScaleHandler.EXPECT().GetScalers(uniquelyNamedScaledObject).Return([]scalers.Scaler{s}, nil)
				mockClient.EXPECT().Status().Return(mockStatusWriter)
				mockStatusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any())

				// Call function to be tested
				metricSpecs, err := metricNameTestReconciler.getScaledObjectMetricSpecs(testLogger, uniquelyNamedScaledObject)

				// Test that the status was updated
				Ω(uniquelyNamedScaledObject.Status.ExternalMetricNames).Should(Equal(expectedExternalMetricNames))

				// Test returned values
				Ω(len(metricSpecs)).Should(Equal(1))
				Ω(err).Should(BeNil())
			})
		})

		Context("With Duplicate Values", func() {
			var duplicateNamedScaledObject = &kedav1alpha1.ScaledObject{}

			It("should pass metric name validation", func() {
				// Generate test data
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

				// Set up expectations
				mockScaleHandler.EXPECT().GetScalers(duplicateNamedScaledObject).Return(testScalers, nil)

				// Call function tobe tested
				metricSpecs, err := metricNameTestReconciler.getScaledObjectMetricSpecs(testLogger, duplicateNamedScaledObject)

				// Test that the status was not updated
				Ω(duplicateNamedScaledObject.Status.ExternalMetricNames).Should(BeNil())

				// Test returned values
				Ω(metricSpecs).Should(BeNil())
				Ω(err).ShouldNot(BeNil())
			})
		})
	})
})
