package scalers

import (
	"context"
	"fmt"
	"time"

	apps_v1 "k8s.io/api/apps/v1"
	core_v1 "k8s.io/api/core/v1"

	kore_v1alpha1 "github.com/Azure/Kore/pkg/apis/kore/v1alpha1"
	clientset "github.com/Azure/Kore/pkg/client/clientset/versioned"
	log "github.com/Sirupsen/logrus"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// ScaleHandler encapsulates the logic of calling the right scalers for
// each ScaledObject and making the final scale decision and operation
type ScaleHandler struct {
	koreClient clientset.Interface
	kubeClient kubernetes.Interface
}

const defaultPollingInterval = 30

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
	go h.handleScaleLoop(ctx, scaledObject)
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

	var scaleDecision int32

	for i, trigger := range scaledObject.Spec.Triggers {
		scaler, err := getScaler(trigger, resolvedSecrets)
		if err != nil {
			log.Errorf("error for trigger #%d: %s", i, err)
			continue
		}

		sd, err := scaler.GetScaleDecision(ctx)
		if err != nil {
			log.Errorf("error getting scale decision for trigger #%d: %s", i, err)
			continue
		}

		scaleDecision += sd
	}

	log.Debugf("scaledObject: %s, target deployment: %s, scale decision: %d", scaledObject.GetName(), deployment.GetName(), scaleDecision)

	h.scaleDeployment(deployment, scaleDecision)

	return
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
