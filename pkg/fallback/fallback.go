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
	"fmt"
	"reflect"
	"strconv"

	v2 "k8s.io/api/autoscaling/v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/scale"
	"k8s.io/metrics/pkg/apis/external_metrics"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scaling/resolver"
)

var log = logf.Log.WithName("fallback")

func isFallbackEnabled(scaledObject *kedav1alpha1.ScaledObject) bool {
	return scaledObject.Spec.Fallback != nil
}

func GetMetricsWithFallback(ctx context.Context, client runtimeclient.Client, scaleClient scale.ScalesGetter, metrics []external_metrics.ExternalMetricValue, suppressedError error, metricName string, scaledObject *kedav1alpha1.ScaledObject, metricSpec v2.MetricSpec) ([]external_metrics.ExternalMetricValue, bool, error) {
	status := scaledObject.Status.DeepCopy()

	initHealthStatus(status)
	healthStatus := getHealthStatus(status, metricName)

	if suppressedError == nil {
		zero := int32(0)
		healthStatus.NumberOfFailures = &zero
		healthStatus.Status = kedav1alpha1.HealthStatusHappy
		status.Health[metricName] = *healthStatus

		updateStatus(ctx, client, scaledObject, status)

		return metrics, false, nil
	}

	healthStatus.Status = kedav1alpha1.HealthStatusFailing
	*healthStatus.NumberOfFailures++
	status.Health[metricName] = *healthStatus

	updateStatus(ctx, client, scaledObject, status)

	switch {
	case !isFallbackEnabled(scaledObject):
		return nil, false, suppressedError
	case !HasValidFallback(scaledObject):
		log.Info("Failed to validate ScaledObject Spec. Please check that parameters are positive integers", "scaledObject.Namespace", scaledObject.Namespace, "scaledObject.Name", scaledObject.Name)
		return nil, false, suppressedError
	case *healthStatus.NumberOfFailures > scaledObject.Spec.Fallback.FailureThreshold:
		// For triggerScoped behavior, return metrics (or placeholder) so formula can handle nil values
		if scaledObject.Spec.Fallback.Behavior == kedav1alpha1.FallbackBehaviorTriggerScoped {
			// Return existing metrics if available, otherwise create a placeholder
			if len(metrics) > 0 {
				return metrics, false, suppressedError
			}
			// Create placeholder metric with zero value for formula evaluation
			placeholderMetric := external_metrics.ExternalMetricValue{
				MetricName: metricName,
				Value:      *resource.NewQuantity(0, resource.DecimalSI),
				Timestamp:  metav1.Now(),
			}
			return []external_metrics.ExternalMetricValue{placeholderMetric}, false, suppressedError
		}

		var currentReplicas int32
		var err error

		if scaledObject.Spec.Fallback.Behavior != kedav1alpha1.FallbackBehaviorStatic {
			currentReplicas, err = resolver.GetCurrentReplicas(ctx, client, scaleClient, scaledObject)
			if err != nil {
				return nil, false, suppressedError
			}
		}

		l := doFallback(ctx, client, scaleClient, scaledObject, metricSpec, metricName, currentReplicas, suppressedError)
		if l == nil {
			return l, false, fmt.Errorf("error performing fallback")
		}
		return l, true, nil

	default:
		return nil, false, suppressedError
	}
}

func fallbackExistsInScaledObject(scaledObject *kedav1alpha1.ScaledObject) bool {
	for _, element := range scaledObject.Status.Health {
		if element.Status == kedav1alpha1.HealthStatusFailing && *element.NumberOfFailures > scaledObject.Spec.Fallback.FailureThreshold {
			return true
		}
	}

	return false
}

func HasValidFallback(scaledObject *kedav1alpha1.ScaledObject) bool {
	modifierChecking := true
	if scaledObject.IsUsingModifiers() {
		value, err := strconv.ParseInt(scaledObject.Spec.Advanced.ScalingModifiers.Target, 10, 64)
		modifierChecking = err == nil && value > 0
	}
	return scaledObject.Spec.Fallback.FailureThreshold >= 0 &&
		scaledObject.Spec.Fallback.Replicas >= 0 &&
		modifierChecking
}

