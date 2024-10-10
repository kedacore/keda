package scalers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	v2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsv1beta1 "k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type cpuMemoryScaler struct {
	metadata         *cpuMemoryMetadata
	resourceName     corev1.ResourceName
	logger           logr.Logger
	kubeClient       client.Client
	podMetricsClient metricsv1beta1.PodMetricsesGetter
}

const (
	cpuMetricName          = "cpu"
	memoryMetricName       = "memory"
	defaultActivationValue = "0"
	scaledObjectType       = "ScaledObject"
)

type cpuMemoryMetadata struct {
	Type                         string `keda:"name=type,            order=triggerMetadata, enum=Utilization;AverageValue, optional"`
	Value                        string `keda:"name=value,           order=triggerMetadata"`
	ActivationValue              string `keda:"name=activationValue, order=triggerMetadata, optional"`
	ContainerName                string `keda:"name=containerName,   order=triggerMetadata, optional"`
	MetricType                   v2.MetricTargetType
	AverageValue                 *resource.Quantity
	AverageUtilization           *int32
	ActivationAverageValue       *resource.Quantity
	ActivationAverageUtilization *int32
	ScalableObjectType           string
	Namespace                    string
	ScaleTargetGVK               schema.GroupVersionKind
	ScaleTargetNamespacedName    types.NamespacedName
}

// NewCPUMemoryScaler creates a new cpuMemoryScaler
func NewCPUMemoryScaler(resourceName corev1.ResourceName, config *scalersconfig.ScalerConfig, kubeClient client.Client, podMetricsClient metricsv1beta1.PodMetricsesGetter) (Scaler, error) {
	logger := InitializeLogger(config, "cpu_memory_scaler")

	meta, err := parseResourceMetadata(config, logger, kubeClient)
	if err != nil {
		return nil, fmt.Errorf("error parsing %s metadata: %w", resourceName, err)
	}

	return &cpuMemoryScaler{
		metadata:         meta,
		resourceName:     resourceName,
		logger:           logger,
		kubeClient:       kubeClient,
		podMetricsClient: podMetricsClient,
	}, nil
}

// getScaleTarget retrieves the GVK and its namespaced name of the scale target (e.g., Deployment, StatefulSet)
// from a ScaledObject. It returns an error if the ScaledObject is not found or if it doesn't
// have a scale target reference.
func getScaleTarget(scalableObjectName, scalableObjectNamespace, scalableObjectType string, kubeClient client.Client) (schema.GroupVersionKind, types.NamespacedName, error) {
	if scalableObjectType != scaledObjectType {
		return schema.GroupVersionKind{}, types.NamespacedName{}, fmt.Errorf("unsupported scalable object type: %s", scalableObjectType)
	}

	scaledObject := &kedav1alpha1.ScaledObject{}
	err := kubeClient.Get(context.Background(), types.NamespacedName{
		Name:      scalableObjectName,
		Namespace: scalableObjectNamespace,
	}, scaledObject)

	if err != nil {
		return schema.GroupVersionKind{}, types.NamespacedName{}, err
	}

	if scaledObject.Spec.ScaleTargetRef == nil {
		return schema.GroupVersionKind{}, types.NamespacedName{}, fmt.Errorf("scaled object %s has no scale target ref", scalableObjectName)
	}

	gvk := scaledObject.Status.ScaleTargetGVKR.GroupVersionKind()
	namespacedName := types.NamespacedName{
		Name:      scaledObject.Spec.ScaleTargetRef.Name,
		Namespace: scaledObject.Namespace,
	}

	return gvk, namespacedName, nil
}

