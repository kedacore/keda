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

type cacheEntry struct {
	config *aws.Config
	usages map[string]bool // Tracks the resources which have requested the cache
}
type sharedConfigCache struct {
	sync.Mutex
	items  map[string]cacheEntry
	logger logr.Logger
}

func newSharedConfigsCache() sharedConfigCache {
	return sharedConfigCache{items: map[string]cacheEntry{}, logger: logf.Log.WithName("aws_credentials_cache")}
}

func (a *sharedConfigCache) getCacheKey(awsAuthorization AuthorizationMetadata) string {
	key := "keda"
	if awsAuthorization.AwsAccessKeyID != "" {
		key = fmt.Sprintf("%s-%s-%s", awsAuthorization.AwsAccessKeyID, awsAuthorization.AwsSecretAccessKey, awsAuthorization.AwsSessionToken)
	} else if awsAuthorization.AwsRoleArn != "" {
		key = awsAuthorization.AwsRoleArn
	}
	// to avoid sensitive data as key and to use a constant key size,
	// we hash the key with sha3
	hash := sha3.Sum224([]byte(key))
	return hex.EncodeToString(hash[:])
}

func (a *sharedConfigCache) GetCredentials(ctx context.Context, awsRegion string, awsAuthorization AuthorizationMetadata) (*aws.Config, error) {
	a.Lock()
	defer a.Unlock()
	key := a.getCacheKey(awsAuthorization)
	if cachedEntry, exists := a.items[key]; exists {
		cachedEntry.usages[awsAuthorization.TriggerUniqueKey] = true
		a.items[key] = cachedEntry
		return cachedEntry.config, nil
	}

	configOptions := make([]func(*config.LoadOptions) error, 0)
	configOptions = append(configOptions, config.WithRegion(awsRegion))
	cfg, err := config.LoadDefaultConfig(ctx, configOptions...)
	if err != nil {
		return nil, err
	}

	if awsAuthorization.UsingPodIdentity {
		if awsAuthorization.AwsRoleArn != "" {
			cfg.Credentials = a.retrievePodIdentityCredentials(cfg, awsAuthorization.AwsRoleArn)
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

func (a *sharedConfigCache) retrievePodIdentityCredentials(cfg aws.Config, roleArn string) *aws.CredentialsCache {
	stsSvc := sts.NewFromConfig(cfg)
	webIdentityTokenFile := os.Getenv("AWS_WEB_IDENTITY_TOKEN_FILE")

	webIdentityCredentialProvider := stscreds.NewWebIdentityRoleProvider(stsSvc, roleArn, stscreds.IdentityTokenFile(webIdentityTokenFile), func(options *stscreds.WebIdentityRoleOptions) {
		options.RoleSessionName = "KEDA"
	})
	var cachedProvider *aws.CredentialsCache

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	_, err := webIdentityCredentialProvider.Retrieve(ctx)
	if err != nil {
		a.logger.V(1).Error(err, fmt.Sprintf("error retreiving arnRole %s via WebIdentity", roleArn))
		// Fallback to Assume Role
		assumeRoleCredentialProvider := stscreds.NewAssumeRoleProvider(stsSvc, roleArn, func(options *stscreds.AssumeRoleOptions) {
			options.RoleSessionName = "KEDA"
		})
		cachedProvider = aws.NewCredentialsCache(assumeRoleCredentialProvider)
		a.logger.V(1).Info(fmt.Sprintf("using assume role to retrieve token for arnRole %s", roleArn))
	} else {
		cachedProvider = aws.NewCredentialsCache(webIdentityCredentialProvider)
		a.logger.V(1).Info(fmt.Sprintf("using assume web identity role to retrieve token for arnRole %s", roleArn))
	}
	return cachedProvider
}

func (*sharedConfigCache) retrieveStaticCredentials(awsAuthorization AuthorizationMetadata) *aws.CredentialsCache {
	staticCredentialsProvider := aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(awsAuthorization.AwsAccessKeyID, awsAuthorization.AwsSecretAccessKey, awsAuthorization.AwsSessionToken))
	return staticCredentialsProvider
}

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
