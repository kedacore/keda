package scalers

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// ErrAwsNoAccessKey is returned when awsAccessKeyID is missing.
var ErrAwsNoAccessKey = errors.New("awsAccessKeyID not found")

type awsAuthorizationMetadata struct {
	awsRoleArn string

	awsAccessKeyID     string
	awsSecretAccessKey string
	awsSessionToken    string

	podIdentityOwner bool
}

type awsConfigMetadata struct {
	awsRegion        string
	awsEndpoint      string
	awsAuthorization awsAuthorizationMetadata
}

func getAwsConfig(ctx context.Context, awsRegion string, awsEndpoint string, awsAuthorization awsAuthorizationMetadata) (*aws.Config, error) {
	metadata := &awsConfigMetadata{
		awsRegion:        awsRegion,
		awsEndpoint:      awsEndpoint,
		awsAuthorization: awsAuthorization,
	}

	configOptions := make([]func(*config.LoadOptions) error, 0)
	configOptions = append(configOptions, config.WithRegion(metadata.awsRegion))
	cfg, err := config.LoadDefaultConfig(ctx, configOptions...)
	if err != nil {
		return nil, err
	}
	if !metadata.awsAuthorization.podIdentityOwner {
		return &cfg, nil
	}
	if metadata.awsAuthorization.awsAccessKeyID != "" && metadata.awsAuthorization.awsSecretAccessKey != "" {
		staticCredentialsProvider := aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(metadata.awsAuthorization.awsAccessKeyID, metadata.awsAuthorization.awsSecretAccessKey, ""))
		cfg.Credentials = staticCredentialsProvider
	}

	if metadata.awsAuthorization.awsRoleArn != "" {
		stsSvc := sts.NewFromConfig(cfg)
		stsCredentialProvider := stscreds.NewAssumeRoleProvider(stsSvc, metadata.awsAuthorization.awsRoleArn, func(options *stscreds.AssumeRoleOptions) {})
		cfg.Credentials = aws.NewCredentialsCache(stsCredentialProvider)
	}

	return &cfg, err
}

func getAwsAuthorization(authParams, metadata, resolvedEnv map[string]string) (awsAuthorizationMetadata, error) {
	meta := awsAuthorizationMetadata{}

	if metadata["identityOwner"] == "operator" {
		meta.podIdentityOwner = false
	} else if metadata["identityOwner"] == "" || metadata["identityOwner"] == "pod" {
		meta.podIdentityOwner = true
		switch {
		case authParams["awsRoleArn"] != "":
			meta.awsRoleArn = authParams["awsRoleArn"]
		case (authParams["awsAccessKeyID"] != "" || authParams["awsAccessKeyId"] != "") && authParams["awsSecretAccessKey"] != "":
			meta.awsAccessKeyID = authParams["awsAccessKeyID"]
			if meta.awsAccessKeyID == "" {
				meta.awsAccessKeyID = authParams["awsAccessKeyId"]
			}
			meta.awsSecretAccessKey = authParams["awsSecretAccessKey"]
			meta.awsSessionToken = authParams["awsSessionToken"]
		default:
			if metadata["awsAccessKeyID"] != "" {
				meta.awsAccessKeyID = metadata["awsAccessKeyID"]
			} else if metadata["awsAccessKeyIDFromEnv"] != "" {
				meta.awsAccessKeyID = resolvedEnv[metadata["awsAccessKeyIDFromEnv"]]
			}

			if len(meta.awsAccessKeyID) == 0 {
				return meta, ErrAwsNoAccessKey
			}

			if metadata["awsSecretAccessKeyFromEnv"] != "" {
				meta.awsSecretAccessKey = resolvedEnv[metadata["awsSecretAccessKeyFromEnv"]]
			}

			if len(meta.awsSecretAccessKey) == 0 {
				return meta, fmt.Errorf("awsSecretAccessKey not found")
			}
		}
	}

	return meta, nil
}
