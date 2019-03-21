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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
	// Default polling interval for a ScaledObject triggers if no pollingInterval is defined.
	defaultPollingInterval = 30
	// Default cooldown period for a deployment if no cooldownPeriod is defined on the scaledObject
	defaultCooldownPeriod = 5 * 60 // 5 minutes
	minReplicas           = 1
	maxReplicas           = 100
)

// NewScaleHandler creates a ScaleHandler object
func NewScaleHandler(koreClient clientset.Interface, kubeClient kubernetes.Interface) *ScaleHandler {
	handler := &ScaleHandler{
		koreClient: koreClient,
		kubeClient: kubeClient,
	}

	return handler
}

// WatchScaledObjectWithContext runs a handleScaleLoop go-routine for the scaledObject
func (h *ScaleHandler) WatchScaledObjectWithContext(ctx context.Context, scaledObject *kore_v1alpha1.ScaledObject, isDue bool) {
	h.createHPAForNewScaledObject(ctx, scaledObject)
	go h.handleScaleLoop(ctx, scaledObject, isDue)
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
	scaledObjects, err := scaledObjectQuerier.List(meta_v1.ListOptions{LabelSelector: metricSelector.String()})
	if err != nil {
		return nil, err
	} else if len(scaledObjects.Items) != 1 {
		return nil, fmt.Errorf("Exactly one scaled object should match label %s", metricSelector.String())
	}

	scaledObject := &scaledObjects.Items[0]
	matchingMetrics := []external_metrics.ExternalMetricValue{}
	scalers, _ := h.getScalers(scaledObject)
	for _, scaler := range scalers {
		metrics, err := scaler.GetMetrics(context.TODO(), merticName, metricSelector)
		if err != nil {
			log.Errorf("error getting metric for scaler : %s", err)
		} else {
			matchingMetrics = append(matchingMetrics, metrics...)
		}
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
	if apierrors.IsNotFound(err) {
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
	scalers, _ := h.getScalers(scaledObject)
	for _, scaler := range scalers {
		metricSpecs := scaler.GetMetricSpecForScaling()

		// add the deploymentName label. This is how the MetricsAdapter will know which scaledobject a metric is for when the HPA queries it.
		for _, metricSpec := range metricSpecs {
			metricSpec.External.MetricSelector = &meta_v1.LabelSelector{MatchLabels: make(map[string]string)}
			metricSpec.External.MetricSelector.MatchLabels["deploymentName"] = deploymentName
		}
		scaledObjectMetricSpecs = append(scaledObjectMetricSpecs, metricSpecs...)
	}

	kvd := &v2beta1.CrossVersionObjectReference{Name: deploymentName, Kind: "Deployment", APIVersion: "apps/v1"}
	var minReplicasVar int32 = minReplicas
	scaledObjectNamespace := scaledObject.GetNamespace()
	hpaName := "kore-hpa-" + deploymentName
	newHPASpec := &v2beta1.HorizontalPodAutoscalerSpec{MinReplicas: &minReplicasVar, MaxReplicas: maxReplicas, Metrics: scaledObjectMetricSpecs, ScaleTargetRef: *kvd}
	objectSpec := &meta_v1.ObjectMeta{Name: hpaName, Namespace: scaledObjectNamespace}
	newHPA := &v2beta1.HorizontalPodAutoscaler{Spec: *newHPASpec, ObjectMeta: *objectSpec}
	newHPA, err := h.kubeClient.AutoscalingV2beta1().HorizontalPodAutoscalers(scaledObjectNamespace).Create(newHPA)
	if apierrors.IsAlreadyExists(err) {
		log.Warnf("HPA with namespace %s and name %s already exists", scaledObjectNamespace, hpaName)
	} else if err != nil {
		log.Errorf("Error creating HPA with namespace %s and name %s : %s\n", scaledObjectNamespace, hpaName, err)
	} else {
		log.Debugf("Created HPA with namespace %s and name %s", scaledObjectNamespace, hpaName)
	}
}

// This method blocks for ever and checks the scaledObject based on its pollingInterval
// if isDue is set to true, the method will check the scaledObject right away. Otherwise
// it'll wait for pollingInterval then check.
func (h *ScaleHandler) handleScaleLoop(ctx context.Context, scaledObject *kore_v1alpha1.ScaledObject, isDue bool) {
	h.handleScale(ctx, scaledObject)

	var pollingInterval time.Duration
	if scaledObject.Spec.PollingInterval != nil {
		pollingInterval = time.Second * time.Duration(*scaledObject.Spec.PollingInterval)
	} else {
		pollingInterval = time.Second * time.Duration(defaultPollingInterval)
	}

	getPollingInterval := func() time.Duration {
		if isDue {
			isDue = false
			return 0
		}
		return pollingInterval
	}

	log.Debugf("watching scaledObject (%s/%s) with pollingInterval: %d", scaledObject.GetNamespace(), scaledObject.GetName(), pollingInterval)

	for {
		select {
		case <-time.After(getPollingInterval()):
			h.handleScale(ctx, scaledObject)
		case <-ctx.Done():
			log.Debugf("context for scaledObject (%s/%s) canceled", scaledObject.GetNamespace(), scaledObject.GetName())
			return
		}
	}
}

// handleScale contains the main logic for the ScaleHandler scaling logic.
// It'll check each trigger active status then call scaleDeployment
func (h *ScaleHandler) handleScale(ctx context.Context, scaledObject *kore_v1alpha1.ScaledObject) {
	isScaledObjectActive := false

	scalers, deployment := h.getScalers(scaledObject)
	for _, scaler := range scalers {
		isTriggerActive, err := scaler.IsActive(ctx)
		if err != nil {
			log.Errorf("Error getting scale decision: %s", err)
			continue
		} else if isTriggerActive {
			isScaledObjectActive = true
			log.Debugf("Scallr %s for scaledObject %s/%s is active", scaler, scaledObject.GetNamespace(), scaledObject.GetName())
		}
	}

	h.scaleDeployment(deployment, scaledObject, isScaledObjectActive)

	return
}

func (h *ScaleHandler) scaleDeployment(deployment *apps_v1.Deployment, scaledObject *kore_v1alpha1.ScaledObject, isActive bool) {
	if *deployment.Spec.Replicas == 0 && isActive {
		// current replica count is 0, but there is an active trigger.
		// scale the deployment up
		h.scaleFromZero(deployment, scaledObject)
	} else if !isActive && *deployment.Spec.Replicas > 0 {
		// there are no active triggers, but the deployment has replicas.
		// Try to scale it down.
		h.scaleToZero(deployment, scaledObject)
	} else if isActive {
		// triggers are active, but we didn't need to scale (replica count > 0)
		// Update LastActiveTime to now.
		now := meta_v1.Now()
		scaledObject.Status.LastActiveTime = &now
		h.updateScaledObject(scaledObject)
	} else {
		log.Debugf("deployment (%s/%s) no change", deployment.GetNamespace(), deployment.GetName())
	}
}

func (h *ScaleHandler) updateScaledObject(scaledObject *kore_v1alpha1.ScaledObject) error {
	_, err := h.koreClient.KoreV1alpha1().ScaledObjects(scaledObject.GetNamespace()).Update(scaledObject)
	if err != nil {
		log.Errorf("Error updating scaledObject (%s/%s) status: %s", scaledObject.GetNamespace(), scaledObject.GetName(), err.Error())
	}
	return err
}

// A deployment will be scaled down to 0 only if it's passed its cooldown period
// or if LastActiveTime is nil
func (h *ScaleHandler) scaleToZero(deployment *apps_v1.Deployment, scaledObject *kore_v1alpha1.ScaledObject) {
	var cooldownPeriod time.Duration

	if scaledObject.Spec.CooldownPeriod != nil {
		cooldownPeriod = time.Second * time.Duration(*scaledObject.Spec.CooldownPeriod)
	} else {
		cooldownPeriod = time.Second * time.Duration(defaultCooldownPeriod)
	}

	// LastActiveTime can be nil if the deployment was scaled outside of Kore.
	// In this case we will ignore the cooldown period and scale it down
	if scaledObject.Status.LastActiveTime == nil ||
		scaledObject.Status.LastActiveTime.Add(cooldownPeriod).Before(time.Now()) {
		// or last time a trigger was active was > cooldown period, so scale down.
		*deployment.Spec.Replicas = 0
		err := h.updateDeployment(deployment)
		if err == nil {
			log.Debugf("Successfully scaled deployment (%s/%s) to 0 replicas", deployment.GetNamespace(), deployment.GetName())
		}
	} else {
		log.Debugf("scaledObject (%s/%s) cooling down. Last active time %v, cooldownPeriod %d",
			scaledObject.GetNamespace(),
			scaledObject.GetName(),
			scaledObject.Status.LastActiveTime,
			cooldownPeriod)
	}
}

func (h *ScaleHandler) scaleFromZero(deployment *apps_v1.Deployment, scaledObject *kore_v1alpha1.ScaledObject) {
	currentReplicas := *deployment.Spec.Replicas
	*deployment.Spec.Replicas = 1
	err := h.updateDeployment(deployment)

	if err == nil {
		log.Debugf("Successfully updated deployment (%s/%s) from %d to %d replicas",
			deployment.GetNamespace(),
			deployment.GetName(),
			currentReplicas,
			*deployment.Spec.Replicas)

		// Scale was successful. Update lastScaleTime and lastActiveTime on the scaledObject
		now := meta_v1.Now()
		scaledObject.Status.LastActiveTime = &now
		h.updateScaledObject(scaledObject)
	}
}

func (h *ScaleHandler) updateDeployment(deployment *apps_v1.Deployment) error {
	_, err := h.kubeClient.AppsV1().Deployments(deployment.GetNamespace()).Update(deployment)
	if err != nil {
		log.Errorf("Error updating deployment (%s/%s)  Error: %s", deployment.GetNamespace(), deployment.GetName(), err)
	}
	return err
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

func (h *ScaleHandler) getScalers(scaledObject *kore_v1alpha1.ScaledObject) ([]scalers.Scaler, *apps_v1.Deployment) {
	scalers := []scalers.Scaler{}
	deploymentName := scaledObject.Spec.ScaleTargetRef.DeploymentName
	if deploymentName == "" {
		log.Errorf("Notified about ScaledObject with missing deployment name: %s", scaledObject.GetName())
		return scalers, nil
	}

	deployment, err := h.kubeClient.AppsV1().Deployments(scaledObject.GetNamespace()).Get(deploymentName, meta_v1.GetOptions{})
	if err != nil {
		log.Errorf("Error getting deployment: %s", err)
		return scalers, nil
	}

	resolvedSecrets, err := h.resolveSecrets(deployment)
	if err != nil {
		log.Errorf("Error resolving secrets for deployment: %s", err)
		return scalers, nil
	}

	for i, trigger := range scaledObject.Spec.Triggers {
		scaler, err := h.getScaler(trigger, resolvedSecrets)
		if err != nil {
			log.Errorf("error for trigger #%d: %s", i, err)
			continue
		}

		scalers = append(scalers, scaler)
	}

	return scalers, deployment
}

func (h *ScaleHandler) getScaler(trigger kore_v1alpha1.ScaleTriggers, resolvedSecrets map[string]string) (scalers.Scaler, error) {
	switch trigger.Type {
	case "azure-queue":
		return scalers.NewAzureQueueScaler(resolvedSecrets, trigger.Metadata), nil
	case "kafka":
		return scalers.NewKafkaScaler(resolvedSecrets, trigger.Metadata), nil
	default:
		return nil, fmt.Errorf("no scaler found for type: %s", trigger.Type)
	}
}
