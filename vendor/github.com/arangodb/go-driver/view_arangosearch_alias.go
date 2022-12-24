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
)

// ArangoSearchViewAlias provides access to the information of a view alias
// Views aliases are only available in ArangoDB 3.10 and higher.
type ArangoSearchViewAlias interface {
	// View Includes generic View functions
	View

	// Properties fetches extended information about the view.
	Properties(ctx context.Context) (ArangoSearchAliasViewProperties, error)

	// SetProperties changes properties of the view.
	SetProperties(ctx context.Context, options ArangoSearchAliasViewProperties) (ArangoSearchAliasViewProperties, error)
}

type ArangoSearchAliasViewProperties struct {
	ArangoSearchViewBase

	// Indexes A list of inverted indexes to add to the View.
	Indexes []ArangoSearchAliasIndex `json:"indexes,omitempty"`
}

type ArangoSearchAliasIndex struct {
	// Collection The name of a collection.
	Collection string `json:"collection"`
	// Index The name of an inverted index of the collection.
	Index string `json:"index"`
}
