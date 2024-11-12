//nolint:revive
package ai

import (
	"encoding/json"
	"fmt"
)

// AiWorkflowsNRQLConfiguration - Enrichment configuration for NRQL type
type AiWorkflowsNRQLConfiguration struct {
	// The NRQL query
	Query string `json:"query"`
}

func (x *AiWorkflowsNRQLConfiguration) ImplementsAiWorkflowsConfiguration() {}

// UnmarshalAiWorkflowsConfigurationInterface unmarshals the interface into the correct type
// based on __typename provided by GraphQL
func UnmarshalAiWorkflowsConfigurationInterface(b []byte) (*AiWorkflowsConfigurationInterface, error) {
	var err error

	var rawMessageAiWorkflowsConfiguration map[string]*json.RawMessage
	err = json.Unmarshal(b, &rawMessageAiWorkflowsConfiguration)
	if err != nil {
		return nil, err
	}

	// Nothing to unmarshal
	if len(rawMessageAiWorkflowsConfiguration) < 1 {
		return nil, nil
	}

	var typeName string

	if rawTypeName, ok := rawMessageAiWorkflowsConfiguration["__typename"]; ok {
		err = json.Unmarshal(*rawTypeName, &typeName)
		if err != nil {
			return nil, err
		}

		switch typeName {
		case "AiWorkflowsNRQLConfiguration":
			var interfaceType AiWorkflowsNRQLConfiguration
			err = json.Unmarshal(b, &interfaceType)
			if err != nil {
				return nil, err
			}

			var xxx AiWorkflowsConfigurationInterface = &interfaceType

			return &xxx, nil
		}
	} else {
		keys := []string{}
		for k := range rawMessageAiWorkflowsConfiguration {
			keys = append(keys, k)
		}
		return nil, fmt.Errorf("interface AiWorkflowsConfiguration did not include a __typename field for inspection: %s", keys)
	}

	return nil, fmt.Errorf("interface AiWorkflowsConfiguration was not matched against all PossibleTypes: %s", typeName)
}

// AiNotificationsSuggestionError - Object for suggestion errors
type AiNotificationsSuggestionError struct {
	// SuggestionError description
	Description string `json:"description"`
	// SuggestionError details
	Details string `json:"details"`
	// SuggestionError type
	Type AiNotificationsErrorType `json:"type"`
}

// AiNotificationsErrorType - Error types
type AiNotificationsErrorType string

func (x *AiNotificationsSuggestionError) ImplementsAiNotificationsError() {}

// AiNotificationsDataValidationError - Object for validation errors
type AiNotificationsDataValidationError struct {
	// Top level error details
	Details string `json:"details"`
	// List of invalid fields
	Fields []AiNotificationsFieldError `json:"fields"`
}

// AiNotificationsFieldError - Invalid field object
type AiNotificationsFieldError struct {
	// Field name
	Field string `json:"field"`
	// Validation error
	Message string `json:"message"`
}

func (x *AiNotificationsDataValidationError) ImplementsAiNotificationsError() {}

// AiNotificationsConstraintError - Missing constraint error. Constraints can be retrieved using suggestion api
type AiNotificationsConstraintError struct {
	// Names of other constraints this constraint is dependent on
	Dependencies []string `json:"dependencies"`
	// Name of the missing constraint
	Name string `json:"name"`
}

func (x *AiNotificationsConstraintError) ImplementsAiNotificationsError() {}

// AiNotificationsResponseError - Response error object
type AiNotificationsResponseError struct {
	// Error description
	Description string `json:"description"`
	// Error details
	Details string `json:"details"`
	// Error type
	Type AiNotificationsErrorType `json:"type"`
}

func (x *AiNotificationsResponseError) ImplementsAiNotificationsError() {}

