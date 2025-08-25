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
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	awsutils "github.com/kedacore/keda/v2/keda-scalers/aws"
)

type AwsSecretManagerHandler struct {
	secretManager *kedav1alpha1.AwsSecretManager
	session       *secretsmanager.Client
	awsMetadata   awsutils.AuthorizationMetadata
}

func NewAwsSecretManagerHandler(a *kedav1alpha1.AwsSecretManager) *AwsSecretManagerHandler {
	return &AwsSecretManagerHandler{
		secretManager: a,
	}
}

// Read fetches the secret value from AWS Secret Manager using the provided secret name, version ID(optional), version stage(optional), and secretKey(optional).
// It returns the secret value as a string.
func (ash *AwsSecretManagerHandler) Read(ctx context.Context, logger logr.Logger, secretName, versionID, versionStage string, secretKey string) (string, error) {
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	}
	if versionID != "" {
		input.VersionId = aws.String(versionID)
	}
	if versionStage != "" {
		input.VersionStage = aws.String(versionStage)
	}
	result, err := ash.session.GetSecretValue(ctx, input)
	if err != nil {
		logger.Error(err, "Error getting credentials")
		return "", err
	}
	if secretKey != "" {
		// Parse the secret string as JSON
		var secretMap map[string]interface{}
		err = json.Unmarshal([]byte(*result.SecretString), &secretMap)
		if err != nil {
			logger.Error(err, "Error parsing secret string as JSON")
			return "", err
		}

		// Check if the specified secret key exists
		if val, ok := secretMap[secretKey]; ok {
			// Convert the value to a string and return it
			if strVal, isString := val.(string); isString {
				return strVal, nil
			}
			logger.Error(nil, "SecretKey value is not a string")
			return "", fmt.Errorf("SecretKey value is not a string")
		}
		logger.Error(nil, "SecretKey Not Found")
		return "", fmt.Errorf("SecretKey Not Found")
	}
	return *result.SecretString, nil
}

// Initialize sets up the AWS Secret Manager handler by configuring AWS credentials, AWS region, or using pod identity.
// It initializes the AWS Secret Manager session and metadata.
func (ash *AwsSecretManagerHandler) Initialize(ctx context.Context, client client.Client, logger logr.Logger, triggerNamespace string, secretsLister corev1listers.SecretLister, podSpec *corev1.PodSpec) error {
	ash.awsMetadata = awsutils.AuthorizationMetadata{
		TriggerUniqueKey: fmt.Sprintf("aws-secret-manager-%s", triggerNamespace),
	}
	awsRegion := ""
	if ash.secretManager.Region != "" {
		awsRegion = ash.secretManager.Region
	}
	ash.awsMetadata.AwsRegion = awsRegion
	podIdentity := ash.secretManager.PodIdentity
	if podIdentity == nil {
		podIdentity = &kedav1alpha1.AuthPodIdentity{}
	}

	switch podIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		ash.awsMetadata.AwsAccessKeyID = resolveAuthSecret(ctx, client, logger, ash.secretManager.Credentials.AccessKey.ValueFrom.SecretKeyRef.Name, triggerNamespace, ash.secretManager.Credentials.AccessKey.ValueFrom.SecretKeyRef.Key, secretsLister)
		ash.awsMetadata.AwsSecretAccessKey = resolveAuthSecret(ctx, client, logger, ash.secretManager.Credentials.AccessSecretKey.ValueFrom.SecretKeyRef.Name, triggerNamespace, ash.secretManager.Credentials.AccessSecretKey.ValueFrom.SecretKeyRef.Key, secretsLister)
		if ash.awsMetadata.AwsAccessKeyID == "" || ash.awsMetadata.AwsSecretAccessKey == "" {
			return fmt.Errorf("AccessKeyID and AccessSecretKey are expected when not using a pod identity provider")
		}
	case kedav1alpha1.PodIdentityProviderAws:
		ash.awsMetadata.UsingPodIdentity = true
		if ash.secretManager.PodIdentity.IsWorkloadIdentityOwner() {
			awsRoleArn, err := resolveServiceAccountAnnotation(ctx, client, podSpec.ServiceAccountName, triggerNamespace, kedav1alpha1.PodIdentityAnnotationEKS, true)
			if err != nil {
				return fmt.Errorf("error resolving role arn for aws: %w", err)
			}
			ash.awsMetadata.AwsRoleArn = awsRoleArn
		} else if ash.secretManager.PodIdentity.RoleArn != nil {
			ash.awsMetadata.AwsRoleArn = *ash.secretManager.PodIdentity.RoleArn
		}
	default:
		return fmt.Errorf("pod identity provider %s not supported", podIdentity.Provider)
	}

	config, err := awsutils.GetAwsConfig(ctx, ash.awsMetadata)
	if err != nil {
		logger.Error(err, "Error getting credentials")
		return err
	}
	ash.session = secretsmanager.NewFromConfig(*config)
	return nil
}

func (ash *AwsSecretManagerHandler) Stop() {
	awsutils.ClearAwsConfig(ash.awsMetadata)
}
