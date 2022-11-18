package nerdstorage

import (
	"context"
	"fmt"
	"strconv"

	"github.com/imdario/mergo"

	"github.com/newrelic/newrelic-client-go/internal/http"
	"github.com/newrelic/newrelic-client-go/pkg/config"
	"github.com/newrelic/newrelic-client-go/pkg/logging"
)

const (
	accountScope = "ACCOUNT"
	actorScope   = "ACTOR"
	entityScope  = "ENTITY"
)

// GetDocumentInput represents the input data required for retrieving a NerdStorage document.
type GetDocumentInput struct {
	Collection string
	DocumentID string
	PackageID  string
}

// GetDocumentWithAccountScope retrieves a NerdStorage document with account scope.
func (e *NerdStorage) GetDocumentWithAccountScope(accountID int, input GetDocumentInput) (interface{}, error) {
	return e.GetDocumentWithAccountScopeWithContext(context.Background(), accountID, input)
}

// GetDocumentWithAccountScopeWithContext retrieves a NerdStorage document with account scope.
func (e *NerdStorage) GetDocumentWithAccountScopeWithContext(ctx context.Context, accountID int, input GetDocumentInput) (interface{}, error) {
	if accountID == 0 {
		return nil, fmt.Errorf("account ID is required when using account scope")
	}

	scopeID := strconv.Itoa(accountID)
	vars := map[string]interface{}{"accountId": accountID}

	resp, err := e.getDocumentWithScope(ctx, accountScope, scopeID, getDocumentWithAccountScopeQuery, vars, input)
	if err != nil {
		return nil, err
	}

	return resp.Actor.Account.NerdStorage.Document, nil
}

// GetDocumentWithUserScope retrieves a NerdStorage document with user scope.
func (e *NerdStorage) GetDocumentWithUserScope(input GetDocumentInput) (interface{}, error) {
	return e.GetDocumentWithUserScopeWithContext(context.Background(), input)
}

// GetDocumentWithUserScopeWithContext retrieves a NerdStorage document with user scope.
func (e *NerdStorage) GetDocumentWithUserScopeWithContext(ctx context.Context, input GetDocumentInput) (interface{}, error) {
	userID, err := e.getUserID(ctx)
	if err != nil {
		return nil, err
	}

	scopeID := strconv.Itoa(userID)

	resp, err := e.getDocumentWithScope(ctx, actorScope, scopeID, getDocumentWithUserScopeQuery, nil, input)
	if err != nil {
		return nil, err
	}

	return resp.Actor.NerdStorage.Document, nil
}

// GetDocumentWithEntityScope retrieves a NerdStorage document with entity scope.
func (e *NerdStorage) GetDocumentWithEntityScope(entityGUID string, input GetDocumentInput) (interface{}, error) {
	return e.GetDocumentWithEntityScopeWithContext(context.Background(), entityGUID, input)
}

// GetDocumentWithEntityScopeWithContext retrieves a NerdStorage document with entity scope.
func (e *NerdStorage) GetDocumentWithEntityScopeWithContext(ctx context.Context, entityGUID string, input GetDocumentInput) (interface{}, error) {
	if entityGUID == "" {
		return nil, fmt.Errorf("entity GUID is required when using entity scope")
	}

	vars := map[string]interface{}{"entityGuid": entityGUID}
	resp, err := e.getDocumentWithScope(ctx, entityScope, entityGUID, getDocumentWithEntityScopeQuery, vars, input)
	if err != nil {
		return nil, err
	}

	return resp.Actor.Entity.NerdStorage.Document, nil
}

// GetCollectionInput represents the input data required for retrieving a NerdStorage collection.
type GetCollectionInput struct {
	Collection string
	PackageID  string
}

// GetCollectionWithAccountScope retrieves a NerdStorage collection with account scope.
func (e *NerdStorage) GetCollectionWithAccountScope(accountID int, input GetCollectionInput) ([]interface{}, error) {
	return e.GetCollectionWithAccountScopeWithContext(context.Background(), accountID, input)
}

// GetCollectionWithAccountScopeWithContext retrieves a NerdStorage collection with account scope.
func (e *NerdStorage) GetCollectionWithAccountScopeWithContext(ctx context.Context, accountID int, input GetCollectionInput) ([]interface{}, error) {
	if accountID == 0 {
		return nil, fmt.Errorf("account ID is required when using account scope")
	}

	scopeID := strconv.Itoa(accountID)
	vars := map[string]interface{}{"accountId": accountID}

	resp, err := e.getCollectionWithScope(ctx, accountScope, scopeID, getCollectionWithAccountScopeQuery, vars, input)
	if err != nil {
		return nil, err
	}

	return resp.Actor.Account.NerdStorage.Collection, nil
}

