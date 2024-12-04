package apiaccess

import (
	"context"
	"errors"
	"fmt"

	"github.com/newrelic/newrelic-client-go/v2/internal/http"
)

type APIKey struct {
	APIAccessKey

	AccountID  *int                   `json:"accountId,omitempty"`
	IngestType APIAccessIngestKeyType `json:"ingestType,omitempty"`
	UserID     *int                   `json:"userId,omitempty"`
}

// Additional Interface methods
func (x *APIAccessIngestKeyError) GetError() error {
	return errors.New(x.Message)
}

func (x *APIAccessUserKeyError) GetError() error {
	return errors.New(x.Message)
}

// CreateAPIAccessKeys create keys. You can create keys for multiple accounts at once.
func (a *APIAccess) CreateAPIAccessKeys(keys APIAccessCreateInput) ([]APIKey, error) {
	return a.CreateAPIAccessKeysWithContext(context.Background(), keys)
}

// CreateAPIAccessKeysWithContext create keys. You can create keys for multiple accounts at once.
func (a *APIAccess) CreateAPIAccessKeysWithContext(ctx context.Context, keys APIAccessCreateInput) ([]APIKey, error) {
	vars := map[string]interface{}{
		"keys": keys,
	}

	resp := apiAccessKeyCreateResponse{}

	if err := a.client.NerdGraphQueryWithContext(ctx, apiAccessKeyCreateKeys, vars, &resp); err != nil {
		return nil, err
	}

	if len(resp.APIAccessCreateKeys.Errors) > 0 {
		return nil, errors.New(formatAPIAccessKeyErrors(resp.APIAccessCreateKeys.Errors))
	}

	return resp.APIAccessCreateKeys.CreatedKeys, nil
}

// GetAPIAccessKey returns a single API access key.
func (a *APIAccess) GetAPIAccessKey(keyID string, keyType APIAccessKeyType) (*APIKey, error) {
	return a.GetAPIAccessKeyWithContext(context.Background(), keyID, keyType)
}

// GetAPIAccessKeyWithContext returns a single API access key.
func (a *APIAccess) GetAPIAccessKeyWithContext(ctx context.Context, keyID string, keyType APIAccessKeyType) (*APIKey, error) {
	vars := map[string]interface{}{
		"id":      keyID,
		"keyType": keyType,
	}

	resp := apiAccessKeyGetResponse{}

	if err := a.client.NerdGraphQueryWithContext(ctx, apiAccessKeyGetKey, vars, &resp); err != nil {
		return nil, err
	}

	if resp.Errors != nil {
		return nil, errors.New(resp.Error())
	}

	return &resp.Actor.APIAccess.Key, nil
}

// SearchAPIAccessKeys returns the relevant keys based on search criteria. Returns keys are scoped to the current user.
func (a *APIAccess) SearchAPIAccessKeys(params APIAccessKeySearchQuery) ([]APIKey, error) {
	return a.SearchAPIAccessKeysWithContext(context.Background(), params)
}

// SearchAPIAccessKeysWithContext returns the relevant keys based on search criteria. Returns keys are scoped to the current user.
func (a *APIAccess) SearchAPIAccessKeysWithContext(ctx context.Context, params APIAccessKeySearchQuery) ([]APIKey, error) {
	vars := map[string]interface{}{
		"query": params,
	}

	resp := apiAccessKeySearchResponse{}

	if err := a.client.NerdGraphQueryWithContext(ctx, apiAccessKeySearch, vars, &resp); err != nil {
		return nil, err
	}

	if resp.Errors != nil {
		return nil, errors.New(resp.Error())
	}

	return resp.Actor.APIAccess.KeySearch.Keys, nil
}

// UpdateAPIAccessKeys updates keys. You can update keys for multiple accounts at once.
func (a *APIAccess) UpdateAPIAccessKeys(keys APIAccessUpdateInput) ([]APIKey, error) {
	return a.UpdateAPIAccessKeysWithContext(context.Background(), keys)
}

// UpdateAPIAccessKeysWithContext updates keys. You can update keys for multiple accounts at once.
func (a *APIAccess) UpdateAPIAccessKeysWithContext(ctx context.Context, keys APIAccessUpdateInput) ([]APIKey, error) {
	vars := map[string]interface{}{
		"keys": keys,
	}

	resp := apiAccessKeyUpdateResponse{}

	if err := a.client.NerdGraphQueryWithContext(ctx, apiAccessKeyUpdateKeys, vars, &resp); err != nil {
		return nil, err
	}

	if len(resp.APIAccessUpdateKeys.Errors) > 0 {
		return nil, errors.New(formatAPIAccessKeyErrors(resp.APIAccessUpdateKeys.Errors))
	}

	return resp.APIAccessUpdateKeys.UpdatedKeys, nil
}

// DeleteAPIAccessKey deletes one or more keys.
func (a *APIAccess) DeleteAPIAccessKey(keys APIAccessDeleteInput) ([]APIAccessDeletedKey, error) {
	return a.DeleteAPIAccessKeyWithContext(context.Background(), keys)
}

// DeleteAPIAccessKeyWithContext deletes one or more keys.
func (a *APIAccess) DeleteAPIAccessKeyWithContext(ctx context.Context, keys APIAccessDeleteInput) ([]APIAccessDeletedKey, error) {
	vars := map[string]interface{}{
		"keys": keys,
	}

	resp := apiAccessKeyDeleteResponse{}

	if err := a.client.NerdGraphQueryWithContext(ctx, apiAccessKeyDeleteKeys, vars, &resp); err != nil {
		return nil, err
	}

	if len(resp.APIAccessDeleteKeys.Errors) > 0 {
		return nil, errors.New(formatAPIAccessKeyErrors(resp.APIAccessDeleteKeys.Errors))
	}

	return resp.APIAccessDeleteKeys.DeletedKeys, nil
}