func parseResourceMetadata(config *scalersconfig.ScalerConfig, logger logr.Logger, kubeClient client.Client) (*cpuMemoryMetadata, error) {
	meta := &cpuMemoryMetadata{
		ActivationValue: defaultActivationValue,
	}

	err := config.TypedConfig(meta)
	if err != nil {
		return meta, err
	}

	if config.MetricType != "" {
		meta.MetricType = config.MetricType
	}

	// This is deprecated and can be removed later
	if meta.Type != "" {
		logger.Info("The 'type' setting is DEPRECATED and will be removed in v2.18 - Use 'metricType' instead.")
		switch meta.Type {
		case "AverageValue":
			meta.MetricType = v2.AverageValueMetricType
		case "Utilization":
			meta.MetricType = v2.UtilizationMetricType
		default:
			return meta, fmt.Errorf("unknown metric type: %s, allowed values are 'Utilization' or 'AverageValue'", meta.Type)
		}
	}

	switch meta.MetricType {
	case v2.AverageValueMetricType:
		averageValueQuantity := resource.MustParse(meta.Value)
		meta.AverageValue = &averageValueQuantity

		activationValueQuantity := resource.MustParse(meta.ActivationValue)
		meta.ActivationAverageValue = &activationValueQuantity
	case v2.UtilizationMetricType:
		utilizationNum, err := parseUtilization(meta.Value)
		if err != nil {
			return meta, err
		}
		meta.AverageUtilization = utilizationNum

		utilizationNum, err = parseUtilization(meta.ActivationValue)
		if err != nil {
			return meta, err
		}
		meta.ActivationAverageUtilization = utilizationNum
	default:
		return meta, fmt.Errorf("unknown metric type: %s, allowed values are 'Utilization' or 'AverageValue'", string(meta.MetricType))
	}

	if config.ScalableObjectType == "ScaledObject" {
		gvk, namespacedName, err := getScaleTarget(config.ScalableObjectName, config.ScalableObjectNamespace, config.ScalableObjectType, kubeClient)
		if err != nil {
			return nil, err
		}

		meta.ScaleTargetGVK = gvk
		meta.ScaleTargetNamespacedName = namespacedName
	}

	meta.ScalableObjectType = config.ScalableObjectType
	meta.Namespace = config.ScalableObjectNamespace

	return meta, nil
}

// parseUtilization parses the utilization value string and returns it as an int32
func parseUtilization(value string) (*int32, error) {
	valueNum, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return nil, err
	}
	utilizationNum := int32(valueNum)
	return &utilizationNum, nil
}

// Close no need for cpuMemory scaler
func (s *cpuMemoryScaler) Close(context.Context) error {
	return nil
}

// GetMetricSpecForScaling returns the metric spec for the HPA
func (s *cpuMemoryScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	metricType := s.metadata.MetricType

	var metricSpec v2.MetricSpec
	if s.metadata.ContainerName != "" {
		containerCPUMemoryMetric := &v2.ContainerResourceMetricSource{
			Name: s.resourceName,
			Target: v2.MetricTarget{
				Type:               metricType,
				AverageUtilization: s.metadata.AverageUtilization,
				AverageValue:       s.metadata.AverageValue,
			},
			Container: s.metadata.ContainerName,
		}
		metricSpec = v2.MetricSpec{ContainerResource: containerCPUMemoryMetric, Type: v2.ContainerResourceMetricSourceType}
	} else {
		cpuMemoryMetric := &v2.ResourceMetricSource{
			Name: s.resourceName,
			Target: v2.MetricTarget{
				Type:               metricType,
				AverageUtilization: s.metadata.AverageUtilization,
				AverageValue:       s.metadata.AverageValue,
			},
		}
		metricSpec = v2.MetricSpec{Resource: cpuMemoryMetric, Type: v2.ResourceMetricSourceType}
	}

	return []v2.MetricSpec{metricSpec}
}

// calculateAverage computes the average value of for the given total resource quantity and count
func calculateAverage(total *resource.Quantity, count int64) *resource.Quantity {
	if count == 0 {
		return &resource.Quantity{}
	}

	// Convert the total to milli-units
	nanoValue := total.ScaledValue(resource.Nano)

	// Perform the division
	averageNanoValue := nanoValue / count

	// Create a new Quantity from the average milli-value
	return resource.NewScaledQuantity(averageNanoValue, resource.Nano)
}

