package notifications

import (
	"context"

	"github.com/newrelic/newrelic-client-go/v2/pkg/ai"
)

func (a *Notifications) AiNotificationsCreateDestinationWithScope(
	accountID int,
	destination AiNotificationsDestinationInput,
	scope *EntityScopeInput,
) (*AiNotificationsDestinationResponse, error) {
	return a.AiNotificationsCreateDestinationWithScopeWithContext(context.Background(),
		accountID,
		destination,
		scope,
	)
}

func (a *Notifications) AiNotificationsCreateDestinationWithScopeWithContext(
	ctx context.Context,
	accountID int,
	destination AiNotificationsDestinationInput,
	scope *EntityScopeInput,
) (*AiNotificationsDestinationResponse, error) {

	resp := AiNotificationsCreateDestinationWithScopeQueryResponse{}
	vars := map[string]interface{}{
		"accountId":   accountID,
		"destination": destination,
	}
	var mutation string
	if scope != nil {
		vars["scopeId"] = scope.ID
		if scope.Type == EntityScopeTypeInputTypes.ORGANIZATION {
			mutation = aiNotificationsCreateDestinationWithOrgScopeMutation
		} else {
			mutation = aiNotificationsCreateDestinationWithAccountScopeMutation
		}
	} else {
		mutation = aiNotificationsCreateDestinationNoScopeMutation
	}

	if err := a.client.NerdGraphQueryWithContext(ctx, mutation, vars, &resp); err != nil {
		return nil, err
	}

	return &resp.AiNotificationsDestinationResponse, nil
}

type AiNotificationsCreateDestinationWithScopeQueryResponse struct {
	AiNotificationsDestinationResponse AiNotificationsDestinationResponse `json:"AiNotificationsCreateDestination"`
}

const aiNotificationsCreateDestinationNoScopeMutation = `mutation(
	$accountId: Int,
	$destination: AiNotificationsDestinationInput!,
) { aiNotificationsCreateDestination(
	accountId: $accountId,
	destination: $destination,
) {
	destination {
		accountId
		active
		auth {
			... on AiNotificationsBasicAuth {
			  authType
			  user
			}
			... on AiNotificationsOAuth2Auth {
			  accessTokenUrl
			  scope
			  refreshable
			  refreshInterval
			  prefix
			  clientId
			  authorizationUrl
			  authType
			}
			... on AiNotificationsTokenAuth {
			  authType
			  prefix
			}
			... on AiNotificationsCustomHeadersAuth {
			  authType
        	  customHeaders {
          	    key
			  }
			}
		}
		createdAt
		id
		guid
		isUserAuthenticated
		lastSent
		name
		properties {
			displayValue
			key
			label
			value
		}
		secureUrl {
			prefix
		}
		status
		type
		updatedAt
		updatedBy
	}
	errors {
      ... on AiNotificationsConstraintsError {
        constraints {
          dependencies
          name
        }
      }
      ... on AiNotificationsDataValidationError {
        details
        fields {
          field
          message
        }
      }
      ... on AiNotificationsResponseError {
        description
        details
        type
      }
      ... on AiNotificationsSuggestionError {
        description
        type
        details
      }
    }
    error {
      ... on AiNotificationsSuggestionError {
        description
        type
        details
      }
      ... on AiNotificationsResponseError {
        description
        type
        details
      }
      ... on AiNotificationsDataValidationError {
        details
        fields {
          message
          field
        }
      }
      ... on AiNotificationsConstraintsError {
        constraints {
          name
          dependencies
        }
      }
    }
} }`

