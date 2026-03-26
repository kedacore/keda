/*
Copyright 2024 The KEDA Authors

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
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	awsutils "github.com/kedacore/keda/v2/pkg/scalers/aws"
)

type AwsParameterStoreHandler struct {
	parameterStore *kedav1alpha1.AwsParameterStore
	session        *ssm.Client
	awsMetadata    awsutils.AuthorizationMetadata
}

func NewAwsParameterStoreHandler(a *kedav1alpha1.AwsParameterStore) *AwsParameterStoreHandler {
	return &AwsParameterStoreHandler{
		parameterStore: a,
	}
}

func (apsh *AwsParameterStoreHandler) Read(ctx context.Context, logger logr.Logger, parameterName string, withDecryption *bool) (string, error) {
	decrypt := true
	if withDecryption != nil {
		decrypt = *withDecryption
	}

	input := &ssm.GetParameterInput{
		Name:           aws.String(parameterName),
		WithDecryption: aws.Bool(decrypt),
	}

	result, err := apsh.session.GetParameter(ctx, input)
	if err != nil {
		logger.Error(err, "Error getting parameter from Parameter Store")
		return "", err
	}

	if result.Parameter == nil || result.Parameter.Value == nil {
		logger.Error(nil, "Parameter value is nil")
		return "", fmt.Errorf("parameter value is nil")
	}

	return *result.Parameter.Value, nil
}

func (apsh *AwsParameterStoreHandler) Initialize(ctx context.Context, client client.Client, logger logr.Logger, triggerNamespace string, secretsLister corev1listers.SecretLister, podSpec *corev1.PodSpec) error {
	apsh.awsMetadata = awsutils.AuthorizationMetadata{
		TriggerUniqueKey: fmt.Sprintf("aws-parameter-store-%s", triggerNamespace),
	}
	awsRegion := ""
	if apsh.parameterStore.Region != "" {
		awsRegion = apsh.parameterStore.Region
	}
	apsh.awsMetadata.AwsRegion = awsRegion
	podIdentity := apsh.parameterStore.PodIdentity
	if podIdentity == nil {
		podIdentity = &kedav1alpha1.AuthPodIdentity{}
	}

	switch podIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		apsh.awsMetadata.AwsAccessKeyID = resolveAuthSecret(ctx, client, logger, apsh.parameterStore.Credentials.AccessKey.ValueFrom.SecretKeyRef.Name, triggerNamespace, apsh.parameterStore.Credentials.AccessKey.ValueFrom.SecretKeyRef.Key, secretsLister)
		apsh.awsMetadata.AwsSecretAccessKey = resolveAuthSecret(ctx, client, logger, apsh.parameterStore.Credentials.AccessSecretKey.ValueFrom.SecretKeyRef.Name, triggerNamespace, apsh.parameterStore.Credentials.AccessSecretKey.ValueFrom.SecretKeyRef.Key, secretsLister)
		if apsh.awsMetadata.AwsAccessKeyID == "" || apsh.awsMetadata.AwsSecretAccessKey == "" {
			return fmt.Errorf("AccessKeyID and AccessSecretKey are expected when not using a pod identity provider")
		}
	case kedav1alpha1.PodIdentityProviderAws:
		apsh.awsMetadata.UsingPodIdentity = true
		if apsh.parameterStore.PodIdentity.IsWorkloadIdentityOwner() {
			awsRoleArn, err := resolveServiceAccountAnnotation(ctx, client, podSpec.ServiceAccountName, triggerNamespace, kedav1alpha1.PodIdentityAnnotationEKS, true)
			if err != nil {
				return fmt.Errorf("error resolving role arn for aws: %w", err)
			}
			apsh.awsMetadata.AwsRoleArn = awsRoleArn
		} else if apsh.parameterStore.PodIdentity.RoleArn != nil {
			apsh.awsMetadata.AwsRoleArn = *apsh.parameterStore.PodIdentity.RoleArn
		}
	default:
		return fmt.Errorf("pod identity provider %s not supported", podIdentity.Provider)
	}

	config, err := awsutils.GetAwsConfig(ctx, apsh.awsMetadata)
	if err != nil {
		logger.Error(err, "Error getting credentials")
		return err
	}
	apsh.session = ssm.NewFromConfig(*config)
	return nil
}

func (apsh *AwsParameterStoreHandler) Stop() {
	awsutils.ClearAwsConfig(apsh.awsMetadata)
}
