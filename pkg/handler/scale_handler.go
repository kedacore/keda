package handler

import (
	"context"
	"fmt"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	keda_v1alpha1 "github.com/kedacore/keda/pkg/apis/keda/v1alpha1"
	clientset "github.com/kedacore/keda/pkg/client/clientset/versioned"
	"github.com/kedacore/keda/pkg/scalers"
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
	kedaClient          clientset.Interface
	kubeClient          kubernetes.Interface
	externalMetricNames map[string]int
	metricNamesLock     sync.RWMutex
	hpasToCreate        map[string]bool
	hpaCreateLock       sync.RWMutex
}

const (
	// Default polling interval for a ScaledObject triggers if no pollingInterval is defined.
	defaultPollingInterval = 30
	// Default cooldown period for a deployment if no cooldownPeriod is defined on the scaledObject
	defaultCooldownPeriod       = 5 * 60 // 5 minutes
	defaultHPAMinReplicas int32 = 1
	defaultHPAMaxReplicas int32 = 100
)

// NewScaleHandler creates a ScaleHandler object
func NewScaleHandler(kedaClient clientset.Interface, kubeClient kubernetes.Interface) *ScaleHandler {
	handler := &ScaleHandler{
		kedaClient:          kedaClient,
		kubeClient:          kubeClient,
		externalMetricNames: make(map[string]int),
		hpasToCreate:        make(map[string]bool),
	}

	return handler
}

// TODO confusing naming switching from isUpdate (controller) -> isDue (here)[]
// WatchScaledObjectWithContext runs a handleScaleLoop go-routine for the scaledObject
func (h *ScaleHandler) WatchScaledObjectWithContext(ctx context.Context, scaledObject *keda_v1alpha1.ScaledObject, isDue bool) {
	h.createHPAWithRetry(scaledObject, true)
	go h.handleScaleLoop(ctx, scaledObject, isDue)
}

// HandleScaledObjectDelete handles any cleanup when a scaled object is deleted
func (h *ScaleHandler) HandleScaledObjectDelete(scaledObject *keda_v1alpha1.ScaledObject) {
	h.deleteHPAForScaledObject(scaledObject)
}

// GetExternalMetricNames returns the exteral metrics of the triggers of the current scaled objects
func (h *ScaleHandler) GetExternalMetricNames() []string {
	h.metricNamesLock.RLock()
	defer h.metricNamesLock.RUnlock()
	returnedMetrics := make([]string, 0, len(h.externalMetricNames))
	for k := range h.externalMetricNames {
		returnedMetrics = append(returnedMetrics, k)
	}

	return returnedMetrics
}

// GetScaledObjectMetrics is used by the  metric adapter in provider.go to get the value for a metric for a scaled object
func (h *ScaleHandler) GetScaledObjectMetrics(namespace string, metricSelector labels.Selector, metricName string) ([]external_metrics.ExternalMetricValue, error) {
	// get the scaled objects matching namespace and labels
	scaledObjectQuerier := h.kedaClient.KedaV1alpha1().ScaledObjects(namespace)
	scaledObjects, err := scaledObjectQuerier.List(meta_v1.ListOptions{LabelSelector: metricSelector.String()})
	if err != nil {
		return nil, err
	} else if len(scaledObjects.Items) != 1 {
		return nil, fmt.Errorf("Exactly one scaled object should match label %s", metricSelector.String())
	}

	scaledObject := &scaledObjects.Items[0]
	matchingMetrics := []external_metrics.ExternalMetricValue{}
	scalers, _, err := h.getScalers(scaledObject)
	if err != nil {
		return nil, fmt.Errorf("Error when getting scalers %s", err)
	}

	for _, scaler := range scalers {
		metrics, err := scaler.GetMetrics(context.TODO(), metricName, metricSelector)
		if err != nil {
			log.Errorf("error getting metric for scaler : %s", err)
		} else {
			matchingMetrics = append(matchingMetrics, metrics...)
		}

		scaler.Close()
	}

	return matchingMetrics, nil
}

