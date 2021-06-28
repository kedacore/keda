package resolver

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/apis/duck"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
)

const (
	referenceOperator = '$'
	referenceOpener   = '('
	referenceCloser   = ')'
)

// ResolveScaleTargetPodSpec for given scalableObject inspects the scale target workload,
// which could be almost any k8s resource (Deployment, StatefulSet, CustomResource...)
// and for the given resource returns *corev1.PodTemplateSpec and a name of the container
// which is being used for referencing environment variables
func ResolveScaleTargetPodSpec(kubeClient client.Client, logger logr.Logger, scalableObject interface{}) (*corev1.PodTemplateSpec, string, error) {
	switch obj := scalableObject.(type) {
	case *kedav1alpha1.ScaledObject:
		// Try to get a real object instance for better cache usage, but fall back to an Unstructured if needed.
		podTemplateSpec := corev1.PodTemplateSpec{}
		gvk := obj.Status.ScaleTargetGVKR.GroupVersionKind()
		objKey := client.ObjectKey{Namespace: obj.Namespace, Name: obj.Spec.ScaleTargetRef.Name}
		switch {
		// For core types, use a typed client so we get an informer-cache-backed Get to reduce API load.
		case gvk.Group == "apps" && gvk.Kind == "Deployment":
			deployment := &appsv1.Deployment{}
			if err := kubeClient.Get(context.TODO(), objKey, deployment); err != nil {
				// resource doesn't exist
				logger.Error(err, "Target deployment doesn't exist", "resource", gvk.String(), "name", objKey.Name)
				return nil, "", err
			}
			podTemplateSpec.ObjectMeta = deployment.ObjectMeta
			podTemplateSpec.Spec = deployment.Spec.Template.Spec
		case gvk.Group == "apps" && gvk.Kind == "StatefulSet":
			statefulSet := &appsv1.StatefulSet{}
			if err := kubeClient.Get(context.TODO(), objKey, statefulSet); err != nil {
				// resource doesn't exist
				logger.Error(err, "Target deployment doesn't exist", "resource", gvk.String(), "name", objKey.Name)
				return nil, "", err
			}
			podTemplateSpec.ObjectMeta = statefulSet.ObjectMeta
			podTemplateSpec.Spec = statefulSet.Spec.Template.Spec
		default:
			unstruct := &unstructured.Unstructured{}
			unstruct.SetGroupVersionKind(gvk)
			if err := kubeClient.Get(context.TODO(), objKey, unstruct); err != nil {
				// resource doesn't exist
				logger.Error(err, "Target resource doesn't exist", "resource", gvk.String(), "name", objKey.Name)
				return nil, "", err
			}
			withPods := &duckv1.WithPod{}
			if err := duck.FromUnstructured(unstruct, withPods); err != nil {
				logger.Error(err, "Cannot convert Unstructured into PodSpecable Duck-type", "object", unstruct)
			}
			podTemplateSpec.ObjectMeta = withPods.ObjectMeta
			podTemplateSpec.Spec = withPods.Spec.Template.Spec
		}

		if podTemplateSpec.Spec.Containers == nil || len(podTemplateSpec.Spec.Containers) == 0 {
			logger.V(1).Info("There aren't any containers found in the ScaleTarget, therefore it is no possible to inject environment properties", "resource", gvk.String(), "name", obj.Spec.ScaleTargetRef.Name)
			return nil, "", nil
		}

		return &podTemplateSpec, obj.Spec.ScaleTargetRef.EnvSourceContainerName, nil
	case *kedav1alpha1.ScaledJob:
		return &obj.Spec.JobTargetRef.Template, obj.Spec.EnvSourceContainerName, nil
	default:
		return nil, "", fmt.Errorf("unknown scalable object type %v", scalableObject)
	}
}

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

// ResolveAuthRefAndPodIdentity provides authentication parameters and pod identity needed authenticate scaler with the environment.
func ResolveAuthRefAndPodIdentity(client client.Client, logger logr.Logger, triggerAuthRef *kedav1alpha1.ScaledObjectAuthRef, podTemplateSpec *corev1.PodTemplateSpec, namespace string) (map[string]string, kedav1alpha1.PodIdentityProvider, error) {
	if podTemplateSpec != nil {
		authParams, podIdentity := resolveAuthRef(client, logger, triggerAuthRef, &podTemplateSpec.Spec, namespace)

		if podIdentity == kedav1alpha1.PodIdentityProviderAwsEKS {
			serviceAccountName := podTemplateSpec.Spec.ServiceAccountName
			serviceAccount := &corev1.ServiceAccount{}
			err := client.Get(context.TODO(), types.NamespacedName{Name: serviceAccountName, Namespace: namespace}, serviceAccount)
			if err != nil {
				return nil, kedav1alpha1.PodIdentityProviderNone, fmt.Errorf("error getting service account: %s", err)
			}
			authParams["awsRoleArn"] = serviceAccount.Annotations[kedav1alpha1.PodIdentityAnnotationEKS]
		} else if podIdentity == kedav1alpha1.PodIdentityProviderAwsKiam {
			authParams["awsRoleArn"] = podTemplateSpec.ObjectMeta.Annotations[kedav1alpha1.PodIdentityAnnotationKiam]
		}
		return authParams, podIdentity, nil
	}

	authParams, _ := resolveAuthRef(client, logger, triggerAuthRef, nil, namespace)
	return authParams, kedav1alpha1.PodIdentityProviderNone, nil
}