const aiNotificationsCreateDestinationWithOrgScopeMutation = `mutation(
	$accountId: Int,
	$destination: AiNotificationsDestinationInput!,
	$scopeId: String!,
) { aiNotificationsCreateDestination(
	accountId: $accountId,
	destination: $destination,
	scope: {type: ORGANIZATION, id: $scopeId},
) {
	destination {
		accountId
		active
		auth {
			... on AiNotificationsBasicAuth {
			  authType
			  user
			}
			... on AiNotificationsOAuth2Auth {
			  accessTokenUrl
			  scope
			  refreshable
			  refreshInterval
			  prefix
			  clientId
			  authorizationUrl
			  authType
			}
			... on AiNotificationsTokenAuth {
			  authType
			  prefix
			}
			... on AiNotificationsCustomHeadersAuth {
			  authType
        	  customHeaders {
          	    key
			  }
			}
		}
		createdAt
		id
		guid
		isUserAuthenticated
		lastSent
		name
		properties {
			displayValue
			key
			label
			value
		}
		secureUrl {
			prefix
		}
		status
		type
		updatedAt
		updatedBy
	}
	errors {
      ... on AiNotificationsConstraintsError {
        constraints {
          dependencies
          name
        }
      }
      ... on AiNotificationsDataValidationError {
        details
        fields {
          field
          message
        }
      }
      ... on AiNotificationsResponseError {
        description
        details
        type
      }
      ... on AiNotificationsSuggestionError {
        description
        type
        details
      }
    }
    error {
      ... on AiNotificationsSuggestionError {
        description
        type
        details
      }
      ... on AiNotificationsResponseError {
        description
        type
        details
      }
      ... on AiNotificationsDataValidationError {
        details
        fields {
          message
          field
        }
      }
      ... on AiNotificationsConstraintsError {
        constraints {
          name
          dependencies
        }
      }
    }
} }`

const aiNotificationsCreateDestinationWithAccountScopeMutation = `mutation(
	$accountId: Int,
	$destination: AiNotificationsDestinationInput!,
	$scopeId: String!,
) { aiNotificationsCreateDestination(
	accountId: $accountId,
	destination: $destination,
	scope: {type: ACCOUNT, id: $scopeId},
) {
	destination {
		accountId
		active
		auth {
			... on AiNotificationsBasicAuth {
			  authType
			  user
			}
			... on AiNotificationsOAuth2Auth {
			  accessTokenUrl
			  scope
			  refreshable
			  refreshInterval
			  prefix
			  clientId
			  authorizationUrl
			  authType
			}
			... on AiNotificationsTokenAuth {
			  authType
			  prefix
			}
			... on AiNotificationsCustomHeadersAuth {
			  authType
        	  customHeaders {
          	    key
			  }
			}
		}
		createdAt
		id
		guid
		isUserAuthenticated
		lastSent
		name
		properties {
			displayValue
			key
			label
			value
		}
		secureUrl {
			prefix
		}
		status
		type
		updatedAt
		updatedBy
	}
	errors {
      ... on AiNotificationsConstraintsError {
        constraints {
          dependencies
          name
        }
      }
      ... on AiNotificationsDataValidationError {
        details
        fields {
          field
          message
        }
      }
      ... on AiNotificationsResponseError {
        description
        details
        type
      }
      ... on AiNotificationsSuggestionError {
        description
        type
        details
      }
    }
    error {
      ... on AiNotificationsSuggestionError {
        description
        type
        details
      }
      ... on AiNotificationsResponseError {
        description
        type
        details
      }
      ... on AiNotificationsDataValidationError {
        details
        fields {
          message
          field
        }
      }
      ... on AiNotificationsConstraintsError {
        constraints {
          name
          dependencies
        }
      }
    }
} }`

func (a *Notifications) GetDestinationsWithScope(
	accountID int,
	cursor string,
	filters ai.AiNotificationsDestinationFilter,
	sorter AiNotificationsDestinationSorter,
) (*AiNotificationsDestinationsWithScopeResponse, error) {
	return a.GetDestinationsWithScopeWithContext(context.Background(),
		accountID,
		cursor,
		filters,
		sorter,
	)
}

