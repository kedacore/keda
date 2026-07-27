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
	"strconv"
	"sync"

	v2 "k8s.io/api/autoscaling/v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/scale"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"k8s.io/utils/ptr"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scaling/resolver"
)

var log = logf.Log.WithName("fallback")

// ScaledObjectHandler encapsulates ScaledObject with necessary clients and locks for performing fallback operations safely in concurrent code
type ScaledObjectHandler struct {
	Ctx          context.Context
	KubeClient   runtimeclient.Client
	ScaleClient  scale.ScalesGetter
	UpdateLock   *sync.RWMutex
	ScaledObject *kedav1alpha1.ScaledObject
}

// UpdateHealthStatus safely serializes updates to ScaledObject's health status and ships them to kube-api
func (h *ScaledObjectHandler) UpdateHealthStatus(metricName string, healthStatus kedav1alpha1.HealthStatus) {
	h.UpdateLock.Lock()
	defer h.UpdateLock.Unlock()

	original := h.ScaledObject.DeepCopy()
	if h.ScaledObject.Status.Health == nil {
		h.ScaledObject.Status.Health = make(map[string]kedav1alpha1.HealthStatus)
	}
	h.ScaledObject.Status.Health[metricName] = healthStatus
	h.updateStatus(original)
}

// IncrementFailure increments the failure count for a specific metric in the ScaledObject's status and persists the change to kube-api
func (h *ScaledObjectHandler) IncrementFailure(metricName string) kedav1alpha1.HealthStatus {
	h.UpdateLock.Lock()
	defer h.UpdateLock.Unlock()

	original := h.ScaledObject.DeepCopy()
	if h.ScaledObject.Status.Health == nil {
		h.ScaledObject.Status.Health = make(map[string]kedav1alpha1.HealthStatus)
	}
	healthStatus := h.ScaledObject.Status.Health[metricName]
	if healthStatus.NumberOfFailures == nil {
		healthStatus.NumberOfFailures = ptr.To(int32(0))
	}
	*healthStatus.NumberOfFailures++
	healthStatus.Status = kedav1alpha1.HealthStatusFailing
	h.ScaledObject.Status.Health[metricName] = healthStatus
	h.updateStatus(original)
	return healthStatus
}

// updateStatus recomputes the fallback condition and patches the status to kube-api if it changed.
// It must be called with h.UpdateLock held; original is a snapshot taken before the health mutation
// so the merge patch carries both the health and condition diffs.
func (h *ScaledObjectHandler) updateStatus(original *kedav1alpha1.ScaledObject) {
	if !isFallbackEnabled(h.ScaledObject) || !HasValidFallback(h.ScaledObject) {
		log.V(1).Info("Fallback is not enabled, hence skipping the health update to the scaledobject", "scaledObject.Namespace", h.ScaledObject.Namespace, "scaledObject.Name", h.ScaledObject.Name)
		return
	}

	if fallbackExceededThreshold(h.ScaledObject.Status, h.ScaledObject.Spec.Fallback.FailureThreshold) {
		h.ScaledObject.Status.Conditions.SetFallbackCondition(metav1.ConditionTrue, "FallbackExists", "At least one trigger is falling back on this scaled object")
	} else {
		h.ScaledObject.Status.Conditions.SetFallbackCondition(metav1.ConditionFalse, "NoFallbackFound", "No fallbacks are active on this scaled object")
	}

	// Update status only if it has changed
	if equality.Semantic.DeepEqual(original.Status, h.ScaledObject.Status) {
		return
	}

	patch := runtimeclient.MergeFrom(original)
	if err := h.KubeClient.Status().Patch(h.Ctx, h.ScaledObject, patch); err != nil {
		log.Error(err, "failed to patch ScaledObjects Status", "scaledObject.Namespace", h.ScaledObject.Namespace, "scaledObject.Name", h.ScaledObject.Name)
	}
}

func isFallbackEnabled(scaledObject *kedav1alpha1.ScaledObject) bool {
	return scaledObject.Spec.Fallback != nil
}