// getAverageValue calculates the average resource (CPU/Memory) usage across all running pods
// that match the scaler's target
func (s *cpuMemoryScaler) getAverageValue(ctx context.Context, metricName string) (*resource.Quantity, error) {
	podList, labelSelector, err := s.getPodList(ctx)
	if err != nil {
		return nil, err
	}

	if podList == nil || labelSelector == nil {
		var q resource.Quantity
		return &q, nil
	}

	podMetricsList, err := s.getPodMetricsList(ctx, labelSelector)
	if err != nil {
		return nil, err
	}

	totalValue := &resource.Quantity{}
	podCount := 0

	for _, pod := range podList.Items {
		if pod.Status.Phase != corev1.PodRunning {
			continue
		}

		podMetrics := getPodMetrics(podMetricsList, pod.Name)
		if podMetrics == nil {
			continue
		}

		var metricValue *resource.Quantity
		if s.metadata.ContainerName != "" {
			containerMetrics := getContainerMetrics(podMetrics, s.metadata.ContainerName)
			if containerMetrics == nil {
				continue
			}
			metricValue = getResourceValue(containerMetrics, metricName)
		} else {
			metricValue = getPodResourceValue(podMetrics, metricName)
		}

		if metricValue == nil {
			return nil, fmt.Errorf("unsupported metric name: %s", metricName)
		}

		totalValue.Add(*metricValue)
		podCount++
	}

	if podCount == 0 {
		return nil, fmt.Errorf("no running pods found")
	}

	averageValue := calculateAverage(totalValue, int64(podCount))
	return averageValue, nil
}

// getAverageUtilization calculates the average resource utilization (as a percentage) across
// all running pods that match the scaler's target
func (s *cpuMemoryScaler) getAverageUtilization(ctx context.Context, metricName string) (*int32, error) {
	podList, labelSelector, err := s.getPodList(ctx)
	if err != nil {
		return nil, err
	}

	if podList == nil || labelSelector == nil {
		var p int32
		return &p, nil
	}

	podMetricsList, err := s.getPodMetricsList(ctx, labelSelector)
	if err != nil {
		return nil, err
	}

	var totalUtilization int64
	podCount := 0

	for _, pod := range podList.Items {
		if pod.Status.Phase != corev1.PodRunning {
			continue
		}

		podMetrics := getPodMetrics(podMetricsList, pod.Name)
		if podMetrics == nil {
			continue
		}

		var metricValue, capacity int64
		if s.metadata.ContainerName != "" {
			containerMetrics := getContainerMetrics(podMetrics, s.metadata.ContainerName)
			if containerMetrics == nil {
				continue
			}
			metricValue = getResourceValueInMillis(containerMetrics, metricName)
			capacity = getContainerResourceCapacity(&pod, s.metadata.ContainerName, getResourceName(metricName))
		} else {
			metricValue = getPodResourceValueInMillis(podMetrics, metricName)
			capacity = getPodResourceCapacity(&pod, getResourceName(metricName))
		}

		if capacity == 0 {
			continue
		}

		utilization := (metricValue * 100) / capacity
		totalUtilization += utilization
		podCount++
	}

	if podCount == 0 {
		return nil, fmt.Errorf("no running pods found with non-zero capacity")
	}

	averageUtilization := int32(totalUtilization / int64(podCount))
	return &averageUtilization, nil
}

// getResourceValue extracts the appropriate resource (CPU/Memory) value from container metrics
func getResourceValue(containerMetrics *v1beta1.ContainerMetrics, metricName string) *resource.Quantity {
	switch metricName {
	case cpuMetricName:
		return containerMetrics.Usage.Cpu()
	case memoryMetricName:
		return containerMetrics.Usage.Memory()
	default:
		return nil
	}
}

// getPodResourceValue calculates the total resource usage for a pod by summing
// all of its containers' resource usage
func getPodResourceValue(podMetrics *v1beta1.PodMetrics, metricName string) *resource.Quantity {
	var total resource.Quantity
	for _, container := range podMetrics.Containers {
		if value := getResourceValue(&container, metricName); value != nil {
			total.Add(*value)
		}
	}
	return &total
}

// getResourceValueInMillis returns the resource value in milli-units for CPU
// or bytes for memory
func getResourceValueInMillis(containerMetrics *v1beta1.ContainerMetrics, metricName string) int64 {
	switch metricName {
	case cpuMetricName:
		return containerMetrics.Usage.Cpu().MilliValue()
	case memoryMetricName:
		return containerMetrics.Usage.Memory().Value()
	default:
		return 0
	}
}

// getPodResourceCapacity calculates the total resource capacity (requests) for a pod
// by summing the requests of all its containers
func getPodResourceValueInMillis(podMetrics *v1beta1.PodMetrics, metricName string) int64 {
	var total int64
	for _, container := range podMetrics.Containers {
		total += getResourceValueInMillis(&container, metricName)
	}
	return total
}

