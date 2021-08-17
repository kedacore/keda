package provider

import (
	"context"
	"fmt"

	"k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers"
)

func isFallbackEnabled(scaledObject *kedav1alpha1.ScaledObject, metricSpec v2beta2.MetricSpec) bool {
	return scaledObject.Spec.Fallback != nil && metricSpec.External.Target.Type == v2beta2.AverageValueMetricType
}

func (p *KedaProvider) getMetricsWithFallback(scaler scalers.Scaler, metricName string, metricSelector labels.Selector, scaledObject *kedav1alpha1.ScaledObject, metricSpec v2beta2.MetricSpec) ([]external_metrics.ExternalMetricValue, error) {
	status := scaledObject.Status.DeepCopy()

	initHealthStatus(status)
	metrics, err := scaler.GetMetrics(context.TODO(), metricName, metricSelector)
	healthStatus := getHealthStatus(status, metricName)

	if err == nil {
		zero := int32(0)
		healthStatus.NumberOfFailures = &zero
		healthStatus.Status = kedav1alpha1.HealthStatusHappy
		status.Health[metricName] = *healthStatus

		p.updateStatus(scaledObject, status, metricSpec)
		return metrics, nil
	}

	healthStatus.Status = kedav1alpha1.HealthStatusFailing
	*healthStatus.NumberOfFailures++
	status.Health[metricName] = *healthStatus

	p.updateStatus(scaledObject, status, metricSpec)

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

func (p *KedaProvider) updateStatus(scaledObject *kedav1alpha1.ScaledObject, status *kedav1alpha1.ScaledObjectStatus, metricSpec v2beta2.MetricSpec) {
	patch := runtimeclient.MergeFrom(scaledObject.DeepCopy())

	if fallbackExistsInScaledObject(scaledObject, metricSpec) {
		status.Conditions.SetFallbackCondition(metav1.ConditionTrue, "FallbackExists", "At least one trigger is falling back on this scaled object")
	} else {
		status.Conditions.SetFallbackCondition(metav1.ConditionFalse, "NoFallbackFound", "No fallbacks are active on this scaled object")
	}

	scaledObject.Status = *status
	err := p.client.Status().Patch(context.TODO(), scaledObject, patch)
	if err != nil {
		logger.Error(err, "Failed to patch ScaledObjects Status")
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
