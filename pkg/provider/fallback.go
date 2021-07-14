package provider

import (
	"context"
	"fmt"

	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers"
	"k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

func isFallbackEnabled(scaledObject *kedav1alpha1.ScaledObject, metricSpec v2beta2.MetricSpec) bool {
	return scaledObject.Spec.Fallback != nil && metricSpec.External.Target.Type == v2beta2.AverageValueMetricType
}

func (p *KedaProvider) getMetricsWithFallback(scaler scalers.Scaler, metricName string, metricSelector labels.Selector, scaledObject *kedav1alpha1.ScaledObject, metricSpec v2beta2.MetricSpec) ([]external_metrics.ExternalMetricValue, error) {
	initHealthStatus(scaledObject)
	metrics, err := scaler.GetMetrics(context.TODO(), metricName, metricSelector)
	healthStatus := getHealthStatus(scaledObject, metricName)

	if err == nil {
		zero := int32(0)
		healthStatus.NumberOfFailures = &zero
		healthStatus.Status = kedav1alpha1.HealthStatusHappy
		scaledObject.Status.Health[metricName] = *healthStatus

		p.updateStatus(scaledObject, metricSpec)
		return metrics, nil
	}

	healthStatus.Status = kedav1alpha1.HealthStatusFailing
	*healthStatus.NumberOfFailures++
	scaledObject.Status.Health[metricName] = *healthStatus

	p.updateStatus(scaledObject, metricSpec)

	switch {
	case !isFallbackEnabled(scaledObject, metricSpec):
		return nil, err
	case !validateFallback(scaledObject):
		logger.Info("Failed to validate ScaledObject Spec. Please check that parameters are positive integers")
		return nil, err
	case *healthStatus.NumberOfFailures > scaledObject.Spec.Fallback.FailureThreshold:
		return doFallback(scaledObject, metricSpec, metricName, err), nil
	default:
		return nil, err
	}
}

func fallbackExistsInScaledObject(scaledObject *kedav1alpha1.ScaledObject, metricSpec v2beta2.MetricSpec) bool {
	if !isFallbackEnabled(scaledObject, metricSpec) || !validateFallback(scaledObject) {
		return false
	}

	for _, element := range scaledObject.Status.Health {
		if element.Status == kedav1alpha1.HealthStatusFailing && *element.NumberOfFailures > scaledObject.Spec.Fallback.FailureThreshold {
			return true
		}
	}

	return false
}

func validateFallback(scaledObject *kedav1alpha1.ScaledObject) bool {
	return scaledObject.Spec.Fallback.FailureThreshold >= 0 &&
		scaledObject.Spec.Fallback.Replicas >= 0
}

func doFallback(scaledObject *kedav1alpha1.ScaledObject, metricSpec v2beta2.MetricSpec, metricName string, suppressedError error) []external_metrics.ExternalMetricValue {
	replicas := int64(scaledObject.Spec.Fallback.Replicas)
	normalisationValue, _ := metricSpec.External.Target.AverageValue.AsInt64()
	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(normalisationValue*replicas, resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}
	fallbackMetrics := []external_metrics.ExternalMetricValue{metric}

	logger.Info(fmt.Sprintf("Suppressing error %s, falling back to %d replicas", suppressedError, replicas))
	return fallbackMetrics
}

func (p *KedaProvider) updateStatus(scaledObject *kedav1alpha1.ScaledObject, metricSpec v2beta2.MetricSpec) {
	if fallbackExistsInScaledObject(scaledObject, metricSpec) {
		scaledObject.Status.Conditions.SetFallbackCondition(metav1.ConditionTrue, "FallbackExists", "At least one trigger is falling back on this scaled object")
	} else {
		scaledObject.Status.Conditions.SetFallbackCondition(metav1.ConditionFalse, "NoFallbackFound", "No fallbacks are active on this scaled object")
	}

	err := p.client.Status().Update(context.TODO(), scaledObject)
	if err != nil {
		logger.Error(err, "Error updating ScaledObject status", "Error")
	}
}

func getHealthStatus(scaledObject *kedav1alpha1.ScaledObject, metricName string) *kedav1alpha1.HealthStatus {
	// Get health status for a specific metric
	_, healthStatusExists := scaledObject.Status.Health[metricName]
	if !healthStatusExists {
		zero := int32(0)
		status := kedav1alpha1.HealthStatus{
			NumberOfFailures: &zero,
			Status:           kedav1alpha1.HealthStatusHappy,
		}
		scaledObject.Status.Health[metricName] = status
	}
	healthStatus := scaledObject.Status.Health[metricName]
	return &healthStatus
}

func initHealthStatus(scaledObject *kedav1alpha1.ScaledObject) {
	// Init health status if missing
	if scaledObject.Status.Health == nil {
		scaledObject.Status.Health = make(map[string]kedav1alpha1.HealthStatus)
	}
}
