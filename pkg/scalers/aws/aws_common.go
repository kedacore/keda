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

package aws

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

// ErrAwsNoAccessKey is returned when awsAccessKeyID is missing.
var ErrAwsNoAccessKey = errors.New("awsAccessKeyID not found")

type awsConfigMetadata struct {
	awsRegion        string
	awsAuthorization AuthorizationMetadata
}

var awsSharedCredentialsCache = newSharedConfigsCache()

func GetAwsConfig(ctx context.Context, awsRegion string, awsAuthorization AuthorizationMetadata) (*aws.Config, error) {
	metadata := &awsConfigMetadata{
		awsRegion:        awsRegion,
		awsAuthorization: awsAuthorization,
	}

	if metadata.awsAuthorization.UsingPodIdentity ||
		(metadata.awsAuthorization.AwsAccessKeyID != "" && metadata.awsAuthorization.AwsSecretAccessKey != "") {
		return awsSharedCredentialsCache.GetCredentials(ctx, metadata.awsRegion, metadata.awsAuthorization)
	}

	// TODO, remove when aws-kiam and aws-eks are removed
	configOptions := make([]func(*config.LoadOptions) error, 0)
	configOptions = append(configOptions, config.WithRegion(metadata.awsRegion))
	cfg, err := config.LoadDefaultConfig(ctx, configOptions...)
	if err != nil {
		return nil, err
	}

	if !metadata.awsAuthorization.PodIdentityOwner {
		return &cfg, nil
	}

	if metadata.awsAuthorization.AwsRoleArn != "" {
		stsSvc := sts.NewFromConfig(cfg)
		stsCredentialProvider := stscreds.NewAssumeRoleProvider(stsSvc, metadata.awsAuthorization.AwsRoleArn, func(options *stscreds.AssumeRoleOptions) {})
		cfg.Credentials = aws.NewCredentialsCache(stsCredentialProvider)
	}
	return &cfg, err
	// END remove when aws-kiam and aws-eks are removed
}

func GetAwsAuthorization(uniqueKey string, podIdentity kedav1alpha1.AuthPodIdentity, triggerMetadata, authParams, resolvedEnv map[string]string) (AuthorizationMetadata, error) {
	meta := AuthorizationMetadata{
		TriggerUniqueKey: uniqueKey,
	}

	if podIdentity.Provider == kedav1alpha1.PodIdentityProviderAws {
		meta.UsingPodIdentity = true
		if val, ok := authParams["awsRoleArn"]; ok && val != "" {
			meta.AwsRoleArn = val
		}
		return meta, nil
	}
	// TODO, remove all the logic below and just keep the logic for
	// parsing awsAccessKeyID, awsSecretAccessKey and awsSessionToken
	// when aws-kiam and aws-eks are removed
	if triggerMetadata["identityOwner"] == "operator" {
		meta.PodIdentityOwner = false
	} else if triggerMetadata["identityOwner"] == "" || triggerMetadata["identityOwner"] == "pod" {
		meta.PodIdentityOwner = true
		switch {
		case authParams["awsRoleArn"] != "":
			meta.AwsRoleArn = authParams["awsRoleArn"]
		case (authParams["awsAccessKeyID"] != "" || authParams["awsAccessKeyId"] != "") && authParams["awsSecretAccessKey"] != "":
			meta.AwsAccessKeyID = authParams["awsAccessKeyID"]
			if meta.AwsAccessKeyID == "" {
				meta.AwsAccessKeyID = authParams["awsAccessKeyId"]
			}
			meta.AwsSecretAccessKey = authParams["awsSecretAccessKey"]
			meta.AwsSessionToken = authParams["awsSessionToken"]
		default:
			if triggerMetadata["awsAccessKeyID"] != "" {
				meta.AwsAccessKeyID = triggerMetadata["awsAccessKeyID"]
			} else if triggerMetadata["awsAccessKeyIDFromEnv"] != "" {
				meta.AwsAccessKeyID = resolvedEnv[triggerMetadata["awsAccessKeyIDFromEnv"]]
			}

			if len(meta.AwsAccessKeyID) == 0 {
				return meta, ErrAwsNoAccessKey
			}

			if triggerMetadata["awsSecretAccessKeyFromEnv"] != "" {
				meta.AwsSecretAccessKey = resolvedEnv[triggerMetadata["awsSecretAccessKeyFromEnv"]]
			}

			if len(meta.AwsSecretAccessKey) == 0 {
				return meta, fmt.Errorf("awsSecretAccessKey not found")
			}
		}
	}

	return meta, nil
}

func ClearAwsConfig(awsAuthorization AuthorizationMetadata) {
	awsSharedCredentialsCache.RemoveCachedEntry(awsAuthorization)
}
