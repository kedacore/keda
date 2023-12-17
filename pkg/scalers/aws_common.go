package scalers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

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
	awsAuthorization awsAuthorizationMetadata
}

type CacheEntry struct {
	credentials *aws.CredentialsCache
	usages      map[string]bool // Tracks the scaledObjects requested the cache
}

var (
	roleCredentialsCache = make(map[string]CacheEntry)
	mu                   sync.Mutex
)

func getCredentialsForRole(cfg aws.Config, roleArn string, scalerUniqueKey string) (*aws.CredentialsCache, error) {
	mu.Lock()
	defer mu.Unlock()

	if cachedEntry, exists := roleCredentialsCache[roleArn]; exists {
		cachedEntry.usages[scalerUniqueKey] = true
		roleCredentialsCache[roleArn] = cachedEntry
		return cachedEntry.credentials, nil
	}

	cachedProvider, err := retrieveCredentials(cfg, roleArn)
	if err != nil {
		return nil, err
	}

	newCacheEntry := CacheEntry{
		credentials: cachedProvider,
		usages:      map[string]bool{scalerUniqueKey: true},
	}

	roleCredentialsCache[roleArn] = newCacheEntry

	return cachedProvider, nil

}

func retrieveCredentials(cfg aws.Config, roleArn string) (*aws.CredentialsCache, error) {
	stsSvc := sts.NewFromConfig(cfg)
	webIdentityTokenFile := os.Getenv("AWS_WEB_IDENTITY_TOKEN_FILE")

	webIdentityCredentialProvider := stscreds.NewWebIdentityRoleProvider(stsSvc, roleArn, stscreds.IdentityTokenFile(webIdentityTokenFile), func(options *stscreds.WebIdentityRoleOptions) {
		options.RoleSessionName = "KEDA"
	})
	var cachedProvider *aws.CredentialsCache

	_, err := webIdentityCredentialProvider.Retrieve(context.Background())
	if err != nil {
		// Fallback to Assume Role
		assumeRoleCredentialProvider := stscreds.NewAssumeRoleProvider(stsSvc, roleArn, func(options *stscreds.AssumeRoleOptions) {
			options.RoleSessionName = "KEDA"
		})
		cachedProvider = aws.NewCredentialsCache(assumeRoleCredentialProvider)
	} else {
		cachedProvider = aws.NewCredentialsCache(webIdentityCredentialProvider)
	}
	return cachedProvider, nil
}

func removeCachedEntry(scalerUniqueKey string, roleArn string) error {
	mu.Lock()
	defer mu.Unlock()

	if cachedEntry, exists := roleCredentialsCache[roleArn]; exists {
		// Delete the scalerUniqueKey from usages
		delete(cachedEntry.usages, scalerUniqueKey)

		// If no more usages, delete the entire entry from the cache
		if len(cachedEntry.usages) == 0 {
			delete(roleCredentialsCache, roleArn)
		} else {
			roleCredentialsCache[roleArn] = cachedEntry
		}
	}

	return nil

}

func getAwsConfig(ctx context.Context, awsRegion string, awsAuthorization awsAuthorizationMetadata, scalerUniqueKey string) (*aws.Config, error) {
	metadata := &awsConfigMetadata{
		awsRegion:        awsRegion,
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
		cfg.Credentials, err = getCredentialsForRole(cfg, metadata.awsAuthorization.awsRoleArn, scalerUniqueKey)
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
