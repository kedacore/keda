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

import "context"

// DatabaseViews provides access to all views in a single database.
// Views are only available in ArangoDB 3.4 and higher.
type DatabaseViews interface {
	// View opens a connection to an existing view within the database.
	// If no collection with given name exists, an NotFoundError is returned.
	View(ctx context.Context, name string) (View, error)

	// ViewExists returns true if a view with given name exists within the database.
	ViewExists(ctx context.Context, name string) (bool, error)

	// Views returns a list of all views in the database.
	Views(ctx context.Context) ([]View, error)

	// CreateArangoSearchView creates a new view of type ArangoSearch,
	// with given name and options, and opens a connection to it.
	// If a view with given name already exists within the database, a ConflictError is returned.
	CreateArangoSearchView(ctx context.Context, name string, options *ArangoSearchViewProperties) (ArangoSearchView, error)

	// CreateArangoSearchAliasView creates ArangoSearch alias view with given name and options, and opens a connection to it.
	// If a view with given name already exists within the database, a ConflictError is returned.
	CreateArangoSearchAliasView(ctx context.Context, name string, options *ArangoSearchAliasViewProperties) (ArangoSearchViewAlias, error)
}

// ViewType is the type of view.
type ViewType string

const (
	// ViewTypeArangoSearch specifies an ArangoSearch view type.
	ViewTypeArangoSearch = ViewType("arangosearch")
	// ViewTypeArangoSearchAlias specifies an ArangoSearch view type alias.
	ViewTypeArangoSearchAlias = ViewType("search-alias")
)
