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
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

type fakeTokenCredential struct {
	calls int
	token azcore.AccessToken
	err   error
}

func (f *fakeTokenCredential) GetToken(_ context.Context, _ policy.TokenRequestOptions) (azcore.AccessToken, error) {
	f.calls++
	return f.token, f.err
}

func TestAzureADWorkloadIdentityTokenProviderReturnsToken(t *testing.T) {
	cred := &fakeTokenCredential{token: azcore.AccessToken{Token: "test-token", ExpiresOn: time.Now().Add(time.Hour)}}
	provider := OAuthAzureADWorkloadIdentityTokenProvider(cred, []string{"scope"}, map[string]string{"logicalCluster": "lkc-1"})

	token, err := provider.Token()
	if err != nil {
		t.Fatal("Expected success but got error", err)
	}
	if token.Token != "test-token" {
		t.Errorf("Expected token to be %v but got %v", "test-token", token.Token)
	}
	if token.Extensions["logicalCluster"] != "lkc-1" {
		t.Errorf("Expected extensions to be carried through, got %v", token.Extensions)
	}
	if provider.String() != "AzureADWorkloadIdentity" {
		t.Errorf("Expected provider name to be AzureADWorkloadIdentity but got %v", provider.String())
	}
}

func TestAzureADWorkloadIdentityTokenProviderCachesUntilExpiry(t *testing.T) {
	cred := &fakeTokenCredential{token: azcore.AccessToken{Token: "cached-token", ExpiresOn: time.Now().Add(time.Hour)}}
	provider := OAuthAzureADWorkloadIdentityTokenProvider(cred, []string{"scope"}, nil)

	if _, err := provider.Token(); err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Token(); err != nil {
		t.Fatal(err)
	}

	if cred.calls != 1 {
		t.Errorf("Expected credential to be called once due to caching but got %v calls", cred.calls)
	}
}

func TestAzureADWorkloadIdentityTokenProviderRefreshesAfterExpiry(t *testing.T) {
	cred := &fakeTokenCredential{token: azcore.AccessToken{Token: "expiring-token", ExpiresOn: time.Now().Add(-time.Minute)}}
	provider := OAuthAzureADWorkloadIdentityTokenProvider(cred, []string{"scope"}, nil)

	if _, err := provider.Token(); err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Token(); err != nil {
		t.Fatal(err)
	}

	if cred.calls != 2 {
		t.Errorf("Expected credential to be called twice since the cached token was already expired, got %v calls", cred.calls)
	}
}

func TestAzureADWorkloadIdentityTokenProviderPropagatesError(t *testing.T) {
	cred := &fakeTokenCredential{err: errors.New("boom")}
	provider := OAuthAzureADWorkloadIdentityTokenProvider(cred, []string{"scope"}, nil)

	if _, err := provider.Token(); err == nil {
		t.Fatal("Expected error but got success")
	}
}