func (h *ScaleHandler) createHPAWithRetry(scaledObject *keda_v1alpha1.ScaledObject, createUpdateOverride bool) {
	deploymentName := scaledObject.Spec.ScaleTargetRef.DeploymentName
	if deploymentName == "" {
		log.Errorf("Notified about ScaledObject with missing deployment name: %s", scaledObject.GetName())
		return
	}
	hpaName := fmt.Sprintf("keda-hpa-%s", deploymentName)
	existsInRetryList := h.doesHPAExistInRetryList(hpaName)
	if existsInRetryList || createUpdateOverride {
		err := h.createOrUpdateHPAForScaledObject(scaledObject)
		if err != nil {
			log.Errorf("Error creating or updating HPA for scaled object %s: %s", scaledObject.GetName(), err)
		}

		h.hpaCreateLock.Lock()
		defer h.hpaCreateLock.Unlock()
		if err != nil {
			h.hpasToCreate[hpaName] = true
			if !existsInRetryList {
				log.Debugf("createHPAWithRetry ScaledObject %s is added to retry list", scaledObject.GetName())
			}
		} else if existsInRetryList {
			delete(h.hpasToCreate, hpaName)
			log.Debugf("createHPAWithRetry ScaledObject %s is removed from retry list", scaledObject.GetName())
		}
	}
}

func (h *ScaleHandler) doesHPAExistInRetryList(hpaName string) bool {
	h.hpaCreateLock.RLock()
	defer h.hpaCreateLock.RUnlock()
	_, found := h.hpasToCreate[hpaName]
	return found
}

func (h *ScaleHandler) deleteHPAForScaledObject(scaledObject *keda_v1alpha1.ScaledObject) {
	deploymentName := scaledObject.Spec.ScaleTargetRef.DeploymentName
	if deploymentName == "" {
		log.Errorf("Notified about ScaledObject with missing deployment name: %s", scaledObject.GetName())
		return
	}

	scaledObjectNamespace := scaledObject.GetNamespace()
	scalers, _, err := h.getScalers(scaledObject)
	if err != nil {
		log.Errorf("Error when getting scalers %s", err)
	}

	for _, scaler := range scalers {
		metricSpecs := scaler.GetMetricSpecForScaling()
		for _, metricSpec := range metricSpecs {
			h.removeExternalMetricName(metricSpec.External.MetricName)
		}
		scaler.Close()
	}

	hpaName := fmt.Sprintf("keda-hpa-%s", deploymentName)
	deleteOptions := &meta_v1.DeleteOptions{}
	err = h.kubeClient.AutoscalingV2beta1().HorizontalPodAutoscalers(scaledObjectNamespace).Delete(hpaName, deleteOptions)
	if apierrors.IsNotFound(err) {
		log.Warnf("HPA with namespace %s and name %s is not found", scaledObjectNamespace, hpaName)
	} else if err != nil {
		log.Errorf("Error deleting HPA with namespace %s and name %s : %s\n", scaledObjectNamespace, hpaName, err)
	} else {
		log.Infof("Deleted HPA with namespace %s and name %s", scaledObjectNamespace, hpaName)
	}

	h.hpaCreateLock.Lock()
	defer h.hpaCreateLock.Unlock()
	delete(h.hpasToCreate, hpaName)
}

func (h *ScaleHandler) addExternalMetricName(metricName string) {
	h.metricNamesLock.Lock()
	defer h.metricNamesLock.Unlock()
	h.externalMetricNames[metricName] = h.externalMetricNames[metricName] + 1
	log.Debugf("ExternalMetricList: Incremented metricName %s with ref count %d", metricName, h.externalMetricNames[metricName])
}

func (h *ScaleHandler) removeExternalMetricName(metricName string) {
	h.metricNamesLock.Lock()
	defer h.metricNamesLock.Unlock()
	h.externalMetricNames[metricName] = h.externalMetricNames[metricName] - 1
	log.Debugf("ExternalMetricList: Decremented metricName %s with ref count %d", metricName, h.externalMetricNames[metricName])
	if h.externalMetricNames[metricName] == 0 {
		delete(h.externalMetricNames, metricName)
		log.Debugf("ExternalMetricList: Removed metric name %s as ref count is 0", metricName)
	}
}

