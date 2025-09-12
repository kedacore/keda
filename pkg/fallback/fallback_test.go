/*
Copyright 2022 The KEDA Authors

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

package fallback

import (
	"context"
	"errors"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"go.uber.org/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"k8s.io/utils/ptr"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/mock/mock_client"
	"github.com/kedacore/keda/v2/pkg/mock/mock_scale"
	mock_scalers "github.com/kedacore/keda/v2/pkg/mock/mock_scaler"
)

const metricName = "some_metric_name"

func TestFallback(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var _ = Describe("fallback", func() {
	var (
		client      *mock_client.MockClient
		scaler      *mock_scalers.MockScaler
		scaleClient *mock_scale.MockScalesGetter
		ctrl        *gomock.Controller
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		client = mock_client.NewMockClient(ctrl)
		scaler = mock_scalers.NewMockScaler(ctrl)
		scaleClient = mock_scale.NewMockScalesGetter(ctrl)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("should return the expected metric when fallback is disabled", func() {
		expectedMetricValue := float64(5)
		primeGetMetrics(scaler, expectedMetricValue)
		so := buildScaledObject(nil, nil)
		metricSpec := createMetricSpec(3)

		metrics, _, err := scaler.GetMetricsAndActivity(context.Background(), metricName)
		metrics, _, err = GetMetricsWithFallback(context.Background(), client, scaleClient, metrics, err, metricName, so, metricSpec)

		Expect(err).ToNot(HaveOccurred())
		value := metrics[0].Value.AsApproximateFloat64()
		Expect(value).Should(Equal(expectedMetricValue))
	})

	It("should reset the health status when scaler metrics are available when fallback is enabled", func() {
		expectedMetricValue := float64(6)
		startingNumberOfFailures := int32(5)
		primeGetMetrics(scaler, expectedMetricValue)

		so := buildScaledObject(
			&kedav1alpha1.Fallback{
				FailureThreshold: int32(3),
				Replicas:         int32(10),
			},
			&kedav1alpha1.ScaledObjectStatus{
				Health: map[string]kedav1alpha1.HealthStatus{
					metricName: {
						NumberOfFailures: &startingNumberOfFailures,
						Status:           kedav1alpha1.HealthStatusFailing,
					},
				},
			},
		)

		metricSpec := createMetricSpec(3)
		expectStatusPatch(ctrl, client)

		metrics, _, err := scaler.GetMetricsAndActivity(context.Background(), metricName)
		metrics, _, err = GetMetricsWithFallback(context.Background(), client, scaleClient, metrics, err, metricName, so, metricSpec)

		Expect(err).ToNot(HaveOccurred())
		value := metrics[0].Value.AsApproximateFloat64()
		Expect(value).Should(Equal(expectedMetricValue))
		Expect(so.Status.Health[metricName]).To(haveFailureAndStatus(0, kedav1alpha1.HealthStatusHappy))
	})

	It("should not reset the health status when fallback is not enabled", func() {
		expectedMetricValue := float64(6)
		startingNumberOfFailures := int32(5)
		primeGetMetrics(scaler, expectedMetricValue)

		so := buildScaledObject(
			nil,
			&kedav1alpha1.ScaledObjectStatus{
				Health: map[string]kedav1alpha1.HealthStatus{
					metricName: {
						NumberOfFailures: &startingNumberOfFailures,
						Status:           kedav1alpha1.HealthStatusFailing,
					},
				},
			},
		)

		metricSpec := createMetricSpec(3)
		expectNoStatusPatch(ctrl)

		metrics, _, err := scaler.GetMetricsAndActivity(context.Background(), metricName)
		metrics, _, err = GetMetricsWithFallback(context.Background(), client, scaleClient, metrics, err, metricName, so, metricSpec)

		Expect(err).ToNot(HaveOccurred())
		value := metrics[0].Value.AsApproximateFloat64()
		Expect(value).Should(Equal(expectedMetricValue))
		Expect(so.Status.Health[metricName]).To(haveFailureAndStatus(5, kedav1alpha1.HealthStatusFailing))
	})

	It("should propagate the error when fallback is disabled", func() {
		scaler.EXPECT().GetMetricsAndActivity(gomock.Any(), gomock.Eq(metricName)).Return(nil, false, errors.New("some error"))

		so := buildScaledObject(nil, nil)
		metricSpec := createMetricSpec(3)
		expectNoStatusPatch(ctrl)

		metrics, _, err := scaler.GetMetricsAndActivity(context.Background(), metricName)
		_, _, err = GetMetricsWithFallback(context.Background(), client, scaleClient, metrics, err, metricName, so, metricSpec)

		Expect(err).ShouldNot(BeNil())
		Expect(err.Error()).Should(Equal("some error"))
	})

	It("should bump the number of failures when metrics call fails", func() {
		scaler.EXPECT().GetMetricsAndActivity(gomock.Any(), gomock.Eq(metricName)).Return(nil, false, errors.New("some error"))
		startingNumberOfFailures := int32(0)

		so := buildScaledObject(
			&kedav1alpha1.Fallback{
				FailureThreshold: int32(3),
				Replicas:         int32(10),
			},
			&kedav1alpha1.ScaledObjectStatus{
				Health: map[string]kedav1alpha1.HealthStatus{
					metricName: {
						NumberOfFailures: &startingNumberOfFailures,
						Status:           kedav1alpha1.HealthStatusHappy,
					},
				},
			},
		)

		metricSpec := createMetricSpec(10)
		expectStatusPatch(ctrl, client)

		metrics, _, err := scaler.GetMetricsAndActivity(context.Background(), metricName)
		_, _, err = GetMetricsWithFallback(context.Background(), client, scaleClient, metrics, err, metricName, so, metricSpec)

		Expect(err).ShouldNot(BeNil())
		Expect(err.Error()).Should(Equal("some error"))
		Expect(so.Status.Health[metricName]).To(haveFailureAndStatus(1, kedav1alpha1.HealthStatusFailing))
	})

	It("should return a normalised metric when number of failures are beyond threshold", func() {
		scaler.EXPECT().GetMetricsAndActivity(gomock.Any(), gomock.Eq(metricName)).Return(nil, false, errors.New("some error"))
		startingNumberOfFailures := int32(3)
		expectedMetricValue := float64(100)

		so := buildScaledObject(
			&kedav1alpha1.Fallback{
				FailureThreshold: int32(3),
				Replicas:         int32(10),
			},
			&kedav1alpha1.ScaledObjectStatus{
				Health: map[string]kedav1alpha1.HealthStatus{
					metricName: {
						NumberOfFailures: &startingNumberOfFailures,
						Status:           kedav1alpha1.HealthStatusHappy,
					},
				},
			},
		)
		metricSpec := createMetricSpec(10)
		expectStatusPatch(ctrl, client)

		mockScaleAndDeployment(ctrl, client, scaleClient, 5)

		metrics, _, err := scaler.GetMetricsAndActivity(context.Background(), metricName)
		metrics, _, err = GetMetricsWithFallback(context.Background(), client, scaleClient, metrics, err, metricName, so, metricSpec)

		Expect(err).ToNot(HaveOccurred())
		value := metrics[0].Value.AsApproximateFloat64()
		Expect(value).Should(Equal(expectedMetricValue))
		Expect(so.Status.Health[metricName]).To(haveFailureAndStatus(4, kedav1alpha1.HealthStatusFailing))
	})

	It("should behave as if fallback is disabled when the metrics spec target type is not average value metric", func() {
		so := buildScaledObject(
			&kedav1alpha1.Fallback{
				FailureThreshold: int32(3),
				Replicas:         int32(10),
			}, nil,
		)

		qty := resource.NewQuantity(int64(3), resource.DecimalSI)
		metricsSpec := v2.MetricSpec{
			External: &v2.ExternalMetricSource{
				Target: v2.MetricTarget{
					Type:  v2.UtilizationMetricType,
					Value: qty,
				},
			},
		}

		isEnabled := isFallbackEnabled(so, metricsSpec)
		Expect(isEnabled).Should(BeFalse())
	})

	It("should ignore error if we fail to update kubernetes status", func() {
		scaler.EXPECT().GetMetricsAndActivity(gomock.Any(), gomock.Eq(metricName)).Return(nil, false, errors.New("some error"))
		startingNumberOfFailures := int32(3)
		expectedMetricValue := float64(100)

		so := buildScaledObject(
			&kedav1alpha1.Fallback{
				FailureThreshold: int32(3),
				Replicas:         int32(10),
			},
			&kedav1alpha1.ScaledObjectStatus{
				Health: map[string]kedav1alpha1.HealthStatus{
					metricName: {
						NumberOfFailures: &startingNumberOfFailures,
						Status:           kedav1alpha1.HealthStatusHappy,
					},
				},
			},
		)
		metricSpec := createMetricSpec(10)

		statusWriter := mock_client.NewMockStatusWriter(ctrl)
		statusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("some error"))
		client.EXPECT().Status().Return(statusWriter)

		mockScaleAndDeployment(ctrl, client, scaleClient, 5)

		metrics, _, err := scaler.GetMetricsAndActivity(context.Background(), metricName)
		metrics, _, err = GetMetricsWithFallback(context.Background(), client, scaleClient, metrics, err, metricName, so, metricSpec)

		Expect(err).ToNot(HaveOccurred())
		value := metrics[0].Value.AsApproximateFloat64()
		Expect(value).Should(Equal(expectedMetricValue))
		Expect(so.Status.Health[metricName]).To(haveFailureAndStatus(4, kedav1alpha1.HealthStatusFailing))
	})

	It("should return error when fallback is enabled but scaledobject has invalid parameter", func() {
		scaler.EXPECT().GetMetricsAndActivity(gomock.Any(), gomock.Eq(metricName)).Return(nil, false, errors.New("some error"))
		startingNumberOfFailures := int32(3)

		so := buildScaledObject(
			&kedav1alpha1.Fallback{
				FailureThreshold: int32(-3),
				Replicas:         int32(10),
			},
			&kedav1alpha1.ScaledObjectStatus{
				Health: map[string]kedav1alpha1.HealthStatus{
					metricName: {
						NumberOfFailures: &startingNumberOfFailures,
						Status:           kedav1alpha1.HealthStatusHappy,
					},
				},
			},
		)
		metricSpec := createMetricSpec(10)

		metrics, _, err := scaler.GetMetricsAndActivity(context.Background(), metricName)
		_, _, err = GetMetricsWithFallback(context.Background(), client, scaleClient, metrics, err, metricName, so, metricSpec)

		Expect(err).ShouldNot(BeNil())
		Expect(err.Error()).Should(Equal("some error"))
	})

	It("should set the fallback condition when a fallback exists in the scaled object", func() {
		scaler.EXPECT().GetMetricsAndActivity(gomock.Any(), gomock.Eq(metricName)).Return(nil, false, errors.New("some error"))
		startingNumberOfFailures := int32(3)
		failingNumberOfFailures := int32(6)
		anotherMetricName := "another metric name"

		so := buildScaledObject(
			&kedav1alpha1.Fallback{
				FailureThreshold: int32(3),
				Replicas:         int32(10),
			},
			&kedav1alpha1.ScaledObjectStatus{
				Health: map[string]kedav1alpha1.HealthStatus{
					anotherMetricName: {
						NumberOfFailures: &failingNumberOfFailures,
						Status:           kedav1alpha1.HealthStatusFailing,
					},
					metricName: {
						NumberOfFailures: &startingNumberOfFailures,
						Status:           kedav1alpha1.HealthStatusHappy,
					},
				},
			},
		)
		metricSpec := createMetricSpec(10)
		expectStatusPatch(ctrl, client)

		mockScaleAndDeployment(ctrl, client, scaleClient, 5)

		metrics, _, err := scaler.GetMetricsAndActivity(context.Background(), metricName)
		_, _, err = GetMetricsWithFallback(context.Background(), client, scaleClient, metrics, err, metricName, so, metricSpec)

		Expect(err).ToNot(HaveOccurred())
		condition := so.Status.Conditions.GetFallbackCondition()
		Expect(condition.IsTrue()).Should(BeTrue())
	})

	It("should set the fallback condition to false if the config is invalid", func() {
		scaler.EXPECT().GetMetricsAndActivity(gomock.Any(), gomock.Eq(metricName)).Return(nil, false, errors.New("some error"))
		startingNumberOfFailures := int32(3)
		failingNumberOfFailures := int32(6)
		anotherMetricName := "another metric name"

		so := buildScaledObject(
			&kedav1alpha1.Fallback{
				FailureThreshold: int32(-3),
				Replicas:         int32(10),
			},
			&kedav1alpha1.ScaledObjectStatus{
				Health: map[string]kedav1alpha1.HealthStatus{
					anotherMetricName: {
						NumberOfFailures: &failingNumberOfFailures,
						Status:           kedav1alpha1.HealthStatusFailing,
					},
					metricName: {
						NumberOfFailures: &startingNumberOfFailures,
						Status:           kedav1alpha1.HealthStatusHappy,
					},
				},
			},
		)
		metricSpec := createMetricSpec(10)

		metrics, _, err := scaler.GetMetricsAndActivity(context.Background(), metricName)
		_, _, err = GetMetricsWithFallback(context.Background(), client, scaleClient, metrics, err, metricName, so, metricSpec)

		Expect(err).ShouldNot(BeNil())
		Expect(err.Error()).Should(Equal("some error"))
		condition := so.Status.Conditions.GetFallbackCondition()
		Expect(condition.IsTrue()).Should(BeFalse())
	})

	It("should use fallback replicas when current replicas is lower when behavior is 'currentReplicasIfHigher'", func() {
		scaler.EXPECT().GetMetricsAndActivity(gomock.Any(), gomock.Eq(metricName)).Return(nil, false, errors.New("some error"))
		startingNumberOfFailures := int32(3)
		behavior := "currentReplicasIfHigher"

		so := buildScaledObject(
			&kedav1alpha1.Fallback{
				FailureThreshold: int32(3),
				Replicas:         int32(10),
				Behavior:         behavior,
			},
			&kedav1alpha1.ScaledObjectStatus{
				Health: map[string]kedav1alpha1.HealthStatus{
					metricName: {
						NumberOfFailures: &startingNumberOfFailures,
						Status:           kedav1alpha1.HealthStatusHappy,
					},
				},
			},
		)
		metricSpec := createMetricSpec(10)
		expectStatusPatch(ctrl, client)

		mockScaleAndDeployment(ctrl, client, scaleClient, 4)

		metrics, _, err := scaler.GetMetricsAndActivity(context.Background(), metricName)
		metrics, _, err = GetMetricsWithFallback(context.Background(), client, scaleClient, metrics, err, metricName, so, metricSpec)

		Expect(err).ToNot(HaveOccurred())
		value := metrics[0].Value.AsApproximateFloat64()
		expectedValue := float64(100) // 10 replicas * 10 target value
		Expect(value).Should(Equal(expectedValue))
	})

	It("should use current replicas when behavior is 'currentReplicas'", func() {
		scaler.EXPECT().GetMetricsAndActivity(gomock.Any(), gomock.Eq(metricName)).Return(nil, false, errors.New("some error"))
		startingNumberOfFailures := int32(3)
		behavior := "currentReplicas"

		so := buildScaledObject(
			&kedav1alpha1.Fallback{
				FailureThreshold: int32(3),
				Replicas:         int32(10),
				Behavior:         behavior,
			},
			&kedav1alpha1.ScaledObjectStatus{
				Health: map[string]kedav1alpha1.HealthStatus{
					metricName: {
						NumberOfFailures: &startingNumberOfFailures,
						Status:           kedav1alpha1.HealthStatusHappy,
					},
				},
			},
		)
		metricSpec := createMetricSpec(10)
		expectStatusPatch(ctrl, client)

		mockScaleAndDeployment(ctrl, client, scaleClient, 6)

		metrics, _, err := scaler.GetMetricsAndActivity(context.Background(), metricName)
		metrics, _, err = GetMetricsWithFallback(context.Background(), client, scaleClient, metrics, err, metricName, so, metricSpec)

		Expect(err).ToNot(HaveOccurred())
		value := metrics[0].Value.AsApproximateFloat64()
		expectedValue := float64(60) // 6 replicas * 10 target value
		Expect(value).Should(Equal(expectedValue))
	})

	It("should ignore current replicas when behavior is 'static'", func() {
		scaler.EXPECT().GetMetricsAndActivity(gomock.Any(), gomock.Eq(metricName)).Return(nil, false, errors.New("some error"))
		startingNumberOfFailures := int32(3)
		behavior := "static"

		so := buildScaledObject(
			&kedav1alpha1.Fallback{
				FailureThreshold: int32(3),
				Replicas:         int32(10),
				Behavior:         behavior,
			},
			&kedav1alpha1.ScaledObjectStatus{
				Health: map[string]kedav1alpha1.HealthStatus{
					metricName: {
						NumberOfFailures: &startingNumberOfFailures,
						Status:           kedav1alpha1.HealthStatusHappy,
					},
				},
			},
		)
		metricSpec := createMetricSpec(10)
		expectStatusPatch(ctrl, client)

		metrics, _, err := scaler.GetMetricsAndActivity(context.Background(), metricName)
		metrics, _, err = GetMetricsWithFallback(context.Background(), client, scaleClient, metrics, err, metricName, so, metricSpec)

		Expect(err).ToNot(HaveOccurred())
		value := metrics[0].Value.AsApproximateFloat64()
		expectedValue := float64(100) // 10 replicas * 10 target value, ignoring current 15
		Expect(value).Should(Equal(expectedValue))
	})
})

// Helper functions
func mockScaleAndDeployment(
	ctrl *gomock.Controller,
	client *mock_client.MockClient,
	scaleClient *mock_scale.MockScalesGetter,
	replicas int32,
) {
	// Mock Scale interface
	mockScaleInterface := mock_scale.NewMockScaleInterface(ctrl)
	scale := &autoscalingv1.Scale{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "myapp",
			Namespace: "default",
		},
		Spec: autoscalingv1.ScaleSpec{
			Replicas: replicas,
		},
	}

	scaleClient.EXPECT().Scales(gomock.Eq("default")).Return(mockScaleInterface).MinTimes(1).MaxTimes(2)
	mockScaleInterface.EXPECT().Get(
		gomock.Any(),
		gomock.Eq("apps"),
		gomock.Eq("deployment"),
		gomock.Eq("myapp"),
	).Return(scale, nil).MinTimes(1).MaxTimes(2)

	// Mock Deployment lookup
	deployment := &appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To[int32](replicas),
		},
	}
	client.EXPECT().Get(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Return(nil).SetArg(2, *deployment)
}

func haveFailureAndStatus(numberOfFailures int, status kedav1alpha1.HealthStatusType) types.GomegaMatcher {
	return &healthStatusMatcher{numberOfFailures: numberOfFailures, status: status}
}

type healthStatusMatcher struct {
	numberOfFailures int
	status           kedav1alpha1.HealthStatusType
}

func (h *healthStatusMatcher) Match(actual interface{}) (success bool, err error) {
	switch v := actual.(type) {
	case kedav1alpha1.HealthStatus:
		return *v.NumberOfFailures == int32(h.numberOfFailures) && v.Status == h.status, nil
	default:
		return false, fmt.Errorf("expected kedav1alpha1.HealthStatus, got %v", actual)
	}
}

func (h *healthStatusMatcher) FailureMessage(actual interface{}) (message string) {
	switch v := actual.(type) {
	case kedav1alpha1.HealthStatus:
		return fmt.Sprintf("expected HealthStatus with NumberOfFailures %d and Status %s, but got NumberOfFailures %d and Status %s", h.numberOfFailures, h.status, *v.NumberOfFailures, v.Status)
	default:
		return "unexpected error"
	}
}

func (h *healthStatusMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	switch v := actual.(type) {
	case kedav1alpha1.HealthStatus:
		return fmt.Sprintf("did not expect HealthStatus with NumberOfFailures %d and Status %s, but got NumberOfFailures %d and Status %s", h.numberOfFailures, h.status, *v.NumberOfFailures, v.Status)
	default:
		return "unexpected error"
	}
}

func expectStatusPatch(ctrl *gomock.Controller, client *mock_client.MockClient) {
	statusWriter := mock_client.NewMockStatusWriter(ctrl)
	statusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any())
	client.EXPECT().Status().Return(statusWriter)
}

func expectNoStatusPatch(ctrl *gomock.Controller) {
	statusWriter := mock_client.NewMockStatusWriter(ctrl)
	statusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
}

func buildScaledObject(fallbackConfig *kedav1alpha1.Fallback, status *kedav1alpha1.ScaledObjectStatus) *kedav1alpha1.ScaledObject {
	scaledObject := &kedav1alpha1.ScaledObject{
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
			},
			Fallback: fallbackConfig,
		},
		Status: kedav1alpha1.ScaledObjectStatus{
			ScaleTargetGVKR: &kedav1alpha1.GroupVersionKindResource{
				Group: "apps",
				Kind:  "Deployment",
			},
		},
	}

	if status != nil {
		// Preserve ScaleTargetGVKR when overwriting status
		gvkr := scaledObject.Status.ScaleTargetGVKR
		scaledObject.Status = *status
		scaledObject.Status.ScaleTargetGVKR = gvkr
	}

	scaledObject.Status.Conditions = *kedav1alpha1.GetInitializedConditions()

	return scaledObject
}

func primeGetMetrics(scaler *mock_scalers.MockScaler, value float64) {
	expectedMetric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(int64(value), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	scaler.EXPECT().GetMetricsAndActivity(gomock.Any(), gomock.Eq(metricName)).Return([]external_metrics.ExternalMetricValue{expectedMetric}, true, nil)
}

func createMetricSpec(averageValue int) v2.MetricSpec {
	qty := resource.NewQuantity(int64(averageValue), resource.DecimalSI)
	return v2.MetricSpec{
		External: &v2.ExternalMetricSource{
			Target: v2.MetricTarget{
				Type:         v2.AverageValueMetricType,
				AverageValue: qty,
			},
		},
	}
}
