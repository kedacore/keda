package openstack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"

	openstackutil "github.com/kedacore/keda/v2/pkg/scalers/openstack/utils"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const tokensEndpoint = "/v3/auth/tokens"
const catalogEndpoint = "/v3/auth/catalog"

// Client is a struct containing an authentication token and an HTTP client for HTTP requests.
// It can also have a public URL for an specific OpenStack project or service.
// "authMetadata" is an unexported attribute used to validate the current token or to renew it against Keystone when it is expired.
type Client struct {
	// Token is the authentication token for querying an OpenStack API.
	Token string

	// URL is the public URL for an OpenStack project.
	URL string

	// HTTPClient is the client used for launching HTTP requests.
	HTTPClient *http.Client

	// authMetadata contains the properties needed for retrieving an authentication token, renew it, and dynamically discover services public URLs from Keystone.
	authMetadata *KeystoneAuthRequest
}

// KeystoneAuthRequest contains all the necessary metadata for building an authentication request for Keystone, the official OpenStack Identity Provider.
type KeystoneAuthRequest struct {
	// AuthURL is the Keystone URL.
	AuthURL string `json:"-"`

	// HTTPClientTimeout is the HTTP client for querying the OpenStack service API.
	HTTPClientTimeout time.Duration `json:"-"`

	// Properties contains the authentication metadata to build the body of a token request.
	Properties *authProps `json:"auth"`
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

type keystoneCatalog struct {
	Catalog []service `json:"catalog"`
}

type service struct {
	Endpoints []endpoint `json:"endpoints"`
	Type      string     `json:"type"`
	ID        string     `json:"id"`
	Name      string     `json:"name"`
}

type endpoint struct {
	URL       string `json:"url"`
	Interface string `json:"interface"`
	Region    string `json:"region"`
	RegionID  string `json:"region_id"`
	ID        string `json:"id"`
}

// IsTokenValid checks if a authentication token is valid
func (client *Client) IsTokenValid(ctx context.Context) (bool, error) {
	var token = client.Token

	if token == "" {
		return false, fmt.Errorf("no authentication token provided")
	}

	tokenURL, err := url.Parse(client.authMetadata.AuthURL)

	if err != nil {
		return false, fmt.Errorf("the authURL is invalid: %w", err)
	}

	tokenURL.Path = path.Join(tokenURL.Path, tokensEndpoint)

	checkTokenRequest, err := http.NewRequestWithContext(ctx, "HEAD", tokenURL.String(), nil)
	if err != nil {
		return false, err
	}
	checkTokenRequest.Header.Set("X-Subject-Token", token)
	checkTokenRequest.Header.Set("X-Auth-Token", token)

	response, err := client.HTTPClient.Do(checkTokenRequest)

	if err != nil {
		return false, err
	}

	defer response.Body.Close()

	if response.StatusCode >= 400 {
		return false, nil
	}

	return true, nil
}

// RenewToken retrieves another token from Keystone
func (client *Client) RenewToken(ctx context.Context) error {
	token, err := client.authMetadata.getToken(ctx)

	if err != nil {
		return err
	}

	client.Token = token

	return nil
}

// NewPasswordAuth creates a struct containing metadata for authentication using the password method
func NewPasswordAuth(authURL string, userID string, userPassword string, projectID string, httpTimeout int) (*KeystoneAuthRequest, error) {
	passAuth := new(KeystoneAuthRequest)

	passAuth.Properties = new(authProps)

	passAuth.Properties.Scope = new(scopeProps)
	passAuth.Properties.Scope.Project = new(projectProps)

	passAuth.Properties.Identity = new(identityProps)
	passAuth.Properties.Identity.Password = new(passwordProps)
	passAuth.Properties.Identity.Password.User = new(userProps)

	url, err := url.Parse(authURL)

	if err != nil {
		return nil, fmt.Errorf("authURL is invalid: %w", err)
	}

	url.Path = path.Join(url.Path, "")

	passAuth.AuthURL = url.String()

	passAuth.HTTPClientTimeout = time.Duration(httpTimeout) * time.Second

	passAuth.Properties.Identity.Methods = []string{"password"}

	passAuth.Properties.Identity.Password.User.ID = userID
	passAuth.Properties.Identity.Password.User.Password = userPassword

	passAuth.Properties.Scope.Project.ID = projectID

	return passAuth, nil
}

// NewAppCredentialsAuth creates a struct containing metadata for authentication using the application credentials method
func NewAppCredentialsAuth(authURL string, id string, secret string, httpTimeout int) (*KeystoneAuthRequest, error) {
	appAuth := new(KeystoneAuthRequest)

	appAuth.Properties = new(authProps)

	appAuth.Properties.Identity = new(identityProps)

	url, err := url.Parse(authURL)

	if err != nil {
		return nil, fmt.Errorf("authURL is invalid: %w", err)
	}

	url.Path = path.Join(url.Path, "")

	appAuth.AuthURL = url.String()

	appAuth.Properties.Identity.AppCredential = new(appCredentialProps)
	appAuth.Properties.Identity.Methods = []string{"application_credential"}
	appAuth.Properties.Identity.AppCredential.ID = id
	appAuth.Properties.Identity.AppCredential.Secret = secret

	appAuth.HTTPClientTimeout = time.Duration(httpTimeout) * time.Second

	return appAuth, nil
}

// RequestClient returns a Client containing an HTTP client and a token.
// If an OpenStack project name is provided as first parameter, it will try to retrieve its API URL using the current credentials.
// If an OpenStack region or availability zone is provided as second parameter, it will retrieve the service API URL for that region.
// Otherwise, if the service API URL was found, it retrieves the first public URL for that service.
func (keystone *KeystoneAuthRequest) RequestClient(ctx context.Context, projectProps ...string) (Client, error) {
	var client = Client{
		HTTPClient:   kedautil.CreateHTTPClient(keystone.HTTPClientTimeout, false),
		authMetadata: keystone,
	}

	token, err := keystone.getToken(ctx)

	if err != nil {
		return client, err
	}

	client.Token = token

	var serviceURL string

	switch len(projectProps) {
	case 2:
		serviceURL, err = keystone.getServiceURL(ctx, token, projectProps[0], projectProps[1])
	case 1:
		serviceURL, err = keystone.getServiceURL(ctx, token, projectProps[0], "")
	default:
		serviceURL = ""
	}

	if err != nil {
		return client, fmt.Errorf("scaler could not find the service URL dynamically. Either provide it in the scaler parameters or check your OpenStack configuration: %w", err)
	}

	client.URL = serviceURL

	return client, nil
}

func (keystone *KeystoneAuthRequest) getToken(ctx context.Context) (string, error) {
	var httpClient = kedautil.CreateHTTPClient(keystone.HTTPClientTimeout, false)

	jsonBody, err := json.Marshal(keystone)

	if err != nil {
		return "", err
	}

	jsonBodyReader := bytes.NewReader(jsonBody)

	tokenURL, err := url.Parse(keystone.AuthURL)

	if err != nil {
		return "", fmt.Errorf("the authURL is invalid: %w", err)
	}

	tokenURL.Path = path.Join(tokenURL.Path, tokensEndpoint)

	tokenRequest, err := http.NewRequestWithContext(ctx, "POST", tokenURL.String(), jsonBodyReader)

	if err != nil {
		return "", err
	}

	resp, err := httpClient.Do(tokenRequest)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return resp.Header["X-Subject-Token"][0], nil
	}

	errBody, err := io.ReadAll(resp.Body)

	if err != nil {
		return "", err
	}

	return "", fmt.Errorf("%s", string(errBody))
}