// resolveAuthRef provides authentication parameters needed authenticate scaler with the environment.
// based on authentication method defined in TriggerAuthentication, authParams and podIdentity is returned
func resolveAuthRef(client client.Client, logger logr.Logger, triggerAuthRef *kedav1alpha1.ScaledObjectAuthRef, podSpec *corev1.PodSpec, namespace string) (map[string]string, kedav1alpha1.PodIdentityProvider) {
	result := make(map[string]string)
	var podIdentity kedav1alpha1.PodIdentityProvider

	if namespace != "" && triggerAuthRef != nil && triggerAuthRef.Name != "" {
		triggerAuthSpec, triggerNamespace, err := getTriggerAuthSpec(client, triggerAuthRef, namespace)
		if err != nil {
			logger.Error(err, "Error getting triggerAuth", "triggerAuthRef.Name", triggerAuthRef.Name)
		} else {
			if triggerAuthSpec.PodIdentity != nil {
				podIdentity = triggerAuthSpec.PodIdentity.Provider
			}
			if triggerAuthSpec.Env != nil {
				for _, e := range triggerAuthSpec.Env {
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
			if triggerAuthSpec.SecretTargetRef != nil {
				for _, e := range triggerAuthSpec.SecretTargetRef {
					result[e.Parameter] = resolveAuthSecret(client, logger, e.Name, triggerNamespace, e.Key)
				}
			}
			if triggerAuthSpec.HashiCorpVault != nil && len(triggerAuthSpec.HashiCorpVault.Secrets) > 0 {
				vault := NewHashicorpVaultHandler(triggerAuthSpec.HashiCorpVault)
				err := vault.Initialize(logger)
				if err != nil {
					logger.Error(err, "Error authenticate to Vault", "triggerAuthRef.Name", triggerAuthRef.Name)
				} else {
					for _, e := range triggerAuthSpec.HashiCorpVault.Secrets {
						secret, err := vault.Read(e.Path)
						if err != nil {
							logger.Error(err, "Error trying to read secret from Vault", "triggerAuthRef.Name", triggerAuthRef.Name,
								"secret.path", e.Path)
						} else {
							if secret == nil {
								// sometimes there is no error, but `vault.Read(e.Path)` is not being able to parse the secret and returns nil
								logger.Error(fmt.Errorf("unable to parse secret, is the provided path correct?"), "Error trying to read secret from Vault",
									"triggerAuthRef.Name", triggerAuthRef.Name, "secret.path", e.Path)
							} else {
								result[e.Parameter] = resolveVaultSecret(logger, secret.Data, e.Key)
							}
						}
					}

					vault.Stop()
				}
			}
		}
	}

	return result, podIdentity
}

var clusterObjectNamespaceCache *string

func getClusterObjectNamespace() (string, error) {
	// Check if a cached value is available.
	if clusterObjectNamespaceCache != nil {
		return *clusterObjectNamespaceCache, nil
	}
	env := os.Getenv("KEDA_CLUSTER_OBJECT_NAMESPACE")
	if env != "" {
		clusterObjectNamespaceCache = &env
		return env, nil
	}
	data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return "", err
	}
	strData := string(data)
	clusterObjectNamespaceCache = &strData
	return strData, nil
}

func getTriggerAuthSpec(client client.Client, triggerAuthRef *kedav1alpha1.ScaledObjectAuthRef, namespace string) (*kedav1alpha1.TriggerAuthenticationSpec, string, error) {
	if triggerAuthRef.Kind == "" || triggerAuthRef.Kind == "TriggerAuthentication" {
		triggerAuth := &kedav1alpha1.TriggerAuthentication{}
		err := client.Get(context.TODO(), types.NamespacedName{Name: triggerAuthRef.Name, Namespace: namespace}, triggerAuth)
		if err != nil {
			return nil, "", err
		}
		return &triggerAuth.Spec, namespace, nil
	} else if triggerAuthRef.Kind == "ClusterTriggerAuthentication" {
		clusterNamespace, err := getClusterObjectNamespace()
		if err != nil {
			return nil, "", err
		}
		triggerAuth := &kedav1alpha1.ClusterTriggerAuthentication{}
		err = client.Get(context.TODO(), types.NamespacedName{Name: triggerAuthRef.Name}, triggerAuth)
		if err != nil {
			return nil, "", err
		}
		return &triggerAuth.Spec, clusterNamespace, nil
	}
	return nil, "", fmt.Errorf("unknown trigger auth kind %s", triggerAuthRef.Kind)
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
					logger.V(1).Info("cannot resolve env to a value. fieldRef and resourceFieldRef env are skipped", "env-var-name", envVar.Name)
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
