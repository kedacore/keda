// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"

	client "github.com/DataDog/datadog-api-client-go"
)

// contextKeys are used to identify the type of value in the context.
// Since these are string, it is possible to get a short description of the
// context key for logging and debugging using key.String().

type contextKey string

func (c contextKey) String() string {
	return "auth " + string(c)
}

var (
	// ContextOAuth2 takes an oauth2.TokenSource as authentication for the request.
	ContextOAuth2 = contextKey("token")

	// ContextBasicAuth takes BasicAuth as authentication for the request.
	ContextBasicAuth = contextKey("basic")

	// ContextAccessToken takes a string oauth2 access token as authentication for the request.
	ContextAccessToken = contextKey("accesstoken")

	// ContextAPIKeys takes a string apikey as authentication for the request
	ContextAPIKeys = contextKey("apiKeys")

	// ContextHttpSignatureAuth takes HttpSignatureAuth as authentication for the request.
	ContextHttpSignatureAuth = contextKey("httpsignature")

	// ContextServerIndex uses a server configuration from the index.
	ContextServerIndex = contextKey("serverIndex")

	// ContextOperationServerIndices uses a server configuration from the index mapping.
	ContextOperationServerIndices = contextKey("serverOperationIndices")

	// ContextServerVariables overrides a server configuration variables.
	ContextServerVariables = contextKey("serverVariables")

	// ContextOperationServerVariables overrides a server configuration variables using operation specific values.
	ContextOperationServerVariables = contextKey("serverOperationVariables")
)

// BasicAuth provides basic http authentication to a request passed via context using ContextBasicAuth.
type BasicAuth struct {
	UserName string `json:"userName,omitempty"`
	Password string `json:"password,omitempty"`
}

// APIKey provides API key based authentication to a request passed via context using ContextAPIKey.
type APIKey struct {
	Key    string
	Prefix string
}

// ServerVariable stores the information about a server variable.
type ServerVariable struct {
	Description  string
	DefaultValue string
	EnumValues   []string
}

// ServerConfiguration stores the information about a server.
type ServerConfiguration struct {
	URL         string
	Description string
	Variables   map[string]ServerVariable
}

// ServerConfigurations stores multiple ServerConfiguration items.
type ServerConfigurations []ServerConfiguration

// Configuration stores the configuration of the API client
type Configuration struct {
	Host               string            `json:"host,omitempty"`
	Scheme             string            `json:"scheme,omitempty"`
	DefaultHeader      map[string]string `json:"defaultHeader,omitempty"`
	UserAgent          string            `json:"userAgent,omitempty"`
	Debug              bool              `json:"debug,omitempty"`
	Compress           bool              `json:"compress,omitempty"`
	Servers            ServerConfigurations
	OperationServers   map[string]ServerConfigurations
	HTTPClient         *http.Client
	unstableOperations map[string]bool
}

