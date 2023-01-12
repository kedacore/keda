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

	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/mock/mock_client"
	mock_scalers "github.com/kedacore/keda/v2/pkg/mock/mock_scaler"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

const metricName = "some_metric_name"

func TestFallback(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = Describe("fallback", func() {
	var (
		client *mock_client.MockClient
		scaler *mock_scalers.MockScaler
		ctrl   *gomock.Controller
		logger logr.Logger
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		client = mock_client.NewMockClient(ctrl)
		scaler = mock_scalers.NewMockScaler(ctrl)

		logger = logr.Discard()
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("should return the expected metric when fallback is disabled", func() {

		expectedMetricValue := int64(5)
		primeGetMetrics(scaler, expectedMetricValue)
		so := buildScaledObject(nil, nil)
		metricSpec := createMetricSpec(3)
		expectStatusPatch(ctrl, client)

		metrics, _, err := scaler.GetMetricsAndActivity(context.Background(), metricName)
		metrics, err = GetMetricsWithFallback(context.Background(), client, logger, metrics, err, metricName, so, metricSpec)

		Expect(err).ToNot(HaveOccurred())
		value, _ := metrics[0].Value.AsInt64()
		Expect(value).Should(Equal(expectedMetricValue))
	})

	It("should reset the health status when scaler metrics are available", func() {
		expectedMetricValue := int64(6)
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
		metrics, err = GetMetricsWithFallback(context.Background(), client, logger, metrics, err, metricName, so, metricSpec)

		Expect(err).ToNot(HaveOccurred())
		value, _ := metrics[0].Value.AsInt64()
		Expect(value).Should(Equal(expectedMetricValue))
		Expect(so.Status.Health[metricName]).To(haveFailureAndStatus(0, kedav1alpha1.HealthStatusHappy))
	})

	It("should propagate the error when fallback is disabled", func() {
		scaler.EXPECT().GetMetricsAndActivity(gomock.Any(), gomock.Eq(metricName)).Return(nil, false, errors.New("Some error"))

		so := buildScaledObject(nil, nil)
		metricSpec := createMetricSpec(3)
		expectStatusPatch(ctrl, client)

		metrics, _, err := scaler.GetMetricsAndActivity(context.Background(), metricName)
		_, err = GetMetricsWithFallback(context.Background(), client, logger, metrics, err, metricName, so, metricSpec)

		Expect(err).ShouldNot(BeNil())
		Expect(err.Error()).Should(Equal("Some error"))
	})

	It("should bump the number of failures when metrics call fails", func() {
		scaler.EXPECT().GetMetricsAndActivity(gomock.Any(), gomock.Eq(metricName)).Return(nil, false, errors.New("Some error"))
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
		_, err = GetMetricsWithFallback(context.Background(), client, logger, metrics, err, metricName, so, metricSpec)

		Expect(err).ShouldNot(BeNil())
		Expect(err.Error()).Should(Equal("Some error"))
		Expect(so.Status.Health[metricName]).To(haveFailureAndStatus(1, kedav1alpha1.HealthStatusFailing))
	})

	It("should return a normalised metric when number of failures are beyond threshold", func() {
		scaler.EXPECT().GetMetricsAndActivity(gomock.Any(), gomock.Eq(metricName)).Return(nil, false, errors.New("Some error"))
		startingNumberOfFailures := int32(3)
		expectedMetricValue := int64(100)

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
		metrics, err = GetMetricsWithFallback(context.Background(), client, logger, metrics, err, metricName, so, metricSpec)

		Expect(err).ToNot(HaveOccurred())
		value, _ := metrics[0].Value.AsInt64()
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

		isEnabled := isFallbackEnabled(logger, so, metricsSpec)
		Expect(isEnabled).Should(BeFalse())
	})

	It("should ignore error if we fail to update kubernetes status", func() {
		scaler.EXPECT().GetMetricsAndActivity(gomock.Any(), gomock.Eq(metricName)).Return(nil, false, errors.New("Some error"))
		startingNumberOfFailures := int32(3)
		expectedMetricValue := int64(100)

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
		statusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("Some error"))
		client.EXPECT().Status().Return(statusWriter)

		metrics, _, err := scaler.GetMetricsAndActivity(context.Background(), metricName)
		metrics, err = GetMetricsWithFallback(context.Background(), client, logger, metrics, err, metricName, so, metricSpec)

		Expect(err).ToNot(HaveOccurred())
		value, _ := metrics[0].Value.AsInt64()
		Expect(value).Should(Equal(expectedMetricValue))
		Expect(so.Status.Health[metricName]).To(haveFailureAndStatus(4, kedav1alpha1.HealthStatusFailing))
	})

	It("should return error when fallback is enabled but scaledobject has invalid parameter", func() {
		scaler.EXPECT().GetMetricsAndActivity(gomock.Any(), gomock.Eq(metricName)).Return(nil, false, errors.New("Some error"))
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
		expectStatusPatch(ctrl, client)

		metrics, _, err := scaler.GetMetricsAndActivity(context.Background(), metricName)
		_, err = GetMetricsWithFallback(context.Background(), client, logger, metrics, err, metricName, so, metricSpec)

		Expect(err).ShouldNot(BeNil())
		Expect(err.Error()).Should(Equal("Some error"))
	})

	It("should set the fallback condition when a fallback exists in the scaled object", func() {
		scaler.EXPECT().GetMetricsAndActivity(gomock.Any(), gomock.Eq(metricName)).Return(nil, false, errors.New("Some error"))
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

		metrics, _, err := scaler.GetMetricsAndActivity(context.Background(), metricName)
		_, err = GetMetricsWithFallback(context.Background(), client, logger, metrics, err, metricName, so, metricSpec)
		Expect(err).ToNot(HaveOccurred())
		condition := so.Status.Conditions.GetFallbackCondition()
		Expect(condition.IsTrue()).Should(BeTrue())
	})

	It("should set the fallback condition to false if the config is invalid", func() {
		scaler.EXPECT().GetMetricsAndActivity(gomock.Any(), gomock.Eq(metricName)).Return(nil, false, errors.New("Some error"))
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
		expectStatusPatch(ctrl, client)

		metrics, _, err := scaler.GetMetricsAndActivity(context.Background(), metricName)
		_, err = GetMetricsWithFallback(context.Background(), client, logger, metrics, err, metricName, so, metricSpec)
		Expect(err).ShouldNot(BeNil())
		Expect(err.Error()).Should(Equal("Some error"))
		condition := so.Status.Conditions.GetFallbackCondition()
		Expect(condition.IsTrue()).Should(BeFalse())
	})
})

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
	}

	if status != nil {
		scaledObject.Status = *status
	}

	scaledObject.Status.Conditions = *kedav1alpha1.GetInitializedConditions()

	return scaledObject
}

func primeGetMetrics(scaler *mock_scalers.MockScaler, value int64) {
	expectedMetric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(value, resource.DecimalSI),
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
