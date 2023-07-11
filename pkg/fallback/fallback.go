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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/apis/external_metrics"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	cl "github.com/kedacore/keda/v2/pkg/externalscaling/api"
)

var log = logf.Log.WithName("fallback")

const healthStr string = "health"
const externalCalculatorStr string = "externalcalculator"

func isFallbackEnabled(scaledObject *kedav1alpha1.ScaledObject, metricSpec v2.MetricSpec, determiner string) bool {
	switch determiner {
	case healthStr:
		if scaledObject.Spec.Fallback == nil {
			return false
		}
		if metricSpec.External.Target.Type != v2.AverageValueMetricType {
			log.V(0).Info("Fallback can only be enabled for triggers with metric of type AverageValue", "scaledObject.Namespace", scaledObject.Namespace, "scaledObject.Name", scaledObject.Name)
			return false
		}
	case externalCalculatorStr:
		// check for nil pointer & only then check for fallback
		if scaledObject.Spec.Advanced == nil || scaledObject.Spec.Advanced.ComplexScalingLogic.Fallback == nil {
			return false
		}
	default:
		log.V(0).Info("Internal error in isFallbackEnabled - wrong determiner - this should never happen")
		return false
	}
	return true
}

// TODO: possible refactor of fallback funcionality to unify status updates
func GetMetricsWithFallbackExternalCalculator(ctx context.Context, client runtimeclient.Client, metrics *cl.MetricsList, suppressedError error, metricName string, scaledObject *kedav1alpha1.ScaledObject) (bool, error) {
	const determiner string = "externalcalculator"
	status := scaledObject.Status.DeepCopy()

	initHealthStatus(status, determiner)

	healthStatus := getHealthStatus(status, metricName, determiner)

	if healthStatus == nil {
		// should never be nil
		err := fmt.Errorf("internal error getting health status in GetMetricsWithFallbackExternalCalculator - wrong determiner")
		return false, err
	}

	// if there is no error
	if suppressedError == nil {
		zero := int32(0)
		healthStatus.NumberOfFailures = &zero
		healthStatus.Status = kedav1alpha1.HealthStatusHappy
		status.ExternalCalculationHealth[metricName] = *healthStatus

		updateStatus(ctx, client, scaledObject, status, v2.MetricSpec{})

		return false, nil
	}

	healthStatus.Status = kedav1alpha1.HealthStatusFailing
	*healthStatus.NumberOfFailures++
	status.ExternalCalculationHealth[metricName] = *healthStatus
	updateStatus(ctx, client, scaledObject, status, v2.MetricSpec{})

	switch {
	case !isFallbackEnabled(scaledObject, v2.MetricSpec{}, determiner):
		return false, suppressedError
	case !validateFallback(scaledObject, determiner):
		log.Info("Failed to validate ScaledObject ComplexScalingLogic Fallback. Please check that parameters are positive integers", "scaledObject.Namespace", scaledObject.Namespace, "scaledObject.Name", scaledObject.Name)
		return false, suppressedError
	case *healthStatus.NumberOfFailures > scaledObject.Spec.Advanced.ComplexScalingLogic.Fallback.FailureThreshold:
		doExternalCalculationFallback(scaledObject, metrics, metricName, suppressedError)
		return true, nil

	default:
		return false, suppressedError
	}
}

