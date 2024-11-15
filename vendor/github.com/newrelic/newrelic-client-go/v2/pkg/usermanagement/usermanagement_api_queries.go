package usermanagement

import (
	"context"
)

// GET User(s)

func (a *Usermanagement) UserManagementGetUsers(
	authenticationDomainIDs []string,
	userIDs []string,
	name string,
	emailID string,
) (*UserManagementAuthenticationDomains, error) {
	return a.UserManagementGetUsersWithContext(context.Background(),
		authenticationDomainIDs,
		userIDs,
		name,
		emailID,
	)
}

func (a *Usermanagement) UserManagementGetUsersWithContext(
	ctx context.Context,
	authenticationDomainIDs []string,
	userIDs []string,
	name string,
	emailID string,
) (*UserManagementAuthenticationDomains, error) {

	resp := authenticationDomainsResponse{}
	vars := map[string]interface{}{
		"authenticationDomainIDs": authenticationDomainIDs,
		"userIDs":                 userIDs,
		"name":                    name,
		"emailID":                 emailID,
	}

	if len(authenticationDomainIDs) == 0 {
		delete(vars, "authenticationDomainIDs")
	}

	if len(userIDs) == 0 {
		delete(vars, "userIDs")
	}

	if name == "" {
		delete(vars, "name")
	}

	if emailID == "" {
		delete(vars, "emailID")
	}

	if err := a.client.NerdGraphQueryWithContext(ctx, getUsersQuery, vars, &resp); err != nil {
		return nil, err
	}

	return &resp.Actor.Organization.UserManagement.AuthenticationDomains, nil
}

const getUsersQuery = `query(
  $authenticationDomainIDs: [ID!],
  $userIDs: [ID!],
  $name: String,
  $emailID: String
)
{
  actor {
    organization {
      userManagement {
        authenticationDomains(id: $authenticationDomainIDs) {
          authenticationDomains {
            users(
              filter: {
                email: { eq: $emailID }
                name: { eq: $name }
                id: { in: $userIDs }
              }
            ) {
              users {
                id
                emailVerificationState
                email
                lastActive
                name
                timeZone
                pendingUpgradeRequest {
                  id
                  message
                  requestedUserType {
                    displayName
                    id
                  }
                }
                type {
                  displayName
                  id
                }
                groups {
                  groups {
                    id
                    displayName
                  }
                }
              }
            }
			id
			name
          }
		nextCursor
		totalCount
        }
      }
    }
  }
}`

// GET Group(s) With User(s)

func (a *Usermanagement) UserManagementGetGroupsWithUsers(
	authenticationDomainIDs []string,
	groupIDs []string,
	name string,
) (*UserManagementAuthenticationDomains, error) {
	return a.UserManagementGetGroupsWithUsersWithContext(context.Background(),
		authenticationDomainIDs,
		groupIDs,
		name,
	)
}

func (a *Usermanagement) UserManagementGetGroupsWithUsersWithContext(
	ctx context.Context,
	authenticationDomainIDs []string,
	groupIDs []string,
	name string,
) (*UserManagementAuthenticationDomains, error) {

	resp := authenticationDomainsResponse{}
	vars := map[string]interface{}{
		"authenticationDomainIDs": authenticationDomainIDs,
		"groupIDs":                groupIDs,
		"name":                    name,
	}

	if len(authenticationDomainIDs) == 0 {
		delete(vars, "authenticationDomainIDs")
	}

	if len(groupIDs) == 0 {
		delete(vars, "groupIDs")
	}

	if name == "" {
		delete(vars, "name")
	}

	if err := a.client.NerdGraphQueryWithContext(ctx, getGroupsWithUsersQuery, vars, &resp); err != nil {
		return nil, err
	}

	return &resp.Actor.Organization.UserManagement.AuthenticationDomains, nil
}

const getGroupsWithUsersQuery = `query (
  $authenticationDomainIDs: [ID!]
  $groupIDs: [ID!]
  $name: String
) {
  actor {
    organization {
      userManagement {
        authenticationDomains(id: $authenticationDomainIDs) {
          authenticationDomains {
            groups(
              filter: {
                displayName: { eq: $name }
                id: { in: $groupIDs }
              }
            ) {
              groups {
                displayName
                id
                users {
                  users {
                    email
                    id
                    name
                    timeZone
                  }
                }
              }
            }
            id
            name
          }
          nextCursor
          totalCount
        }
      }
    }
  }
}
`