// UnmarshalAiNotificationsErrorInterface unmarshals the interface into the correct type
// based on __typename provided by GraphQL
func UnmarshalAiNotificationsErrorInterface(b []byte) (*AiNotificationsErrorInterface, error) {
	var err error

	var rawMessageAiNotificationsError map[string]*json.RawMessage
	err = json.Unmarshal(b, &rawMessageAiNotificationsError)
	if err != nil {
		return nil, err
	}

	// Nothing to unmarshal
	if len(rawMessageAiNotificationsError) < 1 {
		return nil, nil
	}

	var typeName string

	if rawTypeName, ok := rawMessageAiNotificationsError["__typename"]; ok {
		err = json.Unmarshal(*rawTypeName, &typeName)
		if err != nil {
			return nil, err
		}

		switch typeName {
		case "AiNotificationsSuggestionError":
			var interfaceType AiNotificationsSuggestionError
			err = json.Unmarshal(b, &interfaceType)
			if err != nil {
				return nil, err
			}

			var xxx AiNotificationsErrorInterface = &interfaceType

			return &xxx, nil
		case "AiNotificationsDataValidationError":
			var interfaceType AiNotificationsDataValidationError
			err = json.Unmarshal(b, &interfaceType)
			if err != nil {
				return nil, err
			}

			var xxx AiNotificationsErrorInterface = &interfaceType

			return &xxx, nil
		case "AiNotificationsConstraintError":
			var interfaceType AiNotificationsConstraintError
			err = json.Unmarshal(b, &interfaceType)
			if err != nil {
				return nil, err
			}

			var xxx AiNotificationsErrorInterface = &interfaceType

			return &xxx, nil
		case "AiNotificationsResponseError":
			var interfaceType AiNotificationsResponseError
			err = json.Unmarshal(b, &interfaceType)
			if err != nil {
				return nil, err
			}

			var xxx AiNotificationsErrorInterface = &interfaceType

			return &xxx, nil
		}
	} else {
		keys := []string{}
		for k := range rawMessageAiNotificationsError {
			keys = append(keys, k)
		}
		return nil, fmt.Errorf("interface AiNotificationsError did not include a __typename field for inspection: %s", keys)
	}

	return nil, fmt.Errorf("interface AiNotificationsError was not matched against all PossibleTypes: %s", typeName)
}

// AiNotificationsDestinationFilter - Filter destination object
type AiNotificationsDestinationFilter struct {
	// id
	ID string `json:"id,omitempty"`
	// Name
	Name string `json:"name,omitempty"`
}

// SecureValue - The `SecureValue` scalar represents a secure value, ie a password, an API key, etc.
type SecureValue string

// AiNotificationsAuthType - Authentication types
type AiNotificationsAuthType string

// AiNotificationsTokenAuth - Token based authentication
type AiNotificationsTokenAuth struct {
	// Authentication Type - Token or Oauth2
	AuthType AiNotificationsAuthType `json:"authType"`
	// Token Prefix
	Prefix string `json:"prefix"`
}

func (x AiNotificationsTokenAuth) GetCustomHeaders() []AiNotificationsCustomHeaders {
	return nil
}

func (x AiNotificationsTokenAuth) GetAccessTokenURL() string {
	return ""
}

// GetAuthType returns a pointer to the value of AuthType from AiNotificationsTokenAuth
func (x AiNotificationsTokenAuth) GetAuthType() AiNotificationsAuthType {
	return x.AuthType
}

// GetPrefix returns a pointer to the value of Prefix from AiNotificationsTokenAuth
func (x AiNotificationsTokenAuth) GetPrefix() string {
	return x.Prefix
}

func (x AiNotificationsTokenAuth) GetUser() string {
	return ""
}

// GetAuthorizationURL returns a pointer to the value of AuthorizationURL from AiNotificationsOAuth2Auth
func (x AiNotificationsTokenAuth) GetAuthorizationURL() string {
	return ""
}

// GetClientId returns a pointer to the value of ClientId from AiNotificationsOAuth2Auth
func (x AiNotificationsTokenAuth) GetClientId() string {
	return ""
}

// GetRefreshInterval returns a pointer to the value of RefreshInterval from AiNotificationsOAuth2Auth
func (x AiNotificationsTokenAuth) GetRefreshInterval() int {
	return 0
}

// GetRefreshable returns a pointer to the value of Refreshable from AiNotificationsOAuth2Auth
func (x AiNotificationsTokenAuth) GetRefreshable() bool {
	return false
}

// GetScope returns a pointer to the value of Scope from AiNotificationsOAuth2Auth
func (x AiNotificationsTokenAuth) GetScope() string {
	return ""
}

func (x AiNotificationsTokenAuth) ImplementsAiNotificationsAuth() {}

// AiNotificationsBasicAuth - Basic user and password authentication
type AiNotificationsBasicAuth struct {
	// Authentication Type - Basic
	AuthType AiNotificationsAuthType `json:"authType"`
	// Username
	User string `json:"user"`
}

func (x AiNotificationsBasicAuth) GetCustomHeaders() []AiNotificationsCustomHeaders {
	return nil
}

// GetAuthType returns a pointer to the value of AuthType from AiNotificationsBasicAuth
func (x AiNotificationsBasicAuth) GetAuthType() AiNotificationsAuthType {
	return x.AuthType
}

