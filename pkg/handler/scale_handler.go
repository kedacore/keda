package handler

import (
	"context"
	"fmt"
	"time"

	kore_v1alpha1 "github.com/Azure/Kore/pkg/apis/kore/v1alpha1"
	clientset "github.com/Azure/Kore/pkg/client/clientset/versioned"
	"github.com/Azure/Kore/pkg/scalers"
	log "github.com/Sirupsen/logrus"
	apps_v1 "k8s.io/api/apps/v1"
	v2beta1 "k8s.io/api/autoscaling/v2beta1"
	core_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

// ScaleHandler encapsulates the logic of calling the right scalers for
// each ScaledObject and making the final scale decision and operation
type ScaleHandler struct {
	koreClient clientset.Interface
	kubeClient kubernetes.Interface
}

const (
	defaultPollingInterval = 30
)

// NewScaleHandler creates a ScaleHandler object
func NewScaleHandler(koreClient clientset.Interface, kubeClient kubernetes.Interface) *ScaleHandler {
	handler := &ScaleHandler{
		koreClient: koreClient,
		kubeClient: kubeClient,
	}

	return handler
}

// WatchScaledObjectWithContext enqueues the ScaledObject into the work queue
func (h *ScaleHandler) WatchScaledObjectWithContext(ctx context.Context, scaledObject *kore_v1alpha1.ScaledObject) {
	h.createHPAForNewScaledObject(ctx, scaledObject)
	go h.handleScaleLoop(ctx, scaledObject)
}

// HandleScaledObjectDelete handles any cleanup when a scaled object is deleted
func (h *ScaleHandler) HandleScaledObjectDelete(ctx context.Context, scaledObject *kore_v1alpha1.ScaledObject) {
	h.deleteHPAForScaledObject(scaledObject)
}

// GetScaledObjectMetrics is used by the  metric adapter in provider.go to get the value for a metric for a scaled object
func (h *ScaleHandler) GetScaledObjectMetrics(namespace string, metricSelector labels.Selector, merticName string) ([]external_metrics.ExternalMetricValue, error) {
	// get the scaled objects matching namespace and labels
	log.Debugf("Getting metrics for namespace %s MetricName %s Metric Selector %s", namespace, merticName, metricSelector.String())
	scaledObjectQuerier := h.koreClient.KoreV1alpha1().ScaledObjects(namespace)
	scaledObjects, error := scaledObjectQuerier.List(meta_v1.ListOptions{LabelSelector: metricSelector.String()})
	if error != nil {
		return nil, error
	}

	matchingMetrics := []external_metrics.ExternalMetricValue{}
	metricFunc := func(ctx context.Context, scaler scalers.Scaler, deployment *apps_v1.Deployment) {
		metrics, err := scaler.GetMetrics(context.TODO(), merticName, metricSelector)
		if err != nil {
			log.Errorf("error getting metric for scaler : %s", err)
		} else {
			matchingMetrics = append(matchingMetrics, metrics...)
		}
	}

	for _, scaledObject := range scaledObjects.Items {
		h.executeCallbackForEachScaler(nil, &scaledObject, metricFunc)
	}

	return matchingMetrics, nil
}

func (h *ScaleHandler) deleteHPAForScaledObject(scaledObject *kore_v1alpha1.ScaledObject) {
	deploymentName := scaledObject.Spec.ScaleTargetRef.DeploymentName
	if deploymentName == "" {
		log.Errorf("Notified about ScaledObject with missing deployment name: %s", scaledObject.GetName())
		return
	}
	scaledObjectNamespace := scaledObject.GetNamespace()
	hpaName := "kore-hpa-" + deploymentName
	deleteOptions := &meta_v1.DeleteOptions{}
	err := h.kubeClient.AutoscalingV2beta1().HorizontalPodAutoscalers(scaledObjectNamespace).Delete(hpaName, deleteOptions)
	if errors.IsNotFound(err) {
		log.Warnf("HPA with namespace %s and name %s is not found", scaledObjectNamespace, hpaName)

	} else if err != nil {
		log.Errorf("Error deleting HPA with namespace %s and name %s : %s\n", scaledObjectNamespace, hpaName, err)
	} else {
		log.Debugf("Deleted HPA with namespace %s and name %s", scaledObjectNamespace, hpaName)
	}
}

func (h *ScaleHandler) createHPAForNewScaledObject(ctx context.Context, scaledObject *kore_v1alpha1.ScaledObject) {
	deploymentName := scaledObject.Spec.ScaleTargetRef.DeploymentName
	if deploymentName == "" {
		log.Errorf("Notified about ScaledObject with missing deployment name: %s", scaledObject.GetName())
		return
	}

	var scaledObjectMetricSpecs []v2beta1.MetricSpec
	metricSpecFunc := func(ctx context.Context, scaler scalers.Scaler, deployment *apps_v1.Deployment) {
		metricSpecs := scaler.GetMetricSpecForScaling()

		// add the deploymentName label. This is how the MetricsAdapter will know which scaledobject a metric is for when the HPA queries it.
		for _, metricSpec := range metricSpecs {
			metricSpec.External.MetricSelector = &meta_v1.LabelSelector{MatchLabels: make(map[string]string)}
			metricSpec.External.MetricSelector.MatchLabels["deploymentName"] = deploymentName
		}
		scaledObjectMetricSpecs = append(scaledObjectMetricSpecs, metricSpecs...)
	}

	h.executeCallbackForEachScaler(ctx, scaledObject, metricSpecFunc)
	kvd := &v2beta1.CrossVersionObjectReference{Name: deploymentName, Kind: "Deployment", APIVersion: "apps/v1"}
	var minReplicas int32 = 1
	var maxReplicas int32 = 100
	scaledObjectNamespace := scaledObject.GetNamespace()
	hpaName := "kore-hpa-" + deploymentName
	newHPASpec := &v2beta1.HorizontalPodAutoscalerSpec{MinReplicas: &minReplicas, MaxReplicas: maxReplicas, Metrics: scaledObjectMetricSpecs, ScaleTargetRef: *kvd}
	objectSpec := &meta_v1.ObjectMeta{Name: hpaName, Namespace: scaledObjectNamespace}
	newHPA := &v2beta1.HorizontalPodAutoscaler{Spec: *newHPASpec, ObjectMeta: *objectSpec}
	newHPA, err := h.kubeClient.AutoscalingV2beta1().HorizontalPodAutoscalers(scaledObjectNamespace).Create(newHPA)
	if errors.IsAlreadyExists(err) {
		log.Warnf("HPA with namespace %s and name %s already exists", scaledObjectNamespace, hpaName)
	} else if err != nil {
		log.Errorf("Error creating HPA with namespace %s and name %s : %s\n", scaledObjectNamespace, hpaName, err)
	} else {
		log.Debugf("Created HPA with namespace %s and name %s", scaledObjectNamespace, hpaName)
	}
}

func (h *ScaleHandler) handleScaleLoop(ctx context.Context, scaledObject *kore_v1alpha1.ScaledObject) {
	h.handleScale(ctx, scaledObject)

	pollingInterval := time.Second * time.Duration(defaultPollingInterval)

	if scaledObject.Spec.PollingInterval != nil {
		pollingInterval = time.Second * time.Duration(*scaledObject.Spec.PollingInterval)
	}

	log.Debugf("watching scaledObject (%s/%s) with pollingInterval: %d", scaledObject.GetNamespace(), scaledObject.GetName(), pollingInterval)

	for {
		select {
		case <-time.After(pollingInterval):
			h.handleScale(ctx, scaledObject)
		case <-ctx.Done():
			log.Debugf("context for scaledObject (%s/%s) canceled", scaledObject.GetNamespace(), scaledObject.GetName())
			return
		}
	}
}

func (h *ScaleHandler) handleScale(ctx context.Context, scaledObject *kore_v1alpha1.ScaledObject) {

	var scaleDecision int32
	var scaledObjectDeployment *apps_v1.Deployment
	scaleDecisionFunc := func(ctx context.Context, scaler scalers.Scaler, deployment *apps_v1.Deployment) {
		sd, err := scaler.GetScaleDecision(ctx)
		if err != nil {
			log.Errorf("error getting scale decision: %s", err)
			return
		}
		scaledObjectDeployment = deployment
		scaleDecision += sd
	}

	h.executeCallbackForEachScaler(ctx, scaledObject, scaleDecisionFunc)
	log.Debugf("scaledObject: %s, target deployment: %s, scale decision: %d", scaledObject.GetName(), scaledObjectDeployment.GetName(), scaleDecision)
	h.scaleDeployment(scaledObjectDeployment, scaleDecision)
	return
}

func (h *ScaleHandler) scaleDeployment(deployment *apps_v1.Deployment, scaleDecision int32) {
	if *deployment.Spec.Replicas != scaleDecision {
		currentReplicas := *deployment.Spec.Replicas
		// skip setting replica count as 1 if HPA has already scaled the replica count to >1
		if scaleDecision != 0 && currentReplicas > 1 {
			log.Debugf("Calculated replica count of 1 for deployment (%s/%s) is lesser than the current replica count %d. Skipping..",
				deployment.GetNamespace(),
				deployment.GetName(),
				currentReplicas)
			return
		}
		*deployment.Spec.Replicas = scaleDecision
		deployment, err := h.kubeClient.AppsV1().Deployments(deployment.GetNamespace()).Update(deployment)
		if err != nil {
			log.Errorf("Error updating replica count on deployment (%s/%s) from %d to %d. Error: %s",
				deployment.GetNamespace(),
				deployment.GetName(),
				currentReplicas,
				*deployment.Spec.Replicas,
				err)
		} else {
			log.Debugf("Successfully updated deployment (%s/%s) from %d to %d replicas",
				deployment.GetNamespace(),
				deployment.GetName(),
				currentReplicas,
				*deployment.Spec.Replicas)
		}
	} else {
		log.Debugf("Current replica count for deployment (%s/%s) is the same as update replica count. Skipping..",
			deployment.GetNamespace(),
			deployment.GetName())
	}
}
func (h *ScaleHandler) resolveSecrets(deployment *apps_v1.Deployment) (map[string]string, error) {
	deploymentKey, err := cache.MetaNamespaceKeyFunc(deployment)
	if err != nil {
		return nil, err
	}

	if len(deployment.Spec.Template.Spec.Containers) < 1 {
		return nil, fmt.Errorf("Deployment (%s) doesn't have containers", deploymentKey)
	} else if len(deployment.Spec.Template.Spec.Containers) > 1 {
		return nil, fmt.Errorf("Deployment (%s) has more than one container", deploymentKey)
	}

	container := deployment.Spec.Template.Spec.Containers[0]
	resolved := make(map[string]string)
	for _, envVar := range container.Env {
		if envVar.Value != "" {
			resolved[envVar.Name] = envVar.Value
		} else if envVar.ValueFrom != nil && envVar.ValueFrom.SecretKeyRef != nil {
			value, err := h.resolveSecretValue(envVar.ValueFrom.SecretKeyRef, envVar.Name, deployment.GetNamespace())
			if err != nil {
				return nil, err
			}

			resolved[envVar.Name] = value
		}
	}

	return resolved, nil
}

func (h *ScaleHandler) resolveSecretValue(secretKeyRef *core_v1.SecretKeySelector, keyName, namespace string) (string, error) {
	secretCollection, err := h.kubeClient.CoreV1().Secrets(namespace).Get(secretKeyRef.Name, meta_v1.GetOptions{})

	if err != nil {
		return "", err
	}

	return string(secretCollection.Data[keyName]), nil
}

func (h *ScaleHandler) executeCallbackForEachScaler(ctx context.Context, scaledObject *kore_v1alpha1.ScaledObject, callback func(context.Context, scalers.Scaler, *apps_v1.Deployment)) {
	deploymentName := scaledObject.Spec.ScaleTargetRef.DeploymentName
	if deploymentName == "" {
		log.Errorf("Notified about ScaledObject with missing deployment name: %s", scaledObject.GetName())
		return
	}

	deployment, err := h.kubeClient.AppsV1().Deployments(scaledObject.GetNamespace()).Get(deploymentName, meta_v1.GetOptions{})
	if err != nil {
		log.Errorf("Error getting deployment: %s", err)
		return
	}

	resolvedSecrets, err := h.resolveSecrets(deployment)
	if err != nil {
		log.Errorf("Error resolving secrets for deployment: %s", err)
		return
	}

	for i, trigger := range scaledObject.Spec.Triggers {
		scaler, err := getScaler(trigger, resolvedSecrets)
		if err != nil {
			log.Errorf("error for trigger #%d: %s", i, err)
			continue
		}

		callback(ctx, scaler, deployment)
	}
}

func getScaler(trigger kore_v1alpha1.ScaleTriggers, resolvedSecrets map[string]string) (scalers.Scaler, error) {
	switch trigger.Type {
	case "azure-queue":
		return &scalers.AzureQueueScaler{
			Metadata:        trigger.Metadata,
			ResolvedSecrets: resolvedSecrets,
		}, nil
	default:
		return nil, fmt.Errorf("no scaler found for type: %s", trigger.Type)
	}
}