// IsPodReady returns true if a pod is ready; false otherwise.
func IsPodReady(pod *v1.Pod) bool {
	if pod != nil {
		for _, c := range pod.Status.Conditions {
			if c.Type == v1.PodReady {
				return c.Status == v1.ConditionTrue
			}
		}
	}
	return false
}

// Similar to how it's done in HPA's code: https://github.com/kubernetes/kubernetes/blob/091f87c10bc3532041b77a783a5f832de5506dc8/pkg/controller/podautoscaler/replica_calculator.go#L323
func getReadyReplicasCount(ctx context.Context, client runtimeclient.Client, scaleClient scale.ScalesGetter, scaledObject *kedav1alpha1.ScaledObject) (int32, error) {
	if scaledObject == nil || scaledObject.Spec.ScaleTargetRef == nil || scaledObject.Status.ScaleTargetGVKR == nil {
		return -1, fmt.Errorf("")
	}

	scale, err := scaleClient.Scales(scaledObject.Namespace).Get(ctx, scaledObject.Status.ScaleTargetGVKR.GroupResource(), scaledObject.Spec.ScaleTargetRef.Name, metav1.GetOptions{})
	if err != nil {
		return -1, err
	}

	parsedSelector, err := labels.Parse(scale.Status.Selector)
	if err != nil {
		return -1, err
	}

	podList := v1.PodList{}
	err = client.List(ctx, &podList, runtimeclient.InNamespace(scaledObject.Namespace), runtimeclient.MatchingLabelsSelector{Selector: parsedSelector})
	if err != nil {
		return -1, err
	}

	var readyPodCount int32
	for _, pod := range podList.Items {
		if pod.Status.Phase == v1.PodRunning && IsPodReady(&pod) {
			readyPodCount++
		}
	}

	return readyPodCount, nil
}