// GetUser returns a pointer to the value of User from AiNotificationsBasicAuth
func (x AiNotificationsBasicAuth) GetUser() string {
	return x.User
}

func (x AiNotificationsBasicAuth) GetPrefix() string {
	return ""
}

func (x AiNotificationsBasicAuth) GetAccessTokenURL() string {
	return ""
}

func (x AiNotificationsBasicAuth) GetAuthorizationURL() string {
	return ""
}

func (x AiNotificationsBasicAuth) GetClientId() string {
	return ""
}

func (x AiNotificationsBasicAuth) GetRefreshInterval() int {
	return 0
}

func (x AiNotificationsBasicAuth) GetRefreshable() bool {
	return false
}

func (x AiNotificationsBasicAuth) GetScope() string {
	return ""
}

func (x AiNotificationsBasicAuth) ImplementsAiNotificationsAuth() {}

// AiNotificationsOAuth2Auth - OAuth2 based authentication
type AiNotificationsOAuth2Auth struct {
	// OAuth2 access token url
	AccessTokenURL string `json:"accessTokenUrl"`
	// Authentication Type - Token or Oauth2
	AuthType AiNotificationsAuthType `json:"authType"`
	// OAuth2 authorization url
	AuthorizationURL string `json:"authorizationUrl"`
	// OAuth2 clientId
	ClientId string `json:"clientId"`
	// Token prefix
	Prefix string `json:"prefix"`
	// Interval of how often should the access token be refreshed
	RefreshInterval int `json:"refreshInterval,omitempty"`
	// Is the OAuth2 access token refreshable
	Refreshable bool `json:"refreshable"`
	// OAuth2 token's scope
	Scope string `json:"scope,omitempty"`
}

// GetAccessTokenURL returns a pointer to the value of AccessTokenURL from AiNotificationsOAuth2Auth
func (x AiNotificationsOAuth2Auth) GetAccessTokenURL() string {
	return x.AccessTokenURL
}

// GetAuthType returns a pointer to the value of AuthType from AiNotificationsOAuth2Auth
func (x AiNotificationsOAuth2Auth) GetAuthType() AiNotificationsAuthType {
	return x.AuthType
}

// GetAuthorizationURL returns a pointer to the value of AuthorizationURL from AiNotificationsOAuth2Auth
func (x AiNotificationsOAuth2Auth) GetAuthorizationURL() string {
	return x.AuthorizationURL
}

// GetClientId returns a pointer to the value of ClientId from AiNotificationsOAuth2Auth
func (x AiNotificationsOAuth2Auth) GetClientId() string {
	return x.ClientId
}

// GetPrefix returns a pointer to the value of Prefix from AiNotificationsOAuth2Auth
func (x AiNotificationsOAuth2Auth) GetPrefix() string {
	return x.Prefix
}

// GetRefreshInterval returns a pointer to the value of RefreshInterval from AiNotificationsOAuth2Auth
func (x AiNotificationsOAuth2Auth) GetRefreshInterval() int {
	return x.RefreshInterval
}

// GetRefreshable returns a pointer to the value of Refreshable from AiNotificationsOAuth2Auth
func (x AiNotificationsOAuth2Auth) GetRefreshable() bool {
	return x.Refreshable
}

// GetScope returns a pointer to the value of Scope from AiNotificationsOAuth2Auth
func (x AiNotificationsOAuth2Auth) GetScope() string {
	return x.Scope
}

// AiNotificationsAuth - Authentication interface
type AiNotificationsAuth struct {
	// OAuth2 access token url
	AccessTokenURL string `json:"accessTokenUrl,omitempty"`
	// Authentication Type - Token, Oauth2, or Basic
	AuthType AiNotificationsAuthType `json:"authType,omitempty"`
	// OAuth2 authorization url
	AuthorizationURL string `json:"authorizationUrl,omitempty"`
	// OAuth2 clientId
	ClientId string `json:"clientId,omitempty"`
	// Token prefix
	Prefix string `json:"prefix,omitempty"`
	// Interval of how often should the OAuth2 access token be refreshed
	RefreshInterval int `json:"refreshInterval,omitempty"`
	// Is the OAuth2 access token refreshable
	Refreshable bool `json:"refreshable,omitempty"`
	// OAuth2 token's scope
	Scope string `json:"scope,omitempty"`
	// Basic auth password
	Password SecureValue `json:"password,omitempty"`
	// Basic auth user
	User string `json:"user,omitempty"`
	// Custom headers
	CustomHeaders []AiNotificationsCustomHeaders `json:"customHeaders,omitempty"`
}

func (x *AiNotificationsAuth) ImplementsAiNotificationsAuth() {}