func (h *ScaleHandler) createOrUpdateHPAForScaledObject(scaledObject *keda_v1alpha1.ScaledObject) error {
	deploymentName := scaledObject.Spec.ScaleTargetRef.DeploymentName
	if deploymentName == "" {
		return fmt.Errorf("Notified about ScaledObject with missing deployment name: %s", scaledObject.GetName())
	}

	var scaledObjectMetricSpecs []v2beta1.MetricSpec

	scalers, _, err := h.getScalers(scaledObject)
	if err != nil {
		return fmt.Errorf("Error getting scalers %s", err)
	}

	for _, scaler := range scalers {
		metricSpecs := scaler.GetMetricSpecForScaling()

		// add the deploymentName label. This is how the MetricsAdapter will know which scaledobject a metric is for when the HPA queries it.
		for _, metricSpec := range metricSpecs {
			metricSpec.External.MetricSelector = &meta_v1.LabelSelector{MatchLabels: make(map[string]string)}
			metricSpec.External.MetricSelector.MatchLabels["deploymentName"] = deploymentName
			h.addExternalMetricName(metricSpec.External.MetricName)
		}
		scaledObjectMetricSpecs = append(scaledObjectMetricSpecs, metricSpecs...)
		scaler.Close()
	}

	var minReplicas *int32
	if scaledObject.Spec.MinReplicaCount != nil && *scaledObject.Spec.MinReplicaCount > 0 {
		minReplicas = scaledObject.Spec.MinReplicaCount
	} else {
		tmp := defaultHPAMinReplicas
		minReplicas = &tmp
	}

	var maxReplicas int32
	if scaledObject.Spec.MaxReplicaCount != nil {
		maxReplicas = *scaledObject.Spec.MaxReplicaCount
	} else {
		maxReplicas = defaultHPAMaxReplicas
	}

	scaledObjectNamespace := scaledObject.GetNamespace()
	hpaName := fmt.Sprintf("keda-hpa-%s", deploymentName)
	hpa := &v2beta1.HorizontalPodAutoscaler{
		Spec: v2beta1.HorizontalPodAutoscalerSpec{
			MinReplicas: minReplicas,
			MaxReplicas: maxReplicas,
			Metrics:     scaledObjectMetricSpecs,
			ScaleTargetRef: v2beta1.CrossVersionObjectReference{
				Name:       deploymentName,
				Kind:       "Deployment",
				APIVersion: "apps/v1",
			}},
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      hpaName,
			Namespace: scaledObjectNamespace,
		},
		TypeMeta: meta_v1.TypeMeta{
			APIVersion: "v2beta1",
		},
	}

	_, err = h.kubeClient.AutoscalingV2beta1().HorizontalPodAutoscalers(scaledObjectNamespace).Create(hpa)
	if apierrors.IsAlreadyExists(err) {
		log.Infof("HPA with namespace %s and name %s already exists. Updating..", scaledObjectNamespace, hpaName)
		_, err := h.kubeClient.AutoscalingV2beta1().HorizontalPodAutoscalers(scaledObjectNamespace).Update(hpa)
		if err != nil {
			return fmt.Errorf("error updating HPA with namespace %s and name %s : %s", scaledObjectNamespace, hpaName, err)
		} else {
			log.Infof("Updated HPA with namespace %s and name %s", scaledObjectNamespace, hpaName)
		}
	} else if err != nil {
		return fmt.Errorf("error creating HPA with namespace %s and name %s : %s", scaledObjectNamespace, hpaName, err)
	} else {
		log.Infof("Created HPA with namespace %s and name %s", scaledObjectNamespace, hpaName)
	}

	return nil
}

// This method blocks forever and checks the scaledObject based on its pollingInterval
// if isDue is set to true, the method will check the scaledObject right away. Otherwise
// it'll wait for pollingInterval then check.
func (h *ScaleHandler) handleScaleLoop(ctx context.Context, scaledObject *keda_v1alpha1.ScaledObject, isDue bool) {
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
			h.createHPAWithRetry(scaledObject, false)
			h.handleScale(ctx, scaledObject)
		case <-ctx.Done():
			log.Debugf("context for scaledObject (%s/%s) canceled", scaledObject.GetNamespace(), scaledObject.GetName())
			return
		}
	}
}

