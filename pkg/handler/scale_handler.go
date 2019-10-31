package handler

import (
	"context"
	"fmt"

	kedav1alpha1 "github.com/kedacore/keda/pkg/apis/keda/v1alpha1"
	"github.com/kedacore/keda/pkg/scalers"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// ScaleHandler encapsulates the logic of calling the right scalers for
// each ScaledObject and making the final scale decision and operation
type ScaleHandler struct {
	client           client.Client
	logger           logr.Logger
	reconcilerScheme *runtime.Scheme
}

const (
	// Default polling interval for a ScaledObject triggers if no pollingInterval is defined.
	defaultPollingInterval = 30
	// Default cooldown period for a deployment if no cooldownPeriod is defined on the scaledObject
	defaultCooldownPeriod = 5 * 60 // 5 minutes
)

// NewScaleHandler creates a ScaleHandler object
func NewScaleHandler(client client.Client, reconcilerScheme *runtime.Scheme) *ScaleHandler {
	handler := &ScaleHandler{
		client:           client,
		logger:           logf.Log.WithName("scalehandler"),
		reconcilerScheme: reconcilerScheme,
	}
	return handler
}

func (h *ScaleHandler) updateScaledObjectStatus(scaledObject *kedav1alpha1.ScaledObject) error {
	err := h.client.Status().Update(context.TODO(), scaledObject)
	if err != nil {
		h.logger.Error(err, "Error updating scaledObject status")
		return err
	}
	return nil
}

func (h *ScaleHandler) resolveEnv(container *corev1.Container, namespace string) (map[string]string, error) {
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
					h.logger.V(1).Info("cannot resolve env %s to a value. fieldRef and resourceFieldRef env are skipped", envVar.Name)
					continue
				}

			}
			resolved[envVar.Name] = value
		}

	}

	return resolved, nil
}

func (h *ScaleHandler) resolveConfigMap(configMapRef *corev1.ConfigMapEnvSource, namespace string) (map[string]string, error) {
	configMap := &corev1.ConfigMap{}
	err := h.client.Get(context.TODO(), types.NamespacedName{Name: configMapRef.Name, Namespace: namespace}, configMap)
	if err != nil {
		return nil, err
	}
	return configMap.Data, nil
}

func (h *ScaleHandler) resolveSecretMap(secretMapRef *corev1.SecretEnvSource, namespace string) (map[string]string, error) {
	secret := &corev1.Secret{}
	err := h.client.Get(context.TODO(), types.NamespacedName{Name: secretMapRef.Name, Namespace: namespace}, secret)
	if err != nil {
		return nil, err
	}

	secretsStr := make(map[string]string)
	for k, v := range secret.Data {
		secretsStr[k] = string(v)
	}
	return secretsStr, nil
}

func (h *ScaleHandler) resolveSecretValue(secretKeyRef *corev1.SecretKeySelector, keyName, namespace string) (string, error) {
	secret := &corev1.Secret{}
	err := h.client.Get(context.TODO(), types.NamespacedName{Name: secretKeyRef.Name, Namespace: namespace}, secret)
	if err != nil {
		return "", err
	}
	return string(secret.Data[keyName]), nil

}

func (h *ScaleHandler) resolveConfigValue(configKeyRef *corev1.ConfigMapKeySelector, keyName, namespace string) (string, error) {
	configMap := &corev1.ConfigMap{}
	err := h.client.Get(context.TODO(), types.NamespacedName{Name: configKeyRef.Name, Namespace: namespace}, configMap)
	if err != nil {
		return "", err
	}
	return string(configMap.Data[keyName]), nil
}

// GetDeploymentScalers returns list of Scalers and Deployment for the specified ScaledObject
func (h *ScaleHandler) GetDeploymentScalers(scaledObject *kedav1alpha1.ScaledObject) ([]scalers.Scaler, *appsv1.Deployment, error) {
	scalers := []scalers.Scaler{}

	deploymentName := scaledObject.Spec.ScaleTargetRef.DeploymentName
	if deploymentName == "" {
		return scalers, nil, fmt.Errorf("notified about ScaledObject with missing deployment name: %s", scaledObject.GetName())
	}

	deployment := &appsv1.Deployment{}
	err := h.client.Get(context.TODO(), types.NamespacedName{Name: deploymentName, Namespace: scaledObject.GetNamespace()}, deployment)
	if err != nil {
		return scalers, nil, fmt.Errorf("error getting deployment: %s", err)
	}

	resolvedEnv, err := h.resolveDeploymentEnv(deployment, scaledObject.Spec.ScaleTargetRef.ContainerName)
	if err != nil {
		return scalers, nil, fmt.Errorf("error resolving secrets for deployment: %s", err)
	}

	for i, trigger := range scaledObject.Spec.Triggers {
		authParams, podIdentity := h.parseDeploymentAuthRef(trigger.AuthenticationRef, scaledObject, deployment)
		scaler, err := h.getScaler(scaledObject.Name, scaledObject.Namespace, trigger.Type, resolvedEnv, trigger.Metadata, authParams, podIdentity)
		if err != nil {
			return scalers, nil, fmt.Errorf("error getting scaler for trigger #%d: %s", i, err)
		}

		scalers = append(scalers, scaler)
	}

	return scalers, deployment, nil
}