// NewConfiguration returns a new Configuration object.
func NewConfiguration() *Configuration {
	cfg := &Configuration{
		DefaultHeader: make(map[string]string),
		UserAgent:     getUserAgent(),
		Debug:         false,
		Compress:      true,
		Servers: ServerConfigurations{
			{
				URL:         "https://{subdomain}.{site}",
				Description: "No description provided",
				Variables: map[string]ServerVariable{
					"site": {
						Description:  "The regional site for a Datadog customer.",
						DefaultValue: "datadoghq.com",
						EnumValues: []string{
							"datadoghq.com",
							"us3.datadoghq.com",
							"us5.datadoghq.com",
							"datadoghq.eu",
							"ddog-gov.com",
						},
					},
					"subdomain": {
						Description:  "The subdomain where the API is deployed.",
						DefaultValue: "api",
					},
				},
			},
			{
				URL:         "{protocol}://{name}",
				Description: "No description provided",
				Variables: map[string]ServerVariable{
					"name": {
						Description:  "Full site DNS name.",
						DefaultValue: "api.datadoghq.com",
					},
					"protocol": {
						Description:  "The protocol for accessing the API.",
						DefaultValue: "https",
					},
				},
			},
			{
				URL:         "https://{subdomain}.{site}",
				Description: "No description provided",
				Variables: map[string]ServerVariable{
					"site": {
						Description:  "Any Datadog deployment.",
						DefaultValue: "datadoghq.com",
					},
					"subdomain": {
						Description:  "The subdomain where the API is deployed.",
						DefaultValue: "api",
					},
				},
			},
		},
		OperationServers: map[string]ServerConfigurations{
			"IPRangesApiService.GetIPRanges": {
				{
					URL:         "https://{subdomain}.{site}",
					Description: "No description provided",
					Variables: map[string]ServerVariable{
						"site": {
							Description:  "The regional site for Datadog customers.",
							DefaultValue: "datadoghq.com",
							EnumValues: []string{
								"datadoghq.com",
								"us3.datadoghq.com",
								"us5.datadoghq.com",
								"datadoghq.eu",
								"ddog-gov.com",
							},
						},
						"subdomain": {
							Description:  "The subdomain where the API is deployed.",
							DefaultValue: "ip-ranges",
						},
					},
				},
				{
					URL:         "{protocol}://{name}",
					Description: "No description provided",
					Variables: map[string]ServerVariable{
						"name": {
							Description:  "Full site DNS name.",
							DefaultValue: "ip-ranges.datadoghq.com",
						},
						"protocol": {
							Description:  "The protocol for accessing the API.",
							DefaultValue: "https",
						},
					},
				},
				{
					URL:         "https://{subdomain}.datadoghq.com",
					Description: "No description provided",
					Variables: map[string]ServerVariable{
						"subdomain": {
							Description:  "The subdomain where the API is deployed.",
							DefaultValue: "ip-ranges",
						},
					},
				},
			},
			"ServiceLevelObjectivesApiService.SearchSLO": {
				{
					URL:         "https://{subdomain}.{site}",
					Description: "No description provided",
					Variables: map[string]ServerVariable{
						"site": {
							Description:  "The regional site for Datadog customers.",
							DefaultValue: "datadoghq.com",
							EnumValues: []string{
								"datadoghq.com",
								"us3.datadoghq.com",
								"us5.datadoghq.com",
								"ddog-gov.com",
							},
						},
						"subdomain": {
							Description:  "The subdomain where the API is deployed.",
							DefaultValue: "api",
						},
					},
				},
				{
					URL:         "{protocol}://{name}",
					Description: "No description provided",
					Variables: map[string]ServerVariable{
						"name": {
							Description:  "Full site DNS name.",
							DefaultValue: "api.datadoghq.com",
						},
						"protocol": {
							Description:  "The protocol for accessing the API.",
							DefaultValue: "https",
						},
					},
				},
				{
					URL:         "https://{subdomain}.{site}",
					Description: "No description provided",
					Variables: map[string]ServerVariable{
						"site": {
							Description:  "Any Datadog deployment.",
							DefaultValue: "datadoghq.com",
						},
						"subdomain": {
							Description:  "The subdomain where the API is deployed.",
							DefaultValue: "api",
						},
					},
				},
			},
			"LogsApiService.SubmitLog": {
				{
					URL:         "https://{subdomain}.{site}",
					Description: "No description provided",
					Variables: map[string]ServerVariable{
						"site": {
							Description:  "The regional site for Datadog customers.",
							DefaultValue: "datadoghq.com",
							EnumValues: []string{
								"datadoghq.com",
								"us3.datadoghq.com",
								"us5.datadoghq.com",
								"datadoghq.eu",
								"ddog-gov.com",
							},
						},
						"subdomain": {
							Description:  "The subdomain where the API is deployed.",
							DefaultValue: "http-intake.logs",
						},
					},
				},
				{
					URL:         "{protocol}://{name}",
					Description: "No description provided",
					Variables: map[string]ServerVariable{
						"name": {
							Description:  "Full site DNS name.",
							DefaultValue: "http-intake.logs.datadoghq.com",
						},
						"protocol": {
							Description:  "The protocol for accessing the API.",
							DefaultValue: "https",
						},
					},
				},
				{
					URL:         "https://{subdomain}.{site}",
					Description: "No description provided",
					Variables: map[string]ServerVariable{
						"site": {
							Description:  "Any Datadog deployment.",
							DefaultValue: "datadoghq.com",
						},
						"subdomain": {
							Description:  "The subdomain where the API is deployed.",
							DefaultValue: "http-intake.logs",
						},
					},
				},
			},
		},
		unstableOperations: map[string]bool{
			"GetDailyCustomReports":            false,
			"GetSpecifiedDailyCustomReports":   false,
			"GetMonthlyCustomReports":          false,
			"GetSpecifiedMonthlyCustomReports": false,
			"SearchSLO":                        false,
			"GetSLOHistory":                    false,
			"GetUsageAttribution":              false,
		},
	}
	return cfg
}

// AddDefaultHeader adds a new HTTP header to the default header in the request.
func (c *Configuration) AddDefaultHeader(key string, value string) {
	c.DefaultHeader[key] = value
}

// URL formats template on a index using given variables.
func (sc ServerConfigurations) URL(index int, variables map[string]string) (string, error) {
	if index < 0 || len(sc) <= index {
		return "", fmt.Errorf("Index %v out of range %v", index, len(sc)-1)
	}
	server := sc[index]
	url := server.URL

	// go through variables and replace placeholders
	for name, variable := range server.Variables {
		if value, ok := variables[name]; ok {
			found := bool(len(variable.EnumValues) == 0)
			for _, enumValue := range variable.EnumValues {
				if value == enumValue {
					found = true
				}
			}
			if !found {
				return "", fmt.Errorf("The variable %s in the server URL has invalid value %v. Must be %v", name, value, variable.EnumValues)
			}
			url = strings.Replace(url, "{"+name+"}", value, -1)
		} else {
			url = strings.Replace(url, "{"+name+"}", variable.DefaultValue, -1)
		}
	}
	return url, nil
}

