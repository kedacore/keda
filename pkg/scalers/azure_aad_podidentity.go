package scalers

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
)

const (
	MSI_URL = "http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=https%3A%2F%2Fstorage.azure.com%2F"
)

func getAzureADPodIdentityToken() (AADToken, error) {

	var token AADToken

	resp, err := http.Get(MSI_URL)
	if err != nil {
		return token, err
	}

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
