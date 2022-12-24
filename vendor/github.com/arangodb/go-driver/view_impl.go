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
	"fmt"
	"net/http"
	"path"
)

// newView creates a new View implementation.
func newView(name string, viewType ViewType, db *database) (View, error) {
	if name == "" {
		return nil, WithStack(InvalidArgumentError{Message: "name is empty"})
	}
	if viewType == "" {
		return nil, WithStack(InvalidArgumentError{Message: "viewType is empty"})
	}
	if db == nil {
		return nil, WithStack(InvalidArgumentError{Message: "db is nil"})
	}
	return &view{
		name:     name,
		viewType: viewType,
		db:       db,
		conn:     db.conn,
	}, nil
}

type view struct {
	name     string
	viewType ViewType
	db       *database
	conn     Connection
}

// relPath creates the relative path to this view (`_db/<db-name>/_api/view/<view-name>`)
func (v *view) relPath() string {
	escapedName := pathEscape(v.name)
	return path.Join(v.db.relPath(), "_api", "view", escapedName)
}

// Name returns the name of the view.
func (v *view) Name() string {
	return v.name
}

// Type returns the type of this view.
func (v *view) Type() ViewType {
	return v.viewType
}

// ArangoSearchView returns this view as an ArangoSearch view.
// When the type of the view is not ArangoSearch, an error is returned.
func (v *view) ArangoSearchView() (ArangoSearchView, error) {
	if v.viewType != ViewTypeArangoSearch {
		return nil, WithStack(newArangoError(http.StatusConflict, 0, fmt.Sprintf("Type must be '%s', got '%s'", ViewTypeArangoSearch, v.viewType)))
	}
	return &viewArangoSearch{view: *v}, nil
}

func (v *view) ArangoSearchViewAlias() (ArangoSearchViewAlias, error) {
	if v.viewType != ViewTypeArangoSearchAlias {
		return nil, WithStack(newArangoError(http.StatusConflict, 0, fmt.Sprintf("Type must be '%s', got '%s'", ViewTypeArangoSearchAlias, v.viewType)))
	}
	return &viewArangoSearchAlias{view: *v}, nil
}

// Database returns the database containing the view.
func (v *view) Database() Database {
	return v.db
}

func (v *view) Rename(ctx context.Context, newName string) error {
	if newName == "" {
		return WithStack(InvalidArgumentError{Message: "newName is empty"})
	}
	req, err := v.conn.NewRequest("PUT", path.Join(v.relPath(), "rename"))
	if err != nil {
		return WithStack(err)
	}
	input := struct {
		Name string `json:"name"`
	}{
		Name: newName,
	}
	if _, err := req.SetBody(input); err != nil {
		return WithStack(err)
	}
	applyContextSettings(ctx, req)
	resp, err := v.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return WithStack(err)
	}
	v.name = newName
	return nil
}

// Remove removes the entire view.
// If the view does not exist, a NotFoundError is returned.
func (v *view) Remove(ctx context.Context) error {
	req, err := v.conn.NewRequest("DELETE", v.relPath())
	if err != nil {
		return WithStack(err)
	}
	applyContextSettings(ctx, req)
	resp, err := v.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return WithStack(err)
	}
	return nil
}