// ServerURL returns URL based on server settings.
func (c *Configuration) ServerURL(index int, variables map[string]string) (string, error) {
	return c.Servers.URL(index, variables)
}

func getServerIndex(ctx context.Context) (int, error) {
	si := ctx.Value(ContextServerIndex)
	if si != nil {
		if index, ok := si.(int); ok {
			return index, nil
		}
		return 0, reportError("invalid type %T should be int", si)
	}
	return 0, nil
}

func getServerOperationIndex(ctx context.Context, endpoint string) (int, error) {
	osi := ctx.Value(ContextOperationServerIndices)
	if osi != nil {
		operationIndices, ok := osi.(map[string]int)
		if !ok {
			return 0, reportError("invalid type %T should be map[string]int", osi)
		}
		index, ok := operationIndices[endpoint]
		if ok {
			return index, nil
		}
	}
	return getServerIndex(ctx)
}

func getServerVariables(ctx context.Context) (map[string]string, error) {
	sv := ctx.Value(ContextServerVariables)
	if sv != nil {
		if variables, ok := sv.(map[string]string); ok {
			return variables, nil
		}
		return nil, reportError("ctx value of ContextServerVariables has invalid type %T should be map[string]string", sv)
	}
	return nil, nil
}

func getServerOperationVariables(ctx context.Context, endpoint string) (map[string]string, error) {
	osv := ctx.Value(ContextOperationServerVariables)
	if osv != nil {
		operationVariables, ok := osv.(map[string]map[string]string)
		if !ok {
			return nil, reportError("ctx value of ContextOperationServerVariables has invalid type %T should be map[string]map[string]string", osv)
		}
		variables, ok := operationVariables[endpoint]
		if ok {
			return variables, nil
		}
	}
	return getServerVariables(ctx)
}

// ServerURLWithContext returns a new server URL given an endpoint.
func (c *Configuration) ServerURLWithContext(ctx context.Context, endpoint string) (string, error) {
	sc, ok := c.OperationServers[endpoint]
	if !ok {
		sc = c.Servers
	}

	if ctx == nil {
		return sc.URL(0, nil)
	}

	index, err := getServerOperationIndex(ctx, endpoint)
	if err != nil {
		return "", err
	}

	variables, err := getServerOperationVariables(ctx, endpoint)
	if err != nil {
		return "", err
	}

	return sc.URL(index, variables)
}

// GetUnstableOperations returns a slice with all unstable operation Ids.
func (c *Configuration) GetUnstableOperations() []string {
	ids := make([]string, len(c.unstableOperations))
	for id := range c.unstableOperations {
		ids = append(ids, id)
	}
	return ids
}

// SetUnstableOperationEnabled sets an unstable operation as enabled (true) or disabled (false).
// This function accepts operation ID as an argument - this is the name of the method on the API class, e.g. "CreateFoo"
// Returns true if the operation is marked as unstable and thus was enabled/disabled, false otherwise.
func (c *Configuration) SetUnstableOperationEnabled(operation string, enabled bool) bool {
	if _, ok := c.unstableOperations[operation]; ok {
		c.unstableOperations[operation] = enabled
		return true
	}
	log.Printf("WARNING: '%s' is not an unstable operation, can't enable/disable", operation)
	return false
}

// IsUnstableOperation determines whether an operation is an unstable operation.
// This function accepts operation ID as an argument - this is the name of the method on the API class, e.g. "CreateFoo".
func (c *Configuration) IsUnstableOperation(operation string) bool {
	_, present := c.unstableOperations[operation]
	return present
}

// IsUnstableOperationEnabled determines whether an unstable operation is enabled.
// This function accepts operation ID as an argument - this is the name of the method on the API class, e.g. "CreateFoo"
// Returns true if the operation is unstable and it is enabled, false otherwise.
func (c *Configuration) IsUnstableOperationEnabled(operation string) bool {
	if enabled, present := c.unstableOperations[operation]; present {
		return enabled
	}
	log.Printf("WARNING: '%s' is not an unstable operation, is always enabled", operation)
	return false
}

func getUserAgent() string {
	return fmt.Sprintf(
		"datadog-api-client-go/%s (go %s; os %s; arch %s)",
		client.Version,
		runtime.Version(),
		runtime.GOOS,
		runtime.GOARCH,
	)
}

// NewDefaultContext returns a new context setup with environment variables.
func NewDefaultContext(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	if site, ok := os.LookupEnv("DD_SITE"); ok {
		ctx = context.WithValue(
			ctx,
			ContextServerVariables,
			map[string]string{"site": site},
		)
	}

	keys := make(map[string]APIKey)
	if apiKey, ok := os.LookupEnv("DD_API_KEY"); ok {
		keys["apiKeyAuth"] = APIKey{Key: apiKey}
	}
	if apiKey, ok := os.LookupEnv("DD_APP_KEY"); ok {
		keys["appKeyAuth"] = APIKey{Key: apiKey}
	}
	ctx = context.WithValue(
		ctx,
		ContextAPIKeys,
		keys,
	)

	return ctx
}
