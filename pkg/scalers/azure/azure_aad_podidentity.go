package azure

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/kedacore/keda/v2/pkg/util"
)

const (
	msiURL = "http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=%s"
)

// GetAzureADPodIdentityToken returns the AADToken for resource
func GetAzureADPodIdentityToken(httpClient util.HTTPDoer, audience string) (AADToken, error) {
	var token AADToken

	urlStr := fmt.Sprintf(msiURL, url.QueryEscape(audience))
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return token, err
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