func GetMetricsWithFallback(ctx context.Context, client runtimeclient.Client, metrics []external_metrics.ExternalMetricValue, suppressedError error, metricName string, scaledObject *kedav1alpha1.ScaledObject, metricSpec v2.MetricSpec) ([]external_metrics.ExternalMetricValue, error) {
	const determiner string = "health"
	status := scaledObject.Status.DeepCopy()

	initHealthStatus(status, determiner)
	healthStatus := getHealthStatus(status, metricName, determiner)
	if healthStatus == nil {
		// should never be nil
		err := fmt.Errorf("internal error getting health status in GetMetricsWithFallback - wrong determiner")
		return metrics, err
	}

	if suppressedError == nil {
		zero := int32(0)
		healthStatus.NumberOfFailures = &zero
		healthStatus.Status = kedav1alpha1.HealthStatusHappy
		status.Health[metricName] = *healthStatus

		updateStatus(ctx, client, scaledObject, status, metricSpec)
		return metrics, nil
	}

	healthStatus.Status = kedav1alpha1.HealthStatusFailing
	*healthStatus.NumberOfFailures++
	status.Health[metricName] = *healthStatus

	updateStatus(ctx, client, scaledObject, status, metricSpec)

	switch {
	case !isFallbackEnabled(scaledObject, metricSpec, determiner):
		return nil, suppressedError
	case !validateFallback(scaledObject, determiner):
		log.Info("Failed to validate ScaledObject Spec. Please check that parameters are positive integers", "scaledObject.Namespace", scaledObject.Namespace, "scaledObject.Name", scaledObject.Name)
		return nil, suppressedError
	case *healthStatus.NumberOfFailures > scaledObject.Spec.Fallback.FailureThreshold:
		return doFallback(scaledObject, metricSpec, metricName, suppressedError), nil
	default:
		return nil, suppressedError
	}
}

func fallbackExistsInScaledObject(scaledObject *kedav1alpha1.ScaledObject, metricSpec v2.MetricSpec, determiner string) bool {
	if !isFallbackEnabled(scaledObject, metricSpec, determiner) || !validateFallback(scaledObject, determiner) {
		return false
	}

	switch determiner {
	case healthStr:
		for _, element := range scaledObject.Status.Health {
			if element.Status == kedav1alpha1.HealthStatusFailing && *element.NumberOfFailures > scaledObject.Spec.Fallback.FailureThreshold {
				return true
			}
		}
	case externalCalculatorStr:
		for _, element := range scaledObject.Status.ExternalCalculationHealth {
			if element.Status == kedav1alpha1.HealthStatusFailing && *element.NumberOfFailures > scaledObject.Spec.Advanced.ComplexScalingLogic.Fallback.FailureThreshold {
				return true
			}
		}
	default:
		// this should never happen
	}

	return false
}

func validateFallback(scaledObject *kedav1alpha1.ScaledObject, determiner string) bool {
	switch determiner {
	case healthStr:
		return scaledObject.Spec.Fallback.FailureThreshold >= 0 &&
			scaledObject.Spec.Fallback.Replicas >= 0
	case externalCalculatorStr:
		return scaledObject.Spec.Advanced.ComplexScalingLogic.Fallback.FailureThreshold >= 0 &&
			scaledObject.Spec.Advanced.ComplexScalingLogic.Fallback.Replicas >= 0
	default:
	}
	return false
}

func doFallback(scaledObject *kedav1alpha1.ScaledObject, metricSpec v2.MetricSpec, metricName string, suppressedError error) []external_metrics.ExternalMetricValue {
	replicas := int64(scaledObject.Spec.Fallback.Replicas)
	normalisationValue := metricSpec.External.Target.AverageValue.AsApproximateFloat64()
	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewMilliQuantity(int64(normalisationValue*1000)*replicas, resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}
	fallbackMetrics := []external_metrics.ExternalMetricValue{metric}

	log.Info("Suppressing error, falling back to fallback.replicas", "scaledObject.Namespace", scaledObject.Namespace, "scaledObject.Name", scaledObject.Name, "suppressedError", suppressedError, "fallback.replicas", replicas)
	return fallbackMetrics
}

