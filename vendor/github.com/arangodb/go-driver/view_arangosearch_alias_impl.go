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

// viewArangoSearchAlias implements ArangoSearchViewAlias
type viewArangoSearchAlias struct {
	view
}

// Properties fetches extended information about the view.
func (v *viewArangoSearchAlias) Properties(ctx context.Context) (ArangoSearchAliasViewProperties, error) {
	req, err := v.conn.NewRequest("GET", path.Join(v.relPath(), "properties"))
	if err != nil {
		return ArangoSearchAliasViewProperties{}, WithStack(err)
	}
	applyContextSettings(ctx, req)
	resp, err := v.conn.Do(ctx, req)
	if err != nil {
		return ArangoSearchAliasViewProperties{}, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return ArangoSearchAliasViewProperties{}, WithStack(err)
	}
	var data ArangoSearchAliasViewProperties
	if err := resp.ParseBody("", &data); err != nil {
		return ArangoSearchAliasViewProperties{}, WithStack(err)
	}
	return data, nil
}

// SetProperties changes properties of the view.
func (v *viewArangoSearchAlias) SetProperties(ctx context.Context, options ArangoSearchAliasViewProperties) (ArangoSearchAliasViewProperties, error) {
	req, err := v.conn.NewRequest("PUT", path.Join(v.relPath(), "properties"))
	if err != nil {
		return ArangoSearchAliasViewProperties{}, WithStack(err)
	}
	if _, err := req.SetBody(options); err != nil {
		return ArangoSearchAliasViewProperties{}, WithStack(err)
	}
	applyContextSettings(ctx, req)
	resp, err := v.conn.Do(ctx, req)
	if err != nil {
		return ArangoSearchAliasViewProperties{}, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return ArangoSearchAliasViewProperties{}, WithStack(err)
	}
	var data ArangoSearchAliasViewProperties
	if err := resp.ParseBody("", &data); err != nil {
		return ArangoSearchAliasViewProperties{}, WithStack(err)
	}
	return data, nil
}