func getResourceName(metricName string) corev1.ResourceName {
	switch metricName {
	case cpuMetricName:
		return corev1.ResourceCPU
	case memoryMetricName:
		return corev1.ResourceMemory
	default:
		return ""
	}
}

// getPodResourceCapacity calculates the total resource capacity (requests) for a pod
// by summing the requests of all its containers
func getPodResourceCapacity(pod *corev1.Pod, resourceName corev1.ResourceName) int64 {
	var total int64
	for _, container := range pod.Spec.Containers {
		if quantity, ok := container.Resources.Requests[resourceName]; ok {
			total += quantity.MilliValue()
		}
	}
	return total
}

// getPodList retrieves the list of pods matching the scale target's selector
func (s *cpuMemoryScaler) getPodList(ctx context.Context) (*corev1.PodList, labels.Selector, error) {
	var labelSelector labels.Selector

	switch {
	case s.metadata.ScaleTargetGVK.Group == "apps" && s.metadata.ScaleTargetGVK.Kind == "Deployment":
		deployment := &appsv1.Deployment{}
		err := s.kubeClient.Get(ctx, s.metadata.ScaleTargetNamespacedName, deployment)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get deployment: %w", err)
		}

		labelSelector = labels.SelectorFromSet(deployment.Spec.Selector.MatchLabels)
	case s.metadata.ScaleTargetGVK.Group == "apps" && s.metadata.ScaleTargetGVK.Kind == "StatefulSet":
		statefulSet := &appsv1.StatefulSet{}
		err := s.kubeClient.Get(ctx, s.metadata.ScaleTargetNamespacedName, statefulSet)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get statefulset: %w", err)
		}

		labelSelector = labels.SelectorFromSet(statefulSet.Spec.Selector.MatchLabels)
	default:
		return nil, nil, nil
	}

	podList := &corev1.PodList{}
	err := s.kubeClient.List(ctx, podList, &client.ListOptions{
		Namespace:     s.metadata.Namespace,
		LabelSelector: labelSelector,
	})

	if err != nil {
		return nil, nil, fmt.Errorf("failed to list pods: %w", err)
	}

	return podList, labelSelector, nil
}

// getPodMetricsList retrieves metrics for pods matching the given label selector
// using the pod metrics client
func (s *cpuMemoryScaler) getPodMetricsList(ctx context.Context, labelSelector labels.Selector) (*v1beta1.PodMetricsList, error) {
	podsMetricsList, err := s.podMetricsClient.PodMetricses(s.metadata.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector.String(),
	})

	return podsMetricsList, err
}

func getPodMetrics(podMetricsList *v1beta1.PodMetricsList, podName string) *v1beta1.PodMetrics {
	for _, podMetrics := range podMetricsList.Items {
		if podMetrics.Name == podName {
			return &podMetrics
		}
	}
	return nil
}

func getContainerMetrics(podMetrics *v1beta1.PodMetrics, containerName string) *v1beta1.ContainerMetrics {
	for _, containerMetrics := range podMetrics.Containers {
		if containerMetrics.Name == containerName {
			return &containerMetrics
		}
	}
	return nil
}

// getContainerResourceCapacity retrieves the resource capacity (requests) for a specific container within a pod
func getContainerResourceCapacity(pod *corev1.Pod, containerName string, resourceName corev1.ResourceName) int64 {
	for _, container := range pod.Spec.Containers {
		if container.Name == containerName {
			if quantity, ok := container.Resources.Requests[resourceName]; ok {
				return quantity.MilliValue()
			}
		}
	}
	return 0
}

// GetMetricsAndActivity only returns the activity of the cpu/memory scaler
func (s *cpuMemoryScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	if s.metadata.ScalableObjectType == "ScaledJob" {
		return nil, false, nil
	}

	switch s.metadata.MetricType {
	case v2.AverageValueMetricType:
		averageValue, err := s.getAverageValue(ctx, metricName)
		if err != nil {
			return nil, false, err
		}

		return nil, averageValue.Cmp(*s.metadata.ActivationAverageValue) == 1, nil
	case v2.UtilizationMetricType:
		averageUtilization, err := s.getAverageUtilization(ctx, metricName)
		if err != nil {
			return nil, false, err
		}

		return nil, *averageUtilization > *s.metadata.ActivationAverageUtilization, nil
	}

	return nil, false, fmt.Errorf("no matching resource metric found for %s", s.resourceName)
}
