package resolver

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/go-logr/logr"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

const (
	AccessKeyID     = "AWS_ACCESS_KEY_ID"
	SecretAccessKey = "AWS_SECRET_ACCESS_KEY"
)

type AwsSecretManagerHandler struct {
	secretManager *kedav1alpha1.AwsSecretManager
	secretclient  *secretsmanager.SecretsManager
}

func NewAwsSecretManagerHandler(a *kedav1alpha1.AwsSecretManager) *AwsSecretManagerHandler {
	return &AwsSecretManagerHandler{
		secretManager: a,
	}
}

func (ash *AwsSecretManagerHandler) Read(secretName, versionID, versionStage string) (string, error) {
	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(secretName),
		VersionId:    aws.String(versionID),
		VersionStage: aws.String(versionStage),
	}

	result, err := ash.secretclient.GetSecretValue(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case secretsmanager.ErrCodeResourceNotFoundException:
				err = fmt.Errorf(secretsmanager.ErrCodeResourceNotFoundException+": %s", aerr.Error())
				return "", err
			case secretsmanager.ErrCodeInvalidParameterException:
				err = fmt.Errorf(secretsmanager.ErrCodeInvalidParameterException+": %s", aerr.Error())
				return "", err
			case secretsmanager.ErrCodeInvalidRequestException:
				err = fmt.Errorf(secretsmanager.ErrCodeInvalidRequestException+": %s", aerr.Error())
				return "", err
			case secretsmanager.ErrCodeDecryptionFailure:
				err = fmt.Errorf(secretsmanager.ErrCodeDecryptionFailure+": %s", aerr.Error())
				return "", err
			case secretsmanager.ErrCodeInternalServiceError:
				err = fmt.Errorf(secretsmanager.ErrCodeInternalServiceError+": %s", aerr.Error())
				return "", err
			default:
				err = fmt.Errorf(aerr.Error())
				return "", err
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			err = fmt.Errorf(err.Error())
			return "", err
		}
	}
	return *result.SecretString, nil
}

func (ash *AwsSecretManagerHandler) Initialize(ctx context.Context, client client.Client, logger logr.Logger, triggerNamespace string, secretsLister corev1listers.SecretLister) error {
	config, err := ash.getconfig(ctx, client, logger, triggerNamespace, secretsLister)
	if err != nil {
		return err
	}

	sess, err := session.NewSession()

	ash.secretclient = secretsmanager.New(sess, config)
	return err
}

func (ash *AwsSecretManagerHandler) getconfig(ctx context.Context, client client.Client, logger logr.Logger, triggerNamespace string, secretsLister corev1listers.SecretLister) (*aws.Config, error) {
	config := aws.NewConfig()

	podIdentity := ash.secretManager.PodIdentity
	if podIdentity == nil {
		podIdentity = &kedav1alpha1.AuthPodIdentity{}
	}

	switch podIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		secretName := ash.secretManager.Credentials.ValuesFrom
		accessKeyID := resolveAuthSecret(ctx, client, logger, secretName, triggerNamespace, AccessKeyID, secretsLister)
		accessSecretKey := resolveAuthSecret(ctx, client, logger, secretName, triggerNamespace, SecretAccessKey, secretsLister)
		if accessKeyID == "" || accessSecretKey == "" {
			return nil, fmt.Errorf("%s and %s are expected when not using a pod identity provider", AccessKeyID, SecretAccessKey)
		}
		config.WithCredentials(credentials.NewStaticCredentials(accessKeyID, accessSecretKey, ""))
		if ash.secretManager.Cloud.Region != "" {
			config.WithRegion(ash.secretManager.Cloud.Region)
		}
		if ash.secretManager.Cloud.Endpoint != "" {
			config.WithEndpoint(ash.secretManager.Cloud.Endpoint)
		}
		return config, nil

	case kedav1alpha1.PodIdentityProviderAwsKiam, kedav1alpha1.PodIdentityProviderAwsEKS:
		if ash.secretManager.Cloud.Region != "" {
			config.WithRegion(ash.secretManager.Cloud.Region)
		}
		if ash.secretManager.Cloud.Endpoint != "" {
			config.WithEndpoint(ash.secretManager.Cloud.Endpoint)
		}

		return config, nil

	default:
		return nil, fmt.Errorf("pod identity provider %s not supported", podIdentity.Provider)
	}
}
