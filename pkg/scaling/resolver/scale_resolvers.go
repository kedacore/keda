package resolver

import (
	"bytes"
	"context"
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
)

const (
	referenceOperator = '$'
	referenceOpener   = '('
	referenceCloser   = ')'
)

// ResolveContainerEnv resolves all environment variables in a container.
// It returns either map of env variable key and value or error if there is any.
func ResolveContainerEnv(client client.Client, logger logr.Logger, podSpec *corev1.PodSpec, containerName, namespace string) (map[string]string, error) {
	if len(podSpec.Containers) < 1 {
		return nil, fmt.Errorf("target object doesn't have containers")
	}

	var container corev1.Container
	if containerName != "" {
		containerWithNameFound := false
		for _, c := range podSpec.Containers {
			if c.Name == containerName {
				container = c
				containerWithNameFound = true
				break
			}
		}

		if !containerWithNameFound {
			return nil, fmt.Errorf("couldn't find container with name %s on Target object", containerName)
		}
	} else {
		container = podSpec.Containers[0]
	}

	return resolveEnv(client, logger, &container, namespace)
}

// ResolveAuthRef provides authentication parameters needed authenticate scaler with the environment.
// based on authentication method define in TriggerAuthentication, authParams and podIdentity is returned
func ResolveAuthRef(client client.Client, logger logr.Logger, triggerAuthRef *kedav1alpha1.ScaledObjectAuthRef, podSpec *corev1.PodSpec, namespace string) (map[string]string, kedav1alpha1.PodIdentityProvider) {
	result := make(map[string]string)
	var podIdentity kedav1alpha1.PodIdentityProvider

	if namespace != "" && triggerAuthRef != nil && triggerAuthRef.Name != "" {
		triggerAuth := &kedav1alpha1.TriggerAuthentication{}
		err := client.Get(context.TODO(), types.NamespacedName{Name: triggerAuthRef.Name, Namespace: namespace}, triggerAuth)
		if err != nil {
			logger.Error(err, "Error getting triggerAuth", "triggerAuthRef.Name", triggerAuthRef.Name)
		} else {
			if triggerAuth.Spec.PodIdentity != nil {
				podIdentity = triggerAuth.Spec.PodIdentity.Provider
			}
			if triggerAuth.Spec.Env != nil {
				for _, e := range triggerAuth.Spec.Env {
					if podSpec == nil {
						result[e.Parameter] = ""
						continue
					}
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
			if triggerAuth.Spec.HashiCorpVault != nil && len(triggerAuth.Spec.HashiCorpVault.Secrets) > 0 {
				vault := NewHashicorpVaultHandler(triggerAuth.Spec.HashiCorpVault)
				err := vault.Initialize(logger)
				if err != nil {
					logger.Error(err, "Error authenticate to Vault", "triggerAuthRef.Name", triggerAuthRef.Name)
				} else {
					for _, e := range triggerAuth.Spec.HashiCorpVault.Secrets {
						secret, err := vault.Read(e.Path)
						if err != nil {
							logger.Error(err, "Error trying to read secret from Vault", "triggerAuthRef.Name", triggerAuthRef.Name,
								"secret.path", e.Path)
							continue
						}

						result[e.Parameter] = resolveVaultSecret(logger, secret.Data, e.Key)
					}

					vault.Stop()
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
				configMap, err := resolveConfigMap(client, source.ConfigMapRef, namespace)
				switch {
				case err == nil:
					for k, v := range configMap {
						resolved[k] = v
					}
				case source.ConfigMapRef.Optional != nil && *source.ConfigMapRef.Optional:
					// ignore error when ConfigMap is marked as optional
					continue
				default:
					return nil, fmt.Errorf("error reading config ref %s on namespace %s/: %s", source.ConfigMapRef, namespace, err)
				}
			} else if source.SecretRef != nil {
				secretsMap, err := resolveSecretMap(client, source.SecretRef, namespace)
				switch {
				case err == nil:
					for k, v := range secretsMap {
						resolved[k] = v
					}
				case source.SecretRef.Optional != nil && *source.SecretRef.Optional:
					// ignore error when Secret is marked as optional
					continue
				default:
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
				// resolve syntax if environment variables have dependent variables
				value = resolveEnvValue(envVar.Value, resolved)
			} else if envVar.ValueFrom != nil {
				// env is an EnvVarSource, that can be on of the 4 below
				switch {
				case envVar.ValueFrom.SecretKeyRef != nil:
					// env is a secret selector
					value, err = resolveSecretValue(client, envVar.ValueFrom.SecretKeyRef, envVar.ValueFrom.SecretKeyRef.Key, namespace)
					if err != nil {
						return nil, fmt.Errorf("error resolving secret name %s for env %s in namespace %s",
							envVar.ValueFrom.SecretKeyRef,
							envVar.Name,
							namespace)
					}
				case envVar.ValueFrom.ConfigMapKeyRef != nil:
					// env is a configMap selector
					value, err = resolveConfigValue(client, envVar.ValueFrom.ConfigMapKeyRef, envVar.ValueFrom.ConfigMapKeyRef.Key, namespace)
					if err != nil {
						return nil, fmt.Errorf("error resolving config %s for env %s in namespace %s",
							envVar.ValueFrom.ConfigMapKeyRef,
							envVar.Name,
							namespace)
					}
				default:
					logger.V(1).Info("cannot resolve env %s to a value. fieldRef and resourceFieldRef env are skipped", envVar.Name)
					continue
				}
			}
			resolved[envVar.Name] = value
		}
	}
	return resolved, nil
}

func resolveEnvValue(value string, env map[string]string) string {
	var buf bytes.Buffer
	checkpoint := 0

	for cursor := 0; cursor < len(value); cursor++ {
		if value[cursor] == referenceOperator && cursor+3 < len(value) {
			// append value contents since the last checkpoint into the buffer
			buf.WriteString(value[checkpoint:cursor])

			var content string
			length := 1

		OperatorSwitch:
			switch value[cursor+1] {
			case referenceOperator:
				// escaped reference
				content = string(referenceOperator)
			case referenceOpener:
				// read dependent reference;2 indicates operator and opener length
				for i := 2; i < len(value)-cursor; i++ {
					if value[cursor+i] == referenceCloser {
						dependentEnvKey := value[cursor+2 : cursor+i]
						dependentEnvValue, ok := env[dependentEnvKey]
						if ok {
							content = dependentEnvValue
						} else {
							content = string(referenceOperator) + string(referenceOpener) + dependentEnvKey + string(referenceCloser)
						}
						length = i
						break OperatorSwitch
					}
				}
				// not match reference closer
				content = string(referenceOperator) + string(referenceOpener)
			default:
				content = string(referenceOperator) + string(value[cursor+1])
			}

			// append resolved env value into the buffer
			buf.WriteString(content)
			// make cursor continue scan
			cursor += length
			checkpoint = cursor + 1
		}
	}
	return buf.String() + value[checkpoint:]
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
	return configMap.Data[keyName], nil
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

func resolveVaultSecret(logger logr.Logger, data map[string]interface{}, key string) string {
	if v2Data, ok := data["data"].(map[string]interface{}); ok {
		if value, ok := v2Data[key]; ok {
			if s, ok := value.(string); ok {
				return s
			}
		} else {
			logger.Error(fmt.Errorf("key '%s' not found", key), "Error trying to get key from Vault secret")
			return ""
		}
	}

	logger.Error(fmt.Errorf("unable to convert Vault Data value"), "Error trying to convert Data secret vaule")
	return ""
}
