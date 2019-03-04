package scalers

import (
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	apps_v1 "k8s.io/api/apps/v1"
	core_v1 "k8s.io/api/core/v1"

	kore_v1alpha1 "github.com/Azure/Kore/pkg/apis/kore/v1alpha1"
	clientset "github.com/Azure/Kore/pkg/client/clientset/versioned"
	log "github.com/Sirupsen/logrus"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// ScaleHandler encapsulates the logic of calling the right scalers for
// each ScaledObject and making the final scale decision and operation
type ScaleHandler struct {
	koreClient clientset.Interface
	kubeClient kubernetes.Interface
	// A delaying workqueue is used to re-enqueue the watched
	// scaledObjects based on their polling interval.
	// Default pollingInterval is 30 seconds.
	workqueue workqueue.DelayingInterface
	// While scaledObjects are in the workqueue, they could become invalid due to either
	// a delete call on the scaledObject, or an update call. The activeScaledObjects
	// is a shared cache for the current valid scaledObjects.
	cache *scaleHandlerSharedCache
}

const defaultPollingInterval = 30

// NewScaleHandler creates a ScaleHandler object
func NewScaleHandler(koreClient clientset.Interface, kubeClient kubernetes.Interface) *ScaleHandler {
	handler := &ScaleHandler{
		koreClient: koreClient,
		kubeClient: kubeClient,
		workqueue:  workqueue.NewNamedDelayingQueue("ScaledObjects"),
		cache: &scaleHandlerSharedCache{
			activeScaledObjects: make(map[string]*kore_v1alpha1.ScaledObject),
			opsLock:             sync.RWMutex{},
		},
	}

	return handler
}

// Run starts a goroutine that dequeues workitems from the workqueue
func (h *ScaleHandler) Run(stopCh <-chan struct{}) {
	defer h.workqueue.ShutDown()

	log.Info("Starting ScaleHandler workqueue")
	go wait.Until(h.handleScale, time.Second, stopCh)
	<-stopCh
	log.Info("Shutting down ScaleHaldner workqueue")
}

// handleScale just calls processNextItem in a forever loop
// until a shutdown signal is raised.
func (h *ScaleHandler) handleScale() {
	for h.processNextItem() {
	}
}

// processNextItem pulls items off the workqueue and checks for their scale state.
// It'll also re-enqueue the item with its pollingInterval after it's done.
// Returns false only if the queue is shutting down.
func (h *ScaleHandler) processNextItem() bool {
	obj, shutdown := h.workqueue.Get()

	if shutdown {
		// queue is shutting down
		return false
	}

	defer h.workqueue.Done(obj)

	var key string
	var ok bool
	if key, ok = obj.(string); !ok {
		return true
	}

	scaledObject := h.cache.get(key)

	if scaledObject == nil {
		return true
	}

	enqueueAfter := time.Second * time.Duration(defaultPollingInterval)

	if scaledObject.Spec.PollingInterval != nil {
		enqueueAfter = time.Second * time.Duration(*scaledObject.Spec.PollingInterval)
	}

	defer h.workqueue.AddAfter(obj, enqueueAfter)

	deploymentName := scaledObject.Spec.ScaleTargetRef.DeploymentName
	if deploymentName == "" {
		log.Infof("Notified about ScaledObject with missing deployment name: %s", scaledObject.GetName())
		return true
	}

	deployment, err := h.kubeClient.AppsV1().Deployments(scaledObject.GetNamespace()).Get(deploymentName, meta_v1.GetOptions{})
	if err != nil {
		log.Errorf("Error getting deployment: %s", err)
		return true
	}

	resolvedSecrets, err := h.resolveSecrets(deployment)
	if err != nil {
		log.Errorf("Error resolving secrets for deployment: %s", err)
		return true
	}

	var scaleDecision int32

	for i, trigger := range scaledObject.Spec.Triggers {
		scaler, err := getScaler(trigger, resolvedSecrets)
		if err != nil {
			log.Errorf("error for trigger #%d: %s", i, err)
			continue
		}

		sd, err := scaler.GetScaleDecision()
		if err != nil {
			log.Errorf("error getting scale decision for trigger #%d: %s", i, err)
			continue
		}

		scaleDecision += sd
	}

	log.Infof("scaledObject: %s, target deployment: %s, scale decision: %d", scaledObject.GetName(), deployment.GetName(), scaleDecision)

	h.scaleDeployment(deployment, scaleDecision)

	return true
}

// WatchScaledObject enqueues the ScaledObject into the work queue
func (h *ScaleHandler) WatchScaledObject(scaledObject *kore_v1alpha1.ScaledObject) {
	scaledObjectKey, err := cache.MetaNamespaceKeyFunc(scaledObject)
	if err != nil {
		log.Errorf("Cannot get key for scaledObject (%s/%s)", scaledObject.GetNamespace(), scaledObject.GetName())
		return
	}

	h.cache.set(scaledObjectKey, scaledObject)
	h.workqueue.Add(scaledObjectKey)
}

// StopWatchingScaledObject deletes the scaledObject from the cache
func (h *ScaleHandler) StopWatchingScaledObject(scaledObject *kore_v1alpha1.ScaledObject) {
	scaledObjectKey, err := cache.MetaNamespaceKeyFunc(scaledObject)
	if err != nil {
		log.Errorf("Cannot get key for scaledObject (%s/%s)", scaledObject.GetNamespace(), scaledObject.GetName())
		return
	}

	h.cache.delete(scaledObjectKey)
}

func (h *ScaleHandler) scaleDeployment(deployment *apps_v1.Deployment, scaleDecision int32) {
	if *deployment.Spec.Replicas != scaleDecision {
		currentReplicas := *deployment.Spec.Replicas
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
			log.Infof("Successfully updated deployment (%s/%s) from %d to %d replicas",
				deployment.GetNamespace(),
				deployment.GetName(),
				currentReplicas,
				*deployment.Spec.Replicas)
		}
	} else {
		log.Infof("Current replica count for deployment (%s/%s) is the same as update replica count. Skipping..",
			deployment.GetNamespace(),
			deployment.GetName())
	}
}

func (h *ScaleHandler) resolveSecrets(deployment *apps_v1.Deployment) (*map[string]string, error) {
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

	return &resolved, nil
}

func (h *ScaleHandler) resolveSecretValue(secretKeyRef *core_v1.SecretKeySelector, keyName, namespace string) (string, error) {
	secretCollection, err := h.kubeClient.CoreV1().Secrets(namespace).Get(secretKeyRef.Name, meta_v1.GetOptions{})

	if err != nil {
		return "", err
	}

	return string(secretCollection.Data[keyName]), nil
}

func getScaler(trigger kore_v1alpha1.ScaleTriggers, resolvedSecrets *map[string]string) (Scaler, error) {
	switch trigger.Type {
	case "azure-queue":
		return &azureQueueScaler{
			metadata:        &trigger.Metadata,
			resolvedSecrets: resolvedSecrets,
		}, nil
	default:
		return nil, fmt.Errorf("no scaler found for type: %s", trigger.Type)
	}
}
