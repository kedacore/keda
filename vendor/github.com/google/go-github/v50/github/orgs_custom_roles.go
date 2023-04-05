// Copyright 2022 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github

import (
	"context"
	"fmt"
)

// OrganizationCustomRepoRoles represents custom repository roles available in specified organization.
type OrganizationCustomRepoRoles struct {
	TotalCount      *int               `json:"total_count,omitempty"`
	CustomRepoRoles []*CustomRepoRoles `json:"custom_roles,omitempty"`
}

// CustomRepoRoles represents custom repository roles for an organization.
// See https://docs.github.com/en/enterprise-cloud@latest/organizations/managing-peoples-access-to-your-organization-with-roles/managing-custom-repository-roles-for-an-organization
// for more information.
type CustomRepoRoles struct {
	ID          *int64   `json:"id,omitempty"`
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"`
	BaseRole    *string  `json:"base_role,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

// ListCustomRepoRoles lists the custom repository roles available in this organization.
// In order to see custom repository roles in an organization, the authenticated user must be an organization owner.
//
// GitHub API docs: https://docs.github.com/en/rest/orgs/custom-roles#list-custom-repository-roles-in-an-organization
func (s *OrganizationsService) ListCustomRepoRoles(ctx context.Context, org string) (*OrganizationCustomRepoRoles, *Response, error) {
	u := fmt.Sprintf("orgs/%v/custom-repository-roles", org)

	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	customRepoRoles := new(OrganizationCustomRepoRoles)
	resp, err := s.client.Do(ctx, req, customRepoRoles)
	if err != nil {
		return nil, resp, err
	}

	return customRepoRoles, resp, nil
}

// CreateOrUpdateCustomRoleOptions represents options required to create or update a custom repository role.
type CreateOrUpdateCustomRoleOptions struct {
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"`
	BaseRole    *string  `json:"base_role,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

// CreateCustomRepoRole creates a custom repository role in this organization.
// In order to create custom repository roles in an organization, the authenticated user must be an organization owner.
//
// GitHub API docs: https://docs.github.com/en/rest/orgs/custom-roles#create-a-custom-repository-role
func (s *OrganizationsService) CreateCustomRepoRole(ctx context.Context, org string, opts *CreateOrUpdateCustomRoleOptions) (*CustomRepoRoles, *Response, error) {
	u := fmt.Sprintf("orgs/%v/custom-repository-roles", org)

	req, err := s.client.NewRequest("POST", u, opts)
	if err != nil {
		return nil, nil, err
	}

	resultingRole := new(CustomRepoRoles)
	resp, err := s.client.Do(ctx, req, resultingRole)
	if err != nil {
		return nil, resp, err
	}

	return resultingRole, resp, err
}

// UpdateCustomRepoRole updates a custom repository role in this organization.
// In order to update custom repository roles in an organization, the authenticated user must be an organization owner.
//
// GitHub API docs: https://docs.github.com/en/rest/orgs/custom-roles#update-a-custom-repository-role
func (s *OrganizationsService) UpdateCustomRepoRole(ctx context.Context, org, roleID string, opts *CreateOrUpdateCustomRoleOptions) (*CustomRepoRoles, *Response, error) {
	u := fmt.Sprintf("orgs/%v/custom-repository-roles/%v", org, roleID)

	req, err := s.client.NewRequest("PATCH", u, opts)
	if err != nil {
		return nil, nil, err
	}

	resultingRole := new(CustomRepoRoles)
	resp, err := s.client.Do(ctx, req, resultingRole)
	if err != nil {
		return nil, resp, err
	}

	return resultingRole, resp, err
}

// DeleteCustomRepoRole deletes an existing custom repository role in this organization.
// In order to delete custom repository roles in an organization, the authenticated user must be an organization owner.
//
// GitHub API docs: https://docs.github.com/en/rest/orgs/custom-roles#delete-a-custom-repository-role
func (s *OrganizationsService) DeleteCustomRepoRole(ctx context.Context, org, roleID string) (*Response, error) {
	u := fmt.Sprintf("orgs/%v/custom-repository-roles/%v", org, roleID)

	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return nil, err
	}

	resultingRole := new(CustomRepoRoles)
	resp, err := s.client.Do(ctx, req, resultingRole)
	if err != nil {
		return resp, err
	}

	return resp, nil
}
