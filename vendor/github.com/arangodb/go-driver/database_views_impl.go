//
// DISCLAIMER
//
// Copyright 2018 ArangoDB GmbH, Cologne, Germany
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Copyright holder is ArangoDB GmbH, Cologne, Germany
//
// Author Ewout Prangsma
//

package driver

import (
	"context"
	"path"
)

type viewInfo struct {
	Name string   `json:"name,omitempty"`
	Type ViewType `json:"type,omitempty"`
	ArangoID
	ArangoError
}

type getViewResponse struct {
	Result []viewInfo `json:"result,omitempty"`

	ArangoError
}

// View opens a connection to an existing view within the database.
// If no collection with given name exists, an NotFoundError is returned.
func (d *database) View(ctx context.Context, name string) (View, error) {
	escapedName := pathEscape(name)
	req, err := d.conn.NewRequest("GET", path.Join(d.relPath(), "_api/view", escapedName))
	if err != nil {
		return nil, WithStack(err)
	}
	applyContextSettings(ctx, req)
	resp, err := d.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return nil, WithStack(err)
	}
	var data viewInfo
	if err := resp.ParseBody("", &data); err != nil {
		return nil, WithStack(err)
	}
	view, err := newView(name, data.Type, d)
	if err != nil {
		return nil, WithStack(err)
	}
	return view, nil
}

// ViewExists returns true if a view with given name exists within the database.
func (d *database) ViewExists(ctx context.Context, name string) (bool, error) {
	escapedName := pathEscape(name)
	req, err := d.conn.NewRequest("GET", path.Join(d.relPath(), "_api/view", escapedName))
	if err != nil {
		return false, WithStack(err)
	}
	applyContextSettings(ctx, req)
	resp, err := d.conn.Do(ctx, req)
	if err != nil {
		return false, WithStack(err)
	}
	if err := resp.CheckStatus(200); err == nil {
		return true, nil
	} else if IsNotFound(err) {
		return false, nil
	} else {
		return false, WithStack(err)
	}
}

// Views returns a list of all views in the database.
func (d *database) Views(ctx context.Context) ([]View, error) {
	req, err := d.conn.NewRequest("GET", path.Join(d.relPath(), "_api/view"))
	if err != nil {
		return nil, WithStack(err)
	}
	applyContextSettings(ctx, req)
	resp, err := d.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return nil, WithStack(err)
	}
	var data getViewResponse
	if err := resp.ParseBody("", &data); err != nil {
		return nil, WithStack(err)
	}
	result := make([]View, 0, len(data.Result))
	for _, info := range data.Result {
		view, err := newView(info.Name, info.Type, d)
		if err != nil {
			return nil, WithStack(err)
		}
		result = append(result, view)
	}
	return result, nil
}

// CreateArangoSearchView creates a new view of type ArangoSearch,
// with given name and options, and opens a connection to it.
// If a view with given name already exists within the database, a ConflictError is returned.
func (d *database) CreateArangoSearchView(ctx context.Context, name string, options *ArangoSearchViewProperties) (ArangoSearchView, error) {
	input := struct {
		Name                       string   `json:"name"`
		Type                       ViewType `json:"type"`
		ArangoSearchViewProperties          // `json:"properties"`
	}{
		Name: name,
		Type: ViewTypeArangoSearch,
	}
	if options != nil {
		input.ArangoSearchViewProperties = *options
	}
	req, err := d.conn.NewRequest("POST", path.Join(d.relPath(), "_api/view"))
	if err != nil {
		return nil, WithStack(err)
	}
	if _, err := req.SetBody(input); err != nil {
		return nil, WithStack(err)
	}
	applyContextSettings(ctx, req)
	resp, err := d.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(201); err != nil {
		return nil, WithStack(err)
	}
	view, err := newView(name, input.Type, d)
	if err != nil {
		return nil, WithStack(err)
	}
	result, err := view.ArangoSearchView()
	if err != nil {
		return nil, WithStack(err)
	}

	return result, nil
}

// CreateArangoSearchAliasView creates a new view of type search-alias,
// with given name and options, and opens a connection to it.
// If a view with given name already exists within the database, a ConflictError is returned.
func (d *database) CreateArangoSearchAliasView(ctx context.Context, name string, options *ArangoSearchAliasViewProperties) (ArangoSearchViewAlias, error) {
	input := struct {
		Name string   `json:"name"`
		Type ViewType `json:"type"`
		ArangoSearchAliasViewProperties
	}{
		Name: name,
		Type: ViewTypeArangoSearchAlias,
	}
	if options != nil {
		input.ArangoSearchAliasViewProperties = *options
	}
	req, err := d.conn.NewRequest("POST", path.Join(d.relPath(), "_api/view"))
	if err != nil {
		return nil, WithStack(err)
	}
	if _, err := req.SetBody(input); err != nil {
		return nil, WithStack(err)
	}
	applyContextSettings(ctx, req)
	resp, err := d.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(201); err != nil {
		return nil, WithStack(err)
	}
	view, err := newView(name, input.Type, d)
	if err != nil {
		return nil, WithStack(err)
	}
	result, err := view.ArangoSearchViewAlias()
	if err != nil {
		return nil, WithStack(err)
	}

	return result, nil
}
