/*
Copyright 2026 The KEDA Authors

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

package gcp

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google/externalaccount"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

const workloadIdentityRequestTimeout = 30 * time.Second

var (
	workloadIdentityProviderPattern = regexp.MustCompile(`^//iam\.googleapis\.com/projects/[0-9]+/locations/global/workloadIdentityPools/[a-z0-9-]+/providers/[a-z0-9-]+$`)
	googleServiceAccountPattern     = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*@([a-zA-Z0-9][a-zA-Z0-9.-]*\.iam\.gserviceaccount\.com|developer\.gserviceaccount\.com)$`)
)

// WorkloadIdentityProviderAudience validates an administrator-approved OIDC pool
// provider audience and returns its canonical Google STS resource name. The
// original audience must still be used when requesting the Kubernetes token.
func WorkloadIdentityProviderAudience(audience string) (string, error) {
	canonical := strings.TrimPrefix(audience, "https:")
	if !workloadIdentityProviderPattern.MatchString(canonical) {
		return "", errors.New("service account token audience must identify a Google workload identity pool provider: //iam.googleapis.com/projects/PROJECT_NUMBER/locations/global/workloadIdentityPools/POOL/providers/PROVIDER (optionally prefixed with https:)")
	}
	return canonical, nil
}

func (a *AuthorizationMetadata) workloadIdentityConfig() (externalaccount.Config, error) {
	if a.ServiceAccountTokenProvider == nil || a.ServiceAccountTokenProvider.GetToken == nil {
		return externalaccount.Config{}, errors.New("GCP workload identity requires a service account token provider")
	}
	audience, err := WorkloadIdentityProviderAudience(a.ServiceAccountTokenProvider.Audience)
	if err != nil {
		return externalaccount.Config{}, err
	}
	config := externalaccount.Config{
		Audience:             audience,
		SubjectTokenType:     "urn:ietf:params:oauth:token-type:jwt",
		TokenURL:             "https://sts.googleapis.com/v1/token",
		SubjectTokenSupplier: serviceAccountTokenSupplier{provider: a.ServiceAccountTokenProvider},
	}
	if a.IdentityID != "" {
		if !googleServiceAccountPattern.MatchString(a.IdentityID) {
			return externalaccount.Config{}, errors.New("GCP identityId must be a Google service account email")
		}
		config.ServiceAccountImpersonationURL = "https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/" + a.IdentityID + ":generateAccessToken"
	}
	return config, nil
}

func (a *AuthorizationMetadata) workloadIdentityTokenSource(ctx context.Context, scopes ...string) (oauth2.TokenSource, error) {
	config, err := a.workloadIdentityConfig()
	if err != nil {
		return nil, err
	}
	config.Scopes = scopes

	// externalaccount retains this context for later refreshes, which can outlive
	// the reconciliation or polling request that first constructed the scaler.
	ctx = context.WithoutCancel(ctx)
	client := *http.DefaultClient
	if configured, ok := ctx.Value(oauth2.HTTPClient).(*http.Client); ok && configured != nil {
		client = *configured
	}
	if client.Timeout <= 0 || client.Timeout > workloadIdentityRequestTimeout {
		client.Timeout = workloadIdentityRequestTimeout
	}
	// Assertions and access tokens must stay at the fixed Google endpoints.
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	ctx = context.WithValue(ctx, oauth2.HTTPClient, &client)
	return externalaccount.NewTokenSource(ctx, config)
}

type serviceAccountTokenSupplier struct {
	provider *scalersconfig.ServiceAccountTokenProvider
}

func (s serviceAccountTokenSupplier) SubjectToken(ctx context.Context, _ externalaccount.SupplierOptions) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, workloadIdentityRequestTimeout)
	defer cancel()
	token, err := s.provider.GetToken(ctx)
	if err != nil {
		return "", err
	}
	if token == "" {
		return "", errors.New("service account token provider returned an empty token")
	}
	return token, nil
}
