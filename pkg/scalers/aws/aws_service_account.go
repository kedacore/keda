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
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

var (
	serviceAccountRegionPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*-[0-9]+$`)
	serviceAccountRolePattern   = regexp.MustCompile(`^arn:[a-z0-9-]+:iam::[0-9]{12}:role/[\x21-\x7e]+$`)
)

func validateServiceAccountAuthorization(auth AuthorizationMetadata) error {
	provider := auth.ServiceAccountTokenProvider
	if provider == nil || provider.GetToken == nil || provider.Namespace == "" || provider.ServiceAccountName == "" {
		return fmt.Errorf("AWS service account authentication requires a resolved token provider")
	}
	if provider.Audience != "sts.amazonaws.com" {
		return fmt.Errorf("AWS service account authentication requires the administrator-approved audience sts.amazonaws.com")
	}
	if !auth.UsingPodIdentity || auth.AwsExternalID != "" || auth.AwsAccessKeyID != "" || auth.AwsSecretAccessKey != "" || auth.AwsSessionToken != "" {
		return fmt.Errorf("AWS service account authentication cannot use another credential source or externalID")
	}
	_, err := serviceAccountSTSEndpoint(auth.AwsRoleArn, auth.AwsRegion)
	return err
}

// serviceAccountSTSEndpoint limits assertions to AWS STS endpoints resolved by the
// SDK. Neither scaler endpoint overrides nor ambient SDK configuration select
// where the Kubernetes assertion is sent.
func serviceAccountSTSEndpoint(roleARN, region string) (string, error) {
	if !serviceAccountRegionPattern.MatchString(region) || !serviceAccountRolePattern.MatchString(roleARN) {
		return "", fmt.Errorf("AWS service account authentication requires a valid region and IAM role ARN")
	}
	role, err := arn.Parse(roleARN)
	if err != nil {
		return "", fmt.Errorf("invalid AWS service account role ARN: %w", err)
	}
	partitionSuffixes := map[string]string{
		"aws":        "amazonaws.com",
		"aws-us-gov": "amazonaws.com",
		"aws-cn":     "amazonaws.com.cn",
		"aws-iso":    "c2s.ic.gov",
		"aws-iso-b":  "sc2s.sgov.gov",
		"aws-iso-e":  "cloud.adc-e.uk",
		"aws-iso-f":  "csp.hci.ic.gov",
		"aws-eusc":   "amazonaws.eu",
	}
	suffix, ok := partitionSuffixes[role.Partition]
	if !ok {
		return "", fmt.Errorf("unsupported AWS partition for service account authentication: %s", role.Partition)
	}
	endpoint, err := sts.NewDefaultEndpointResolverV2().ResolveEndpoint(context.Background(), sts.EndpointParameters{Region: aws.String(region)})
	if err != nil {
		return "", fmt.Errorf("failed to resolve AWS STS endpoint: %w", err)
	}
	if endpoint.URI.Scheme != "https" || endpoint.URI.Host != "sts."+region+"."+suffix {
		return "", fmt.Errorf("AWS STS endpoint does not match the role ARN partition")
	}
	return endpoint.URI.String(), nil
}

func newServiceAccountConfig(auth AuthorizationMetadata) (*aws.Config, error) {
	if err := validateServiceAccountAuthorization(auth); err != nil {
		return nil, err
	}
	endpoint, err := serviceAccountSTSEndpoint(auth.AwsRoleArn, auth.AwsRegion)
	if err != nil {
		return nil, err
	}
	httpClient := &http.Client{
		Transport: awshttp.NewBuildableClient().GetTransport(),
		Timeout:   30 * time.Second,
		// A redirect would replay the Kubernetes assertion in the POST body.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	// Do not load the default credential chain: selected service accounts must
	// never fall back to the operator's token file, IAM role, or node identity.
	cfg := &aws.Config{Region: auth.AwsRegion, HTTPClient: httpClient, Credentials: aws.AnonymousCredentials{}}
	stsClient := sts.NewFromConfig(*cfg, func(options *sts.Options) {
		options.BaseEndpoint = aws.String(endpoint)
	})
	cfg.Credentials = aws.NewCredentialsCache(&serviceAccountCredentials{
		client: stsClient, roleARN: auth.AwsRoleArn, tokenProvider: auth.ServiceAccountTokenProvider,
	})
	return cfg, nil
}

type serviceAccountCredentials struct {
	client        stscreds.AssumeRoleWithWebIdentityAPIClient
	roleARN       string
	tokenProvider *scalersconfig.ServiceAccountTokenProvider
}

func (p *serviceAccountCredentials) Retrieve(ctx context.Context) (aws.Credentials, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	token, err := p.tokenProvider.GetToken(ctx)
	if err != nil {
		return aws.Credentials{}, fmt.Errorf("failed to get AWS service account assertion: %w", err)
	}
	if token == "" {
		return aws.Credentials{}, fmt.Errorf("AWS service account assertion is empty")
	}
	response, err := p.client.AssumeRoleWithWebIdentity(ctx, &sts.AssumeRoleWithWebIdentityInput{
		RoleArn: aws.String(p.roleARN), RoleSessionName: aws.String("KEDA"), WebIdentityToken: aws.String(token),
	})
	if err != nil {
		return aws.Credentials{}, fmt.Errorf("failed to assume AWS role with the selected service account: %w", err)
	}
	if response == nil || response.Credentials == nil {
		return aws.Credentials{}, fmt.Errorf("AWS STS returned no credentials")
	}
	credentials := response.Credentials
	if aws.ToString(credentials.AccessKeyId) == "" || aws.ToString(credentials.SecretAccessKey) == "" || aws.ToString(credentials.SessionToken) == "" || !aws.ToTime(credentials.Expiration).After(time.Now()) {
		return aws.Credentials{}, fmt.Errorf("AWS STS returned incomplete or expired credentials")
	}
	return aws.Credentials{
		AccessKeyID: aws.ToString(credentials.AccessKeyId), SecretAccessKey: aws.ToString(credentials.SecretAccessKey),
		SessionToken: aws.ToString(credentials.SessionToken), CanExpire: true, Expires: *credentials.Expiration,
		Source: "KEDAServiceAccountWebIdentity",
	}, nil
}