func (h *ScaleHandler) getJobScalers(scaledObject *kedav1alpha1.ScaledObject) ([]scalers.Scaler, error) {
	scalers := []scalers.Scaler{}

	resolvedEnv, err := h.resolveJobEnv(scaledObject)
	if err != nil {
		return scalers, fmt.Errorf("error resolving secrets for job: %s", err)
	}

	for i, trigger := range scaledObject.Spec.Triggers {
		authParams, podIdentity := h.parseJobAuthRef(trigger.AuthenticationRef, scaledObject)
		scaler, err := h.getScaler(scaledObject.Name, scaledObject.Namespace, trigger.Type, resolvedEnv, trigger.Metadata, authParams, podIdentity)
		if err != nil {
			return scalers, fmt.Errorf("error getting scaler for trigger #%d: %s", i, err)
		}

		scalers = append(scalers, scaler)
	}

	return scalers, nil
}

func (h *ScaleHandler) resolveAuthSecret(name, namespace, key string) string {
	if name == "" || namespace == "" || key == "" {
		h.logger.Error(fmt.Errorf("Error trying to get secret"), "name, namespace and key are required", "Secret.Namespace", namespace, "Secret.Name", name, "key", key)
		return ""
	}

	secret := &corev1.Secret{}
	err := h.client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, secret)
	if err != nil {
		h.logger.Error(err, "Error trying to get secret from namespace", "Secret.Namespace", namespace, "Secret.Name", name)
		return ""
	}
	result := secret.Data[key]

	if result == nil {
		return ""
	}

	return string(result)
}

func (h *ScaleHandler) parseAuthRef(triggerAuthRef kedav1alpha1.ScaledObjectAuthRef, scaledObject *kedav1alpha1.ScaledObject, resolveEnv func(string, string) string) (map[string]string, string) {
	result := make(map[string]string)
	podIdentity := ""

	if triggerAuthRef.Name != "" {
		triggerAuth := &kedav1alpha1.TriggerAuthentication{}
		err := h.client.Get(context.TODO(), types.NamespacedName{Name: triggerAuthRef.Name, Namespace: scaledObject.Namespace}, triggerAuth)
		if err != nil {
			h.logger.Error(err, "Error getting triggerAuth", "triggerAuthRef.Name", triggerAuthRef.Name)
		} else {
			podIdentity = string(triggerAuth.Spec.PodIdentity.Provider)
			if triggerAuth.Spec.Env != nil {
				for _, e := range triggerAuth.Spec.Env {
					result[e.Parameter] = resolveEnv(e.Name, e.ContainerName)
				}
			}
			if triggerAuth.Spec.SecretTargetRef != nil {
				for _, e := range triggerAuth.Spec.SecretTargetRef {
					result[e.Parameter] = h.resolveAuthSecret(e.Name, scaledObject.Namespace, e.Key)
				}
			}
		}
	}

	return result, podIdentity
}

func (h *ScaleHandler) getScaler(name, namespace, triggerType string, resolvedEnv, triggerMetadata, authParams map[string]string, podIdentity string) (scalers.Scaler, error) {
	switch triggerType {
	case "azure-queue":
		return scalers.NewAzureQueueScaler(resolvedEnv, triggerMetadata, authParams, podIdentity)
	case "azure-servicebus":
		return scalers.NewAzureServiceBusScaler(resolvedEnv, triggerMetadata)
	case "aws-sqs-queue":
		return scalers.NewAwsSqsQueueScaler(resolvedEnv, triggerMetadata)
	case "aws-cloudwatch":
		return scalers.NewAwsCloudwatchScaler(resolvedEnv, triggerMetadata)
	case "kafka":
		return scalers.NewKafkaScaler(resolvedEnv, triggerMetadata)
	case "rabbitmq":
		return scalers.NewRabbitMQScaler(resolvedEnv, triggerMetadata)
	case "azure-eventhub":
		return scalers.NewAzureEventHubScaler(resolvedEnv, triggerMetadata)
	case "prometheus":
		return scalers.NewPrometheusScaler(resolvedEnv, triggerMetadata)
	case "redis":
		return scalers.NewRedisScaler(resolvedEnv, triggerMetadata)
	case "gcp-pubsub":
		return scalers.NewPubSubScaler(resolvedEnv, triggerMetadata)
	case "external":
		return scalers.NewExternalScaler(name, namespace, resolvedEnv, triggerMetadata)
	case "liiklus":
		return scalers.NewLiiklusScaler(resolvedEnv, triggerMetadata)
	case "stan":
		return scalers.NewStanScaler(resolvedEnv, triggerMetadata)
	default:
		return nil, fmt.Errorf("no scaler found for type: %s", triggerType)
	}
}
