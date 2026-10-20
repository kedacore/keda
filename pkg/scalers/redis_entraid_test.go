package scalers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/go-logr/logr"
	"github.com/redis/go-redis-entraid/manager"
	"github.com/redis/go-redis-entraid/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

type fakeRedisTokenManager struct {
	startStop manager.StopFunc
	startErr  error
}

func (m *fakeRedisTokenManager) GetToken(bool) (*token.Token, error) {
	return nil, nil
}

func (m *fakeRedisTokenManager) Start(manager.TokenListener) (manager.StopFunc, error) {
	if m.startErr != nil {
		return nil, m.startErr
	}
	return m.startStop, nil
}

func TestStoppableTokenManager(t *testing.T) {
	t.Run("stop before start is a no-op", func(t *testing.T) {
		manager := &stoppableTokenManager{TokenManager: &fakeRedisTokenManager{}}
		assert.NotPanics(t, manager.Stop)
	})

	t.Run("start records stop function", func(t *testing.T) {
		var calls atomic.Int32
		fake := &fakeRedisTokenManager{
			startStop: func() error {
				calls.Add(1)
				return nil
			},
		}
		wrapped := &stoppableTokenManager{TokenManager: fake}
		stop, err := wrapped.Start(nil)
		require.NoError(t, err)
		require.NotNil(t, stop)
		wrapped.Stop()
		wrapped.Stop()
		assert.Equal(t, int32(2), calls.Load())
	})

	t.Run("start error does not record stop function", func(t *testing.T) {
		wrapped := &stoppableTokenManager{
			TokenManager: &fakeRedisTokenManager{startErr: errors.New("start failed")},
		}
		_, err := wrapped.Start(nil)
		require.Error(t, err)
		assert.NotPanics(t, wrapped.Stop)
	})
}

type fakeRedisCredential struct {
	getToken func() (azcore.AccessToken, error)
}

func (f *fakeRedisCredential) GetToken(context.Context, policy.TokenRequestOptions) (azcore.AccessToken, error) {
	return f.getToken()
}

func redisTestJWT() string {
	header, _ := json.Marshal(map[string]string{"alg": "none", "typ": "JWT"})
	payload, _ := json.Marshal(map[string]string{"oid": "test-object-id"})
	return strings.Join([]string{
		base64.RawURLEncoding.EncodeToString(header),
		base64.RawURLEncoding.EncodeToString(payload),
		"signature",
	}, ".")
}

func TestRedisClientAzureWorkloadBranch(t *testing.T) {
	var calls atomic.Int32
	original := newRedisCredential
	t.Cleanup(func() { newRedisCredential = original })
	newRedisCredential = func(logr.Logger, kedav1alpha1.AuthPodIdentity) (azcore.TokenCredential, error) {
		return &fakeRedisCredential{getToken: func() (azcore.AccessToken, error) {
			calls.Add(1)
			return azcore.AccessToken{Token: redisTestJWT(), ExpiresOn: time.Now().Add(time.Hour)}, nil
		}}, nil
	}

	info := redisConnectionInfo{Addresses: []string{"127.0.0.1:1"}, EnableTLS: true}
	identity := kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAzureWorkload}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := getRedisClient(ctx, info, 0, identity, logr.Discard())
	require.Error(t, err)
	assert.Positive(t, calls.Load())
}

func TestRedisClusterClientAzureWorkloadBranch(t *testing.T) {
	var calls atomic.Int32
	original := newRedisCredential
	t.Cleanup(func() { newRedisCredential = original })
	newRedisCredential = func(logr.Logger, kedav1alpha1.AuthPodIdentity) (azcore.TokenCredential, error) {
		return &fakeRedisCredential{getToken: func() (azcore.AccessToken, error) {
			calls.Add(1)
			return azcore.AccessToken{Token: redisTestJWT(), ExpiresOn: time.Now().Add(time.Hour)}, nil
		}}, nil
	}

	info := redisConnectionInfo{Addresses: []string{"127.0.0.1:1"}, EnableTLS: true}
	identity := kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAzureWorkload}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := getRedisClusterClient(ctx, info, identity, logr.Discard())
	require.Error(t, err)
	assert.Positive(t, calls.Load())
}

func TestRedisClientDoesNotUseCredentialWithoutPodIdentity(t *testing.T) {
	var calls atomic.Int32
	original := newRedisCredential
	t.Cleanup(func() { newRedisCredential = original })
	newRedisCredential = func(logr.Logger, kedav1alpha1.AuthPodIdentity) (azcore.TokenCredential, error) {
		calls.Add(1)
		return &fakeRedisCredential{}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := getRedisClient(ctx, redisConnectionInfo{Addresses: []string{"127.0.0.1:1"}}, 0, kedav1alpha1.AuthPodIdentity{}, logr.Discard())
	require.Error(t, err)
	assert.Zero(t, calls.Load())
}

func TestRedisClusterClientDoesNotUseCredentialWithoutPodIdentity(t *testing.T) {
	var calls atomic.Int32
	original := newRedisCredential
	t.Cleanup(func() { newRedisCredential = original })
	newRedisCredential = func(logr.Logger, kedav1alpha1.AuthPodIdentity) (azcore.TokenCredential, error) {
		calls.Add(1)
		return &fakeRedisCredential{}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := getRedisClusterClient(ctx, redisConnectionInfo{Addresses: []string{"127.0.0.1:1"}}, kedav1alpha1.AuthPodIdentity{}, logr.Discard())
	require.Error(t, err)
	assert.Zero(t, calls.Load())
}

func TestNewRedisEntraIDCredentialsProviderErrors(t *testing.T) {
	t.Run("credential constructor", func(t *testing.T) {
		original := newRedisCredential
		t.Cleanup(func() { newRedisCredential = original })
		expected := errors.New("credential failed")
		newRedisCredential = func(logr.Logger, kedav1alpha1.AuthPodIdentity) (azcore.TokenCredential, error) {
			return nil, expected
		}

		_, stop, err := newRedisEntraIDCredentialsProvider(logr.Discard(), kedav1alpha1.AuthPodIdentity{})
		require.ErrorIs(t, err, expected)
		assert.NotPanics(t, stop)
	})

	t.Run("token acquisition", func(t *testing.T) {
		original := newRedisCredential
		t.Cleanup(func() { newRedisCredential = original })
		expected := errors.New("token failed")
		newRedisCredential = func(logr.Logger, kedav1alpha1.AuthPodIdentity) (azcore.TokenCredential, error) {
			return &fakeRedisCredential{getToken: func() (azcore.AccessToken, error) {
				return azcore.AccessToken{}, expected
			}}, nil
		}

		_, stop, err := newRedisEntraIDCredentialsProvider(logr.Discard(), kedav1alpha1.AuthPodIdentity{})
		require.Error(t, err)
		assert.ErrorContains(t, err, expected.Error())
		assert.NotPanics(t, stop)
	})
}
