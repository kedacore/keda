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
	"reflect"
	"strconv"

	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/scale"
	"k8s.io/metrics/pkg/apis/external_metrics"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scaling/resolver"
)

var log = logf.Log.WithName("fallback")

func isFallbackEnabled(scaledObject *kedav1alpha1.ScaledObject, metricSpec v2.MetricSpec) bool {
	if scaledObject.Spec.Fallback == nil {
		return false
	}

	// If we are using ScalingModifiers, we only care whether its metric type is AverageValue.
	// If not, test the type of metricSpec passed.
	if scaledObject.IsUsingModifiers() && scaledObject.Spec.Advanced.ScalingModifiers.MetricType != v2.AverageValueMetricType {
		log.V(0).Info("Fallback can only be enabled for scalingModifiers with metric of type AverageValue", "scaledObject.Namespace", scaledObject.Namespace, "scaledObject.Name", scaledObject.Name)
		return false
	} else if !scaledObject.IsUsingModifiers() && metricSpec.External.Target.Type != v2.AverageValueMetricType {
		log.V(0).Info("Fallback can only be enabled for triggers with metric of type AverageValue", "scaledObject.Namespace", scaledObject.Namespace, "scaledObject.Name", scaledObject.Name)
		return false
	}

	return true
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

		updateStatus(ctx, client, scaledObject, status, metricSpec)

		return metrics, false, nil
	}

	healthStatus.Status = kedav1alpha1.HealthStatusFailing
	*healthStatus.NumberOfFailures++
	status.Health[metricName] = *healthStatus

	updateStatus(ctx, client, scaledObject, status, metricSpec)

	switch {
	case !isFallbackEnabled(scaledObject, metricSpec):
		return nil, false, suppressedError
	case !HasValidFallback(scaledObject):
		log.Info("Failed to validate ScaledObject Spec. Please check that parameters are positive integers", "scaledObject.Namespace", scaledObject.Namespace, "scaledObject.Name", scaledObject.Name)
		return nil, false, suppressedError
	case *healthStatus.NumberOfFailures > scaledObject.Spec.Fallback.FailureThreshold:
		var currentReplicas int32
		var err error

		if scaledObject.Spec.Fallback.Behavior != kedav1alpha1.FallbackBehaviorStatic {
			currentReplicas, err = resolver.GetCurrentReplicas(ctx, client, scaleClient, scaledObject)
			if err != nil {
				return nil, false, suppressedError
			}
		}
		return doFallback(scaledObject, metricSpec, metricName, currentReplicas, suppressedError), true, nil
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

func doFallback(scaledObject *kedav1alpha1.ScaledObject, metricSpec v2.MetricSpec, metricName string, currentReplicas int32, suppressedError error) []external_metrics.ExternalMetricValue {
	fallbackBehavior := scaledObject.Spec.Fallback.Behavior
	fallbackReplicas := int64(scaledObject.Spec.Fallback.Replicas)
	var replicas int64

	switch fallbackBehavior {
	case kedav1alpha1.FallbackBehaviorStatic:
		replicas = fallbackReplicas
	case kedav1alpha1.FallbackBehaviorCurrentReplicas:
		replicas = int64(currentReplicas)
	case kedav1alpha1.FallbackBehaviorCurrentReplicasIfHigher:
		currentReplicasCount := int64(currentReplicas)
		if currentReplicasCount > fallbackReplicas {
			replicas = currentReplicasCount
		} else {
			replicas = fallbackReplicas
		}
	case kedav1alpha1.FallbackBehaviorCurrentReplicasIfLower:
		currentReplicasCount := int64(currentReplicas)
		if currentReplicasCount < fallbackReplicas {
			replicas = currentReplicasCount
		} else {
			replicas = fallbackReplicas
		}
	default:
		replicas = fallbackReplicas
	}

	var normalisationValue int64
	if !scaledObject.IsUsingModifiers() {
		normalisationValue = int64(metricSpec.External.Target.AverageValue.AsApproximateFloat64())
	} else {
		value, _ := strconv.ParseInt(scaledObject.Spec.Advanced.ScalingModifiers.Target, 10, 64)
		normalisationValue = value
		metricName = kedav1alpha1.CompositeMetricName
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewMilliQuantity(normalisationValue*1000*replicas, resource.DecimalSI),
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

func updateStatus(ctx context.Context, client runtimeclient.Client, scaledObject *kedav1alpha1.ScaledObject, status *kedav1alpha1.ScaledObjectStatus, metricSpec v2.MetricSpec) {
	patch := runtimeclient.MergeFrom(scaledObject.DeepCopy())

	if !isFallbackEnabled(scaledObject, metricSpec) || !HasValidFallback(scaledObject) {
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
