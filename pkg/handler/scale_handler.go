package handler

import (
	"context"
	"fmt"
	"sync"
	"time"

	keda_v1alpha1 "github.com/kedacore/keda/pkg/apis/keda/v1alpha1"
	clientset "github.com/kedacore/keda/pkg/client/clientset/versioned"
	"github.com/kedacore/keda/pkg/scalers"
	log "github.com/sirupsen/logrus"
	apps_v1 "k8s.io/api/apps/v1"
	v2beta1 "k8s.io/api/autoscaling/v2beta1"
	batch_v1 "k8s.io/api/batch/v1"
	core_v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
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
	jobInformerStopCh   chan struct{}
}

// defaultPollingInterval - Default polling interval for a ScaledObject triggers if no pollingInterval is defined
// defaultCooldownPeriod - Default cooldown period for a deployment if no cooldownPeriod is defined on the scaledObject
// defaultJobInformerResync - Default period for the job informer to resync
const (
	defaultPollingInterval         = 30
	defaultCooldownPeriod          = 5 * 60 // 5 minutes
	defaultHPAMinReplicas    int32 = 1
	defaultHPAMaxReplicas    int32 = 100
	defaultJobInformerResync       = 30
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
	switch scaledObject.Spec.ScaleType {
	case keda_v1alpha1.ScaleTypeJob:
		h.createOrUpdateJobInformerForScaledObject(scaledObject)
	default:
		h.createHPAWithRetry(scaledObject, true)
	}
	go h.handleScaleLoop(ctx, scaledObject, isDue)
}

// HandleScaledObjectDelete handles any cleanup when a scaled object is deleted
func (h *ScaleHandler) HandleScaledObjectDelete(scaledObject *keda_v1alpha1.ScaledObject) {
	switch scaledObject.Spec.ScaleType {
	case keda_v1alpha1.ScaleTypeJob:
		h.deleteJobsForScaledObject(scaledObject)
	default:
		h.deleteHPAForScaledObject(scaledObject)
	}
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
	scalers, _, err := h.getDeploymentScalers(scaledObject)
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

func (h *ScaleHandler) deleteJobsForScaledObject(scaledObject *keda_v1alpha1.ScaledObject) {
	// end the job informer
	if h.jobInformerStopCh != nil {
		close(h.jobInformerStopCh)
	}

	// delete all running jobs for this scaled object
	propagationPolicy := meta_v1.DeletePropagationBackground
	err := h.kubeClient.BatchV1().Jobs(scaledObject.GetNamespace()).DeleteCollection(
		&meta_v1.DeleteOptions{
			PropagationPolicy: &propagationPolicy,
		},
		meta_v1.ListOptions{
			LabelSelector: "scaledobject=" + scaledObject.GetName(),
		},
	)
	if err != nil {
		log.Errorf("Failed to delete jobs of ScaledObject %s", scaledObject.GetName())
	}
}

func (h *ScaleHandler) deleteHPAForScaledObject(scaledObject *keda_v1alpha1.ScaledObject) {
	deploymentName := scaledObject.Spec.ScaleTargetRef.DeploymentName
	if deploymentName == "" {
		log.Errorf("Notified about ScaledObject with missing deployment name: %s", scaledObject.GetName())
		return
	}

	scaledObjectNamespace := scaledObject.GetNamespace()
	scalers, _, err := h.getDeploymentScalers(scaledObject)
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

func (h *ScaleHandler) createOrUpdateJobInformerForScaledObject(scaledObject *keda_v1alpha1.ScaledObject) error {
	sharedInformerFactory := informers.NewSharedInformerFactory(h.kubeClient, time.Second*time.Duration(defaultJobInformerResync))
	jobInformer := sharedInformerFactory.Batch().V1().Jobs().Informer()

	jobInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{

		UpdateFunc: func(oldObj, newObj interface{}) {
			new := newObj.(*batch_v1.Job)
			if new.Status.CompletionTime != nil {
				// TODO(stgricci): job labels / orphaning jobs (scaleobject option?)
				propagationPolicy := meta_v1.DeletePropagationBackground
				err := h.kubeClient.BatchV1().Jobs(scaledObject.GetNamespace()).Delete(new.Name, &meta_v1.DeleteOptions{
					PropagationPolicy: &propagationPolicy,
				})
				if err != nil {
					log.Errorf("Failed to delete job: %s", new.Name)
				}
				log.Infof("Cleaned up job %s", new.Name)
			}
		},
	})

	h.jobInformerStopCh = make(chan struct{})
	defer close(h.jobInformerStopCh)

	go jobInformer.Run(h.jobInformerStopCh)
	return nil
}

func (h *ScaleHandler) createOrUpdateHPAForScaledObject(scaledObject *keda_v1alpha1.ScaledObject) error {
	deploymentName := scaledObject.Spec.ScaleTargetRef.DeploymentName
	if deploymentName == "" {
		return fmt.Errorf("Notified about ScaledObject with missing deployment name: %s", scaledObject.GetName())
	}

	var scaledObjectMetricSpecs []v2beta1.MetricSpec

	scalers, _, err := h.getDeploymentScalers(scaledObject)
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
		}
		log.Infof("Updated HPA with namespace %s and name %s", scaledObjectNamespace, hpaName)
	} else if err != nil {
		return fmt.Errorf("error creating HPA with namespace %s and name %s : %s", scaledObjectNamespace, hpaName, err)
	} else {
		log.Infof("Created HPA with namespace %s and name %s", scaledObjectNamespace, hpaName)
	}

	return nil
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