// getCatalog retrieves the OpenStack catalog according to the current authorization
func (keystone *KeystoneAuthRequest) getCatalog(ctx context.Context, token string) ([]service, error) {
	var httpClient = kedautil.CreateHTTPClient(keystone.HTTPClientTimeout, false)

	catalogURL, err := url.Parse(keystone.AuthURL)

	if err != nil {
		return nil, fmt.Errorf("the authURL is invalid: %w", err)
	}

	catalogURL.Path = path.Join(catalogURL.Path, catalogEndpoint)

	getCatalog, err := http.NewRequestWithContext(ctx, "GET", catalogURL.String(), nil)

	if err != nil {
		return nil, err
	}

	getCatalog.Header.Set("X-Auth-Token", token)

	resp, err := httpClient.Do(getCatalog)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var keystoneCatalog = keystoneCatalog{}

		err := json.NewDecoder(resp.Body).Decode(&keystoneCatalog)

		if err != nil {
			return nil, fmt.Errorf("error parsing the catalog request response body: %w", err)
		}

		return keystoneCatalog.Catalog, nil
	}

	errBody, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	return nil, fmt.Errorf("%s", string(errBody))
}

// getServiceURL retrieves a public URL for an OpenStack project from the OpenStack catalog
func (keystone *KeystoneAuthRequest) getServiceURL(ctx context.Context, token string, projectName string, region string) (string, error) {
	serviceTypes, err := openstackutil.GetServiceTypes(ctx, projectName)

	if err != nil {
		return "", err
	}

	serviceCatalog, err := keystone.getCatalog(ctx, token)

	if err != nil {
		return "", err
	}

	if len(serviceCatalog) == 0 {
		return "", fmt.Errorf("no catalog provided based upon the current authorization. Service URL cannot be dynamically retrieved")
	}

	for _, serviceType := range serviceTypes {
		for _, service := range serviceCatalog {
			if serviceType == service.Type {
				for _, endpoint := range service.Endpoints {
					if endpoint.Interface == "public" {
						if region != "" {
							if endpoint.Region == region {
								return endpoint.URL, nil
							}
							continue
						}
						return endpoint.URL, nil
					}
				}
				return "", fmt.Errorf("service '%s' does not have a public URL or the public URL for a specific region is not registered in the catalog", projectName)
			}
		}
	}

	// If getServiceTypes() timed-out or failed or if serviceType is not in the catalog, try to find by project name
	for _, service := range serviceCatalog {
		if projectName == service.Name {
			for _, endpoint := range service.Endpoints {
				if endpoint.Interface == "public" {
					if region != "" {
						if endpoint.Region == region {
							return endpoint.URL, nil
						}
						continue
					}
					return endpoint.URL, nil
				}
			}
			return "", fmt.Errorf("service '%s' does not have a public URL or the public URL for a specific region is not registered in the catalog", projectName)
		}
	}

	return "", fmt.Errorf("service '%s' not found in catalog. Please, provide different credentials or reach to your OpenStack cluster manager", projectName)
}
