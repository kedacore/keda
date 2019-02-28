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
	koreClient          clientset.Interface
	kubeClient          kubernetes.Interface
	workqueue           workqueue.DelayingInterface
	activeScaledObjects map[string]*kore_v1alpha1.ScaledObject
	opsLock             sync.RWMutex
}

// NewScaleHandler creates a ScaleHandler object
func NewScaleHandler(koreClient clientset.Interface, kubeClient kubernetes.Interface) *ScaleHandler {
	handler := &ScaleHandler{
		koreClient:          koreClient,
		kubeClient:          kubeClient,
		workqueue:           workqueue.NewNamedDelayingQueue("ScaledObjects"),
		activeScaledObjects: make(map[string]*kore_v1alpha1.ScaledObject),
		opsLock:             sync.RWMutex{},
	}

	return handler
}

func (h *ScaleHandler) Run(stopCh <-chan struct{}) {
	defer h.workqueue.ShutDown()

	log.Info("Starting ScaleHandler workqueue")
	go wait.Until(h.handleScale, time.Second, stopCh)
	<-stopCh
	log.Info("Shutting down ScaleHaldner workqueue")
}

func (h *ScaleHandler) handleScale() {
	obj, shutdown := h.workqueue.Get()

	if shutdown {
		// queue is shutting down
		return
	}

	defer h.workqueue.Done(obj)

	var key string
	var ok bool
	if key, ok = obj.(string); !ok {
		return
	}

	h.opsLock.RLock()
	scaledObject := h.activeScaledObjects[key]
	h.opsLock.RUnlock()

	if scaledObject == nil {
		return
	}

	defer h.workqueue.AddAfter(obj, time.Second * 30)

	deploymentName := scaledObject.Spec.ScaleTargetRef.DeploymentName
	if deploymentName == "" {
		log.Infof("Notified about ScaledObject with missing deployment name: %s", scaledObject.GetName())
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
}

// WatchScaledObject enqueues the ScaledObject into the work queue
func (h *ScaleHandler) WatchScaledObject(scaledObject *kore_v1alpha1.ScaledObject) {
	scaledObjectKey, err := cache.MetaNamespaceKeyFunc(scaledObject)
	if err != nil {
		log.Errorf("Cannot get key for scaledObject (%s/%s)", scaledObject.GetNamespace(), scaledObject.GetName())
		return
	}

	h.opsLock.Lock()
	h.activeScaledObjects[scaledObjectKey] = scaledObject
	h.opsLock.Unlock()

	h.workqueue.Add(scaledObjectKey)
}

func (h *ScaleHandler) StopWatchingScaledObject(scaledObject *kore_v1alpha1.ScaledObject) {
	scaledObjectKey, err := cache.MetaNamespaceKeyFunc(scaledObject)
	if err != nil {
		log.Errorf("Cannot get key for scaledObject (%s/%s)", scaledObject.GetNamespace(), scaledObject.GetName())
		return
	}

	h.opsLock.Lock()
	defer h.opsLock.Unlock()

	delete(h.activeScaledObjects, scaledObjectKey)
}

func (h *ScaleHandler) scaleDeployment(deployment *apps_v1.Deployment, scaleDecision int32) {
	if *deployment.Spec.Replicas != scaleDecision {

		// TODO: we should also have a "status" for the ScaledObject
		// TODO: where we can store information like:
		// TODO: LastScaleTime, LastActiveTime, CurrentReplicas, and DesiredReplicas
		// scaledObject.Status.LastScaleTime = &currentTime
		// scaledObject.Status.LastActiveTime = &currentTime
		// scaledObject.Status.CurrentReplicas = *deploymentCopy.Spec.Replicas
		// scaledObject.Status.DesiredReplicas = scaleDecision

		*deployment.Spec.Replicas = scaleDecision
		deployment, err := h.kubeClient.AppsV1().Deployments(deployment.GetNamespace()).Update(deployment)
		if err != nil {
			log.Errorf("Error updating replica count on deployment (%s/%s) from %d to %d. Error: %s",
				deployment.GetNamespace(),
				deployment.GetName(),
				*deployment.Spec.Replicas,
				*deployment.Spec.Replicas,
				err)
		} else {
			log.Infof("Successfully updated deployment (%s/%s) from %d to %d replicas",
				deployment.GetNamespace(),
				deployment.GetName(),
				*deployment.Spec.Replicas,
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
	if trigger.Type == "azure-queue" {
		return &azureQueueScaler{
			metadata:        &trigger.Metadata,
			resolvedSecrets: resolvedSecrets,
		}, nil
	}

	return nil, fmt.Errorf("no scaler found for type: %s", trigger.Type)
}