func doExternalCalculationFallback(scaledObject *kedav1alpha1.ScaledObject, metrics *cl.MetricsList, metricName string, suppressedError error) {
	replicas := int64(scaledObject.Spec.Advanced.ComplexScalingLogic.Fallback.Replicas)
	normalisationValue, err := strconv.ParseFloat(scaledObject.Spec.Advanced.ComplexScalingLogic.Target, 64)
	if err != nil {
		log.Error(err, "error converting string to float in ExternalCalculation fallback")
		return
	}
	metric := cl.Metric{
		Name:  metricName,
		Value: float32(normalisationValue * float64(replicas)),
	}
	metrics.MetricValues = []*cl.Metric{&metric}
	log.Info(fmt.Sprintf("Suppressing error, externalCalculator falling back to %d fallback.replicas", scaledObject.Spec.Advanced.ComplexScalingLogic.Fallback.Replicas), "scaledObject.Namespace", scaledObject.Namespace, "scaledObject.Name", scaledObject.Name, "suppressedError", suppressedError)
}

func updateStatus(ctx context.Context, client runtimeclient.Client, scaledObject *kedav1alpha1.ScaledObject, status *kedav1alpha1.ScaledObjectStatus, metricSpec v2.MetricSpec) {
	patch := runtimeclient.MergeFrom(scaledObject.DeepCopy())

	// if metricSpec is empty, expect to update externalCalculator
	if reflect.DeepEqual(metricSpec, v2.MetricSpec{}) {
		if fallbackExistsInScaledObject(scaledObject, metricSpec, externalCalculatorStr) {
			status.Conditions.SetExternalFallbackCondition(metav1.ConditionTrue, "ExternalFallbackExists", "At least one external calculator is failing on this scaled object")
		} else {
			status.Conditions.SetExternalFallbackCondition(metav1.ConditionFalse, "NoExternalFallbackFound", "No external fallbacks are active on this scaled object")
		}
	} else {
		if fallbackExistsInScaledObject(scaledObject, metricSpec, healthStr) {
			status.Conditions.SetFallbackCondition(metav1.ConditionTrue, "FallbackExists", "At least one trigger is falling back on this scaled object")
		} else {
			status.Conditions.SetFallbackCondition(metav1.ConditionFalse, "NoFallbackFound", "No fallbacks are active on this scaled object")
		}
	}

	scaledObject.Status = *status
	err := client.Status().Patch(ctx, scaledObject, patch)
	if err != nil {
		log.Error(err, "failed to patch ScaledObjects Status", "scaledObject.Namespace", scaledObject.Namespace, "scaledObject.Name", scaledObject.Name)
	}
}

func getHealthStatus(status *kedav1alpha1.ScaledObjectStatus, metricName string, determiner string) *kedav1alpha1.HealthStatus {
	switch determiner {
	case healthStr:
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
	case externalCalculatorStr:
		// Get health status for a specific metric
		_, healthStatusExists := status.ExternalCalculationHealth[metricName]
		if !healthStatusExists {
			zero := int32(0)
			healthStatus := kedav1alpha1.HealthStatus{
				NumberOfFailures: &zero,
				Status:           kedav1alpha1.HealthStatusHappy,
			}
			status.ExternalCalculationHealth[metricName] = healthStatus
		}
		healthStatus := status.ExternalCalculationHealth[metricName]
		return &healthStatus
	default:
		// if wrong determiner was given
		return nil
	}
}

// Init health status of given structure. Possible determiners are "health" and
// "externalcalculator". These represent (1) default health status for ScaledObjectStatus.Health
// and (2) externalCalculators health status for ScaledObjectStatus.ExternalCalculationHealth
func initHealthStatus(status *kedav1alpha1.ScaledObjectStatus, determiner string) {
	// Init specific health status if missing ("health" for standard; "external" for external calculator health)
	switch determiner {
	case healthStr:
		if status.Health == nil {
			status.Health = make(map[string]kedav1alpha1.HealthStatus)
		}
	case externalCalculatorStr:
		if status.ExternalCalculationHealth == nil {
			status.ExternalCalculationHealth = make(map[string]kedav1alpha1.HealthStatus)
		}
	default:
	}
}
