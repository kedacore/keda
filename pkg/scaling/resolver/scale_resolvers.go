/*
Copyright 2021 The KEDA Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package resolver

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/scale"
	"knative.dev/pkg/apis/duck"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/util"
)

const (
	referenceOperator     = '$'
	referenceOpener       = '('
	referenceCloser       = ')'
	boolTrue              = true
	boolFalse             = false
	defaultServiceAccount = "default"
	appsGroup             = "apps"
	deploymentKind        = "Deployment"
	statefulSetKind       = "StatefulSet"
)

var (
	kedaNamespace, _     = util.GetClusterObjectNamespace()
	restrictSecretAccess = util.GetRestrictSecretAccess()
	log                  = logf.Log.WithName("scale_resolvers")
)

// isSecretAccessRestricted returns whether secret access need to be restricted in KEDA namespace
func isSecretAccessRestricted(logger logr.Logger) bool {
	if restrictSecretAccess == "" {
		return boolFalse
	}
	if strings.ToLower(restrictSecretAccess) == strconv.FormatBool(boolTrue) {
		logger.V(1).Info("Secret Access is restricted to be in Cluster Object Namespace, please use ClusterTriggerAuthentication instead of TriggerAuthentication", "Cluster Object Namespace", kedaNamespace, "Env Var", util.RestrictSecretAccessEnvVar, "Env Value", strings.ToLower(restrictSecretAccess))
		return boolTrue
	}
	return boolFalse
}

// ResolveScaleTargetPodSpec for given scalableObject inspects the scale target workload,
// which could be almost any k8s resource (Deployment, StatefulSet, CustomResource...)
// and for the given resource returns *corev1.PodTemplateSpec and a name of the container
// which is being used for referencing environment variables
func ResolveScaleTargetPodSpec(ctx context.Context, kubeClient client.Client, scalableObject interface{}) (*corev1.PodTemplateSpec, string, error) {
	switch obj := scalableObject.(type) {
	case *kedav1alpha1.ScaledObject:
		// Try to get a real object instance for better cache usage, but fall back to an Unstructured if needed.
		podTemplateSpec := corev1.PodTemplateSpec{}

		// trying to prevent operator crashes, due to some race condition, sometimes obj.Status.ScaleTargetGVKR is nil
		// see https://github.com/kedacore/keda/issues/4389
		// Tracking issue: https://github.com/kedacore/keda/issues/4955
		if obj.Status.ScaleTargetGVKR == nil {
			scaledObject := &kedav1alpha1.ScaledObject{}
			err := kubeClient.Get(ctx, types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, scaledObject)
			if err != nil {
				log.Error(err, "failed to get ScaledObject", "name", obj.Name, "namespace", obj.Namespace)
				return nil, "", err
			}
			obj = scaledObject
		}
		if obj.Status.ScaleTargetGVKR == nil {
			err := fmt.Errorf("failed to get ScaledObject.Status.ScaleTargetGVKR, probably invalid ScaledObject cache")
			log.Error(err, "failed to get ScaledObject.Status.ScaleTargetGVKR, probably invalid ScaledObject cache", "scaledObject.Name", obj.Name, "scaledObject.Namespace", obj.Namespace)
			return nil, "", err
		}

		gvk := obj.Status.ScaleTargetGVKR.GroupVersionKind()
		objKey := client.ObjectKey{Namespace: obj.Namespace, Name: obj.Spec.ScaleTargetRef.Name}

		logger := log.WithValues("scaledObject.Namespace", obj.Namespace, "scaledObject.Name", obj.Name, "resource", gvk.String(), "name", objKey.Name)

		switch {
		// For core types, use a typed client so we get an informer-cache-backed Get to reduce API load.
		case gvk.Group == "apps" && gvk.Kind == "Deployment":
			deployment := &appsv1.Deployment{}
			if err := kubeClient.Get(ctx, objKey, deployment); err != nil {
				// resource doesn't exist
				logger.Error(err, "target deployment doesn't exist")
				return nil, "", err
			}
			podTemplateSpec.ObjectMeta = deployment.ObjectMeta
			podTemplateSpec.Spec = deployment.Spec.Template.Spec
		case gvk.Group == "apps" && gvk.Kind == "StatefulSet":
			statefulSet := &appsv1.StatefulSet{}
			if err := kubeClient.Get(ctx, objKey, statefulSet); err != nil {
				// resource doesn't exist
				logger.Error(err, "target statefulset doesn't exist")
				return nil, "", err
			}
			podTemplateSpec.ObjectMeta = statefulSet.ObjectMeta
			podTemplateSpec.Spec = statefulSet.Spec.Template.Spec
		default:
			unstruct := &unstructured.Unstructured{}
			unstruct.SetGroupVersionKind(gvk)
			if err := kubeClient.Get(ctx, objKey, unstruct); err != nil {
				// resource doesn't exist
				logger.Error(err, "target resource doesn't exist")
				return nil, "", err
			}
			withPods := &duckv1.WithPod{}
			if err := duck.FromUnstructured(unstruct, withPods); err != nil {
				logger.Error(err, "cannot convert Unstructured into PodSpecable Duck-type", "object", unstruct)
			}
			podTemplateSpec.ObjectMeta = withPods.ObjectMeta
			podTemplateSpec.Spec = withPods.Spec.Template.Spec
		}

		if len(podTemplateSpec.Spec.Containers) == 0 {
			logger.V(1).Info("There aren't any containers found in the ScaleTarget, therefore it is no possible to inject environment properties", "scaleTargetRef.Name", obj.Spec.ScaleTargetRef.Name)
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
func ResolveContainerEnv(ctx context.Context, client client.Client, logger logr.Logger, podSpec *corev1.PodSpec, containerName, namespace string, secretsLister corev1listers.SecretLister) (map[string]string, error) {
	if len(podSpec.Containers) < 1 {
		return nil, fmt.Errorf("target object doesn't have containers")
	}

	var container corev1.Container
	if containerName != "" {
		containerWithNameFound := boolFalse
		for _, c := range podSpec.Containers {
			if c.Name == containerName {
				container = c
				containerWithNameFound = boolTrue
				break
			}
		}

		if !containerWithNameFound {
			return nil, fmt.Errorf("couldn't find container with name %s on Target object", containerName)
		}
	} else {
		container = podSpec.Containers[0]
	}

	return resolveEnv(ctx, client, logger, &container, namespace, secretsLister)
}

// ResolveAuthRefAndPodIdentity provides authentication parameters and pod identity needed authenticate scaler with the environment.
func ResolveAuthRefAndPodIdentity(ctx context.Context, client client.Client, logger logr.Logger,
	triggerAuthRef *kedav1alpha1.AuthenticationRef, podTemplateSpec *corev1.PodTemplateSpec,
	namespace string, secretsLister corev1listers.SecretLister) (map[string]string, kedav1alpha1.AuthPodIdentity, error) {
	if podTemplateSpec != nil {
		authParams, podIdentity, err := resolveAuthRef(ctx, client, logger, triggerAuthRef, &podTemplateSpec.Spec, namespace, secretsLister)

		if err != nil {
			return authParams, podIdentity, err
		}
		switch podIdentity.Provider {
		case kedav1alpha1.PodIdentityProviderAws:
			if podIdentity.RoleArn != nil {
				if podIdentity.IsWorkloadIdentityOwner() {
					return nil, kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
						fmt.Errorf("roleArn can't be set if KEDA isn't identity owner, current value: '%s'", *podIdentity.IdentityOwner)
				}
				authParams["awsRoleArn"] = *podIdentity.RoleArn
			}
			if podIdentity.IsWorkloadIdentityOwner() {
				value, err := resolveServiceAccountAnnotation(ctx, client, podTemplateSpec.Spec.ServiceAccountName, namespace, kedav1alpha1.PodIdentityAnnotationEKS, true)
				if err != nil {
					return nil, kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
						fmt.Errorf("error getting service account: '%s', error: %w", podTemplateSpec.Spec.ServiceAccountName, err)
				}
				authParams["awsRoleArn"] = value
			}
		case kedav1alpha1.PodIdentityProviderAwsEKS:
			value, err := resolveServiceAccountAnnotation(ctx, client, podTemplateSpec.Spec.ServiceAccountName, namespace, kedav1alpha1.PodIdentityAnnotationEKS, false)
			if err != nil {
				return nil, kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
					fmt.Errorf("error getting service account: '%s', error: %w", podTemplateSpec.Spec.ServiceAccountName, err)
			}
			authParams["awsRoleArn"] = value
			// FIXME: Delete this for v3
			logger.Info("WARNING: AWS EKS Identity has been deprecated (https://github.com/kedacore/keda/discussions/5343) and will be removed from KEDA on v3")
		case kedav1alpha1.PodIdentityProviderAzureWorkload:
			if podIdentity.IdentityID != nil && *podIdentity.IdentityID == "" {
				return nil, kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone}, fmt.Errorf("IdentityID of PodIdentity should not be empty")
			}
		default:
		}
		return authParams, podIdentity, nil
	}

	return resolveAuthRef(ctx, client, logger, triggerAuthRef, nil, namespace, secretsLister)
}

// resolveAuthRef provides authentication parameters needed authenticate scaler with the environment.
// based on authentication method defined in TriggerAuthentication, authParams and podIdentity is returned
func resolveAuthRef(ctx context.Context, client client.Client, logger logr.Logger,
	triggerAuthRef *kedav1alpha1.AuthenticationRef, podSpec *corev1.PodSpec,
	namespace string, secretsLister corev1listers.SecretLister) (map[string]string, kedav1alpha1.AuthPodIdentity, error) {
	result := make(map[string]string)
	podIdentity := kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone}
	var err error

	if namespace != "" && triggerAuthRef != nil && triggerAuthRef.Name != "" {
		triggerAuthSpec, triggerNamespace, err := getTriggerAuthSpec(ctx, client, triggerAuthRef, namespace)
		if err != nil {
			logger.Error(err, "error getting triggerAuth", "triggerAuthRef.Name", triggerAuthRef.Name)
		} else {
			if triggerAuthSpec.PodIdentity != nil {
				podIdentity = *triggerAuthSpec.PodIdentity
			}
			if triggerAuthSpec.Env != nil {
				for _, e := range triggerAuthSpec.Env {
					if podSpec == nil {
						result[e.Parameter] = ""
						continue
					}
					env, err := ResolveContainerEnv(ctx, client, logger, podSpec, e.ContainerName, namespace, secretsLister)
					if err != nil {
						result[e.Parameter] = ""
					} else {
						result[e.Parameter] = env[e.Name]
					}
				}
			}
			if triggerAuthSpec.ConfigMapTargetRef != nil {
				for _, e := range triggerAuthSpec.ConfigMapTargetRef {
					result[e.Parameter] = resolveAuthConfigMap(ctx, client, logger, e.Name, triggerNamespace, e.Key)
				}
			}
			if triggerAuthSpec.SecretTargetRef != nil {
				for _, e := range triggerAuthSpec.SecretTargetRef {
					result[e.Parameter] = resolveAuthSecret(ctx, client, logger, e.Name, triggerNamespace, e.Key, secretsLister)
				}
			}
			if triggerAuthSpec.HashiCorpVault != nil && len(triggerAuthSpec.HashiCorpVault.Secrets) > 0 {
				vault := NewHashicorpVaultHandler(triggerAuthSpec.HashiCorpVault)
				err := vault.Initialize(logger)
				defer vault.Stop()
				if err != nil {
					logger.Error(err, "error authenticating to Vault", "triggerAuthRef.Name", triggerAuthRef.Name)
					return result, podIdentity, err
				}

				secrets, err := vault.ResolveSecrets(triggerAuthSpec.HashiCorpVault.Secrets)
				if err != nil {
					logger.Error(err, "could not get secrets from vault",
						"triggerAuthRef.Name", triggerAuthRef.Name,
					)
					return result, podIdentity, err
				}

				for _, e := range secrets {
					result[e.Parameter] = e.Value
				}
			}
			if triggerAuthSpec.AzureKeyVault != nil && len(triggerAuthSpec.AzureKeyVault.Secrets) > 0 {
				vaultHandler := NewAzureKeyVaultHandler(triggerAuthSpec.AzureKeyVault)
				err := vaultHandler.Initialize(ctx, client, logger, triggerNamespace, secretsLister)
				if err != nil {
					logger.Error(err, "error authenticating to Azure Key Vault", "triggerAuthRef.Name", triggerAuthRef.Name)
					return result, podIdentity, err
				}

				for _, secret := range triggerAuthSpec.AzureKeyVault.Secrets {
					res, err := vaultHandler.Read(ctx, secret.Name, secret.Version)
					if err != nil {
						logger.Error(err, "error trying to read secret from Azure Key Vault", "triggerAuthRef.Name", triggerAuthRef.Name,
							"secret.Name", secret.Name, "secret.Version", secret.Version)
						return result, podIdentity, err
					}

					result[secret.Parameter] = res
				}
			}
			if triggerAuthSpec.GCPSecretManager != nil && len(triggerAuthSpec.GCPSecretManager.Secrets) > 0 {
				secretManagerHandler := NewGCPSecretManagerHandler(triggerAuthSpec.GCPSecretManager)
				err := secretManagerHandler.Initialize(ctx, client, logger, triggerNamespace, secretsLister)
				if err != nil {
					logger.Error(err, "error authenticating to GCP Secret Manager", "triggerAuthRef.Name", triggerAuthRef.Name)
				} else {
					for _, secret := range triggerAuthSpec.GCPSecretManager.Secrets {
						version := "latest"
						if secret.Version != "" {
							version = secret.Version
						}
						res, err := secretManagerHandler.Read(ctx, secret.ID, version)
						if err != nil {
							logger.Error(err, "error trying to read secret from GCP Secret Manager", "triggerAuthRef.Name", triggerAuthRef.Name,
								"secret.Name", secret.ID, "secret.Version", secret.Version)
						} else {
							result[secret.Parameter] = res
						}
					}
				}
			}
			if triggerAuthSpec.AwsSecretManager != nil && len(triggerAuthSpec.AwsSecretManager.Secrets) > 0 {
				awsSecretManagerHandler := NewAwsSecretManagerHandler(triggerAuthSpec.AwsSecretManager)
				err := awsSecretManagerHandler.Initialize(ctx, client, logger, triggerNamespace, secretsLister, podSpec)
				defer awsSecretManagerHandler.Stop()
				if err != nil {
					logger.Error(err, "error authenticating to Aws Secret Manager", "triggerAuthRef.Name", triggerAuthRef.Name)
				} else {
					for _, secret := range triggerAuthSpec.AwsSecretManager.Secrets {
						res, err := awsSecretManagerHandler.Read(ctx, logger, secret.Name, secret.VersionID, secret.VersionStage, secret.SecretKey)
						if err != nil {
							logger.Error(err, "error trying to read secret from Aws Secret Manager", "triggerAuthRef.Name", triggerAuthRef.Name,
								"secret.Name", secret.Name, "secret.Version", secret.VersionID, "secret.VersionStage", secret.VersionStage, "secret.SecretKey", secret.SecretKey)
						} else {
							result[secret.Parameter] = res
						}
					}
				}
			}
		}
	}

	return result, podIdentity, err
}

func getTriggerAuthSpec(ctx context.Context, client client.Client, triggerAuthRef *kedav1alpha1.AuthenticationRef, namespace string) (*kedav1alpha1.TriggerAuthenticationSpec, string, error) {
	if triggerAuthRef.Kind == "" || triggerAuthRef.Kind == "TriggerAuthentication" {
		triggerAuth := &kedav1alpha1.TriggerAuthentication{}
		err := client.Get(ctx, types.NamespacedName{Name: triggerAuthRef.Name, Namespace: namespace}, triggerAuth)
		if err != nil {
			return nil, "", err
		}
		return &triggerAuth.Spec, namespace, nil
	} else if triggerAuthRef.Kind == "ClusterTriggerAuthentication" {
		clusterNamespace, err := util.GetClusterObjectNamespace()
		if err != nil {
			return nil, "", err
		}
		triggerAuth := &kedav1alpha1.ClusterTriggerAuthentication{}
		err = client.Get(ctx, types.NamespacedName{Name: triggerAuthRef.Name}, triggerAuth)
		if err != nil {
			return nil, "", err
		}
		return &triggerAuth.Spec, clusterNamespace, nil
	}
	return nil, "", fmt.Errorf("unknown trigger auth kind %s", triggerAuthRef.Kind)
}

func resolveEnv(ctx context.Context, client client.Client, logger logr.Logger, container *corev1.Container, namespace string, secretsLister corev1listers.SecretLister) (map[string]string, error) {
	resolved := make(map[string]string)
	secretAccessRestricted := isSecretAccessRestricted(logger)
	accessSecrets := readSecrets(secretAccessRestricted, namespace)
	if container.EnvFrom != nil {
		for _, source := range container.EnvFrom {
			if source.ConfigMapRef != nil {
				configMap, err := resolveConfigMap(ctx, client, source.ConfigMapRef, namespace)
				switch {
				case err == nil:
					for k, v := range configMap {
						resolved[k] = v
					}
				case source.ConfigMapRef.Optional != nil && *source.ConfigMapRef.Optional:
					// ignore error when ConfigMap is marked as optional
					continue
				default:
					return nil, fmt.Errorf("error reading config ref %s on namespace %s/: %w", source.ConfigMapRef, namespace, err)
				}
			} else if source.SecretRef != nil && accessSecrets {
				secretsMap, err := resolveSecretMap(ctx, client, logger, source.SecretRef, namespace, secretsLister)
				switch {
				case err == nil:
					for k, v := range secretsMap {
						resolved[k] = v
					}
				case source.SecretRef.Optional != nil && *source.SecretRef.Optional:
					// ignore error when Secret is marked as optional
					continue
				default:
					return nil, fmt.Errorf("error reading secret ref %s on namespace %s: %w", source.SecretRef, namespace, err)
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
				case envVar.ValueFrom.SecretKeyRef != nil && accessSecrets:
					// env is a secret selector
					value, err = resolveSecretValue(ctx, client, logger, envVar.ValueFrom.SecretKeyRef, envVar.ValueFrom.SecretKeyRef.Key, namespace, secretsLister)
					if err != nil {
						if envVar.ValueFrom.SecretKeyRef.Optional != nil && *envVar.ValueFrom.SecretKeyRef.Optional {
							continue
						}
						return nil, fmt.Errorf("error resolving secret name %s for env %s in namespace %s",
							envVar.ValueFrom.SecretKeyRef,
							envVar.Name,
							namespace)
					}
				case envVar.ValueFrom.ConfigMapKeyRef != nil:
					// env is a configMap selector
					value, err = resolveConfigValue(ctx, client, envVar.ValueFrom.ConfigMapKeyRef, envVar.ValueFrom.ConfigMapKeyRef.Key, namespace)
					if err != nil {
						if envVar.ValueFrom.ConfigMapKeyRef.Optional != nil && *envVar.ValueFrom.ConfigMapKeyRef.Optional {
							continue
						}
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

func readSecrets(secretAccessRestricted bool, namespace string) bool {
	if secretAccessRestricted && (namespace != kedaNamespace) {
		return boolFalse
	}
	return boolTrue
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

func resolveConfigMap(ctx context.Context, client client.Client, configMapRef *corev1.ConfigMapEnvSource, namespace string) (map[string]string, error) {
	configMap := &corev1.ConfigMap{}
	err := client.Get(ctx, types.NamespacedName{Name: configMapRef.Name, Namespace: namespace}, configMap)
	if err != nil {
		return nil, err
	}
	return configMap.Data, nil
}

func resolveSecretMap(ctx context.Context, client client.Client, logger logr.Logger, secretMapRef *corev1.SecretEnvSource, namespace string, secretsLister corev1listers.SecretLister) (map[string]string, error) {
	secret := &corev1.Secret{}
	var err error
	if isSecretAccessRestricted(logger) {
		secret, err = secretsLister.Secrets(kedaNamespace).Get(secretMapRef.Name)
	} else {
		err = client.Get(ctx, types.NamespacedName{Name: secretMapRef.Name, Namespace: namespace}, secret)
	}
	if err != nil {
		return nil, err
	}

	secretsStr := make(map[string]string)
	for k, v := range secret.Data {
		secretsStr[k] = string(v)
	}
	return secretsStr, nil
}

func resolveSecretValue(ctx context.Context, client client.Client, logger logr.Logger, secretKeyRef *corev1.SecretKeySelector, keyName, namespace string, secretsLister corev1listers.SecretLister) (string, error) {
	secret := &corev1.Secret{}
	var err error
	if isSecretAccessRestricted(logger) {
		secret, err = secretsLister.Secrets(kedaNamespace).Get(secretKeyRef.Name)
	} else {
		err = client.Get(ctx, types.NamespacedName{Name: secretKeyRef.Name, Namespace: namespace}, secret)
	}
	if err != nil {
		return "", err
	}
	return string(secret.Data[keyName]), nil
}

func resolveConfigValue(ctx context.Context, client client.Client, configKeyRef *corev1.ConfigMapKeySelector, keyName, namespace string) (string, error) {
	configMap := &corev1.ConfigMap{}
	err := client.Get(ctx, types.NamespacedName{Name: configKeyRef.Name, Namespace: namespace}, configMap)
	if err != nil {
		return "", err
	}
	return configMap.Data[keyName], nil
}

func resolveAuthConfigMap(ctx context.Context, client client.Client, logger logr.Logger, name, namespace, key string) string {
	ref := &corev1.ConfigMapKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: name}, Key: key}
	val, err := resolveConfigValue(ctx, client, ref, key, namespace)
	if err != nil {
		logger.Error(err, "error trying to get config map from namespace", "ConfigMap.Namespace", namespace, "ConfigMap.Name", name)
		return ""
	}
	return val
}

func resolveAuthSecret(ctx context.Context, client client.Client, logger logr.Logger, name, namespace, key string, secretsLister corev1listers.SecretLister) string {
	if name == "" || namespace == "" || key == "" {
		logger.Error(fmt.Errorf("error trying to get secret"), "name, namespace and key are required", "Secret.Namespace", namespace, "Secret.Name", name, "key", key)
		return ""
	}

	secret := &corev1.Secret{}
	var err error
	if isSecretAccessRestricted(logger) {
		secret, err = secretsLister.Secrets(kedaNamespace).Get(name)
	} else {
		err = client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, secret)
	}
	if err != nil {
		logger.Error(err, "error trying to get secret from namespace", "Secret.Namespace", namespace, "Secret.Name", name)
		return ""
	}
	result := secret.Data[key]

	if result == nil {
		return ""
	}

	return string(result)
}

// resolveServiceAccountAnnotation retrieves the value of a specific annotation
// from the annotations of a given Kubernetes ServiceAccount.
func resolveServiceAccountAnnotation(ctx context.Context, client client.Client, name, namespace, annotation string, required bool) (string, error) {
	serviceAccountName := defaultServiceAccount
	if name != "" {
		serviceAccountName = name
	}
	serviceAccount := &corev1.ServiceAccount{}
	err := client.Get(ctx, types.NamespacedName{Name: serviceAccountName, Namespace: namespace}, serviceAccount)
	if err != nil {
		return "", fmt.Errorf("error getting service account: '%s', error: %w", serviceAccountName, err)
	}
	value, ok := serviceAccount.Annotations[annotation]
	if !ok && required {
		return "", fmt.Errorf("annotation '%s' not found", annotation)
	}
	return value, nil
}

// GetCurrentReplicas returns the current replica count for a ScaledObject
func GetCurrentReplicas(ctx context.Context, client client.Client, scaleClient scale.ScalesGetter, scaledObject *kedav1alpha1.ScaledObject) (int32, error) {
	targetName := scaledObject.Spec.ScaleTargetRef.Name
	targetGVKR := scaledObject.Status.ScaleTargetGVKR

	logger := log.WithValues("scaledObject.Namespace", scaledObject.Namespace,
		"scaledObject.Name", scaledObject.Name,
		"resource", fmt.Sprintf("%s/%s", targetGVKR.Group, targetGVKR.Kind),
		"name", targetName)

	// Get the current replica count. As a special case, Deployments and StatefulSets fetch directly from the object so they can use the informer cache to reduce API calls.
	// Everything else uses the scale subresource.
	switch {
	case targetGVKR.Group == appsGroup && targetGVKR.Kind == deploymentKind:
		deployment := &appsv1.Deployment{}
		if err := client.Get(ctx, types.NamespacedName{Name: targetName, Namespace: scaledObject.Namespace}, deployment); err != nil {
			logger.Error(err, "target deployment doesn't exist")
			return 0, err
		}
		return *deployment.Spec.Replicas, nil
	case targetGVKR.Group == appsGroup && targetGVKR.Kind == statefulSetKind:
		statefulSet := &appsv1.StatefulSet{}
		if err := client.Get(ctx, types.NamespacedName{Name: targetName, Namespace: scaledObject.Namespace}, statefulSet); err != nil {
			logger.Error(err, "target statefulset doesn't exist")
			return 0, err
		}
		return *statefulSet.Spec.Replicas, nil
	default:
		scale, err := scaleClient.Scales(scaledObject.Namespace).Get(ctx, targetGVKR.GroupResource(), targetName, metav1.GetOptions{})
		if err != nil {
			logger.Error(err, "error getting scale subresource")
			return 0, err
		}
		return scale.Spec.Replicas, nil
	}
}