var AccessKeyErrorPrefix = "The following errors have been thrown.\n"

func formatAPIAccessKeyErrors(errs []APIAccessKeyErrorResponse) string {
	errorString := AccessKeyErrorPrefix
	for _, e := range errs {
		IDAsString := ""
		if len(e.ID) != 0 {
			// Id is returned in the 'error' block only in the case of update and delete but not with create.
			// So; in the case of create, it is made an empty string to generalize the usage of IDAsString.
			IDAsString = fmt.Sprintf("%s: ", e.ID)
		}
		if e.Type == "USER" {
			errorString += fmt.Sprintf("%s: %s%s\n", e.UserKeyErrorType, IDAsString, e.Message)
		} else if e.Type == "INGEST" {
			errorString += fmt.Sprintf("%s: %s%s\n", e.IngestKeyErrorType, IDAsString, e.Message)
		} else if len(e.Type) == 0 {
			// When Ingest Keys are deleted, the "type" attribute is currently null in the response sent,
			// in the "errors" attribute - hence, this condition. However, this is not the case with User Keys.
			errorString += fmt.Sprintf("%s: %s%s\n", e.IngestKeyErrorType, IDAsString, e.Message)
		} else {
			errorString += e.Message
		}
	}
	return errorString
}

// apiAccessKeyCreateResponse represents the JSON response returned from creating key(s).
type apiAccessKeyCreateResponse struct {
	APIAccessCreateKeys struct {
		CreatedKeys []APIKey                    `json:"createdKeys"`
		Errors      []APIAccessKeyErrorResponse `json:"errors,omitempty"`
	} `json:"apiAccessCreateKeys"`
}

// apiAccessKeyUpdateResponse represents the JSON response returned from updating key(s).
type apiAccessKeyUpdateResponse struct {
	APIAccessUpdateKeys struct {
		UpdatedKeys []APIKey                    `json:"updatedKeys"`
		Errors      []APIAccessKeyErrorResponse `json:"errors,omitempty"`
	} `json:"apiAccessUpdateKeys"`
}

// apiAccessKeyGetResponse represents the JSON response returned from getting an access key.
type apiAccessKeyGetResponse struct {
	Actor struct {
		APIAccess struct {
			Key APIKey `json:"key,omitempty"`
		} `json:"apiAccess"`
	} `json:"actor"`
	http.GraphQLErrorResponse
}

type apiAccessKeySearchResponse struct {
	Actor struct {
		APIAccess struct {
			KeySearch struct {
				Keys []APIKey `json:"keys"`
			} `json:"keySearch,omitempty"`
		} `json:"apiAccess"`
	} `json:"actor"`
	http.GraphQLErrorResponse
}

type apiAccessKeyDeleteResponse struct {
	APIAccessDeleteKeys struct {
		DeletedKeys []APIAccessDeletedKey       `json:"deletedKeys,omitempty"`
		Errors      []APIAccessKeyErrorResponse `json:"errors,omitempty"`
	} `json:"apiAccessDeleteKeys"`
}

const (
	graphqlAPIAccessKeyBaseFields = `
		id
		key
		name
		createdAt
		notes
		type
		... on ApiAccessIngestKey {
			id
			name
			accountId
			ingestType
			key
			notes
			type
		}
		... on ApiAccessUserKey {
			id
			name
			accountId
			key
			notes
			type
			userId
		}
		... on ApiAccessKey {
			id
			name
			key
			notes
			type
		}`

	graphqlAPIAccessCreateKeyFields = `createdKeys {` + graphqlAPIAccessKeyBaseFields + `}`

	graphqlAPIAccessUpdatedKeyFields = `updatedKeys {` + graphqlAPIAccessKeyBaseFields + `}`

	graphqlAPIAccessKeyErrorFields = `errors {
		  message
		  type
		... on ApiAccessIngestKeyError {
			id
			ingestErrorType: errorType
			accountId
			ingestType
			message
			type
		  }
		... on ApiAccessKeyError {
			message
			type
		  }
		... on ApiAccessUserKeyError {
			id
			accountId
			userErrorType: errorType
			message
			type
			userId
		  }
		}
	`

	apiAccessKeyCreateKeys = `mutation($keys: ApiAccessCreateInput!) {
			apiAccessCreateKeys(keys: $keys) {` + graphqlAPIAccessCreateKeyFields + graphqlAPIAccessKeyErrorFields + `
		}}`

	apiAccessKeyGetKey = `query($id: ID!, $keyType: ApiAccessKeyType!) {
		actor {
			apiAccess {
				key(id: $id, keyType: $keyType) {` + graphqlAPIAccessKeyBaseFields + `}}}}`

	apiAccessKeySearch = `query($query: ApiAccessKeySearchQuery!) {
		actor {
			apiAccess {
				keySearch(query: $query) {
					keys {` + graphqlAPIAccessKeyBaseFields + `}
				}}}}`

	apiAccessKeyUpdateKeys = `mutation($keys: ApiAccessUpdateInput!) {
			apiAccessUpdateKeys(keys: $keys) {` + graphqlAPIAccessUpdatedKeyFields + graphqlAPIAccessKeyErrorFields + `
		}}`

	apiAccessKeyDeleteKeys = `mutation($keys: ApiAccessDeleteInput!) {
			apiAccessDeleteKeys(keys: $keys) {
				deletedKeys {
					id
				}` + graphqlAPIAccessKeyErrorFields + `}}`
)