// GetCollectionWithUserScope retrieves a NerdStorage collection with user scope.
func (e *NerdStorage) GetCollectionWithUserScope(input GetCollectionInput) ([]interface{}, error) {
	return e.GetCollectionWithUserScopeWithContext(context.Background(), input)
}

// GetCollectionWithUserScopeWithContext retrieves a NerdStorage collection with user scope.
func (e *NerdStorage) GetCollectionWithUserScopeWithContext(ctx context.Context, input GetCollectionInput) ([]interface{}, error) {
	userID, err := e.getUserID(ctx)
	if err != nil {
		return nil, err
	}

	scopeID := strconv.Itoa(userID)
	resp, err := e.getCollectionWithScope(ctx, actorScope, scopeID, getCollectionWithUserScopeQuery, nil, input)
	if err != nil {
		return nil, err
	}

	return resp.Actor.NerdStorage.Collection, nil
}

// GetCollectionWithEntityScope wretrieves a NerdStorage collection with entity scope.
func (e *NerdStorage) GetCollectionWithEntityScope(entityGUID string, input GetCollectionInput) ([]interface{}, error) {
	return e.GetCollectionWithEntityScopeWithContext(context.Background(), entityGUID, input)
}

// GetCollectionWithEntityScopeWithContext wretrieves a NerdStorage collection with entity scope.
func (e *NerdStorage) GetCollectionWithEntityScopeWithContext(ctx context.Context, entityGUID string, input GetCollectionInput) ([]interface{}, error) {
	if entityGUID == "" {
		return nil, fmt.Errorf("entity GUID is required when using entity scope")
	}

	vars := map[string]interface{}{"entityGuid": entityGUID}
	resp, err := e.getCollectionWithScope(ctx, entityScope, entityGUID, getCollectionWithEntityScopeQuery, vars, input)
	if err != nil {
		return nil, err
	}

	return resp.Actor.Entity.NerdStorage.Collection, nil
}

// WriteDocumentInput represents the input data required for the WriteDocument mutation.
type WriteDocumentInput struct {
	Collection string
	Document   interface{}
	DocumentID string
	PackageID  string
}

// WriteDocumentWithAccountScope writes a NerdStorage document with account scope.
func (e *NerdStorage) WriteDocumentWithAccountScope(accountID int, input WriteDocumentInput) (interface{}, error) {
	return e.WriteDocumentWithAccountScopeWithContext(context.Background(), accountID, input)
}

// WriteDocumentWithAccountScopeWithContext writes a NerdStorage document with account scope.
func (e *NerdStorage) WriteDocumentWithAccountScopeWithContext(ctx context.Context, accountID int, input WriteDocumentInput) (interface{}, error) {
	if accountID == 0 {
		return nil, fmt.Errorf("account ID is required when using account scope")
	}

	scopeID := strconv.Itoa(accountID)

	return e.writeDocumentWithScope(ctx, accountScope, scopeID, input)
}

// WriteDocumentWithUserScope writes a NerdStorage document with user scope.
func (e *NerdStorage) WriteDocumentWithUserScope(input WriteDocumentInput) (interface{}, error) {
	return e.WriteDocumentWithUserScopeWithContext(context.Background(), input)
}

// WriteDocumentWithUserScopeWithContext writes a NerdStorage document with user scope.
func (e *NerdStorage) WriteDocumentWithUserScopeWithContext(ctx context.Context, input WriteDocumentInput) (interface{}, error) {
	userID, err := e.getUserID(ctx)
	if err != nil {
		return nil, err
	}

	scopeID := strconv.Itoa(userID)
	return e.writeDocumentWithScope(ctx, actorScope, scopeID, input)
}

// WriteDocumentWithEntityScope writes a NerdStorage document with entity scope.
func (e *NerdStorage) WriteDocumentWithEntityScope(entityGUID string, input WriteDocumentInput) (interface{}, error) {
	return e.WriteDocumentWithEntityScopeWithContext(context.Background(), entityGUID, input)
}

