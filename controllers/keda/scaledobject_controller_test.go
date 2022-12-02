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
	"fmt"
	"sync/atomic"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/mock/mock_client"
	"github.com/kedacore/keda/v2/pkg/mock/mock_scaling"
	"github.com/kedacore/keda/v2/pkg/scalers"
	"github.com/kedacore/keda/v2/pkg/scaling/cache"
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
				ScaleHandler: mockScaleHandler,
				Client:       mockClient,
			}
		})

		Context("With Unique Values", func() {
			var uniquelyNamedScaledObject = &kedav1alpha1.ScaledObject{}

			It("should pass metric name validation", func() {
				// Generate test data
				testScalers := make([]cache.ScalerBuilder, 0)
				expectedExternalMetricNames := make([]string, 0)

				for i, tm := range triggerMeta {
					config := &scalers.ScalerConfig{
						ScalableObjectName:      fmt.Sprintf("test.%d", i),
						ScalableObjectNamespace: "test",
						TriggerMetadata:         tm,
						ResolvedEnv:             nil,
						AuthParams:              nil,
					}

					s, err := scalers.NewPrometheusScaler(config)
					if err != nil {
						Fail(err.Error())
					}

					testScalers = append(testScalers, cache.ScalerBuilder{
						Scaler: s,
						Factory: func() (scalers.Scaler, *scalers.ScalerConfig, error) {
							scaler, err := scalers.NewPrometheusScaler(config)
							return scaler, config, err
						},
					})
					for _, metricSpec := range s.GetMetricSpecForScaling(context.Background()) {
						if metricSpec.External != nil {
							expectedExternalMetricNames = append(expectedExternalMetricNames, metricSpec.External.Metric.Name)
						}
					}
				}

				// Set up expectations
				scalerCache := cache.ScalersCache{
					Scalers: testScalers,
				}
				mockScaleHandler.EXPECT().GetScalersCache(context.Background(), uniquelyNamedScaledObject).Return(&scalerCache, nil)
				mockClient.EXPECT().Status().Return(mockStatusWriter)
				mockStatusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any())

				// Call function to be tested
				metricSpecs, err := metricNameTestReconciler.getScaledObjectMetricSpecs(context.Background(), testLogger, uniquelyNamedScaledObject)

				// Test that the status was updated with metric names
				Ω(uniquelyNamedScaledObject.Status.ExternalMetricNames).Should(Equal(expectedExternalMetricNames))

				// Test returned values
				Ω(len(metricSpecs)).Should(Equal(len(testScalers)))
				Ω(err).Should(BeNil())
				scalerCache.Close(ctx)
			})

			It("should pass metric name validation with single value", func() {
				// Generate test data
				expectedExternalMetricNames := make([]string, 0)

				config := &scalers.ScalerConfig{
					ScalableObjectName:      "test",
					ScalableObjectNamespace: "test",
					TriggerMetadata:         triggerMeta[0],
					ResolvedEnv:             nil,
					AuthParams:              nil,
				}

				s, err := scalers.NewPrometheusScaler(config)
				if err != nil {
					Fail(err.Error())
				}
				for _, metricSpec := range s.GetMetricSpecForScaling(context.Background()) {
					if metricSpec.External != nil {
						expectedExternalMetricNames = append(expectedExternalMetricNames, metricSpec.External.Metric.Name)
					}
				}

				scalersCache := cache.ScalersCache{
					Scalers: []cache.ScalerBuilder{{
						Scaler: s,
						Factory: func() (scalers.Scaler, *scalers.ScalerConfig, error) {
							return s, config, nil
						},
					}},
				}
				// Set up expectations
				mockScaleHandler.EXPECT().GetScalersCache(context.Background(), uniquelyNamedScaledObject).Return(&scalersCache, nil)
				mockClient.EXPECT().Status().Return(mockStatusWriter)
				mockStatusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any())

				// Call function to be tested
				metricSpecs, err := metricNameTestReconciler.getScaledObjectMetricSpecs(context.Background(), testLogger, uniquelyNamedScaledObject)

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
				testScalers := make([]cache.ScalerBuilder, 0)
				for i := 0; i < 4; i++ {
					config := &scalers.ScalerConfig{
						ScalableObjectName:      fmt.Sprintf("test.%d", i),
						ScalableObjectNamespace: "test",
						TriggerMetadata:         triggerMeta[0],
						ResolvedEnv:             nil,
						AuthParams:              nil,
					}

					s, err := scalers.NewPrometheusScaler(config)
					if err != nil {
						Fail(err.Error())
					}

					testScalers = append(testScalers, cache.ScalerBuilder{
						Scaler: s,
						Factory: func() (scalers.Scaler, *scalers.ScalerConfig, error) {
							return s, config, nil
						},
					})
				}
				scalersCache := cache.ScalersCache{
					Scalers: testScalers,
				}

				// Set up expectations
				mockScaleHandler.EXPECT().GetScalersCache(context.Background(), duplicateNamedScaledObject).Return(&scalersCache, nil)

				// Call function tobe tested
				metricSpecs, err := metricNameTestReconciler.getScaledObjectMetricSpecs(context.Background(), testLogger, duplicateNamedScaledObject)

				// Test that the status was not updated
				Ω(duplicateNamedScaledObject.Status.ExternalMetricNames).Should(BeNil())

				// Test returned values
				Ω(metricSpecs).Should(BeNil())
				Ω(err).ShouldNot(BeNil())
			})
		})
	})

	Describe("functional tests", func() {
		It("cleans up a deleted trigger from the HPA", func() {
			// Create the scaling target.
			err := k8sClient.Create(context.Background(), generateDeployment("clean-up"))
			Expect(err).ToNot(HaveOccurred())

			// Create the ScaledObject with two triggers.
			so := &kedav1alpha1.ScaledObject{
				ObjectMeta: metav1.ObjectMeta{Name: "clean-up-test", Namespace: "default"},
				Spec: kedav1alpha1.ScaledObjectSpec{
					ScaleTargetRef: &kedav1alpha1.ScaleTarget{
						Name: "clean-up",
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
			hpa := &autoscalingv2.HorizontalPodAutoscaler{}
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: "keda-hpa-clean-up-test", Namespace: "default"}, hpa)
			}).ShouldNot(HaveOccurred())
			Expect(hpa.Spec.Metrics).To(HaveLen(2))
			Expect(hpa.Spec.Metrics[0].External.Metric.Name).To(Equal("s0-cron-UTC-0xxxx-1xxxx"))
			Expect(hpa.Spec.Metrics[1].External.Metric.Name).To(Equal("s1-cron-UTC-2xxxx-3xxxx"))

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
			Expect(hpa.Spec.Metrics[0].External.Metric.Name).To(Equal("s0-cron-UTC-0xxxx-1xxxx"))
		})

		It("cleans up old hpa when hpa name is updated", func() {
			// Create the scaling target.
			deploymentName := "changing-name"
			soName := "so-" + deploymentName
			err := k8sClient.Create(context.Background(), generateDeployment(deploymentName))
			Expect(err).ToNot(HaveOccurred())

			// Create the ScaledObject without specifying name.
			so := &kedav1alpha1.ScaledObject{
				ObjectMeta: metav1.ObjectMeta{Name: soName, Namespace: "default"},
				Spec: kedav1alpha1.ScaledObjectSpec{
					ScaleTargetRef: &kedav1alpha1.ScaleTarget{
						Name: deploymentName,
					},
					Advanced: &kedav1alpha1.AdvancedConfig{
						HorizontalPodAutoscalerConfig: &kedav1alpha1.HorizontalPodAutoscalerConfig{},
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
					},
				},
			}
			err = k8sClient.Create(context.Background(), so)
			Expect(err).ToNot(HaveOccurred())

			// Get and confirm the HPA.
			hpa := &autoscalingv2.HorizontalPodAutoscaler{}
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: fmt.Sprintf("keda-hpa-%s", soName), Namespace: "default"}, hpa)
			}).ShouldNot(HaveOccurred())
			Expect(hpa.Name).To(Equal(fmt.Sprintf("keda-hpa-%s", soName)))

			// Update hpa name
			Eventually(func() error {
				err = k8sClient.Get(context.Background(), types.NamespacedName{Name: soName, Namespace: "default"}, so)
				Expect(err).ToNot(HaveOccurred())
				so.Spec.Advanced.HorizontalPodAutoscalerConfig.Name = fmt.Sprintf("new-%s", soName)
				return k8sClient.Update(context.Background(), so)
			}).ShouldNot(HaveOccurred())

			// Wait until the HPA is updated.
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: fmt.Sprintf("new-%s", soName), Namespace: "default"}, hpa)
			}).ShouldNot(HaveOccurred())

			// And validate that old hpa is deleted.
			err = k8sClient.Get(context.Background(), types.NamespacedName{Name: fmt.Sprintf("keda-hpa-%s", soName), Namespace: "default"}, hpa)
			Expect(err).Should(HaveOccurred())
			Expect(errors.IsNotFound(err)).To(Equal(true))
		})

		//https://github.com/kedacore/keda/issues/2407
		It("cache is correctly recreated if SO is deleted and created", func() {
			// Create the scaling target.
			err := k8sClient.Create(context.Background(), generateDeployment("cache-regenerate"))
			Expect(err).ToNot(HaveOccurred())

			// Create the ScaledObject with one trigger.
			so := &kedav1alpha1.ScaledObject{
				ObjectMeta: metav1.ObjectMeta{Name: "cache-regenerate", Namespace: "default"},
				Spec: kedav1alpha1.ScaledObjectSpec{
					ScaleTargetRef: &kedav1alpha1.ScaleTarget{
						Name: "cache-regenerate",
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
					},
				},
			}
			err = k8sClient.Create(context.Background(), so)
			Expect(err).ToNot(HaveOccurred())

			// Get and confirm the HPA.
			hpa := &autoscalingv2.HorizontalPodAutoscaler{}
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: "keda-hpa-cache-regenerate", Namespace: "default"}, hpa)
			}).ShouldNot(HaveOccurred())
			Expect(hpa.Spec.Metrics).To(HaveLen(1))
			Expect(hpa.Spec.Metrics[0].External.Metric.Name).To(Equal("s0-cron-UTC-0xxxx-1xxxx"))

			// Delete the ScaledObject
			err = k8sClient.Delete(context.Background(), so)
			Expect(err).ToNot(HaveOccurred())
			time.Sleep(30 * time.Second)

			// Create the same ScaledObject with a change in the trigger.
			so = &kedav1alpha1.ScaledObject{
				ObjectMeta: metav1.ObjectMeta{Name: "cache-regenerate", Namespace: "default"},
				Spec: kedav1alpha1.ScaledObjectSpec{
					ScaleTargetRef: &kedav1alpha1.ScaleTarget{
						Name: "cache-regenerate",
					},
					Triggers: []kedav1alpha1.ScaleTriggers{
						{
							Type: "cron",
							Metadata: map[string]string{
								"timezone":        "CET",
								"start":           "0 * * * *",
								"end":             "1 * * * *",
								"desiredReplicas": "1",
							},
						},
					},
				},
			}
			err = k8sClient.Create(context.Background(), so)
			Expect(err).ToNot(HaveOccurred())
			time.Sleep(30 * time.Second)

			// Get and confirm the HPA.
			hpa2 := &autoscalingv2.HorizontalPodAutoscaler{}
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: "keda-hpa-cache-regenerate", Namespace: "default"}, hpa2)
			}).ShouldNot(HaveOccurred())
			Expect(hpa2.Spec.Metrics).To(HaveLen(1))
			Expect(hpa2.Spec.Metrics[0].External.Metric.Name).To(Equal("s0-cron-CET-0xxxx-1xxxx"))
		})

		It("deploys ScaledObject and creates HPA, when IdleReplicaCount, MinReplicaCount and MaxReplicaCount is defined", func() {

			deploymentName := "idleminmax"
			soName := "so-" + deploymentName

			// Create the scaling target.
			err := k8sClient.Create(context.Background(), generateDeployment(deploymentName))
			Expect(err).ToNot(HaveOccurred())

			var one int32 = 1
			var five int32 = 5
			var ten int32 = 10

			// Create the ScaledObject
			so := &kedav1alpha1.ScaledObject{
				ObjectMeta: metav1.ObjectMeta{Name: soName, Namespace: "default"},
				Spec: kedav1alpha1.ScaledObjectSpec{
					ScaleTargetRef: &kedav1alpha1.ScaleTarget{
						Name: deploymentName,
					},
					IdleReplicaCount: &one,
					MinReplicaCount:  &five,
					MaxReplicaCount:  &ten,
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
					},
				},
			}
			err = k8sClient.Create(context.Background(), so)
			Ω(err).ToNot(HaveOccurred())

			// Get and confirm the HPA
			hpa := &autoscalingv2.HorizontalPodAutoscaler{}
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: "keda-hpa-" + soName, Namespace: "default"}, hpa)
			}).ShouldNot(HaveOccurred())

			Ω(*hpa.Spec.MinReplicas).To(Equal(five))
			Ω(hpa.Spec.MaxReplicas).To(Equal(ten))

			Eventually(func() metav1.ConditionStatus {
				err = k8sClient.Get(context.Background(), types.NamespacedName{Name: soName, Namespace: "default"}, so)
				Ω(err).ToNot(HaveOccurred())
				return so.Status.Conditions.GetReadyCondition().Status
			}, 20*time.Second).Should(Equal(metav1.ConditionTrue))
		})

		It("deploys ScaledObject and creates HPA, when metadata.Annotations is configured", func() {

			deploymentName := "annotations"
			soName := "so-" + deploymentName

			// Create the scaling target.
			err := k8sClient.Create(context.Background(), generateDeployment(deploymentName))
			Expect(err).ToNot(HaveOccurred())

			// Create the ScaledObject
			so := &kedav1alpha1.ScaledObject{
				ObjectMeta: metav1.ObjectMeta{
					Name:      soName,
					Namespace: "default",
					Annotations: map[string]string{
						"annotation-email": "email@example.com",
						"annotation-url":   "https://example.com",
					}},
				Spec: kedav1alpha1.ScaledObjectSpec{
					ScaleTargetRef: &kedav1alpha1.ScaleTarget{
						Name: deploymentName,
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
					},
				},
			}
			err = k8sClient.Create(context.Background(), so)
			Ω(err).ToNot(HaveOccurred())

			// Get and confirm the HPA
			hpa := &autoscalingv2.HorizontalPodAutoscaler{}
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: "keda-hpa-" + soName, Namespace: "default"}, hpa)
			}).ShouldNot(HaveOccurred())

			Ω(hpa.Annotations).To(Equal(so.Annotations))
		})

		It("doesn't allow MinReplicaCount > MaxReplicaCount", func() {
			deploymentName := "minmax"
			soName := "so-" + deploymentName

			// Create the scaling target.
			err := k8sClient.Create(context.Background(), generateDeployment(deploymentName))
			Expect(err).ToNot(HaveOccurred())

			var five int32 = 5
			var ten int32 = 10

			// Create the ScaledObject
			so := &kedav1alpha1.ScaledObject{
				ObjectMeta: metav1.ObjectMeta{Name: soName, Namespace: "default"},
				Spec: kedav1alpha1.ScaledObjectSpec{
					ScaleTargetRef: &kedav1alpha1.ScaleTarget{
						Name: deploymentName,
					},
					MinReplicaCount: &ten,
					MaxReplicaCount: &five,
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
					},
				},
			}
			err = k8sClient.Create(context.Background(), so)
			Ω(err).ToNot(HaveOccurred())

			Eventually(func() metav1.ConditionStatus {
				err = k8sClient.Get(context.Background(), types.NamespacedName{Name: soName, Namespace: "default"}, so)
				Ω(err).ToNot(HaveOccurred())
				return so.Status.Conditions.GetReadyCondition().Status
			}, 20*time.Second).Should(Equal(metav1.ConditionFalse))
		})

		It("doesn't allow IdleReplicaCount > MinReplicaCount", func() {
			deploymentName := "idlemin"
			soName := "so-" + deploymentName

			// Create the scaling target.
			err := k8sClient.Create(context.Background(), generateDeployment(deploymentName))
			Expect(err).ToNot(HaveOccurred())

			var five int32 = 5
			var ten int32 = 10

			// Create the ScaledObject with two triggers
			so := &kedav1alpha1.ScaledObject{
				ObjectMeta: metav1.ObjectMeta{Name: soName, Namespace: "default"},
				Spec: kedav1alpha1.ScaledObjectSpec{
					ScaleTargetRef: &kedav1alpha1.ScaleTarget{
						Name: deploymentName,
					},
					IdleReplicaCount: &ten,
					MinReplicaCount:  &five,
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
					},
				},
			}
			err = k8sClient.Create(context.Background(), so)
			Ω(err).ToNot(HaveOccurred())

			Eventually(func() metav1.ConditionStatus {
				err = k8sClient.Get(context.Background(), types.NamespacedName{Name: soName, Namespace: "default"}, so)
				Ω(err).ToNot(HaveOccurred())
				return so.Status.Conditions.GetReadyCondition().Status
			}, 20*time.Second).Should(Equal(metav1.ConditionFalse))
		})

		It("doesn't allow IdleReplicaCount > MaxReplicaCount, when MinReplicaCount is not explicitly defined", func() {
			deploymentName := "idlemax"
			soName := "so-" + deploymentName

			// Create the scaling target.
			err := k8sClient.Create(context.Background(), generateDeployment(deploymentName))
			Expect(err).ToNot(HaveOccurred())

			var five int32 = 5
			var ten int32 = 10

			// Create the ScaledObject with two triggers
			so := &kedav1alpha1.ScaledObject{
				ObjectMeta: metav1.ObjectMeta{Name: soName, Namespace: "default"},
				Spec: kedav1alpha1.ScaledObjectSpec{
					ScaleTargetRef: &kedav1alpha1.ScaleTarget{
						Name: deploymentName,
					},
					IdleReplicaCount: &ten,
					MaxReplicaCount:  &five,
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
					},
				},
			}
			err = k8sClient.Create(context.Background(), so)
			Ω(err).ToNot(HaveOccurred())

			Eventually(func() metav1.ConditionStatus {
				err = k8sClient.Get(context.Background(), types.NamespacedName{Name: soName, Namespace: "default"}, so)
				Ω(err).ToNot(HaveOccurred())
				return so.Status.Conditions.GetReadyCondition().Status
			}, 20*time.Second).Should(Equal(metav1.ConditionFalse))
		})

		It("doesn't allow non-unique triggerName in ScaledObject", func() {
			deploymentName := "non-unique-triggername"
			soName := "so-" + deploymentName

			triggerName := "non-unique"

			// Create the scaling target.
			err := k8sClient.Create(context.Background(), generateDeployment(deploymentName))
			Expect(err).ToNot(HaveOccurred())

			var five int32 = 5
			var ten int32 = 10

			// Create the ScaledObject with two triggers
			so := &kedav1alpha1.ScaledObject{
				ObjectMeta: metav1.ObjectMeta{Name: soName, Namespace: "default"},
				Spec: kedav1alpha1.ScaledObjectSpec{
					ScaleTargetRef: &kedav1alpha1.ScaleTarget{
						Name: deploymentName,
					},
					IdleReplicaCount: &ten,
					MinReplicaCount:  &five,
					Triggers: []kedav1alpha1.ScaleTriggers{
						{
							Type: "cron",
							Name: triggerName,
							Metadata: map[string]string{
								"timezone":        "UTC",
								"start":           "0 * * * *",
								"end":             "1 * * * *",
								"desiredReplicas": "1",
							},
						},
						{
							Type: "cron",
							Name: triggerName,
							Metadata: map[string]string{
								"timezone":        "UTC",
								"start":           "10 * * * *",
								"end":             "11 * * * *",
								"desiredReplicas": "1",
							},
						},
					},
				},
			}
			err = k8sClient.Create(context.Background(), so)
			Ω(err).ToNot(HaveOccurred())

			Eventually(func() metav1.ConditionStatus {
				err = k8sClient.Get(context.Background(), types.NamespacedName{Name: soName, Namespace: "default"}, so)
				Ω(err).ToNot(HaveOccurred())
				return so.Status.Conditions.GetReadyCondition().Status
			}, 20*time.Second).Should(Equal(metav1.ConditionFalse))
		})
	})

	It("scaleobject ready condition 'False/Unknow' to 'True' will requeue", func() {
		var (
			deploymentName        = "conditionchange"
			soName                = "so-" + deploymentName
			min             int32 = 1
			max             int32 = 5
			pollingInterVal int32 = 1
		)

		// Create the scaling target.
		err := k8sClient.Create(context.Background(), generateDeployment(deploymentName))
		Expect(err).ToNot(HaveOccurred())

		so := &kedav1alpha1.ScaledObject{
			ObjectMeta: metav1.ObjectMeta{Name: soName, Namespace: "default"},
			Spec: kedav1alpha1.ScaledObjectSpec{
				ScaleTargetRef: &kedav1alpha1.ScaleTarget{
					Name: deploymentName,
				},
				MinReplicaCount: &min,
				MaxReplicaCount: &max,
				PollingInterval: &pollingInterVal,
				Triggers: []kedav1alpha1.ScaleTriggers{
					{
						Type:       "cpu",
						MetricType: autoscalingv2.UtilizationMetricType,
						Metadata: map[string]string{
							"value": "50",
						},
					},
					{
						Type:       "external-mock",
						MetricType: autoscalingv2.AverageValueMetricType,
						Metadata:   map[string]string{},
					},
				},
			},
		}
		err = k8sClient.Create(context.Background(), so)
		Expect(err).ToNot(HaveOccurred())

		// wait so's ready condition Ready
		Eventually(func() metav1.ConditionStatus {
			err = k8sClient.Get(context.Background(), types.NamespacedName{Name: soName, Namespace: "default"}, so)
			Expect(err).ToNot(HaveOccurred())
			return so.Status.Conditions.GetReadyCondition().Status
		}, 5*time.Second).Should(Equal(metav1.ConditionTrue))

		// check hpa
		hpa := &autoscalingv2.HorizontalPodAutoscaler{}
		Eventually(func() int {
			err = k8sClient.Get(context.Background(), types.NamespacedName{Name: getHPAName(so), Namespace: "default"}, hpa)
			Expect(err).ToNot(HaveOccurred())
			return len(hpa.Spec.Metrics)
		}, 1*time.Second).Should(Equal(2))

		// mock external server offline
		atomic.StoreInt32(&scalers.MockExternalServerStatus, scalers.MockExternalServerStatusOffline)

		// wait so's ready condition not
		Eventually(func() metav1.ConditionStatus {
			err = k8sClient.Get(context.Background(), types.NamespacedName{Name: soName, Namespace: "default"}, so)
			Expect(err).ToNot(HaveOccurred())
			return so.Status.Conditions.GetReadyCondition().Status
		}, 5*time.Second).Should(Or(Equal(metav1.ConditionFalse), Equal(metav1.ConditionUnknown)))

		// mock kube-controller-manager request v1beta1.custom.metrics.k8s.io api GetMetrics
		err = k8sClient.Get(context.Background(), types.NamespacedName{Name: getHPAName(so), Namespace: "default"}, hpa)
		Expect(err).ToNot(HaveOccurred())
		hpa.Status.CurrentMetrics = []autoscalingv2.MetricStatus{
			{
				Type: autoscalingv2.ResourceMetricSourceType,
				Resource: &autoscalingv2.ResourceMetricStatus{
					Name: corev1.ResourceCPU,
					Current: autoscalingv2.MetricValueStatus{
						Value: resource.NewQuantity(int64(100), resource.DecimalSI),
					},
				},
			},
		}
		err = k8sClient.Status().Update(ctx, hpa)
		Expect(err).ToNot(HaveOccurred())

		// hpa metrics will only left CPU metric
		Eventually(func() int {
			hpa := &autoscalingv2.HorizontalPodAutoscaler{}
			err = k8sClient.Get(context.Background(), types.NamespacedName{Name: getHPAName(so), Namespace: "default"}, hpa)
			Expect(err).ToNot(HaveOccurred())
			return len(hpa.Spec.Metrics)
		}, 5*time.Second).Should(Equal(1))

		// mock external server online
		atomic.StoreInt32(&scalers.MockExternalServerStatus, scalers.MockExternalServerStatusOnline)

		// wait so's ready condition Ready
		Eventually(func() metav1.ConditionStatus {
			err = k8sClient.Get(context.Background(), types.NamespacedName{Name: soName, Namespace: "default"}, so)
			Expect(err).ToNot(HaveOccurred())
			return so.Status.Conditions.GetReadyCondition().Status
		}, 5*time.Second).Should(Equal(metav1.ConditionTrue))

		// hpa will recover
		Eventually(func() int {
			hpa := &autoscalingv2.HorizontalPodAutoscaler{}
			err = k8sClient.Get(context.Background(), types.NamespacedName{Name: getHPAName(so), Namespace: "default"}, hpa)
			Expect(err).ToNot(HaveOccurred())
			return len(hpa.Spec.Metrics)
		}, 5*time.Second).Should(Equal(2))
	})
})

func generateDeployment(name string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  name,
							Image: name,
						},
					},
				},
			},
		},
	}
}
