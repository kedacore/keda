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

package aws

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

const testServiceAccountRoleARN = "arn:aws:iam::123456789012:role/scaling-reader"

type serviceAccountTestTransport func(*http.Request) (*http.Response, error)

func (f serviceAccountTestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func serviceAccountTestAuthorization() AuthorizationMetadata {
	return AuthorizationMetadata{
		AwsRoleArn: testServiceAccountRoleARN, AwsRegion: "us-east-1", UsingPodIdentity: true, TriggerUniqueKey: "tenant-trigger",
		ServiceAccountTokenProvider: &scalersconfig.ServiceAccountTokenProvider{
			Namespace: "tenant", ServiceAccountName: "scaling-reader", Audience: "sts.amazonaws.com",
			GetToken: func(context.Context) (string, error) { return "tenant-assertion", nil },
		},
	}
}

func serviceAccountTestResponse(req *http.Request, status int, body string) *http.Response {
	return &http.Response{
		Request: req, StatusCode: status, Header: http.Header{"Content-Type": []string{"text/xml"}},
		Body: io.NopCloser(strings.NewReader(body)),
	}
}

func serviceAccountTestCredentials(req *http.Request, accessKey string) *http.Response {
	return serviceAccountTestResponse(req, http.StatusOK, fmt.Sprintf(`<AssumeRoleWithWebIdentityResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/">
<AssumeRoleWithWebIdentityResult><Credentials><AccessKeyId>%s</AccessKeyId><SecretAccessKey>tenant-secret</SecretAccessKey><SessionToken>tenant-session</SessionToken><Expiration>%s</Expiration></Credentials></AssumeRoleWithWebIdentityResult>
</AssumeRoleWithWebIdentityResponse>`, accessKey, time.Now().Add(time.Hour).UTC().Format(time.RFC3339)))
}

func TestAWSServiceAccountExchangeAndRefresh(t *testing.T) {
	// None of these ambient credentials or configuration files may be consulted.
	t.Setenv("AWS_PROFILE", "missing-operator-profile")
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(t.TempDir(), "missing-config"))
	t.Setenv("AWS_WEB_IDENTITY_TOKEN_FILE", filepath.Join(t.TempDir(), "operator-token"))
	t.Setenv("AWS_ACCESS_KEY_ID", "operator-access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "operator-secret")
	t.Setenv("AWS_ENDPOINT_URL_STS", "https://untrusted.example")
	auth := serviceAccountTestAuthorization()
	tokenRequests := 0
	approvalRevoked := false
	auth.ServiceAccountTokenProvider.GetToken = func(ctx context.Context) (string, error) {
		tokenRequests++
		if approvalRevoked {
			return "", errors.New("service account approval revoked")
		}
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		assert.Positive(t, time.Until(deadline))
		assert.LessOrEqual(t, time.Until(deadline), 30*time.Second)
		return fmt.Sprintf("tenant-assertion-%d", tokenRequests), nil
	}
	cfg, err := newServiceAccountConfig(auth)
	require.NoError(t, err)
	exchanges := 0
	cfg.HTTPClient.(*http.Client).Transport = serviceAccountTestTransport(func(req *http.Request) (*http.Response, error) {
		exchanges++
		assert.Equal(t, "https", req.URL.Scheme)
		assert.Equal(t, "sts.us-east-1.amazonaws.com", req.URL.Host)
		assert.Empty(t, req.Header.Get("Authorization"))
		require.NoError(t, req.ParseForm())
		assert.Equal(t, "AssumeRoleWithWebIdentity", req.Form.Get("Action"))
		assert.Equal(t, testServiceAccountRoleARN, req.Form.Get("RoleArn"))
		assert.Equal(t, "KEDA", req.Form.Get("RoleSessionName"))
		assert.Equal(t, fmt.Sprintf("tenant-assertion-%d", exchanges), req.Form.Get("WebIdentityToken"))
		return serviceAccountTestCredentials(req, fmt.Sprintf("tenant-key-%d", exchanges)), nil
	})
	first, err := cfg.Credentials.Retrieve(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "tenant-key-1", first.AccessKeyID)
	assert.Equal(t, "tenant-secret", first.SecretAccessKey)
	assert.Equal(t, "tenant-session", first.SessionToken)
	second, err := cfg.Credentials.Retrieve(t.Context())
	require.NoError(t, err)
	assert.Equal(t, first, second)
	assert.Equal(t, 1, tokenRequests)
	assert.Equal(t, 1, exchanges)
	cfg.Credentials.(*aws.CredentialsCache).Invalidate()
	refreshed, err := cfg.Credentials.Retrieve(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "tenant-key-2", refreshed.AccessKeyID)
	assert.Equal(t, 2, tokenRequests)
	assert.Equal(t, 2, exchanges)
	approvalRevoked = true
	cfg.Credentials.(*aws.CredentialsCache).Invalidate()
	_, err = cfg.Credentials.Retrieve(t.Context())
	require.ErrorContains(t, err, "service account approval revoked")
	assert.Equal(t, 3, tokenRequests)
	assert.Equal(t, 2, exchanges)
}

func TestAWSServiceAccountFailuresNeverFallBack(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "operator-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "operator-secret")
	for _, test := range []struct {
		name       string
		token      string
		tokenError error
		status     int
		body       string
		errorText  string
	}{
		{name: "token rejected", tokenError: errors.New("approval revoked"), errorText: "approval revoked"},
		{name: "empty token", errorText: "assertion is empty"},
		{name: "STS denies", token: "tenant-assertion", status: http.StatusForbidden,
			body: `<ErrorResponse><Error><Code>AccessDenied</Code><Message>role does not trust this subject</Message></Error></ErrorResponse>`, errorText: "AccessDenied"},
		{name: "redirect", token: "tenant-assertion", status: http.StatusTemporaryRedirect, errorText: "failed to assume AWS role"},
		{name: "missing credentials", token: "tenant-assertion", status: http.StatusOK,
			body: `<AssumeRoleWithWebIdentityResponse><AssumeRoleWithWebIdentityResult/></AssumeRoleWithWebIdentityResponse>`, errorText: "no credentials"},
		{name: "expired credentials", token: "tenant-assertion", status: http.StatusOK,
			body: `<AssumeRoleWithWebIdentityResponse><AssumeRoleWithWebIdentityResult><Credentials><AccessKeyId>tenant-key</AccessKeyId><SecretAccessKey>secret</SecretAccessKey><SessionToken>session</SessionToken><Expiration>2000-01-01T00:00:00Z</Expiration></Credentials></AssumeRoleWithWebIdentityResult></AssumeRoleWithWebIdentityResponse>`, errorText: "incomplete or expired"},
	} {
		t.Run(test.name, func(t *testing.T) {
			auth := serviceAccountTestAuthorization()
			auth.ServiceAccountTokenProvider.GetToken = func(context.Context) (string, error) { return test.token, test.tokenError }
			cfg, err := newServiceAccountConfig(auth)
			require.NoError(t, err)
			exchanges := 0
			cfg.HTTPClient.(*http.Client).Transport = serviceAccountTestTransport(func(req *http.Request) (*http.Response, error) {
				exchanges++
				assert.Equal(t, "sts.us-east-1.amazonaws.com", req.URL.Host, "an assertion must not follow redirects")
				require.NoError(t, req.ParseForm())
				assert.Equal(t, "AssumeRoleWithWebIdentity", req.Form.Get("Action"), "AssumeRole fallback must not occur")
				response := serviceAccountTestResponse(req, test.status, test.body)
				response.Header.Set("Location", "https://untrusted.example/token")
				return response, nil
			})
			credentials, err := cfg.Credentials.Retrieve(t.Context())
			require.ErrorContains(t, err, test.errorText)
			assert.Empty(t, credentials.AccessKeyID)
			if test.token == "" {
				assert.Zero(t, exchanges)
			} else {
				assert.Equal(t, 1, exchanges)
			}
		})
	}
}

func TestAWSServiceAccountCacheIsolatesIdentities(t *testing.T) {
	cache := newSharedConfigsCache()
	auth := serviceAccountTestAuthorization()
	first, err := cache.GetCredentials(t.Context(), auth)
	require.NoError(t, err)
	otherTrigger := auth
	otherTrigger.TriggerUniqueKey = "another-trigger"
	sameIdentity, err := cache.GetCredentials(t.Context(), otherTrigger)
	require.NoError(t, err)
	assert.Same(t, first, sameIdentity)
	for _, test := range []struct {
		name   string
		change func(*AuthorizationMetadata)
	}{
		{name: "namespace", change: func(a *AuthorizationMetadata) { a.ServiceAccountTokenProvider.Namespace = "another-tenant" }},
		{name: "service account", change: func(a *AuthorizationMetadata) { a.ServiceAccountTokenProvider.ServiceAccountName = "another-reader" }},
		{name: "audience", change: func(a *AuthorizationMetadata) { a.ServiceAccountTokenProvider.Audience = "different-audience" }},
		{name: "role", change: func(a *AuthorizationMetadata) { a.AwsRoleArn = "arn:aws:iam::123456789012:role/another-role" }},
		{name: "region", change: func(a *AuthorizationMetadata) { a.AwsRegion = "us-west-2" }},
		{name: "operator identity", change: func(a *AuthorizationMetadata) { a.ServiceAccountTokenProvider = nil }},
	} {
		t.Run(test.name, func(t *testing.T) {
			other := serviceAccountTestAuthorization()
			test.change(&other)
			assert.NotEqual(t, cache.getCacheKey(auth), cache.getCacheKey(other))
			if test.name == "audience" || test.name == "operator identity" {
				return
			}
			otherConfig, err := cache.GetCredentials(t.Context(), other)
			require.NoError(t, err)
			assert.NotSame(t, first, otherConfig)
		})
	}
	cache.RemoveCachedEntry(auth)
	assert.Contains(t, cache.items, cache.getCacheKey(auth))
	cache.RemoveCachedEntry(otherTrigger)
	assert.NotContains(t, cache.items, cache.getCacheKey(auth))

	// Even malformed legacy inputs must not be able to construct a key that
	// retrieves already-cached credentials for a selected service account.
	forgedLegacy := AuthorizationMetadata{
		AwsRoleArn:    strings.Join([]string{"service-account", "tenant", "scaling-reader", "sts.amazonaws.com", testServiceAccountRoleARN, "us"}, "\x00"),
		AwsExternalID: "east", AwsRegion: "1", UsingPodIdentity: true,
	}
	assert.NotEqual(t, cache.getCacheKey(auth), cache.getCacheKey(forgedLegacy))
}

func TestAWSServiceAccountEndpointValidation(t *testing.T) {
	for _, test := range []struct {
		region    string
		partition string
		host      string
	}{
		{region: "us-east-1", partition: "aws", host: "sts.us-east-1.amazonaws.com"},
		{region: "us-gov-west-1", partition: "aws-us-gov", host: "sts.us-gov-west-1.amazonaws.com"},
		{region: "cn-north-1", partition: "aws-cn", host: "sts.cn-north-1.amazonaws.com.cn"},
		{region: "eu-isoe-west-1", partition: "aws-iso-e", host: "sts.eu-isoe-west-1.cloud.adc-e.uk"},
		{region: "us-east-1.untrusted.example", partition: "aws"},
		{region: "us-east-1/path", partition: "aws"},
		{region: "us-east-1:443", partition: "aws"},
		{region: "", partition: "aws"},
		{region: "us-east-1", partition: "untrusted"},
		{region: "cn-north-1", partition: "aws"},
	} {
		t.Run(test.partition+"/"+test.region, func(t *testing.T) {
			endpoint, err := serviceAccountSTSEndpoint("arn:"+test.partition+":iam::123456789012:role/scaling-reader", test.region)
			if test.host == "" {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "https://"+test.host, strings.TrimSuffix(endpoint, "/"))
		})
	}
}

func TestAWSServiceAccountAuthorization(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*kedav1alpha1.AuthPodIdentity, *scalersconfig.ServiceAccountTokenProvider)
	}{
		{name: "valid", change: func(*kedav1alpha1.AuthPodIdentity, *scalersconfig.ServiceAccountTokenProvider) {}},
		{name: "wrong audience", change: func(_ *kedav1alpha1.AuthPodIdentity, p *scalersconfig.ServiceAccountTokenProvider) {
			p.Audience = "https://kubernetes.default.svc"
		}},
		{name: "no callback", change: func(_ *kedav1alpha1.AuthPodIdentity, p *scalersconfig.ServiceAccountTokenProvider) { p.GetToken = nil }},
		{name: "wrong account", change: func(_ *kedav1alpha1.AuthPodIdentity, p *scalersconfig.ServiceAccountTokenProvider) {
			p.ServiceAccountName = "another-reader"
		}},
		{name: "missing namespace", change: func(_ *kedav1alpha1.AuthPodIdentity, p *scalersconfig.ServiceAccountTokenProvider) { p.Namespace = "" }},
		{name: "missing role", change: func(p *kedav1alpha1.AuthPodIdentity, _ *scalersconfig.ServiceAccountTokenProvider) { p.RoleArn = nil }},
		{name: "invalid role", change: func(p *kedav1alpha1.AuthPodIdentity, _ *scalersconfig.ServiceAccountTokenProvider) {
			p.RoleArn = new("invalid")
		}},
		{name: "identity owner", change: func(p *kedav1alpha1.AuthPodIdentity, _ *scalersconfig.ServiceAccountTokenProvider) {
			p.IdentityOwner = new("keda")
		}},
		{name: "external ID", change: func(p *kedav1alpha1.AuthPodIdentity, _ *scalersconfig.ServiceAccountTokenProvider) {
			p.ExternalID = new("external-id")
		}},
		{name: "wrong cloud", change: func(p *kedav1alpha1.AuthPodIdentity, _ *scalersconfig.ServiceAccountTokenProvider) {
			p.Provider = kedav1alpha1.PodIdentityProviderGCP
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			provider := serviceAccountTestAuthorization().ServiceAccountTokenProvider
			identity := kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAws, ServiceAccountName: new("scaling-reader"), RoleArn: new(testServiceAccountRoleARN)}
			test.change(&identity, provider)
			auth, err := GetAwsAuthorization("test", "us-east-1", identity, nil, nil, nil, provider)
			if test.name != "valid" {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Same(t, provider, auth.ServiceAccountTokenProvider)
			assert.Equal(t, testServiceAccountRoleARN, auth.AwsRoleArn)
			assert.Empty(t, auth.AwsAccessKeyID)
			_, err = GetAwsAuthorization("test", "us-east-1", identity, nil, nil, nil, nil)
			require.ErrorContains(t, err, "requires a resolved token provider")
		})
	}
}