func doFallback(ctx context.Context, client runtimeclient.Client, scaleClient scale.ScalesGetter, scaledObject *kedav1alpha1.ScaledObject, metricSpec v2.MetricSpec, metricName string, currentReplicas int32, suppressedError error) []external_metrics.ExternalMetricValue {
	fallbackBehavior := scaledObject.Spec.Fallback.Behavior
	fallbackReplicas := int64(scaledObject.Spec.Fallback.Replicas)
	var replicas float64

	switch fallbackBehavior {
	case kedav1alpha1.FallbackBehaviorStatic:
		replicas = float64(fallbackReplicas)
	case kedav1alpha1.FallbackBehaviorCurrentReplicas:
		replicas = float64(currentReplicas)
	case kedav1alpha1.FallbackBehaviorCurrentReplicasIfHigher:
		currentReplicasCount := int64(currentReplicas)
		if currentReplicasCount > fallbackReplicas {
			replicas = float64(currentReplicasCount)
		} else {
			replicas = float64(fallbackReplicas)
		}
	case kedav1alpha1.FallbackBehaviorCurrentReplicasIfLower:
		currentReplicasCount := int64(currentReplicas)
		if currentReplicasCount < fallbackReplicas {
			replicas = float64(currentReplicasCount)
		} else {
			replicas = float64(fallbackReplicas)
		}
	case kedav1alpha1.FallbackBehaviorTriggerScoped:
		// This case should not be reached as triggerScoped returns early in GetMetricsWithFallback
		// But keep as safety check
		return nil
	default:
		replicas = float64(fallbackReplicas)
	}

	// If the metricType is Value, we get the number of readyReplicas, and divide replicas by it.
	if (!scaledObject.IsUsingModifiers() && metricSpec.External.Target.Type == v2.ValueMetricType) ||
		(scaledObject.IsUsingModifiers() && scaledObject.Spec.Advanced.ScalingModifiers.MetricType == v2.ValueMetricType) {
		readyReplicas, err := getReadyReplicasCount(ctx, client, scaleClient, scaledObject)
		if err != nil {
			log.Error(err, "failed to do fallback for metric of type Value", "scaledObject.Namespace", scaledObject.Namespace, "scaledObject.Name", scaledObject.Name, "metricName", metricName)
			return nil
		}
		if readyReplicas == 0 {
			log.Error(fmt.Errorf("readyReplicas is zero, cannot do fallback"), "failed to do fallback for metric of type Value", "scaledObject.Namespace", scaledObject.Namespace, "scaledObject.Name", scaledObject.Name, "metricName", metricName)
			return nil
		}
		replicas /= float64(readyReplicas)
	}

	var normalisationValue float64
	if !scaledObject.IsUsingModifiers() {
		switch metricSpec.External.Target.Type {
		case v2.AverageValueMetricType:
			normalisationValue = metricSpec.External.Target.AverageValue.AsApproximateFloat64()
		case v2.ValueMetricType:
			normalisationValue = metricSpec.External.Target.Value.AsApproximateFloat64()
		}
	} else {
		value, _ := strconv.ParseFloat(scaledObject.Spec.Advanced.ScalingModifiers.Target, 64)
		normalisationValue = value
		metricName = kedav1alpha1.CompositeMetricName
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewMilliQuantity(int64(normalisationValue*1000*replicas), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}
	fallbackMetrics := []external_metrics.ExternalMetricValue{metric}

	log.Info("Suppressing error, using fallback metrics",
		"scaledObject.Namespace", scaledObject.Namespace,
		"scaledObject.Name", scaledObject.Name,
		"suppressedError", suppressedError,
		"fallback.behavior", fallbackBehavior,
		"fallback.replicas", fallbackReplicas,
		"workload.currentReplicas", currentReplicas)
	return fallbackMetrics
}

func updateStatus(ctx context.Context, client runtimeclient.Client, scaledObject *kedav1alpha1.ScaledObject, status *kedav1alpha1.ScaledObjectStatus) {
	patch := runtimeclient.MergeFrom(scaledObject.DeepCopy())

	if !isFallbackEnabled(scaledObject) || !HasValidFallback(scaledObject) {
		log.V(1).Info("Fallback is not enabled, hence skipping the health update to the scaledobject", "scaledObject.Namespace", scaledObject.Namespace, "scaledObject.Name", scaledObject.Name)
		return
	}

	if fallbackExistsInScaledObject(scaledObject) {
		status.Conditions.SetFallbackCondition(metav1.ConditionTrue, "FallbackExists", "At least one trigger is falling back on this scaled object")
	} else {
		status.Conditions.SetFallbackCondition(metav1.ConditionFalse, "NoFallbackFound", "No fallbacks are active on this scaled object")
	}

	// Update status only if it has changed
	if !reflect.DeepEqual(scaledObject.Status, *status) {
		scaledObject.Status = *status
		err := client.Status().Patch(ctx, scaledObject, patch)
		if err != nil {
			log.Error(err, "failed to patch ScaledObjects Status", "scaledObject.Namespace", scaledObject.Namespace, "scaledObject.Name", scaledObject.Name)
		}
	}
}

func getHealthStatus(status *kedav1alpha1.ScaledObjectStatus, metricName string) *kedav1alpha1.HealthStatus {
	// Get health status for a specific metric
	_, healthStatusExists := status.Health[metricName]
	if !healthStatusExists {
		zero := int32(0)
		healthStatus := kedav1alpha1.HealthStatus{
			NumberOfFailures: &zero,
			Status:           kedav1alpha1.HealthStatusHappy,
		}
		status.Health[metricName] = healthStatus
	}
	healthStatus := status.Health[metricName]
	return &healthStatus
}

func initHealthStatus(status *kedav1alpha1.ScaledObjectStatus) {
	// Init health status if missing
	if status.Health == nil {
		status.Health = make(map[string]kedav1alpha1.HealthStatus)
	}
}
