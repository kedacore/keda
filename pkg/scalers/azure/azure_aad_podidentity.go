package azure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/kedacore/keda/v2/pkg/util"
)

const (
	msiURL               = "http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=%s"
	msiURLWithIdentityID = "http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=%s&client_id=%s"
)

func GetAzureServiceBusADPodIdentityToken(ctx context.Context, httpClient util.HTTPDoer, audience string) (AADToken, error) {
	var token AADToken

	urlStr := fmt.Sprintf(msiURL, url.QueryEscape(audience))

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return token, err
	}
	req.Header = map[string][]string{
		"Metadata": {"true"},
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return token, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return token, err
	}

	err = json.Unmarshal(body, &token)
	if err != nil {
		return token, errors.New(string(body))
	}
	return token, nil
}

// GetAzureADPodIdentityToken returns the AADToken for resource
func GetAzureADPodIdentityToken(ctx context.Context, httpClient util.HTTPDoer, audience string, identityID string) (AADToken, error) {
	var token AADToken

	var urlStr string
	if identityID == "" {
		urlStr = fmt.Sprintf(msiURL, url.QueryEscape(audience))
	} else {
		urlStr = fmt.Sprintf(msiURLWithIdentityID, url.QueryEscape(audience), identityID)
	}

	TokenRequest, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return token, err
	}
	TokenRequest.Header = map[string][]string{
		"Metadata": {"true"},
	}

	resp, err := httpClient.Do(TokenRequest)
	if err != nil {
		return token, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return token, err
	}

	err = json.Unmarshal(body, &token)
	if err != nil {
		return token, errors.New(string(body))
	}

	return token, nil
}

// AADToken is the token from Azure AD
type AADToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    string `json:"expires_in"`
	ExpiresOn    string `json:"expires_on"`
	NotBefore    string `json:"not_before"`
	Resource     string `json:"resource"`
	TokenType    string `json:"token_type"`
}
