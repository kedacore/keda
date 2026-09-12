package azure

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/redis/go-redis-entraid/shared"
	"github.com/stretchr/testify/require"
)

type fakeTokenCredential struct {
	token        azcore.AccessToken
	err          error
	calledScopes []string
}

func (f *fakeTokenCredential) GetToken(_ context.Context, options policy.TokenRequestOptions) (azcore.AccessToken, error) {
	f.calledScopes = options.Scopes
	if f.err != nil {
		return azcore.AccessToken{}, f.err
	}
	return f.token, nil
}

func TestRedisEntraIDProviderRequestTokenSuccess(t *testing.T) {
	t.Parallel()

	const scope = "https://redis.azure.com/.default"
	expected := azcore.AccessToken{
		Token:     "token-value",
		ExpiresOn: time.Unix(1_800_000_000, 0),
	}

	cred := &fakeTokenCredential{token: expected}
	provider := NewRedisEntraIDProvider(cred, scope)

	resp, err := provider.RequestToken(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{scope}, cred.calledScopes)
	require.Equal(t, shared.ResponseTypeAccessToken, resp.Type())

	accessTokenResp, ok := resp.(shared.AccessTokenIDPResponse)
	require.True(t, ok)

	token, err := accessTokenResp.AccessToken()
	require.NoError(t, err)
	require.Equal(t, expected, token)
}

func TestRedisEntraIDProviderRequestTokenError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("failed to get token")
	cred := &fakeTokenCredential{err: expectedErr}
	provider := NewRedisEntraIDProvider(cred, "https://redis.azure.com/.default")

	_, err := provider.RequestToken(context.Background())
	require.Error(t, err)
	require.ErrorIs(t, err, expectedErr)
}
