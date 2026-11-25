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
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const workloadIdentityRequestTimeout = 30 * time.Second

var (
	workloadIdentityProviderPattern = regexp.MustCompile(`^//iam\.googleapis\.com/projects/[0-9]+/locations/global/workloadIdentityPools/[a-z0-9-]+/providers/[a-z0-9-]+$`)
	// gkeIdentityNamespacePattern matches the STS audience of a GKE project
	// workload identity pool (PROJECT_ID.svc.id.goog) and the cluster issuer.
	gkeIdentityNamespacePattern = regexp.MustCompile(`^identitynamespace:([a-z][a-z0-9-]{4,28}[a-z0-9]\.svc\.id\.goog):https://container\.googleapis\.com/v1/projects/[a-z][a-z0-9-]{4,28}[a-z0-9]/locations/[a-z0-9-]+/clusters/[a-z](?:[a-z0-9-]{0,38}[a-z0-9])?$`)
)

// WorkloadIdentityAudiences validates an administrator-approved federation
// audience. It returns the audience for the Kubernetes service account token
// and the Google STS audience. Two forms are accepted:
//   - a workload identity pool provider, //iam.googleapis.com/projects/PROJECT_NUMBER/locations/global/workloadIdentityPools/POOL/providers/PROVIDER,
//     optionally prefixed with https:. The Kubernetes token uses the audience as configured.
//   - a GKE project workload identity pool, identitynamespace:PROJECT_ID.svc.id.goog:https://container.googleapis.com/v1/projects/PROJECT_ID/locations/LOCATION/clusters/CLUSTER.
//     The Kubernetes token uses PROJECT_ID.svc.id.goog, as the GKE metadata server does.
func WorkloadIdentityAudiences(audience string) (tokenAudience, stsAudience string, err error) {
	if match := gkeIdentityNamespacePattern.FindStringSubmatch(audience); match != nil {
		return match[1], audience, nil
	}
	canonical := strings.TrimPrefix(audience, "https:")
	if !workloadIdentityProviderPattern.MatchString(canonical) {
		return "", "", errors.New("service account token audience must identify a Google workload identity pool provider: " +
			"//iam.googleapis.com/projects/PROJECT_NUMBER/locations/global/workloadIdentityPools/POOL/providers/PROVIDER (optionally prefixed with https:), " +
			"or a GKE workload identity pool: identitynamespace:PROJECT_ID.svc.id.goog:https://container.googleapis.com/v1/projects/PROJECT_ID/locations/LOCATION/clusters/CLUSTER")
	}
	return audience, canonical, nil
}

func (a *AuthorizationMetadata) workloadIdentityTokenSource(ctx context.Context, scopes ...string) (oauth2.TokenSource, error) {
	_, audience, err := WorkloadIdentityAudiences(a.ServiceAccountTokenProvider.Audience)
	if err != nil {
		return nil, err
	}
	config := externalaccount.Config{
		Audience:             audience,
		SubjectTokenType:     "urn:ietf:params:oauth:token-type:jwt",
		TokenURL:             "https://sts.googleapis.com/v1/token",
		SubjectTokenSupplier: serviceAccountTokenSupplier{provider: a.ServiceAccountTokenProvider},
		Scopes:               scopes,
	}
	// IdentityID is validated as a Google service account email by the API types.
	if a.IdentityID != "" {
		config.ServiceAccountImpersonationURL = "https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/" + a.IdentityID + ":generateAccessToken"
	}

	// externalaccount retains this context for later refreshes, which can outlive
	// the reconciliation or polling request that first constructed the scaler.
	ctx = context.WithoutCancel(ctx)
	client := http.Client{Transport: kedautil.CreateRT(false)}
	if configured, ok := ctx.Value(oauth2.HTTPClient).(*http.Client); ok && configured != nil {
		client = *configured
	}
	client.Timeout = workloadIdentityRequestTimeout
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
	return s.provider.GetToken(ctx)
}
