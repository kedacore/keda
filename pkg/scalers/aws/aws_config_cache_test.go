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
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
)

func TestGetCredentialsReturnNewItemAndStoreItIfNotExist(t *testing.T) {
	cache := newSharedConfigsCache()
	cache.logger = logr.Discard()
	awsAuthorization := AuthorizationMetadata{
		TriggerUniqueKey: "test-key",
		AwsRegion:        "test-region",
	}
	cacheKey := cache.getCacheKey(awsAuthorization)
	_, err := cache.GetCredentials(context.Background(), awsAuthorization)
	assert.NoError(t, err)
	assert.Contains(t, cache.items, cacheKey)
	assert.Contains(t, cache.items[cacheKey].usages, awsAuthorization.TriggerUniqueKey)
}

func TestGetCredentialsReturnCachedItemIfExist(t *testing.T) {
	cache := newSharedConfigsCache()
	cache.logger = logr.Discard()
	awsAuthorization := AuthorizationMetadata{
		TriggerUniqueKey: "test1-key",
		AwsRegion:        "test1-region",
	}
	cfg := aws.Config{}
	cfg.AppID = "test1-app"
	cacheKey := cache.getCacheKey(awsAuthorization)
	cache.items[cacheKey] = cacheEntry{
		config: &cfg,
		usages: map[string]bool{
			"other-usage": true,
		},
	}
	configFromCache, err := cache.GetCredentials(context.Background(), awsAuthorization)
	assert.NoError(t, err)
	assert.Equal(t, &cfg, configFromCache)
	assert.Contains(t, cache.items[cacheKey].usages, awsAuthorization.TriggerUniqueKey)
}

func TestRemoveCachedEntryRemovesCachedItemIfNotUsages(t *testing.T) {
	cache := newSharedConfigsCache()
	cache.logger = logr.Discard()
	awsAuthorization := AuthorizationMetadata{
		TriggerUniqueKey: "test2-key",
		AwsRegion:        "test2-region",
	}
	cfg := aws.Config{}
	cfg.AppID = "test2-app"
	cacheKey := cache.getCacheKey(awsAuthorization)
	cache.items[cacheKey] = cacheEntry{
		config: &cfg,
		usages: map[string]bool{
			awsAuthorization.TriggerUniqueKey: true,
		},
	}
	cache.RemoveCachedEntry(awsAuthorization)
	assert.NotContains(t, cache.items, cacheKey)
}

func TestRemoveCachedEntryNotRemoveCachedItemIfUsages(t *testing.T) {
	cache := newSharedConfigsCache()
	cache.logger = logr.Discard()
	awsAuthorization := AuthorizationMetadata{
		TriggerUniqueKey: "test3-key",
		AwsRegion:        "test3-region",
	}
	cfg := aws.Config{}
	cfg.AppID = "test3-app"
	cacheKey := cache.getCacheKey(awsAuthorization)
	cache.items[cacheKey] = cacheEntry{
		config: &cfg,
		usages: map[string]bool{
			awsAuthorization.TriggerUniqueKey: true,
			"other-usage":                     true,
		},
	}
	cache.RemoveCachedEntry(awsAuthorization)
	assert.Contains(t, cache.items, cacheKey)
}

func TestCredentialsShouldBeCachedPerRegion(t *testing.T) {
	cache := newSharedConfigsCache()
	cache.logger = logr.Discard()
	awsAuthorization1 := AuthorizationMetadata{
		TriggerUniqueKey: "test4-key",
		AwsRegion:        "test4-region1",
	}
	awsAuthorization2 := AuthorizationMetadata{
		TriggerUniqueKey: "test4-key",
		AwsRegion:        "test4-region2",
	}
	cred1, err1 := cache.GetCredentials(context.Background(), awsAuthorization1)
	cred2, err2 := cache.GetCredentials(context.Background(), awsAuthorization2)

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NotEqual(t, cred1, cred2, "Credentials should be stored per region")
}
