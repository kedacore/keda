package azure

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/redis/go-redis-entraid/shared"
)

type RedisEntraIDProvider struct {
	cred   azcore.TokenCredential
	scopes []string
}

func NewRedisEntraIDProvider(cred azcore.TokenCredential, resource string) *RedisEntraIDProvider {
	return &RedisEntraIDProvider{
		cred:   cred,
		scopes: []string{resource},
	}
}

func (p *RedisEntraIDProvider) RequestToken(ctx context.Context) (shared.IdentityProviderResponse, error) {
	token, err := p.cred.GetToken(ctx, policy.TokenRequestOptions{Scopes: p.scopes})
	if err != nil {
		return nil, fmt.Errorf("error getting azure token: %w", err)
	}

	return shared.NewIDPResponse(shared.ResponseTypeAccessToken, &token)
}