// WriteDocumentWithEntityScopeWithContext writes a NerdStorage document with entity scope.
func (e *NerdStorage) WriteDocumentWithEntityScopeWithContext(ctx context.Context, entityGUID string, input WriteDocumentInput) (interface{}, error) {
	if entityGUID == "" {
		return nil, fmt.Errorf("entity GUID is required when using entity scope")
	}

	return e.writeDocumentWithScope(ctx, entityScope, entityGUID, input)
}

// DeleteDocumentInput represents the input data required for the DeleteDocument mutation.
type DeleteDocumentInput struct {
	Collection string
	DocumentID string
	PackageID  string
}

// DeleteDocumentWithAccountScope deletes a NerdStorage document with account scope.
func (e *NerdStorage) DeleteDocumentWithAccountScope(accountID int, input DeleteDocumentInput) (bool, error) {
	return e.DeleteDocumentWithAccountScopeWithContext(context.Background(), accountID, input)
}

// DeleteDocumentWithAccountScopeWithContext deletes a NerdStorage document with account scope.
func (e *NerdStorage) DeleteDocumentWithAccountScopeWithContext(ctx context.Context, accountID int, input DeleteDocumentInput) (bool, error) {
	if accountID == 0 {
		return false, fmt.Errorf("account ID is required when using account scope")
	}

	scopeID := strconv.Itoa(accountID)

	return e.deleteDocumentWithScope(ctx, accountScope, scopeID, input)
}

// DeleteDocumentWithUserScope deletes a NerdStorage document with user scope.
func (e *NerdStorage) DeleteDocumentWithUserScope(input DeleteDocumentInput) (bool, error) {
	return e.DeleteDocumentWithUserScopeWithContext(context.Background(), input)
}

// DeleteDocumentWithUserScopeWithContext deletes a NerdStorage document with user scope.
func (e *NerdStorage) DeleteDocumentWithUserScopeWithContext(ctx context.Context, input DeleteDocumentInput) (bool, error) {
	userID, err := e.getUserID(ctx)
	if err != nil {
		return false, err
	}

	scopeID := strconv.Itoa(userID)
	return e.deleteDocumentWithScope(ctx, actorScope, scopeID, input)
}

// DeleteDocumentWithEntityScope deletes a NerdStorage document with entity scope.
func (e *NerdStorage) DeleteDocumentWithEntityScope(entityGUID string, input DeleteDocumentInput) (bool, error) {
	return e.DeleteDocumentWithEntityScopeWithContext(context.Background(), entityGUID, input)
}

// DeleteDocumentWithEntityScopeWithContext deletes a NerdStorage document with entity scope.
func (e *NerdStorage) DeleteDocumentWithEntityScopeWithContext(ctx context.Context, entityGUID string, input DeleteDocumentInput) (bool, error) {
	if entityGUID == "" {
		return false, fmt.Errorf("entity GUID is required when using entity scope")
	}

	return e.deleteDocumentWithScope(ctx, entityScope, entityGUID, input)
}

// DeleteCollectionInput represents the input data required for the DeleteCollection mutation.
type DeleteCollectionInput struct {
	Collection string
	PackageID  string
}

// DeleteCollectionWithAccountScope deletes a NerdStorage collection with account scope.
func (e *NerdStorage) DeleteCollectionWithAccountScope(accountID int, input DeleteCollectionInput) (bool, error) {
	return e.DeleteCollectionWithAccountScopeWithContext(context.Background(), accountID, input)
}

// DeleteCollectionWithAccountScopeWithContext deletes a NerdStorage collection with account scope.
func (e *NerdStorage) DeleteCollectionWithAccountScopeWithContext(ctx context.Context, accountID int, input DeleteCollectionInput) (bool, error) {
	if accountID == 0 {
		return false, fmt.Errorf("account ID is required when using account scope")
	}

	scopeID := strconv.Itoa(accountID)

	return e.deleteCollectionWithScope(ctx, accountScope, scopeID, input)
}

// DeleteCollectionWithUserScope deletes a NerdStorage collection with user scope.
func (e *NerdStorage) DeleteCollectionWithUserScope(input DeleteCollectionInput) (bool, error) {
	return e.DeleteCollectionWithUserScopeWithContext(context.Background(), input)
}