func (h *ScaleHandler) resolveEnv(container *core_v1.Container, namespace string) (map[string]string, error) {
	resolved := make(map[string]string)

	if container.EnvFrom != nil {
		for _, source := range container.EnvFrom {
			if source.ConfigMapRef != nil {
				if configMap, err := h.resolveConfigMap(source.ConfigMapRef, namespace); err == nil {
					for k, v := range configMap {
						resolved[k] = v
					}
				} else {
					return nil, fmt.Errorf("error reading config ref %s on namespace %s/: %s", source.ConfigMapRef, namespace, err)
				}
			} else if source.SecretRef != nil {
				if secretsMap, err := h.resolveSecretMap(source.SecretRef, namespace); err == nil {
					for k, v := range secretsMap {
						resolved[k] = v
					}
				} else {
					return nil, fmt.Errorf("error reading secret ref %s on namespace %s: %s", source.SecretRef, namespace, err)
				}
			}
		}

	}

	if container.Env != nil {
		for _, envVar := range container.Env {
			var value string
			var err error

			// env is either a name/value pair or an EnvVarSource
			if envVar.Value != "" {
				value = envVar.Value
			} else if envVar.ValueFrom != nil {
				// env is an EnvVarSource, that can be on of the 4 below
				if envVar.ValueFrom.SecretKeyRef != nil {
					// env is a secret selector
					value, err = h.resolveSecretValue(envVar.ValueFrom.SecretKeyRef, envVar.ValueFrom.SecretKeyRef.Key, namespace)
					if err != nil {
						return nil, fmt.Errorf("error resolving secret name %s for env %s in namespace %s",
							envVar.ValueFrom.SecretKeyRef,
							envVar.Name,
							namespace)
					}
				} else if envVar.ValueFrom.ConfigMapKeyRef != nil {
					// env is a configMap selector
					value, err = h.resolveConfigValue(envVar.ValueFrom.ConfigMapKeyRef, envVar.ValueFrom.ConfigMapKeyRef.Key, namespace)
					if err != nil {
						return nil, fmt.Errorf("error resolving config %s for env %s in namespace %s",
							envVar.ValueFrom.ConfigMapKeyRef,
							envVar.Name,
							namespace)
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

func (h *ScaleHandler) getDeploymentScalers(scaledObject *keda_v1alpha1.ScaledObject) ([]scalers.Scaler, *apps_v1.Deployment, error) {
	scalers := []scalers.Scaler{}
	deploymentName := scaledObject.Spec.ScaleTargetRef.DeploymentName
	if deploymentName == "" {
		return scalers, nil, fmt.Errorf("notified about ScaledObject with missing deployment name: %s", scaledObject.GetName())
	}

	deployment, err := h.kubeClient.AppsV1().Deployments(scaledObject.GetNamespace()).Get(deploymentName, meta_v1.GetOptions{})
	if err != nil {
		return scalers, nil, fmt.Errorf("error getting deployment: %s", err)
	}

	resolvedEnv, err := h.resolveDeploymentEnv(deployment, scaledObject.Spec.ScaleTargetRef.ContainerName)
	if err != nil {
		return scalers, nil, fmt.Errorf("error resolving secrets for deployment: %s", err)
	}

	for i, trigger := range scaledObject.Spec.Triggers {
		scaler, err := h.getScaler(scaledObject, trigger, resolvedEnv)
		if err != nil {
			return scalers, nil, fmt.Errorf("error getting scaler for trigger #%d: %s", i, err)
		}

		scalers = append(scalers, scaler)
	}

	return scalers, deployment, nil
}

// TODO(stgricci): implement me
func (h *ScaleHandler) getJobScalers(scaledObject *keda_v1alpha1.ScaledObject) ([]scalers.Scaler, error) {
	scalers := []scalers.Scaler{}

	resolvedEnv, err := h.resolveJobEnv(scaledObject)
	if err != nil {
		return scalers, fmt.Errorf("error resolving secrets for job: %s", err)
	}

	for i, trigger := range scaledObject.Spec.Triggers {
		scaler, err := h.getScaler(scaledObject, trigger, resolvedEnv)
		if err != nil {
			return scalers, fmt.Errorf("error getting scaler for trigger #%d: %s", i, err)
		}

		scalers = append(scalers, scaler)
	}
	log.Printf("Scalers: %d", len(scalers))

	return scalers, nil
}

func (h *ScaleHandler) getScaler(scaledObject *keda_v1alpha1.ScaledObject, trigger keda_v1alpha1.ScaleTriggers, resolvedEnv map[string]string) (scalers.Scaler, error) {
	switch trigger.Type {
	case "azure-queue":
		return scalers.NewAzureQueueScaler(resolvedEnv, trigger.Metadata)
	case "azure-servicebus":
		return scalers.NewAzureServiceBusScaler(resolvedEnv, trigger.Metadata)
	case "aws-sqs-queue":
		return scalers.NewAwsSqsQueueScaler(resolvedEnv, trigger.Metadata)
	case "aws-cloudwatch":
		return scalers.NewAwsCloudwatchScaler(resolvedEnv, trigger.Metadata)
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
	case "external":
		return scalers.NewExternalScaler(scaledObject, resolvedEnv, trigger.Metadata)
	case "liiklus":
		return scalers.NewLiiklusScaler(resolvedEnv, trigger.Metadata)
	default:
		return nil, fmt.Errorf("no scaler found for type: %s", trigger.Type)
	}
}