func (a *Notifications) GetDestinationsWithScopeWithContext(
	ctx context.Context,
	accountID int,
	cursor string,
	filters ai.AiNotificationsDestinationFilter,
	sorter AiNotificationsDestinationSorter,
) (*AiNotificationsDestinationsWithScopeResponse, error) {

	resp := destinationsWithScopeResponse{}

	filtersWithScope := map[string]interface{}{
		"scopeTypes": []string{"ACCOUNT", "ORGANIZATION"},
	}
	if filters.ID != "" {
		filtersWithScope["id"] = filters.ID
	}
	if filters.Name != "" {
		filtersWithScope["name"] = filters.Name
	}
	if filters.ExactName != "" {
		filtersWithScope["exactName"] = filters.ExactName
	}

	vars := map[string]interface{}{
		"accountId": accountID,
		"filters":   filtersWithScope,
	}

	if cursor != "" {
		vars["cursor"] = cursor
	}

	if sorter.Field != "" && sorter.Direction != "" {
		vars["sorter"] = sorter
	}

	if err := a.client.NerdGraphQueryWithContext(ctx, getDestinationsWithScopeQuery, vars, &resp); err != nil {
		return nil, err
	}

	return &resp.Actor.Account.AiNotifications.Destinations, nil
}

type AiNotificationsDestinationWithScope struct {
	AiNotificationsDestination
	Scope *EntityScope `json:"scope,omitempty"`
}

type AiNotificationsDestinationsWithScopeResponse struct {
	Entities   []AiNotificationsDestinationWithScope `json:"entities"`
	Error      AiNotificationsResponseError          `json:"error,omitempty"`
	Errors     []AiNotificationsResponseError        `json:"errors"`
	NextCursor string                                `json:"nextCursor,omitempty"`
	TotalCount int                                   `json:"totalCount"`
}

type destinationsWithScopeResponse struct {
	Actor struct {
		Account struct {
			AiNotifications struct {
				Destinations AiNotificationsDestinationsWithScopeResponse `json:"destinations,omitempty"`
			} `json:"aiNotifications,omitempty"`
		} `json:"account,omitempty"`
	} `json:"actor,omitempty"`
}

const getDestinationsWithScopeQuery = `query($accountId: Int!, $filters: AiNotificationsDestinationFilter, $sorter: AiNotificationsDestinationSorter, $cursor: String) {
	actor {
		account(id: $accountId) {
			aiNotifications {
				destinations(filters: $filters, sorter: $sorter, cursor: $cursor) {
					error {
						description
						type
						details
					}
					totalCount
					entities {
						accountId
						active
						createdAt
						id
						guid
						lastSent
						name
						properties {
							value
							key
						}
						type
						updatedAt
						updatedBy
						auth {
							... on AiNotificationsBasicAuth {
								authType
								user
							}
							... on AiNotificationsOAuth2Auth {
								accessTokenUrl
								authType
								authorizationUrl
								clientId
								prefix
								refreshable
								refreshInterval
								scope
							}
							... on AiNotificationsTokenAuth {
								authType
								prefix
							}
							... on AiNotificationsCustomHeadersAuth {
								authType
								customHeaders {
									key
								}
							}
						}
						secureUrl {
							prefix
						}
						status
						scope {
							id
							type
						}
						isUserAuthenticated
					}
					nextCursor
				}
			}
		}
	}
}`

// AiNotificationsUpdateDestinationWithScope - Update a Destination with optional scope
func (a *Notifications) AiNotificationsUpdateDestinationWithScope(
	accountID int,
	destination AiNotificationsDestinationUpdate,
	destinationId string,
	scope *EntityScopeInput,
) (*AiNotificationsDestinationResponse, error) {
	return a.AiNotificationsUpdateDestinationWithScopeWithContext(context.Background(),
		accountID,
		destination,
		destinationId,
		scope,
	)
}

// AiNotificationsUpdateDestinationWithScopeWithContext - Update a Destination with optional scope and context
func (a *Notifications) AiNotificationsUpdateDestinationWithScopeWithContext(
	ctx context.Context,
	accountID int,
	destination AiNotificationsDestinationUpdate,
	destinationId string,
	scope *EntityScopeInput,
) (*AiNotificationsDestinationResponse, error) {

	resp := AiNotificationsUpdateDestinationWithScopeQueryResponse{}
	vars := map[string]interface{}{
		"accountId":     accountID,
		"destination":   destination,
		"destinationId": destinationId,
	}

	// Choose mutation based on whether scope is provided
	var mutation string
	if scope != nil {
		vars["scopeId"] = scope.ID
		if scope.Type == EntityScopeTypeInputTypes.ORGANIZATION {
			mutation = aiNotificationsUpdateDestinationWithOrgScopeMutation
		} else {
			mutation = aiNotificationsUpdateDestinationWithAccountScopeMutation
		}
	} else {
		mutation = aiNotificationsUpdateDestinationNoScopeMutation
	}

	if err := a.client.NerdGraphQueryWithContext(ctx, mutation, vars, &resp); err != nil {
		return nil, err
	}

	return &resp.AiNotificationsDestinationResponse, nil
}

