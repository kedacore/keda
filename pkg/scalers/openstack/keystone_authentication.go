package openstack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"time"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const tokensEndpoint = "/auth/tokens"

// KeystoneAuthMetadata contains all the necessary metadata for Keystone authentication
type KeystoneAuthMetadata struct {
	AuthURL    string       `json:"-"`
	AuthToken  string       `json:"-"`
	HTTPClient *http.Client `json:"-"`
	Properties *authProps   `json:"auth"`
}

type authProps struct {
	Identity *identityProps `json:"identity"`
	Scope    *scopeProps    `json:"scope,omitempty"`
}

type identityProps struct {
	Methods       []string            `json:"methods"`
	Password      *passwordProps      `json:"password,omitempty"`
	AppCredential *appCredentialProps `json:"application_credential,omitempty"`
}

type passwordProps struct {
	User *userProps `json:"user"`
}

type appCredentialProps struct {
	ID     string `json:"id"`
	Secret string `json:"secret"`
}

type scopeProps struct {
	Project *projectProps `json:"project"`
}

type userProps struct {
	ID       string `json:"id"`
	Password string `json:"password"`
}

type projectProps struct {
	ID string `json:"id"`
}

// GetToken retrieves a token from Keystone
func (authProps *KeystoneAuthMetadata) GetToken() (string, error) {
	jsonBody, jsonError := json.Marshal(authProps)

	if jsonError != nil {
		return "", jsonError
	}

	body := bytes.NewReader(jsonBody)

	tokenURL, err := url.Parse(authProps.AuthURL)

	if err != nil {
		return "", fmt.Errorf("the authURL is invalid: %s", err.Error())
	}

	tokenURL.Path = path.Join(tokenURL.Path, tokensEndpoint)

	getTokenRequest, getTokenRequestError := http.NewRequest("POST", tokenURL.String(), body)

	getTokenRequest.Header.Set("Content-Type", "application/json")

	if getTokenRequestError != nil {
		return "", getTokenRequestError
	}

	resp, requestError := authProps.HTTPClient.Do(getTokenRequest)

	if requestError != nil {
		return "", requestError
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		authProps.AuthToken = resp.Header["X-Subject-Token"][0]
		return resp.Header["X-Subject-Token"][0], nil
	}

	errBody, readBodyErr := ioutil.ReadAll(resp.Body)

	if readBodyErr != nil {
		return "", readBodyErr
	}

	return "", fmt.Errorf(string(errBody))
}

// IsTokenValid checks if a authentication token is valid
func IsTokenValid(authProps KeystoneAuthMetadata) (bool, error) {
	token := authProps.AuthToken

	tokenURL, err := url.Parse(authProps.AuthURL)

	if err != nil {
		return false, fmt.Errorf("the authURL is invalid: %s", err.Error())
	}

	tokenURL.Path = path.Join(tokenURL.Path, tokensEndpoint)

	checkTokenRequest, checkRequestError := http.NewRequest("HEAD", tokenURL.String(), nil)
	checkTokenRequest.Header.Set("X-Subject-Token", token)
	checkTokenRequest.Header.Set("X-Auth-Token", token)

	if checkRequestError != nil {
		return false, checkRequestError
	}

	checkResp, requestError := authProps.HTTPClient.Do(checkTokenRequest)

	if requestError != nil {
		return false, requestError
	}

	defer checkResp.Body.Close()

	if checkResp.StatusCode >= 400 {
		return false, nil
	}

	return true, nil
}

// NewPasswordAuth creates a struct containing metadata for authentication using password method
func NewPasswordAuth(authURL string, userID string, userPassword string, projectID string, httpTimeout int) (*KeystoneAuthMetadata, error) {
	var tokenError error

	passAuth := new(KeystoneAuthMetadata)

	passAuth.Properties = new(authProps)

	passAuth.Properties.Scope = new(scopeProps)
	passAuth.Properties.Scope.Project = new(projectProps)

	passAuth.Properties.Identity = new(identityProps)
	passAuth.Properties.Identity.Password = new(passwordProps)
	passAuth.Properties.Identity.Password.User = new(userProps)

	url, err := url.Parse(authURL)

	if err != nil {
		return nil, fmt.Errorf("authURL is invalid: %s", err.Error())
	}

	url.Path = path.Join(url.Path, "")

	passAuth.AuthURL = url.String()

	passAuth.HTTPClient = kedautil.CreateHTTPClient(time.Duration(httpTimeout) * time.Second)

	passAuth.Properties.Identity.Methods = []string{"password"}
	passAuth.Properties.Identity.Password.User.ID = userID
	passAuth.Properties.Identity.Password.User.Password = userPassword

	passAuth.Properties.Scope.Project.ID = projectID

	passAuth.AuthToken, tokenError = passAuth.GetToken()

	return passAuth, tokenError
}

// NewAppCredentialsAuth creates a struct containing metadata for authentication using application credentials method
func NewAppCredentialsAuth(authURL string, id string, secret string, httpTimeout int) (*KeystoneAuthMetadata, error) {
	var tokenError error

	appAuth := new(KeystoneAuthMetadata)

	appAuth.Properties = new(authProps)

	appAuth.Properties.Identity = new(identityProps)

	url, err := url.Parse(authURL)

	if err != nil {
		return nil, fmt.Errorf("authURL is invalid: %s", err.Error())
	}

	url.Path = path.Join(url.Path, "")

	appAuth.AuthURL = url.String()

	appAuth.HTTPClient = kedautil.CreateHTTPClient(time.Duration(httpTimeout) * time.Second)

	appAuth.Properties.Identity.AppCredential = new(appCredentialProps)
	appAuth.Properties.Identity.Methods = []string{"application_credential"}
	appAuth.Properties.Identity.AppCredential.ID = id
	appAuth.Properties.Identity.AppCredential.Secret = secret

	appAuth.AuthToken, tokenError = appAuth.GetToken()

	return appAuth, tokenError
}
