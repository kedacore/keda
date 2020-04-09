package resolver

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	kedav1alpha1 "github.com/kedacore/keda/pkg/apis/keda/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ResolveContainerEnv(client client.Client, logger logr.Logger, podSpec *corev1.PodSpec, containerName, namespace string) (map[string]string, error) {

	if len(podSpec.Containers) < 1 {
		return nil, fmt.Errorf("Target object doesn't have containers")
	}

	var container corev1.Container
	if containerName != "" {
		for _, c := range podSpec.Containers {
			if c.Name == containerName {
				container = c
				break
			}
		}

		if &container == nil {
			return nil, fmt.Errorf("Couldn't find container with name %s on Target object", containerName)
		}
	} else {
		container = podSpec.Containers[0]
	}

	return resolveEnv(client, logger, &container, namespace)
}

func ResolveAuthRef(client client.Client, logger logr.Logger, triggerAuthRef *kedav1alpha1.ScaledObjectAuthRef, podSpec *corev1.PodSpec, namespace string) (map[string]string, string) {
	result := make(map[string]string)
	podIdentity := ""

	if triggerAuthRef != nil && triggerAuthRef.Name != "" {
		triggerAuth := &kedav1alpha1.TriggerAuthentication{}
		err := client.Get(context.TODO(), types.NamespacedName{Name: triggerAuthRef.Name, Namespace: namespace}, triggerAuth)
		if err != nil {
			logger.Error(err, "Error getting triggerAuth", "triggerAuthRef.Name", triggerAuthRef.Name)
		} else {
			podIdentity = string(triggerAuth.Spec.PodIdentity.Provider)
			if triggerAuth.Spec.Env != nil {
				for _, e := range triggerAuth.Spec.Env {
					env, err := ResolveContainerEnv(client, logger, podSpec, e.ContainerName, namespace)
					if err != nil {
						result[e.Parameter] = ""
					} else {
						result[e.Parameter] = env[e.Name]
					}
				}
			}
			if triggerAuth.Spec.SecretTargetRef != nil {
				for _, e := range triggerAuth.Spec.SecretTargetRef {
					result[e.Parameter] = resolveAuthSecret(client, logger, e.Name, namespace, e.Key)
				}
			}
		}
	}

	return result, podIdentity
}

func resolveEnv(client client.Client, logger logr.Logger, container *corev1.Container, namespace string) (map[string]string, error) {
	resolved := make(map[string]string)

	if container.EnvFrom != nil {
		for _, source := range container.EnvFrom {
			if source.ConfigMapRef != nil {
				if configMap, err := resolveConfigMap(client, source.ConfigMapRef, namespace); err == nil {
					for k, v := range configMap {
						resolved[k] = v
					}
				} else if source.ConfigMapRef.Optional != nil && *source.ConfigMapRef.Optional {
					// ignore error when ConfigMap is marked as optional
					continue
				} else {
					return nil, fmt.Errorf("error reading config ref %s on namespace %s/: %s", source.ConfigMapRef, namespace, err)
				}
			} else if source.SecretRef != nil {
				if secretsMap, err := resolveSecretMap(client, source.SecretRef, namespace); err == nil {
					for k, v := range secretsMap {
						resolved[k] = v
					}
				} else if source.SecretRef.Optional != nil && *source.SecretRef.Optional {
					// ignore error when Secret is marked as optional
					continue
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
					value, err = resolveSecretValue(client, envVar.ValueFrom.SecretKeyRef, envVar.ValueFrom.SecretKeyRef.Key, namespace)
					if err != nil {
						return nil, fmt.Errorf("error resolving secret name %s for env %s in namespace %s",
							envVar.ValueFrom.SecretKeyRef,
							envVar.Name,
							namespace)
					}
				} else if envVar.ValueFrom.ConfigMapKeyRef != nil {
					// env is a configMap selector
					value, err = resolveConfigValue(client, envVar.ValueFrom.ConfigMapKeyRef, envVar.ValueFrom.ConfigMapKeyRef.Key, namespace)
					if err != nil {
						return nil, fmt.Errorf("error resolving config %s for env %s in namespace %s",
							envVar.ValueFrom.ConfigMapKeyRef,
							envVar.Name,
							namespace)
					}
				} else {
					logger.V(1).Info("cannot resolve env %s to a value. fieldRef and resourceFieldRef env are skipped", envVar.Name)
					continue
				}

			}
			resolved[envVar.Name] = value
		}

	}

	return resolved, nil
}

func resolveConfigMap(client client.Client, configMapRef *corev1.ConfigMapEnvSource, namespace string) (map[string]string, error) {
	configMap := &corev1.ConfigMap{}
	err := client.Get(context.TODO(), types.NamespacedName{Name: configMapRef.Name, Namespace: namespace}, configMap)
	if err != nil {
		return nil, err
	}
	return configMap.Data, nil
}

func resolveSecretMap(client client.Client, secretMapRef *corev1.SecretEnvSource, namespace string) (map[string]string, error) {
	secret := &corev1.Secret{}
	err := client.Get(context.TODO(), types.NamespacedName{Name: secretMapRef.Name, Namespace: namespace}, secret)
	if err != nil {
		return nil, err
	}

	secretsStr := make(map[string]string)
	for k, v := range secret.Data {
		secretsStr[k] = string(v)
	}
	return secretsStr, nil
}

func resolveSecretValue(client client.Client, secretKeyRef *corev1.SecretKeySelector, keyName, namespace string) (string, error) {
	secret := &corev1.Secret{}
	err := client.Get(context.TODO(), types.NamespacedName{Name: secretKeyRef.Name, Namespace: namespace}, secret)
	if err != nil {
		return "", err
	}
	return string(secret.Data[keyName]), nil

}

func resolveConfigValue(client client.Client, configKeyRef *corev1.ConfigMapKeySelector, keyName, namespace string) (string, error) {
	configMap := &corev1.ConfigMap{}
	err := client.Get(context.TODO(), types.NamespacedName{Name: configKeyRef.Name, Namespace: namespace}, configMap)
	if err != nil {
		return "", err
	}
	return string(configMap.Data[keyName]), nil
}

func resolveAuthSecret(client client.Client, logger logr.Logger, name, namespace, key string) string {
	if name == "" || namespace == "" || key == "" {
		logger.Error(fmt.Errorf("error trying to get secret"), "name, namespace and key are required", "Secret.Namespace", namespace, "Secret.Name", name, "key", key)
		return ""
	}

	secret := &corev1.Secret{}
	err := client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, secret)
	if err != nil {
		logger.Error(err, "Error trying to get secret from namespace", "Secret.Namespace", namespace, "Secret.Name", name)
		return ""
	}
	result := secret.Data[key]

	if result == nil {
		return ""
	}

	return string(result)
}
