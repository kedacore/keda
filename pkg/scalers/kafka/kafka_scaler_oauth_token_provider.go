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
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/IBM/sarama"
	"github.com/aws/aws-msk-iam-sasl-signer-go/signer"
	"github.com/aws/aws-sdk-go-v2/aws"
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

type azureADWorkloadIdentityTokenProvider struct {
	sync.Mutex
	expireAt   *time.Time
	token      string
	credential azcore.TokenCredential
	scopes     []string
	extensions map[string]string
}

func OAuthAzureADWorkloadIdentityTokenProvider(credential azcore.TokenCredential, scopes []string, extensions map[string]string) TokenProvider {
	return &azureADWorkloadIdentityTokenProvider{
		credential: credential,
		scopes:     scopes,
		extensions: extensions,
	}
}

func (a *azureADWorkloadIdentityTokenProvider) Token() (*sarama.AccessToken, error) {
	a.Lock()
	defer a.Unlock()

	if a.expireAt != nil && time.Now().Before(*a.expireAt) {
		return &sarama.AccessToken{Token: a.token, Extensions: a.extensions}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	token, err := a.credential.GetToken(ctx, policy.TokenRequestOptions{Scopes: a.scopes})
	if err != nil {
		return nil, err
	}

	a.expireAt = &token.ExpiresOn
	a.token = token.Token

	return &sarama.AccessToken{Token: token.Token, Extensions: a.extensions}, nil
}

func (a *azureADWorkloadIdentityTokenProvider) String() string {
	return "AzureADWorkloadIdentity"
}
