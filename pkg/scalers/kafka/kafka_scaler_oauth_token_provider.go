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

package kafka

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/aws/aws-msk-iam-sasl-signer-go/signer"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

type TokenProvider interface {
	sarama.AccessTokenProvider
	String() string
}

type oauthBearerTokenProvider struct {
	tokenSource oauth2.TokenSource
	extensions  map[string]string
}

func OAuthBearerTokenProvider(clientID, clientSecret, tokenURL string, scopes []string, extensions map[string]string) TokenProvider {
	cfg := clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     tokenURL,
		Scopes:       scopes,
	}

	return &oauthBearerTokenProvider{
		tokenSource: cfg.TokenSource(context.Background()),
		extensions:  extensions,
	}
}

const (
	// staticTokenRefreshWindowFactor matches the Kafka Java client's default sasl.login.refresh.window.factor
	staticTokenRefreshWindowFactor = 0.8
	// staticTokenRefreshMinPeriod limits refreshes when the source returns a token that is already due,
	// e.g. a JWT in a Secret that has not been rotated yet
	staticTokenRefreshMinPeriod  = time.Minute
	staticTokenRefreshRetryDelay = 10 * time.Second
	staticTokenRefreshTimeout    = 5 * time.Second
)

type staticTokenProvider struct {
	mu         sync.Mutex
	token      string
	refreshAt  time.Time
	extensions map[string]string
	refresh    func(context.Context) (string, error)
	now        func() time.Time
}

// OAuthStaticTokenProvider returns an already-issued token, e.g. a Kubernetes bound service
// account token. If the token is a JWT and refresh is set, a new token is obtained through
// refresh once 80% of its lifetime has elapsed. If refreshing fails, the current token is
// kept and refreshing is retried after 10 seconds.
func OAuthStaticTokenProvider(token string, extensions map[string]string, refresh func(context.Context) (string, error)) TokenProvider {
	return &staticTokenProvider{
		token:      token,
		refreshAt:  tokenRefreshTime(token, time.Now()),
		extensions: extensions,
		refresh:    refresh,
		now:        time.Now,
	}
}

func (p *staticTokenProvider) Token() (*sarama.AccessToken, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.refresh != nil && !p.refreshAt.IsZero() && !p.now().Before(p.refreshAt) {
		ctx, cancel := context.WithTimeout(context.Background(), staticTokenRefreshTimeout)
		defer cancel()
		now := p.now()
		if token, err := p.refresh(ctx); err == nil && token != "" {
			p.token = token
			p.refreshAt = tokenRefreshTime(token, now)
			if minRefreshAt := now.Add(staticTokenRefreshMinPeriod); !p.refreshAt.IsZero() && p.refreshAt.Before(minRefreshAt) {
				p.refreshAt = minRefreshAt
			}
		} else {
			p.refreshAt = now.Add(staticTokenRefreshRetryDelay)
		}
	}

	return &sarama.AccessToken{Token: p.token, Extensions: p.extensions}, nil
}

func (p *staticTokenProvider) String() string {
	return "OAuthBearer"
}

// tokenRefreshTime returns when 80% of a JWT's lifetime has elapsed, or zero if the token has no expiry.
// The token is not verified here; that is the broker's job.
func tokenRefreshTime(token string, now time.Time) time.Time {
	claims := jwt.RegisteredClaims{}
	// Claims are still decoded if the signing method is unknown to the parser (ErrTokenUnverifiable)
	if _, _, err := jwt.NewParser().ParseUnverified(token, &claims); (err != nil && !errors.Is(err, jwt.ErrTokenUnverifiable)) || claims.ExpiresAt == nil {
		return time.Time{}
	}
	issuedAt := now
	if claims.IssuedAt != nil {
		issuedAt = claims.IssuedAt.Time
	}
	return issuedAt.Add(time.Duration(float64(claims.ExpiresAt.Sub(issuedAt)) * staticTokenRefreshWindowFactor))
}

func (o *oauthBearerTokenProvider) Token() (*sarama.AccessToken, error) {
	token, err := o.tokenSource.Token()
	if err != nil {
		return nil, err
	}

	return &sarama.AccessToken{Token: token.AccessToken, Extensions: o.extensions}, nil
}

func (o *oauthBearerTokenProvider) String() string {
	return "OAuthBearer"
}

type mskTokenProvider struct {
	sync.Mutex
	expireAt            *time.Time
	token               string
	region              string
	credentialsProvider aws.CredentialsProvider
}

func OAuthMSKTokenProvider(cfg *aws.Config) TokenProvider {
	return &mskTokenProvider{
		region:              cfg.Region,
		credentialsProvider: cfg.Credentials,
	}
}

func (m *mskTokenProvider) Token() (*sarama.AccessToken, error) {
	m.Lock()
	defer m.Unlock()

	if m.expireAt != nil && time.Now().Before(*m.expireAt) {
		return &sarama.AccessToken{Token: m.token}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	token, expirationMs, err := signer.GenerateAuthTokenFromCredentialsProvider(ctx, m.region, m.credentialsProvider)
	if err != nil {
		return nil, err
	}

	expirationTime := time.UnixMilli(expirationMs)
	m.expireAt = &expirationTime
	m.token = token

	return &sarama.AccessToken{Token: token}, err
}

func (m *mskTokenProvider) String() string {
	return "MSK"
}