// handleScale contains the main logic for the ScaleHandler scaling logic.
// It'll check each trigger active status then call scaleDeployment
func (h *ScaleHandler) handleScale(ctx context.Context, scaledObject *keda_v1alpha1.ScaledObject) {
	scalers, deployment, err := h.getScalers(scaledObject)

	if deployment == nil {
		return
	}
	if err != nil {
		log.Errorf("Error getting scalers: %s", err)
		return
	}

	isScaledObjectActive := false

	for _, scaler := range scalers {
		isTriggerActive, err := scaler.IsActive(ctx)

		if err != nil {
			log.Debugf("Error getting scale decision: %s", err)
			continue
		} else if isTriggerActive {
			isScaledObjectActive = true
			log.Debugf("Scaler %s for scaledObject %s/%s is active", scaler, scaledObject.GetNamespace(), scaledObject.GetName())
		}
		scaler.Close()
	}

	h.scaleDeployment(deployment, scaledObject, isScaledObjectActive)

	return
}

func (h *ScaleHandler) scaleDeployment(deployment *apps_v1.Deployment, scaledObject *keda_v1alpha1.ScaledObject, isActive bool) {

	if *deployment.Spec.Replicas == 0 && isActive {
		// current replica count is 0, but there is an active trigger.
		// scale the deployment up
		h.scaleFromZero(deployment, scaledObject)
	} else if !isActive &&
		*deployment.Spec.Replicas > 0 &&
		(scaledObject.Spec.MinReplicaCount == nil || *scaledObject.Spec.MinReplicaCount == 0) {
		// there are no active triggers, but the deployment has replicas.
		// AND
		// There is no minimum configured or minumum is set to ZERO. HPA will handles other scale down operations

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

func (h *ScaleHandler) updateScaledObject(scaledObject *keda_v1alpha1.ScaledObject) error {
	newScaledObject, err := h.kedaClient.KedaV1alpha1().ScaledObjects(scaledObject.GetNamespace()).Update(scaledObject)
	if err != nil {
		log.Errorf("Error updating scaledObject (%s/%s) status: %s", scaledObject.GetNamespace(), scaledObject.GetName(), err.Error())
	} else {
		*scaledObject = *newScaledObject
	}
	return err
}

// A deployment will be scaled down to 0 only if it's passed its cooldown period
// or if LastActiveTime is nil
func (h *ScaleHandler) scaleToZero(deployment *apps_v1.Deployment, scaledObject *keda_v1alpha1.ScaledObject) {
	var cooldownPeriod time.Duration

	if scaledObject.Spec.CooldownPeriod != nil {
		cooldownPeriod = time.Second * time.Duration(*scaledObject.Spec.CooldownPeriod)
	} else {
		cooldownPeriod = time.Second * time.Duration(defaultCooldownPeriod)
	}

	// LastActiveTime can be nil if the deployment was scaled outside of Keda.
	// In this case we will ignore the cooldown period and scale it down
	if scaledObject.Status.LastActiveTime == nil ||
		scaledObject.Status.LastActiveTime.Add(cooldownPeriod).Before(time.Now()) {
		// or last time a trigger was active was > cooldown period, so scale down.
		*deployment.Spec.Replicas = 0
		err := h.updateDeployment(deployment)
		if err == nil {
			log.Infof("Successfully scaled deployment (%s/%s) to 0 replicas", deployment.GetNamespace(), deployment.GetName())
		}
	} else {
		log.Debugf("scaledObject (%s/%s) cooling down. Last active time %v, cooldownPeriod %d",
			scaledObject.GetNamespace(),
			scaledObject.GetName(),
			scaledObject.Status.LastActiveTime,
			cooldownPeriod)
	}
}

func (h *ScaleHandler) scaleFromZero(deployment *apps_v1.Deployment, scaledObject *keda_v1alpha1.ScaledObject) {
	currentReplicas := *deployment.Spec.Replicas
	if scaledObject.Spec.MinReplicaCount != nil && *scaledObject.Spec.MinReplicaCount > 0 {
		deployment.Spec.Replicas = scaledObject.Spec.MinReplicaCount
	} else {
		*deployment.Spec.Replicas = 1
	}

	err := h.updateDeployment(deployment)

	if err == nil {
		log.Infof("Successfully updated deployment (%s/%s) from %d to %d replicas",
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

func (h *ScaleHandler) resolveEnv(deployment *apps_v1.Deployment, containerName string) (map[string]string, error) {
	deploymentKey, err := cache.MetaNamespaceKeyFunc(deployment)
	if err != nil {
		return nil, err
	}

	if len(deployment.Spec.Template.Spec.Containers) < 1 {
		return nil, fmt.Errorf("Deployment (%s) doesn't have containers", deploymentKey)
	}

	resolved := make(map[string]string)

	var container core_v1.Container

	if containerName != "" {
		for _, c := range deployment.Spec.Template.Spec.Containers {
			if c.Name == containerName {
				container = c
				break
			}
		}

		if &container == nil {
			return nil, fmt.Errorf("Couldn't find container with name %s on deployment %s", containerName, deployment.GetName())
		}
	} else {
		container = deployment.Spec.Template.Spec.Containers[0]
	}

	if container.EnvFrom != nil {
		for _, source := range container.EnvFrom {
			if source.ConfigMapRef != nil {
				if configMap, err := h.resolveConfigMap(source.ConfigMapRef, deployment.GetNamespace()); err == nil {
					for k, v := range configMap {
						resolved[k] = v
					}
				} else {
					return nil, fmt.Errorf("error reading config ref %s on deployment %s/%s: %s", source.ConfigMapRef, deployment.GetNamespace(), deployment.GetName(), err)
				}
			} else if source.SecretRef != nil {
				if secretsMap, err := h.resolveSecretMap(source.SecretRef, deployment.GetNamespace()); err == nil {
					for k, v := range secretsMap {
						resolved[k] = v
					}
				} else {
					return nil, fmt.Errorf("error reading secret ref %s on deployment %s/%s: %s", source.SecretRef, deployment.GetNamespace(), deployment.GetName(), err)
				}
			}
		}
	}

	if container.Env != nil {
		for _, envVar := range container.Env {
			var value string

			// env is either a name/value pair or an EnvVarSource
			if envVar.Value != "" {
				value = envVar.Value
			} else if envVar.ValueFrom != nil {
				// env is an EnvVarSource, that can be one of the 4 below
				if envVar.ValueFrom.SecretKeyRef != nil {
					// env is a secret selector
					value, err = h.resolveSecretValue(envVar.ValueFrom.SecretKeyRef, envVar.ValueFrom.SecretKeyRef.Key, deployment.GetNamespace())
					if err != nil {
						return nil, fmt.Errorf("error resolving secret name %s for env %s in deployment %s/%s",
							envVar.ValueFrom.SecretKeyRef,
							envVar.Name,
							deployment.GetNamespace(),
							deployment.GetName())
					}
				} else if envVar.ValueFrom.ConfigMapKeyRef != nil {
					// env is a configMap selector
					value, err = h.resolveConfigValue(envVar.ValueFrom.ConfigMapKeyRef, envVar.ValueFrom.ConfigMapKeyRef.Key, deployment.GetNamespace())
					if err != nil {
						return nil, fmt.Errorf("error resolving config %s for env %s in deployment %s/%s",
							envVar.ValueFrom.ConfigMapKeyRef,
							envVar.Name,
							deployment.GetName(),
							deployment.GetNamespace())
					}
				} else {
					return nil, fmt.Errorf("cannot resolve env %s to a value. fieldRef and resourceFieldRef env are skipped", envVar.Name)
				}
			}
			resolved[envVar.Name] = value
		}
	}

	return resolved, nil
}

func (h *ScaleHandler) resolveConfigMap(configMapRef *core_v1.ConfigMapEnvSource, namespace string) (map[string]string, error) {
	configMap, err := h.kubeClient.CoreV1().ConfigMaps(namespace).Get(configMapRef.Name, meta_v1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return configMap.Data, nil
}

func (h *ScaleHandler) resolveSecretMap(secretMapRef *core_v1.SecretEnvSource, namespace string) (map[string]string, error) {
	secrets, err := h.kubeClient.CoreV1().Secrets(namespace).Get(secretMapRef.Name, meta_v1.GetOptions{})
	if err != nil {
		return nil, err
	}

	secretsStr := make(map[string]string)
	for k, v := range secrets.Data {
		secretsStr[k] = string(v)
	}

	return secretsStr, nil
}

func (h *ScaleHandler) resolveSecretValue(secretKeyRef *core_v1.SecretKeySelector, keyName, namespace string) (string, error) {
	secretCollection, err := h.kubeClient.CoreV1().Secrets(namespace).Get(secretKeyRef.Name, meta_v1.GetOptions{})

	if err != nil {
		return "", err
	}

	return string(secretCollection.Data[keyName]), nil
}

func (h *ScaleHandler) resolveConfigValue(configKeyRef *core_v1.ConfigMapKeySelector, keyName, namespace string) (string, error) {
	configCollection, err := h.kubeClient.CoreV1().ConfigMaps(namespace).Get(configKeyRef.Name, meta_v1.GetOptions{})

	if err != nil {
		return "", err
	}

	return string(configCollection.Data[keyName]), nil
}

func (h *ScaleHandler) getScalers(scaledObject *keda_v1alpha1.ScaledObject) ([]scalers.Scaler, *apps_v1.Deployment, error) {
	scalers := []scalers.Scaler{}
	deploymentName := scaledObject.Spec.ScaleTargetRef.DeploymentName
	if deploymentName == "" {
		return scalers, nil, fmt.Errorf("notified about ScaledObject with missing deployment name: %s", scaledObject.GetName())
	}

	deployment, err := h.kubeClient.AppsV1().Deployments(scaledObject.GetNamespace()).Get(deploymentName, meta_v1.GetOptions{})
	if err != nil {
		return scalers, nil, fmt.Errorf("error getting deployment: %s", err)
	}

	resolvedEnv, err := h.resolveEnv(deployment, scaledObject.Spec.ScaleTargetRef.ContainerName)
	if err != nil {
		return scalers, nil, fmt.Errorf("error resolving secrets for deployment: %s", err)
	}

	for i, trigger := range scaledObject.Spec.Triggers {
		scaler, err := h.getScaler(trigger, resolvedEnv)
		if err != nil {
			return scalers, nil, fmt.Errorf("error getting scaler for trigger #%d: %s", i, err)
		}

		scalers = append(scalers, scaler)
	}

	return scalers, deployment, nil
}

func (h *ScaleHandler) getScaler(trigger keda_v1alpha1.ScaleTriggers, resolvedEnv map[string]string) (scalers.Scaler, error) {
	switch trigger.Type {
	case "azure-queue":
		return scalers.NewAzureQueueScaler(resolvedEnv, trigger.Metadata)
	case "azure-servicebus":
		return scalers.NewAzureServiceBusScaler(resolvedEnv, trigger.Metadata)
	case "aws-sqs-queue":
		return scalers.NewAwsSqsQueueScaler(resolvedEnv, trigger.Metadata)
	case "kafka":
		return scalers.NewKafkaScaler(resolvedEnv, trigger.Metadata)
	case "rabbitmq":
		return scalers.NewRabbitMQScaler(resolvedEnv, trigger.Metadata)
	case "azure-eventhub":
		return scalers.NewAzureEventHubScaler(resolvedEnv, trigger.Metadata)
	case "prometheus":
		return scalers.NewPrometheusScaler(resolvedEnv, trigger.Metadata)
	case "redis":
		return scalers.NewRedisScaler(resolvedEnv, trigger.Metadata)
	case "gcp-pubsub":
		return scalers.NewPubSubScaler(resolvedEnv, trigger.Metadata)
	default:
		return nil, fmt.Errorf("no scaler found for type: %s", trigger.Type)
	}
}