// AiNotificationsAuth - Authentication interface
type AiNotificationsAuthInterface interface {
	ImplementsAiNotificationsAuth()
	GetAccessTokenURL() string
	GetAuthType() AiNotificationsAuthType
	GetAuthorizationURL() string
	GetClientId() string
	GetPrefix() string
	GetRefreshInterval() int
	GetRefreshable() bool
	GetScope() string
	GetCustomHeaders() []AiNotificationsCustomHeaders
}

// AiNotificationsCustomHeaders - Custom headers
type AiNotificationsCustomHeaders struct {
	Key string `json:"key"`
}

// AiNotificationsCustomHeadersAuth - Custom headers based authentication
type AiNotificationsCustomHeadersAuth struct {
	// Authentication Type - CustomHeaders
	AuthType AiNotificationsAuthType `json:"authType"`
	// Custom headers
	CustomHeaders []AiNotificationsCustomHeaders `json:"customHeaders"`
}

func (x AiNotificationsCustomHeadersAuth) GetUser() string {
	return ""
}

func (x AiNotificationsCustomHeadersAuth) GetPrefix() string {
	return ""
}

func (x AiNotificationsCustomHeadersAuth) GetAccessTokenURL() string {
	return ""
}

func (x AiNotificationsCustomHeadersAuth) GetAuthorizationURL() string {
	return ""
}

func (x AiNotificationsCustomHeadersAuth) GetClientId() string {
	return ""
}

func (x AiNotificationsCustomHeadersAuth) GetRefreshInterval() int {
	return 0
}

func (x AiNotificationsCustomHeadersAuth) GetRefreshable() bool {
	return false
}

func (x AiNotificationsCustomHeadersAuth) GetScope() string {
	return ""
}

func (x AiNotificationsCustomHeadersAuth) GetAuthType() AiNotificationsAuthType {
	return x.AuthType
}

func (x AiNotificationsCustomHeadersAuth) GetCustomHeaders() []AiNotificationsCustomHeaders {
	return x.CustomHeaders
}

func (x AiNotificationsCustomHeadersAuth) ImplementsAiNotificationsAuth() {}

// UnmarshalAiNotificationsAuthInterface unmarshals the interface into the correct type
// based on __typename provided by GraphQL
func UnmarshalAiNotificationsAuthInterface(b []byte) (*AiNotificationsAuthInterface, error) {
	var err error

	var rawMessageAiNotificationsAuth map[string]*json.RawMessage
	err = json.Unmarshal(b, &rawMessageAiNotificationsAuth)
	if err != nil {
		return nil, err
	}

	// Nothing to unmarshal
	if len(rawMessageAiNotificationsAuth) < 1 {
		return nil, nil
	}

	var typeName string

	if rawTypeName, ok := rawMessageAiNotificationsAuth["__typename"]; ok {
		err = json.Unmarshal(*rawTypeName, &typeName)
		if err != nil {
			return nil, err
		}

		switch typeName {
		case "AiNotificationsBasicAuth":
			var interfaceType AiNotificationsBasicAuth
			err = json.Unmarshal(b, &interfaceType)
			if err != nil {
				return nil, err
			}

			var xxx AiNotificationsAuthInterface = &interfaceType

			return &xxx, nil
		case "AiNotificationsTokenAuth":
			var interfaceType AiNotificationsTokenAuth
			err = json.Unmarshal(b, &interfaceType)
			if err != nil {
				return nil, err
			}

			var xxx AiNotificationsAuthInterface = &interfaceType

			return &xxx, nil
		case "AiNotificationsCustomHeadersAuth":
			var interfaceType AiNotificationsCustomHeadersAuth
			err = json.Unmarshal(b, &interfaceType)
			if err != nil {
				return nil, err
			}

			var xxx AiNotificationsAuthInterface = &interfaceType

			return &xxx, nil
		}
	} else {
		keys := []string{}
		for k := range rawMessageAiNotificationsAuth {
			keys = append(keys, k)
		}
		return nil, fmt.Errorf("interface AiNotificationsAuth did not include a __typename field for inspection: %s", keys)
	}

	return nil, fmt.Errorf("interface AiNotificationsAuth was not matched against all PossibleTypes: %s", typeName)
}

// AiNotificationsChannelFilter - Filter channel object
type AiNotificationsChannelFilter struct {
	// id
	ID string `json:"id,omitempty"`
}

// AiWorkflowsFilters - Filter workflows object
type AiWorkflowsFilters struct {
	// id
	ID string `json:"id,omitempty"`
}
