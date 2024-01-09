// Copyright 2023 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github

import (
	"context"
	"fmt"
)

// ActionsPermissions represents a policy for repositories and allowed actions in an organization.
//
// GitHub API docs: https://docs.github.com/en/rest/actions/permissions
type ActionsPermissions struct {
	EnabledRepositories *string `json:"enabled_repositories,omitempty"`
	AllowedActions      *string `json:"allowed_actions,omitempty"`
	SelectedActionsURL  *string `json:"selected_actions_url,omitempty"`
}

func (a ActionsPermissions) String() string {
	return Stringify(a)
}

// ActionsEnabledOnOrgRepos represents all the repositories in an organization for which Actions is enabled.
type ActionsEnabledOnOrgRepos struct {
	TotalCount   int           `json:"total_count"`
	Repositories []*Repository `json:"repositories"`
}

// ActionsAllowed represents selected actions that are allowed.
//
// GitHub API docs: https://docs.github.com/en/rest/actions/permissions
type ActionsAllowed struct {
	GithubOwnedAllowed *bool    `json:"github_owned_allowed,omitempty"`
	VerifiedAllowed    *bool    `json:"verified_allowed,omitempty"`
	PatternsAllowed    []string `json:"patterns_allowed,omitempty"`
}

func (a ActionsAllowed) String() string {
	return Stringify(a)
}

// GetActionsPermissions gets the GitHub Actions permissions policy for repositories and allowed actions in an organization.
//
// GitHub API docs: https://docs.github.com/en/rest/actions/permissions#get-github-actions-permissions-for-an-organization
func (s *ActionsService) GetActionsPermissions(ctx context.Context, org string) (*ActionsPermissions, *Response, error) {
	u := fmt.Sprintf("orgs/%v/actions/permissions", org)

	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	permissions := new(ActionsPermissions)
	resp, err := s.client.Do(ctx, req, permissions)
	if err != nil {
		return nil, resp, err
	}

	return permissions, resp, nil
}

// EditActionsPermissions sets the permissions policy for repositories and allowed actions in an organization.
//
// GitHub API docs: https://docs.github.com/en/rest/actions/permissions#set-github-actions-permissions-for-an-organization
func (s *ActionsService) EditActionsPermissions(ctx context.Context, org string, actionsPermissions ActionsPermissions) (*ActionsPermissions, *Response, error) {
	u := fmt.Sprintf("orgs/%v/actions/permissions", org)
	req, err := s.client.NewRequest("PUT", u, actionsPermissions)
	if err != nil {
		return nil, nil, err
	}

	p := new(ActionsPermissions)
	resp, err := s.client.Do(ctx, req, p)
	if err != nil {
		return nil, resp, err
	}

	return p, resp, nil
}

// ListEnabledReposInOrg lists the selected repositories that are enabled for GitHub Actions in an organization.
//
// GitHub API docs: https://docs.github.com/en/rest/actions/permissions#list-selected-repositories-enabled-for-github-actions-in-an-organization
func (s *ActionsService) ListEnabledReposInOrg(ctx context.Context, owner string, opts *ListOptions) (*ActionsEnabledOnOrgRepos, *Response, error) {
	u := fmt.Sprintf("orgs/%v/actions/permissions/repositories", owner)
	u, err := addOptions(u, opts)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	repos := &ActionsEnabledOnOrgRepos{}
	resp, err := s.client.Do(ctx, req, repos)
	if err != nil {
		return nil, resp, err
	}

	return repos, resp, nil
}

// SetEnabledReposInOrg replaces the list of selected repositories that are enabled for GitHub Actions in an organization..
//
// GitHub API docs: https://docs.github.com/en/rest/actions/permissions#set-selected-repositories-enabled-for-github-actions-in-an-organization
func (s *ActionsService) SetEnabledReposInOrg(ctx context.Context, owner string, repositoryIDs []int64) (*Response, error) {
	u := fmt.Sprintf("orgs/%v/actions/permissions/repositories", owner)

	req, err := s.client.NewRequest("PUT", u, struct {
		IDs []int64 `json:"selected_repository_ids"`
	}{IDs: repositoryIDs})
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(ctx, req, nil)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

// AddEnabledReposInOrg adds a repository to the list of selected repositories that are enabled for GitHub Actions in an organization.
//
// GitHub API docs: https://docs.github.com/en/rest/actions/permissions#enable-a-selected-repository-for-github-actions-in-an-organization
func (s *ActionsService) AddEnabledReposInOrg(ctx context.Context, owner string, repositoryID int64) (*Response, error) {
	u := fmt.Sprintf("orgs/%v/actions/permissions/repositories/%v", owner, repositoryID)

	req, err := s.client.NewRequest("PUT", u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(ctx, req, nil)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

// RemoveEnabledRepoInOrg removes a single repository from the list of enabled repos for GitHub Actions in an organization.
//
// GitHub API docs: https://docs.github.com/en/rest/actions/permissions#disable-a-selected-repository-for-github-actions-in-an-organization
func (s *ActionsService) RemoveEnabledReposInOrg(ctx context.Context, owner string, repositoryID int64) (*Response, error) {
	u := fmt.Sprintf("orgs/%v/actions/permissions/repositories/%v", owner, repositoryID)

	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(ctx, req, nil)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

// GetActionsAllowed gets the actions that are allowed in an organization.
//
// GitHub API docs: https://docs.github.com/en/rest/actions/permissions#get-allowed-actions-and-reusable-workflows-for-an-organization
func (s *ActionsService) GetActionsAllowed(ctx context.Context, org string) (*ActionsAllowed, *Response, error) {
	u := fmt.Sprintf("orgs/%v/actions/permissions/selected-actions", org)

	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	actionsAllowed := new(ActionsAllowed)
	resp, err := s.client.Do(ctx, req, actionsAllowed)
	if err != nil {
		return nil, resp, err
	}

	return actionsAllowed, resp, nil
}

// EditActionsAllowed sets the actions that are allowed in an organization.
//
// GitHub API docs: https://docs.github.com/en/rest/actions/permissions#set-allowed-actions-and-reusable-workflows-for-an-organization
func (s *ActionsService) EditActionsAllowed(ctx context.Context, org string, actionsAllowed ActionsAllowed) (*ActionsAllowed, *Response, error) {
	u := fmt.Sprintf("orgs/%v/actions/permissions/selected-actions", org)
	req, err := s.client.NewRequest("PUT", u, actionsAllowed)
	if err != nil {
		return nil, nil, err
	}

	p := new(ActionsAllowed)
	resp, err := s.client.Do(ctx, req, p)
	if err != nil {
		return nil, resp, err
	}

	return p, resp, nil
}