// DeleteCollectionWithUserScopeWithContext deletes a NerdStorage collection with user scope.
func (e *NerdStorage) DeleteCollectionWithUserScopeWithContext(ctx context.Context, input DeleteCollectionInput) (bool, error) {
	userID, err := e.getUserID(ctx)
	if err != nil {
		return false, err
	}

	scopeID := strconv.Itoa(userID)

	return e.deleteCollectionWithScope(ctx, actorScope, scopeID, input)
}

// DeleteCollectionWithEntityScope deletes a NerdStorage collection with entity scope.
func (e *NerdStorage) DeleteCollectionWithEntityScope(entityGUID string, input DeleteCollectionInput) (bool, error) {
	return e.DeleteCollectionWithEntityScopeWithContext(context.Background(), entityGUID, input)
}

// DeleteCollectionWithEntityScopeWithContext deletes a NerdStorage collection with entity scope.
func (e *NerdStorage) DeleteCollectionWithEntityScopeWithContext(ctx context.Context, entityGUID string, input DeleteCollectionInput) (bool, error) {
	if entityGUID == "" {
		return false, fmt.Errorf("entity GUID is required when using entity scope")
	}

	return e.deleteCollectionWithScope(ctx, entityScope, entityGUID, input)
}

func (e *NerdStorage) getDocumentWithScope(ctx context.Context, scope string, scopeID string, query string, vars map[string]interface{}, input GetDocumentInput) (*getResponse, error) {
	var resp getResponse

	v := map[string]interface{}{
		"collection": input.Collection,
		"documentId": input.DocumentID,
		"scope": scopeInput{
			Name: scope,
			ID:   scopeID,
		},
	}

	err := mergo.Merge(&v, vars)
	if err != nil {
		return nil, err
	}

	req, err := e.client.NewNerdGraphRequest(query, v, &resp)
	if err != nil {
		return nil, err
	}

	req.WithContext(ctx)
	req.SetHeader("NewRelic-Package-ID", input.PackageID)

	_, err = e.client.Do(req)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

func (e *NerdStorage) getCollectionWithScope(ctx context.Context, scope string, scopeID string, query string, vars map[string]interface{}, input GetCollectionInput) (*getResponse, error) {
	var resp getResponse

	v := map[string]interface{}{
		"collection": input.Collection,
		"scope": scopeInput{
			Name: scope,
			ID:   scopeID,
		},
	}

	err := mergo.Merge(&v, vars)
	if err != nil {
		return nil, err
	}

	req, err := e.client.NewNerdGraphRequest(query, v, &resp)
	if err != nil {
		return nil, err
	}

	req.WithContext(ctx)
	req.SetHeader("NewRelic-Package-ID", input.PackageID)

	_, err = e.client.Do(req)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

func (e *NerdStorage) writeDocumentWithScope(ctx context.Context, scope string, scopeID string, input WriteDocumentInput) (interface{}, error) {
	var resp writeDocumentResponse

	vars := map[string]interface{}{
		"collection": input.Collection,
		"document":   input.Document,
		"documentId": input.DocumentID,
		"scope": scopeInput{
			Name: scope,
			ID:   scopeID,
		},
	}

	req, err := e.client.NewNerdGraphRequest(writeDocumentMutation, vars, &resp)
	if err != nil {
		return "", err
	}

	req.WithContext(ctx)
	req.SetHeader("NewRelic-Package-ID", input.PackageID)

	_, err = e.client.Do(req)
	if err != nil {
		return "", err
	}

	return resp.NerdStorageWriteDocument, nil
}

func (e *NerdStorage) deleteDocumentWithScope(ctx context.Context, scope string, scopeID string, input DeleteDocumentInput) (bool, error) {
	var resp deleteDocumentResponse

	vars := map[string]interface{}{
		"collection": input.Collection,
		"documentId": input.DocumentID,
		"scope": scopeInput{
			Name: scope,
			ID:   scopeID,
		},
	}

	req, err := e.client.NewNerdGraphRequest(deleteDocumentMutation, vars, &resp)
	if err != nil {
		return false, err
	}

	req.WithContext(ctx)
	req.SetHeader("NewRelic-Package-ID", input.PackageID)

	_, err = e.client.Do(req)
	if err != nil {
		return false, err
	}

	return resp.NerdStorageDeleteDocument.Deleted != 0, nil
}

func (e *NerdStorage) deleteCollectionWithScope(ctx context.Context, scope string, scopeID string, input DeleteCollectionInput) (bool, error) {
	var resp deleteCollectionResponse

	vars := map[string]interface{}{
		"collection": input.Collection,
		"scope": scopeInput{
			Name: scope,
			ID:   scopeID,
		},
	}

	req, err := e.client.NewNerdGraphRequest(deleteCollectionMutation, vars, &resp)
	if err != nil {
		return false, err
	}

	req.WithContext(ctx)
	req.SetHeader("NewRelic-Package-ID", input.PackageID)

	_, err = e.client.Do(req)
	if err != nil {
		return false, err
	}

	return resp.NerdStorageDeleteCollection.Deleted != 0, nil
}

func (e *NerdStorage) getUserID(ctx context.Context) (int, error) {
	var resp userIDResponse

	err := e.client.NerdGraphQueryWithContext(ctx, getUserIDQuery, nil, &resp)
	if err != nil {
		return 0, err
	}

	return resp.Actor.User.ID, nil
}

const (
	getDocumentWithAccountScopeQuery = `
		query($accountId: Int!, $documentId: String!, $collection: String!) {
			actor {
				account(id: $accountId) {
					nerdStorage {
						document(collection: $collection, documentId: $documentId)
					}
				}
			}
		}`

	getDocumentWithEntityScopeQuery = `
		query($entityGuid: EntityGuid!, $documentId: String!, $collection: String!) {
			actor {
				entity(guid: $entityGuid) {
					nerdStorage {
						document(collection: $collection, documentId: $documentId)
					}
				}
			}
		}`

	getDocumentWithUserScopeQuery = `
		query($documentId: String!, $collection: String!) {
			actor {
				nerdStorage {
					document(collection: $collection, documentId: $documentId)
				}
			}
		}`

	getCollectionWithAccountScopeQuery = `
		query($accountId: Int!, $collection: String!) {
			actor {
				account(id: $accountId) {
					nerdStorage {
						collection(collection: $collection) {
							document
							id
						}
					}
				}
			}
		}`

	getCollectionWithEntityScopeQuery = `
		query($entityGuid: EntityGuid!, $collection: String!) {
			actor {
				entity(guid: $entityGuid) {
					nerdStorage {
						collection(collection: $collection) {
							document
							id
						}
					}
				}
			}
		}`

	getCollectionWithUserScopeQuery = `
		query($collection: String!) {
			actor {
				nerdStorage {
					collection(collection: $collection) {
							document
							id
						}
				}
			}
		}`

	getUserIDQuery = `
		query {
			actor {
				user {
					id
				}
			}
		}`

	writeDocumentMutation = `
		mutation($collection: String!, $document: NerdStorageDocument!, $documentId: String!, $scope: NerdStorageScopeInput!) {
			nerdStorageWriteDocument(collection: $collection, document: $document, documentId: $documentId, scope: $scope)
		}`

	deleteDocumentMutation = `
		mutation($collection: String!, $documentId: String!, $scope: NerdStorageScopeInput!) {
			nerdStorageDeleteDocument(collection: $collection, documentId: $documentId, scope: $scope) {
				deleted
			}
		}`

	deleteCollectionMutation = `
		mutation($collection: String!, $scope: NerdStorageScopeInput!) {
			nerdStorageDeleteCollection(collection: $collection, scope: $scope) {
				deleted
			}
		}`
)

type scopeInput struct {
	Name string
	ID   string
}

type getResponse struct {
	Actor struct {
		Account struct {
			NerdStorage struct {
				Document   interface{}
				Collection []interface{}
			}
		}
		Entity struct {
			NerdStorage struct {
				Document   interface{}
				Collection []interface{}
			}
		}
		NerdStorage struct {
			Document   interface{}
			Collection []interface{}
		}
	}
}

type userIDResponse struct {
	Actor struct {
		User struct {
			ID int
		}
	}
}

type writeDocumentResponse struct {
	NerdStorageWriteDocument interface{}
}

type deleteDocumentResponse struct {
	NerdStorageDeleteDocument struct {
		Deleted int
	}
}

type deleteCollectionResponse struct {
	NerdStorageDeleteCollection struct {
		Deleted int
	}
}

// NerdStorage is used to communicate with the New Relic Workloads product.
type NerdStorage struct {
	client http.Client
	logger logging.Logger
}

// New returns a new client for interacting with the New Relic One NerdStorage
// document store.
func New(config config.Config) NerdStorage {
	return NerdStorage{
		client: http.NewClient(config),
		logger: config.GetLogger(),
	}
}