func TestAWSServiceAccountSigV4DoesNotSuppressErrors(t *testing.T) {
	config := &scalersconfig.ScalerConfig{
		PodIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAws, ServiceAccountName: new("scaling-reader"), RoleArn: new(testServiceAccountRoleARN)},
	}
	_, err := NewSigV4RoundTripper(config, "us-east-1")
	require.ErrorContains(t, err, "resolved token provider")
	config.ServiceAccountTokenProvider = serviceAccountTestAuthorization().ServiceAccountTokenProvider
	_, err = NewSigV4RoundTripper(config, "")
	require.ErrorContains(t, err, "valid region")
	config.ServiceAccountTokenProvider.GetToken = func(context.Context) (string, error) {
		return "", errors.New("selected SigV4 identity unavailable")
	}
	rt, err := NewSigV4RoundTripper(config, "us-east-1")
	require.NoError(t, err)
	auth, err := parseAwsAMPMetadata(config, "us-east-1")
	require.NoError(t, err)
	t.Cleanup(func() { ClearAwsConfig(*auth) })
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://metrics.example/query", nil)
	require.NoError(t, err)
	response, err := rt.RoundTrip(req)
	if response != nil {
		defer response.Body.Close()
	}
	require.ErrorContains(t, err, "selected SigV4 identity unavailable")
	require.Nil(t, response)
}
