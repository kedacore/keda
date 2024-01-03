package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
)

func TestGetCredentialsReturnNewItemAndStoreItIfNotExist(t *testing.T) {
	cache := newSharedConfigsCache()
	cache.logger = logr.Discard()
	config := awsConfigMetadata{
		awsRegion: "test-region",
		awsAuthorization: AuthorizationMetadata{
			TriggerUniqueKey: "test-key",
		},
	}
	cacheKey := cache.getCacheKey(config.awsAuthorization)
	_, err := cache.GetCredentials(context.Background(), config.awsRegion, config.awsAuthorization)
	assert.NoError(t, err)
	assert.Contains(t, cache.items, cacheKey)
	assert.Contains(t, cache.items[cacheKey].usages, config.awsAuthorization.TriggerUniqueKey)
}

func TestGetCredentialsReturnCachedItemIfExist(t *testing.T) {
	cache := newSharedConfigsCache()
	cache.logger = logr.Discard()
	config := awsConfigMetadata{
		awsRegion: "test1-region",
		awsAuthorization: AuthorizationMetadata{
			TriggerUniqueKey: "test1-key",
		},
	}
	cfg := aws.Config{}
	cfg.AppID = "test1-app"
	cacheKey := cache.getCacheKey(config.awsAuthorization)
	cache.items[cacheKey] = cacheEntry{
		config: &cfg,
		usages: map[string]bool{
			"other-usage": true,
		},
	}
	configFromCache, err := cache.GetCredentials(context.Background(), config.awsRegion, config.awsAuthorization)
	assert.NoError(t, err)
	assert.Equal(t, &cfg, configFromCache)
	assert.Contains(t, cache.items[cacheKey].usages, config.awsAuthorization.TriggerUniqueKey)
}

func TestRemoveCachedEntryRemovesCachedItemIfNotUsages(t *testing.T) {
	cache := newSharedConfigsCache()
	cache.logger = logr.Discard()
	config := awsConfigMetadata{
		awsRegion: "test2-region",
		awsAuthorization: AuthorizationMetadata{
			TriggerUniqueKey: "test2-key",
		},
	}
	cfg := aws.Config{}
	cfg.AppID = "test2-app"
	cacheKey := cache.getCacheKey(config.awsAuthorization)
	cache.items[cacheKey] = cacheEntry{
		config: &cfg,
		usages: map[string]bool{
			config.awsAuthorization.TriggerUniqueKey: true,
		},
	}
	cache.RemoveCachedEntry(config.awsAuthorization)
	assert.NotContains(t, cache.items, cacheKey)
}

func TestRemoveCachedEntryNotRemoveCachedItemIfUsages(t *testing.T) {
	cache := newSharedConfigsCache()
	cache.logger = logr.Discard()
	config := awsConfigMetadata{
		awsRegion: "test3-region",
		awsAuthorization: AuthorizationMetadata{
			TriggerUniqueKey: "test3-key",
		},
	}
	cfg := aws.Config{}
	cfg.AppID = "test3-app"
	cacheKey := cache.getCacheKey(config.awsAuthorization)
	cache.items[cacheKey] = cacheEntry{
		config: &cfg,
		usages: map[string]bool{
			config.awsAuthorization.TriggerUniqueKey: true,
			"other-usage":                            true,
		},
	}
	cache.RemoveCachedEntry(config.awsAuthorization)
	assert.Contains(t, cache.items, cacheKey)
}