type AiNotificationsUpdateDestinationWithScopeQueryResponse struct {
	AiNotificationsDestinationResponse AiNotificationsDestinationResponse `json:"AiNotificationsUpdateDestination"`
}

const aiNotificationsUpdateDestinationNoScopeMutation = `mutation(
	$accountId: Int!,
	$destination: AiNotificationsDestinationUpdate!,
	$destinationId: ID!,
) { aiNotificationsUpdateDestination(
	accountId: $accountId,
	destination: $destination,
	destinationId: $destinationId,
) {
	destination {
		accountId
		active
		auth {
			... on AiNotificationsBasicAuth {
			  authType
			  user
			}
			... on AiNotificationsOAuth2Auth {
			  accessTokenUrl
			  scope
			  refreshable
			  refreshInterval
			  prefix
			  clientId
			  authorizationUrl
			  authType
			}
			... on AiNotificationsTokenAuth {
			  authType
			  prefix
			}
			... on AiNotificationsCustomHeadersAuth {
			  authType
        	  customHeaders {
          	    key
			  }
			}
		}
		createdAt
		id
		guid
		isUserAuthenticated
		lastSent
		name
		properties {
			displayValue
			key
			label
			value
		}
		secureUrl {
			prefix
		}
		status
		type
		updatedAt
		updatedBy
	}
	errors {
      ... on AiNotificationsConstraintsError {
        constraints {
          dependencies
          name
        }
      }
      ... on AiNotificationsDataValidationError {
        details
        fields {
          field
          message
        }
      }
      ... on AiNotificationsResponseError {
        description
        details
        type
      }
      ... on AiNotificationsSuggestionError {
        description
        type
        details
      }
    }
} }`

const aiNotificationsUpdateDestinationWithOrgScopeMutation = `mutation(
	$accountId: Int!,
	$destination: AiNotificationsDestinationUpdate!,
	$destinationId: ID!,
	$scopeId: String!,
) { aiNotificationsUpdateDestination(
	accountId: $accountId,
	destination: $destination,
	destinationId: $destinationId,
	scope: {type: ORGANIZATION, id: $scopeId},
) {
	destination {
		accountId
		active
		auth {
			... on AiNotificationsBasicAuth {
			  authType
			  user
			}
			... on AiNotificationsOAuth2Auth {
			  accessTokenUrl
			  scope
			  refreshable
			  refreshInterval
			  prefix
			  clientId
			  authorizationUrl
			  authType
			}
			... on AiNotificationsTokenAuth {
			  authType
			  prefix
			}
			... on AiNotificationsCustomHeadersAuth {
			  authType
        	  customHeaders {
          	    key
			  }
			}
		}
		createdAt
		id
		guid
		isUserAuthenticated
		lastSent
		name
		properties {
			displayValue
			key
			label
			value
		}
		secureUrl {
			prefix
		}
		status
		type
		updatedAt
		updatedBy
	}
	errors {
      ... on AiNotificationsConstraintsError {
        constraints {
          dependencies
          name
        }
      }
      ... on AiNotificationsDataValidationError {
        details
        fields {
          field
          message
        }
      }
      ... on AiNotificationsResponseError {
        description
        details
        type
      }
      ... on AiNotificationsSuggestionError {
        description
        type
        details
      }
    }
} }`