func GetMetricsWithFallback(soh ScaledObjectHandler, metrics []external_metrics.ExternalMetricValue, suppressedError error, metricName string, metricSpec v2.MetricSpec) ([]external_metrics.ExternalMetricValue, bool, error) {
	soh.UpdateLock.RLock()
	scaledObject := soh.ScaledObject.DeepCopy()
	soh.UpdateLock.RUnlock()

	// Health is only tracked for ScaledObjects with a valid fallback configured. For all others we
	// must leave the status untouched, so the enabled/valid checks have to happen before any mutation.
	fallbackConfigured := isFallbackEnabled(scaledObject) && HasValidFallback(scaledObject)

	if suppressedError == nil {
		if fallbackConfigured {
			soh.UpdateHealthStatus(metricName, kedav1alpha1.HealthStatus{
				NumberOfFailures: ptr.To(int32(0)),
				Status:           kedav1alpha1.HealthStatusHappy,
			})
		}
		return metrics, false, nil
	}

	switch {
	case !isFallbackEnabled(scaledObject):
		return nil, false, suppressedError
	case !HasValidFallback(scaledObject):
		log.Info("Failed to validate ScaledObject Spec. Please check that parameters are positive integers", "scaledObject.Namespace", scaledObject.Namespace, "scaledObject.Name", scaledObject.Name)
		return nil, false, suppressedError
	}

	healthStatus := soh.IncrementFailure(metricName)

	switch {
	case *healthStatus.NumberOfFailures > scaledObject.Spec.Fallback.FailureThreshold:
		if scaledObject.Spec.Fallback.Behavior == kedav1alpha1.FallbackBehaviorScalingModifiers {
			// scaling modifier expression engine will treat this as "nil" for the "??" operator
			placeholderMetric := external_metrics.ExternalMetricValue{
				MetricName: metricName,
				Value:      *resource.NewQuantity(-1, resource.DecimalSI),
				Timestamp:  metav1.Now(),
			}
			return []external_metrics.ExternalMetricValue{placeholderMetric}, false, nil
		}

		var currentReplicas int32
		var err error

		if scaledObject.Spec.Fallback.Behavior != kedav1alpha1.FallbackBehaviorStatic {
			currentReplicas, err = resolver.GetCurrentReplicas(soh.Ctx, soh.KubeClient, soh.ScaleClient, soh.ScaledObject)
			if err != nil {
				return nil, false, suppressedError
			}
		}

		l := doFallback(soh, metricSpec, metricName, currentReplicas, suppressedError)
		if l == nil {
			return l, false, fmt.Errorf("error performing fallback")
		}
		return l, true, nil

	default:
		return nil, false, suppressedError
	}
}

func fallbackExceededThreshold(status kedav1alpha1.ScaledObjectStatus, failureThreshold int32) bool {
	for _, element := range status.Health {
		if element.Status == kedav1alpha1.HealthStatusFailing && *element.NumberOfFailures > failureThreshold {
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
func getReadyReplicasCount(soh ScaledObjectHandler) (int32, error) {
	scaledObject := soh.ScaledObject
	if scaledObject == nil || scaledObject.Spec.ScaleTargetRef == nil || scaledObject.Status.ScaleTargetGVKR == nil {
		return -1, fmt.Errorf("")
	}

	scale, err := soh.ScaleClient.Scales(scaledObject.Namespace).Get(soh.Ctx, scaledObject.Status.ScaleTargetGVKR.GroupResource(), scaledObject.Spec.ScaleTargetRef.Name, metav1.GetOptions{})
	if err != nil {
		return -1, err
	}

	parsedSelector, err := labels.Parse(scale.Status.Selector)
	if err != nil {
		return -1, err
	}

	podList := v1.PodList{}
	err = soh.KubeClient.List(soh.Ctx, &podList, runtimeclient.InNamespace(scaledObject.Namespace), runtimeclient.MatchingLabelsSelector{Selector: parsedSelector})
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

func doFallback(soh ScaledObjectHandler, metricSpec v2.MetricSpec, metricName string, currentReplicas int32, suppressedError error) []external_metrics.ExternalMetricValue {
	scaledObject := soh.ScaledObject
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
	case kedav1alpha1.FallbackBehaviorScalingModifiers:
		return nil
	default:
		replicas = float64(fallbackReplicas)
	}

	// If the metricType is Value, we get the number of readyReplicas, and divide replicas by it.
	if (!scaledObject.IsUsingModifiers() && metricSpec.External.Target.Type == v2.ValueMetricType) ||
		(scaledObject.IsUsingModifiers() && scaledObject.Spec.Advanced.ScalingModifiers.MetricType == v2.ValueMetricType) {
		readyReplicas, err := getReadyReplicasCount(soh)
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
