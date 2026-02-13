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

/*
This file contains all the logic for caching aws.Config across all the (AWS)
triggers. The first time when an aws.Config is requested, it's cached based on
the authentication info (roleArn, Key&Secret, keda itself) and it's returned
every time when an aws.Config is requested for the same authentication info.
This is required because if we don't cache and share them, each scaler
generates and refresh it's own token although all the tokens grants the same
permissions
*/

package aws

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/go-logr/logr"
	"golang.org/x/crypto/sha3"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	webIdentityTokenFile = os.Getenv("AWS_WEB_IDENTITY_TOKEN_FILE")
)

// cacheEntry stores *aws.Config and where is used
type cacheEntry struct {
	config *aws.Config
	usages map[string]bool // Tracks the resources which have requested the cache
}

// sharedConfigCache is a shared cache for storing all *aws.Config
// across all (AWS) triggers
type sharedConfigCache struct {
	sync.Mutex
	items  map[string]cacheEntry
	logger logr.Logger
}

func newSharedConfigsCache() sharedConfigCache {
	return sharedConfigCache{items: map[string]cacheEntry{}, logger: logf.Log.WithName("aws_credentials_cache")}
}

// getCacheKey returns a unique key based on given AuthorizationMetadata.
// As it can contain sensitive data, the key is hashed to not expose secrets
func (a *sharedConfigCache) getCacheKey(awsAuthorization AuthorizationMetadata) string {
	key := "keda-" + awsAuthorization.AwsRegion
	if awsAuthorization.AwsAccessKeyID != "" {
		key = fmt.Sprintf("%s-%s-%s-%s", awsAuthorization.AwsAccessKeyID, awsAuthorization.AwsSecretAccessKey, awsAuthorization.AwsSessionToken, awsAuthorization.AwsRegion)
	} else if awsAuthorization.AwsRoleArn != "" {
		key = fmt.Sprintf("%s-%s-%s", awsAuthorization.AwsRoleArn, awsAuthorization.AwsExternalID, awsAuthorization.AwsRegion)
	}
	// to avoid sensitive data as key and to use a constant key size,
	// we hash the key with sha3
	hash := sha3.Sum224([]byte(key))
	return hex.EncodeToString(hash[:])
}

// GetCredentials returns *aws.Config for a given AuthorizationMetadata.
// The *aws.Config is also cached for next requests with same AuthorizationMetadata,
// sharing it between all the requests. To track if the *aws.Config is used by whom,
// every time when an scaler requests *aws.Config we register it inside
// the cached item.
func (a *sharedConfigCache) GetCredentials(ctx context.Context, awsAuthorization AuthorizationMetadata) (*aws.Config, error) {
	a.Lock()
	defer a.Unlock()
	key := a.getCacheKey(awsAuthorization)
	if cachedEntry, exists := a.items[key]; exists {
		cachedEntry.usages[awsAuthorization.TriggerUniqueKey] = true
		a.items[key] = cachedEntry
		return cachedEntry.config, nil
	}

	configOptions := make([]func(*config.LoadOptions) error, 0)
	configOptions = append(configOptions, config.WithRegion(awsAuthorization.AwsRegion))
	cfg, err := config.LoadDefaultConfig(ctx, configOptions...)
	if err != nil {
		return nil, err
	}

	if awsAuthorization.UsingPodIdentity {
		if awsAuthorization.AwsRoleArn != "" {
			cfg.Credentials = a.retrievePodIdentityCredentials(ctx, cfg, awsAuthorization.AwsRoleArn, awsAuthorization.AwsExternalID)
		}
	} else {
		cfg.Credentials = a.retrieveStaticCredentials(awsAuthorization)
	}

	newCacheEntry := cacheEntry{
		config: &cfg,
		usages: map[string]bool{
			awsAuthorization.TriggerUniqueKey: true,
		},
	}
	a.items[key] = newCacheEntry

	return &cfg, nil
}

// RemoveCachedEntry removes the usage of an AuthorizationMetadata from the cached item.
// If there isn't any usage of a given cached item (because there isn't any trigger using the aws.Config),
// we also remove it from the cache
func (a *sharedConfigCache) RemoveCachedEntry(awsAuthorization AuthorizationMetadata) {
	a.Lock()
	defer a.Unlock()
	key := a.getCacheKey(awsAuthorization)
	if cachedEntry, exists := a.items[key]; exists {
		// Delete the TriggerUniqueKey from usages
		delete(cachedEntry.usages, awsAuthorization.TriggerUniqueKey)

		// If no more usages, delete the entire entry from the cache
		if len(cachedEntry.usages) == 0 {
			delete(a.items, key)
		} else {
			a.items[awsAuthorization.AwsRoleArn] = cachedEntry
		}
	}
}

// retrievePodIdentityCredentials returns an *aws.CredentialsCache to assume given roleArn.
// It tries first to assume the role using WebIdentity (OIDC federation) and if this method fails,
// it tries to assume the role using KEDA's role (AssumeRole)
func (a *sharedConfigCache) retrievePodIdentityCredentials(ctx context.Context, cfg aws.Config, roleArn string, externalID string) *aws.CredentialsCache {
	stsSvc := sts.NewFromConfig(cfg)

	if webIdentityTokenFile != "" {
		webIdentityCredentialProvider := stscreds.NewWebIdentityRoleProvider(stsSvc, roleArn, stscreds.IdentityTokenFile(webIdentityTokenFile), func(options *stscreds.WebIdentityRoleOptions) {
			options.RoleSessionName = "KEDA"
		})

		ctx, cancel := context.WithTimeout(ctx, time.Second*5)
		defer cancel()
		_, err := webIdentityCredentialProvider.Retrieve(ctx)
		if err == nil {
			a.logger.V(1).Info(fmt.Sprintf("using assume web identity role to retrieve token for arnRole %s", roleArn))
			return aws.NewCredentialsCache(webIdentityCredentialProvider)
		}
		a.logger.V(1).Error(err, fmt.Sprintf("error retrieving arnRole %s via WebIdentity", roleArn))
	}

	// Fallback to Assume Role
	a.logger.V(1).Info(fmt.Sprintf("using assume role to retrieve token for arnRole %s", roleArn))
	assumeRoleCredentialProvider := stscreds.NewAssumeRoleProvider(stsSvc, roleArn, func(options *stscreds.AssumeRoleOptions) {
		options.RoleSessionName = "KEDA"
		if externalID != "" {
			options.ExternalID = aws.String(externalID)
		}
	})
	return aws.NewCredentialsCache(assumeRoleCredentialProvider)
}

// retrieveStaticCredentials returns an *aws.CredentialsCache for given
// AuthorizationMetadata (using static credentials). This is used for static
// authentication via AwsAccessKeyID & AwsAccessKeySecret
func (*sharedConfigCache) retrieveStaticCredentials(awsAuthorization AuthorizationMetadata) *aws.CredentialsCache {
	staticCredentialsProvider := aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(awsAuthorization.AwsAccessKeyID, awsAuthorization.AwsSecretAccessKey, awsAuthorization.AwsSessionToken))
	return staticCredentialsProvider
}
