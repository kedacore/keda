package controllers

import (
	"context"
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/mock/mock_client"
	"github.com/kedacore/keda/v2/pkg/mock/mock_scaling"
	"github.com/kedacore/keda/v2/pkg/scalers"
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
		testLogger = zap.New(zap.UseDevMode(true), zap.WriteTo(GinkgoWriter))
	)

	Describe("Metric Names", func() {
		var (
			metricNameTestReconciler ScaledObjectReconciler
			mockScaleHandler         *mock_scaling.MockScaleHandler
			mockClient               *mock_client.MockClient
			mockStatusWriter         *mock_client.MockStatusWriter
		)

		var triggerMeta = []map[string]string{
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

	Describe("functional tests", func() {
		var deployment *appsv1.Deployment

		BeforeEach(func() {
			deployment = &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "myapp", Namespace: "default"},
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "myapp",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app": "myapp",
							},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "app",
									Image: "app",
								},
							},
						},
					},
				},
			}
		})

		It("cleans up a deleted trigger from the HPA", func() {
			// Create the scaling target.
			err := k8sClient.Create(context.Background(), deployment)
			Expect(err).ToNot(HaveOccurred())

			// Create the ScaledObject with two triggers.
			so := &kedav1alpha1.ScaledObject{
				ObjectMeta: metav1.ObjectMeta{Name: "clean-up-test", Namespace: "default"},
				Spec: kedav1alpha1.ScaledObjectSpec{
					ScaleTargetRef: &kedav1alpha1.ScaleTarget{
						Name: "myapp",
					},
					Triggers: []kedav1alpha1.ScaleTriggers{
						{
							Type: "cron",
							Metadata: map[string]string{
								"timezone":        "UTC",
								"start":           "0 * * * *",
								"end":             "1 * * * *",
								"desiredReplicas": "1",
							},
						},
						{
							Type: "cron",
							Metadata: map[string]string{
								"timezone":        "UTC",
								"start":           "2 * * * *",
								"end":             "3 * * * *",
								"desiredReplicas": "2",
							},
						},
					},
				},
			}
			err = k8sClient.Create(context.Background(), so)
			Expect(err).ToNot(HaveOccurred())

			// Get and confirm the HPA.
			hpa := &autoscalingv2beta2.HorizontalPodAutoscaler{}
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: "keda-hpa-clean-up-test", Namespace: "default"}, hpa)
			}).ShouldNot(HaveOccurred())
			Expect(hpa.Spec.Metrics).To(HaveLen(2))
			Expect(hpa.Spec.Metrics[0].External.Metric.Name).To(Equal("cron-UTC-0xxxx-1xxxx"))
			Expect(hpa.Spec.Metrics[1].External.Metric.Name).To(Equal("cron-UTC-2xxxx-3xxxx"))

			// Remove the second trigger.
			Eventually(func() error {
				err = k8sClient.Get(context.Background(), types.NamespacedName{Name: "clean-up-test", Namespace: "default"}, so)
				Expect(err).ToNot(HaveOccurred())
				so.Spec.Triggers = so.Spec.Triggers[:1]
				return k8sClient.Update(context.Background(), so)
			}).ShouldNot(HaveOccurred())

			// Wait until the HPA is updated.
			Eventually(func() int {
				err = k8sClient.Get(context.Background(), types.NamespacedName{Name: "keda-hpa-clean-up-test", Namespace: "default"}, hpa)
				Expect(err).ToNot(HaveOccurred())
				return len(hpa.Spec.Metrics)
			}).Should(Equal(1))
			// And it should only be the first one left.
			Expect(hpa.Spec.Metrics[0].External.Metric.Name).To(Equal("cron-UTC-0xxxx-1xxxx"))
		})
	})
})