const aiNotificationsUpdateDestinationWithAccountScopeMutation = `mutation(
	$accountId: Int!,
	$destination: AiNotificationsDestinationUpdate!,
	$destinationId: ID!,
	$scopeId: String!,
) { aiNotificationsUpdateDestination(
	accountId: $accountId,
	destination: $destination,
	destinationId: $destinationId,
	scope: {type: ACCOUNT, id: $scopeId},
) {
	destination {
		accountId
		active
		auth {
			... on AiNotificationsBasicAuth {
			  authType
			  user
			}
			... on AiNotificationsOAuth2Auth {
			  accessTokenUrl
			  scope
			  refreshable
			  refreshInterval
			  prefix
			  clientId
			  authorizationUrl
			  authType
			}
			... on AiNotificationsTokenAuth {
			  authType
			  prefix
			}
			... on AiNotificationsCustomHeadersAuth {
			  authType
        	  customHeaders {
          	    key
			  }
			}
		}
		createdAt
		id
		guid
		isUserAuthenticated
		lastSent
		name
		properties {
			displayValue
			key
			label
			value
		}
		secureUrl {
			prefix
		}
		status
		type
		updatedAt
		updatedBy
	}
	errors {
      ... on AiNotificationsConstraintsError {
        constraints {
          dependencies
          name
        }
      }
      ... on AiNotificationsDataValidationError {
        details
        fields {
          field
          message
        }
      }
      ... on AiNotificationsResponseError {
        description
        details
        type
      }
      ... on AiNotificationsSuggestionError {
        description
        type
        details
      }
    }
} }`

// AiNotificationsDeleteDestinationWithScope - Delete a Destination with optional scope
func (a *Notifications) AiNotificationsDeleteDestinationWithScope(
	accountID int,
	destinationId string,
	scope *EntityScopeInput,
) (*AiNotificationsDeleteResponse, error) {
	return a.AiNotificationsDeleteDestinationWithScopeWithContext(context.Background(),
		accountID,
		destinationId,
		scope,
	)
}

// AiNotificationsDeleteDestinationWithScopeWithContext - Delete a Destination with optional scope and context
func (a *Notifications) AiNotificationsDeleteDestinationWithScopeWithContext(
	ctx context.Context,
	accountID int,
	destinationId string,
	scope *EntityScopeInput,
) (*AiNotificationsDeleteResponse, error) {

	resp := AiNotificationsDeleteDestinationWithScopeQueryResponse{}
	vars := map[string]interface{}{
		"accountId":     accountID,
		"destinationId": destinationId,
	}

	// Choose mutation based on whether scope is provided
	var mutation string
	if scope != nil {
		vars["scopeId"] = scope.ID
		if scope.Type == EntityScopeTypeInputTypes.ORGANIZATION {
			mutation = aiNotificationsDeleteDestinationWithOrgScopeMutation
		} else {
			mutation = aiNotificationsDeleteDestinationWithAccountScopeMutation
		}
	} else {
		mutation = aiNotificationsDeleteDestinationNoScopeMutation
	}

	if err := a.client.NerdGraphQueryWithContext(ctx, mutation, vars, &resp); err != nil {
		return nil, err
	}

	return &resp.AiNotificationsDeleteResponse, nil
}

type AiNotificationsDeleteDestinationWithScopeQueryResponse struct {
	AiNotificationsDeleteResponse AiNotificationsDeleteResponse `json:"AiNotificationsDeleteDestination"`
}

const aiNotificationsDeleteDestinationNoScopeMutation = `mutation(
	$accountId: Int!,
	$destinationId: ID!,
) { aiNotificationsDeleteDestination(
	accountId: $accountId,
	destinationId: $destinationId,
) {
	error {
		description
		details
		type
	}
	errors {
		description
		details
		type
	}
	ids
} }`

const aiNotificationsDeleteDestinationWithOrgScopeMutation = `mutation(
	$accountId: Int!,
	$destinationId: ID!,
	$scopeId: String!,
) { aiNotificationsDeleteDestination(
	accountId: $accountId,
	destinationId: $destinationId,
	scope: {type: ORGANIZATION, id: $scopeId},
) {
	error {
		description
		details
		type
	}
	errors {
		description
		details
		type
	}
	ids
} }`

const aiNotificationsDeleteDestinationWithAccountScopeMutation = `mutation(
	$accountId: Int!,
	$destinationId: ID!,
	$scopeId: String!,
) { aiNotificationsDeleteDestination(
	accountId: $accountId,
	destinationId: $destinationId,
	scope: {type: ACCOUNT, id: $scopeId},
) {
	error {
		description
		details
		type
	}
	errors {
		description
		details
		type
	}
	ids
} }`
