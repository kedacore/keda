package scalers

import (
	"context"
	"errors"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

var (
	gcpScopeMonitoringRead = "https://www.googleapis.com/auth/monitoring.read"

	errGoogleApplicationCrendentialsNotFound = errors.New("google application credentials not found")
)

type gcpAuthorizationMetadata struct {
	GoogleApplicationCredentials     string
	GoogleApplicationCredentialsFile string
	podIdentityProviderEnabled       bool
}

func (a *gcpAuthorizationMetadata) tokenSource(ctx context.Context, scopes ...string) (oauth2.TokenSource, error) {
	if a.podIdentityProviderEnabled {
		return google.DefaultTokenSource(ctx, scopes...)
	}

	if a.GoogleApplicationCredentials != "" {
		creds, err := google.CredentialsFromJSON(ctx, []byte(a.GoogleApplicationCredentials), scopes...)
		if err != nil {
			return nil, err
		}

		return creds.TokenSource, nil
	}

	if a.GoogleApplicationCredentialsFile != "" {
		data, err := os.ReadFile(a.GoogleApplicationCredentialsFile)
		if err != nil {
			return nil, err
		}

		creds, err := google.CredentialsFromJSON(ctx, data, scopes...)
		if err != nil {
			return nil, err
		}

		return creds.TokenSource, nil
	}

	return nil, errGoogleApplicationCrendentialsNotFound
}

func getGCPAuthorization(config *ScalerConfig) (*gcpAuthorizationMetadata, error) {
	if config.PodIdentity.Provider == kedav1alpha1.PodIdentityProviderGCP {
		return &gcpAuthorizationMetadata{podIdentityProviderEnabled: true}, nil
	}

	if creds := config.AuthParams["GoogleApplicationCredentials"]; creds != "" {
		return &gcpAuthorizationMetadata{GoogleApplicationCredentials: creds}, nil
	}

	if creds := config.TriggerMetadata["credentialsFromEnv"]; creds != "" {
		return &gcpAuthorizationMetadata{GoogleApplicationCredentials: config.ResolvedEnv[creds]}, nil
	}

	if credsFile := config.TriggerMetadata["credentialsFromEnvFile"]; credsFile != "" {
		return &gcpAuthorizationMetadata{GoogleApplicationCredentialsFile: config.ResolvedEnv[credsFile]}, nil
	}

	return nil, errGoogleApplicationCrendentialsNotFound
}

func getGCPOAuth2HTTPTransport(config *ScalerConfig, base http.RoundTripper, scopes ...string) (http.RoundTripper, error) {
	a, err := getGCPAuthorization(config)
	if err != nil {
		return nil, err
	}

	ts, err := a.tokenSource(context.Background(), scopes...)
	if err != nil {
		return nil, err
	}

	return &oauth2.Transport{Source: ts, Base: base}, nil
}
