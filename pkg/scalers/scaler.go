package scalers

import (
	"fmt"

	apps_v1 "k8s.io/api/apps/v1"
	core_v1 "k8s.io/api/core/v1"

	kore_v1alpha1 "github.com/Azure/Kore/pkg/apis/kore/v1alpha1"
	clientset "github.com/Azure/Kore/pkg/client/clientset/versioned"
	log "github.com/Sirupsen/logrus"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type ScaleManager struct {
	scaledObject *kore_v1alpha1.ScaledObject
	koreClient   clientset.Interface
	kubeClient   kubernetes.Interface
}

type Scaler interface {
	GetScaleDecision() (int32, error)
}

func NewScaleManager(scaledObject *kore_v1alpha1.ScaledObject, koreClient clientset.Interface, kubeClient kubernetes.Interface) *ScaleManager {
	m := &ScaleManager{
		scaledObject: scaledObject,
		koreClient:   koreClient,
		kubeClient:   kubeClient,
	}

	return m
}

func (m *ScaleManager) HandleScale() {
	deploymentName := m.scaledObject.Spec.ScaleTargetRef.DeploymentName
	if deploymentName == "" {
		log.Infof("Notified about ScaledObject with missing deployment name: %s", m.scaledObject.GetName())
		return
	}

	deployment, err := m.kubeClient.AppsV1().Deployments(m.scaledObject.GetNamespace()).Get(deploymentName, meta_v1.GetOptions{})
	if err != nil {
		log.Errorf("Error getting deployment: %s", err)
		return
	}

	resolvedSecrets, err := m.resolveSecrets(deployment)
	if err != nil {
		log.Errorf("Error resolving secrets for deployment: %s", err)
	}

	var scaleDecision int32

	for i, trigger := range m.scaledObject.Spec.Triggers {
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

	log.Infof("scaledObject: %s, target deployment: %s, scale decision: %d", m.scaledObject.GetName(), deployment.GetName(), scaleDecision)
}

func (m *ScaleManager) resolveSecrets(deployment *apps_v1.Deployment) (*map[string]string, error) {
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
			value, err := m.resolveSecretValue(envVar.ValueFrom.SecretKeyRef, envVar.Name, deployment.GetNamespace())
			if err != nil {
				return nil, err
			}

			resolved[envVar.Name] = value
		}
	}

	return &resolved, nil
}

func (m *ScaleManager) resolveSecretValue(secretKeyRef *core_v1.SecretKeySelector, keyName, namespace string) (string, error) {
	secretCollection, err := m.kubeClient.CoreV1().Secrets(namespace).Get(secretKeyRef.Name, meta_v1.GetOptions{})

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

	return nil, fmt.Errorf("no scaler found for type: %", trigger.Type)
}
