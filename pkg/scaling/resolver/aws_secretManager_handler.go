package resolver

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

type AwsSecretManagerHandler struct {
	secretManager *kedav1alpha1.AwsSecretManager
	session       *secretsmanager.Client
}

func NewAwsSecretManagerHandler(a *kedav1alpha1.AwsSecretManager) *AwsSecretManagerHandler {
	return &AwsSecretManagerHandler{
		secretManager: a,
	}
}

func (ash *AwsSecretManagerHandler) Read(ctx context.Context, logger logr.Logger, secretName, versionID, versionStage string) (string, error) {
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
	return *result.SecretString, nil
}

func (ash *AwsSecretManagerHandler) Initialize(ctx context.Context, client client.Client, logger logr.Logger, triggerNamespace string, secretsLister corev1listers.SecretLister, podTemplateSpec *corev1.PodTemplateSpec) error {
	config, err := ash.getcredentials(ctx, client, logger, triggerNamespace, secretsLister, podTemplateSpec)
	if err != nil {
		logger.Error(err, "Error getting credentials")
		return err
	}

	logger.Info("Config value", "config", config)

	if ash.secretManager.Cloud != nil {
		if ash.secretManager.Cloud.Region != "" {
			config.Region = ash.secretManager.Cloud.Region
		}
		if ash.secretManager.Cloud.Endpoint != "" {
			logger.Info("Endpoint value", "Endpoint", ash.secretManager.Cloud.Endpoint)
		}
	}

	ash.session = secretsmanager.NewFromConfig(*config)
	return nil
}

func (ash *AwsSecretManagerHandler) getcredentials(ctx context.Context, client client.Client, logger logr.Logger, triggerNamespace string, secretsLister corev1listers.SecretLister, podTemplateSpec *corev1.PodTemplateSpec) (*aws.Config, error) {
	config := &aws.Config{}

	podIdentity := ash.secretManager.PodIdentity
	if podIdentity == nil {
		podIdentity = &kedav1alpha1.AuthPodIdentity{}
	}

	switch podIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		accessKeyID := resolveAuthSecret(ctx, client, logger, ash.secretManager.Credentials.AccessKey.ValueFrom.SecretKeyRef.Name, triggerNamespace, ash.secretManager.Credentials.AccessKey.ValueFrom.SecretKeyRef.Key, secretsLister)
		accessSecretKey := resolveAuthSecret(ctx, client, logger, ash.secretManager.Credentials.AccessSecretKey.ValueFrom.SecretKeyRef.Name, triggerNamespace, ash.secretManager.Credentials.AccessSecretKey.ValueFrom.SecretKeyRef.Key, secretsLister)
		if accessKeyID == "" || accessSecretKey == "" {
			return nil, fmt.Errorf("AccessKeyID and AccessSecretKey are expected when not using a pod identity provider")
		}
		config.Credentials = credentials.NewStaticCredentialsProvider(
			accessKeyID,
			accessSecretKey,
			"",
		)
		return config, nil

	case kedav1alpha1.PodIdentityProviderAwsEKS:
		awsRoleArn, err := ash.getRoleArnAwsEKS(ctx, client, logger, triggerNamespace, podTemplateSpec)
		if err != nil {
			return nil, fmt.Errorf("error resolving role arn for AwsEKS pod identity: %w", err)
		}
		config.Credentials = stscreds.NewAssumeRoleProvider(sts.NewFromConfig(*config), awsRoleArn)
		return config, nil

	case kedav1alpha1.PodIdentityProviderAwsKiam:
		awsRoleArn := podTemplateSpec.ObjectMeta.Annotations[kedav1alpha1.PodIdentityAnnotationKiam]
		config.Credentials = stscreds.NewAssumeRoleProvider(sts.NewFromConfig(*config), awsRoleArn)
		return config, nil
	default:
		return nil, fmt.Errorf("pod identity provider %s not supported", podIdentity.Provider)
	}
}

func (ash *AwsSecretManagerHandler) getRoleArnAwsEKS(ctx context.Context, client client.Client, logger logr.Logger, triggerNamespace string, podTemplateSpec *corev1.PodTemplateSpec) (string, error) {
	serviceAccountName := podTemplateSpec.Spec.ServiceAccountName
	serviceAccount := &corev1.ServiceAccount{}
	logger.Info("serviceAccount value", "serviceAccount", serviceAccount)
	err := client.Get(ctx, types.NamespacedName{Name: serviceAccountName, Namespace: triggerNamespace}, serviceAccount)
	if err != nil {
		return "", err
	}
	return serviceAccount.Annotations[kedav1alpha1.PodIdentityAnnotationEKS], nil
}
