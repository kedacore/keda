//
// DISCLAIMER
//
// Copyright 2018-2025 ArangoDB GmbH, Cologne, Germany
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
// Author Lars Maier
//

package driver

import (
	"context"
	"net/http"
	"path"
	"strings"
)

type analyzer struct {
	definition ArangoSearchAnalyzerDefinition
	db         *database
}

// Name returns the analyzer name
func (a *analyzer) Name() string {
	split := strings.Split(a.definition.Name, "::")
	return split[len(split)-1]
}

// UniqueName returns the unique name: <database>::<analyzer-name>
func (a *analyzer) UniqueName() string {
	return a.definition.Name
}

// Type returns the analyzer type
func (a *analyzer) Type() ArangoSearchAnalyzerType {
	return a.definition.Type
}

// Definition returns the analyzer definition
func (a *analyzer) Definition() ArangoSearchAnalyzerDefinition {
	return a.definition
}

// Properties returns the analyzer properties
func (a *analyzer) Properties() ArangoSearchAnalyzerProperties {
	return a.definition.Properties
}

// Remove the analyzers
func (a *analyzer) Remove(ctx context.Context, force bool) error {
	req, err := a.db.conn.NewRequest("DELETE", path.Join(a.db.relPath(), "_api/analyzer/", a.Name()))
	if err != nil {
		return WithStack(err)
	}
	applyContextSettings(ctx, req)

	if force {
		req.SetQuery("force", "true")
	}

	resp, err := a.db.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return WithStack(err)
	}
	var actualDef ArangoSearchAnalyzerDefinition
	if err := resp.ParseBody("", &actualDef); err != nil {
		return WithStack(err)
	}
	return nil
}

// Database returns the database of this analyzer
func (a *analyzer) Database() Database {
	return a.db
}

// Deprecated: Use EnsureCreatedAnalyzer instead
//
// Ensure ensures that the given analyzer exists. If it does not exist it is created.
// The function returns whether the analyzer already existed or an error.
func (d *database) EnsureAnalyzer(ctx context.Context, definition ArangoSearchAnalyzerDefinition) (bool, ArangoSearchAnalyzer, error) {
	req, err := d.conn.NewRequest("POST", path.Join(d.relPath(), "_api/analyzer"))
	if err != nil {
		return false, nil, WithStack(err)
	}
	applyContextSettings(ctx, req)
	req, err = req.SetBody(definition)
	if err != nil {
		return false, nil, WithStack(err)
	}
	resp, err := d.conn.Do(ctx, req)
	if err != nil {
		return false, nil, WithStack(err)
	}
	if err := resp.CheckStatus(201, 200); err != nil {
		return false, nil, WithStack(err)
	}
	found := resp.StatusCode() == 200
	var actualDef ArangoSearchAnalyzerDefinition
	if err := resp.ParseBody("", &actualDef); err != nil {
		return false, nil, WithStack(err)
	}
	return found, &analyzer{
		db:         d,
		definition: actualDef,
	}, nil
}

// EnsureCreatedAnalyzer creates an Analyzer for the database, if it does not already exist.
// The function returns the Analyser object together with a boolean indicating if the Analyzer was newly created (true) or pre-existing (false).
func (d *database) EnsureCreatedAnalyzer(ctx context.Context, definition *ArangoSearchAnalyzerDefinition) (ArangoSearchAnalyzer, bool, error) {
	req, err := d.conn.NewRequest("POST", path.Join(d.relPath(), "_api/analyzer"))
	if err != nil {
		return nil, false, WithStack(err)
	}
	applyContextSettings(ctx, req)
	req, err = req.SetBody(definition)
	if err != nil {
		return nil, false, WithStack(err)
	}
	resp, err := d.conn.Do(ctx, req)
	if err != nil {
		return nil, false, WithStack(err)
	}

	if err := resp.CheckStatus(http.StatusCreated, http.StatusOK); err != nil {
		return nil, false, WithStack(err)
	}

	var actualDef ArangoSearchAnalyzerDefinition
	if err := resp.ParseBody("", &actualDef); err != nil {
		return nil, false, WithStack(err)
	}

	created := resp.StatusCode() == http.StatusCreated
	return &analyzer{
		db:         d,
		definition: actualDef,
	}, created, nil

}

// Get returns the analyzer definition for the given analyzer or returns an error
func (d *database) Analyzer(ctx context.Context, name string) (ArangoSearchAnalyzer, error) {
	req, err := d.conn.NewRequest("GET", path.Join(d.relPath(), "_api/analyzer/", name))
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
	var actualDef ArangoSearchAnalyzerDefinition
	if err := resp.ParseBody("", &actualDef); err != nil {
		return nil, WithStack(err)
	}
	return &analyzer{
		db:         d,
		definition: actualDef,
	}, nil
}

type analyzerListResponse struct {
	Analyzer []ArangoSearchAnalyzerDefinition `json:"result,omitempty"`
	ArangoError
}

// List returns a list of all analyzers
func (d *database) Analyzers(ctx context.Context) ([]ArangoSearchAnalyzer, error) {
	req, err := d.conn.NewRequest("GET", path.Join(d.relPath(), "_api/analyzer"))
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
	var response analyzerListResponse
	if err := resp.ParseBody("", &response); err != nil {
		return nil, WithStack(err)
	}

	result := make([]ArangoSearchAnalyzer, 0, len(response.Analyzer))
	for _, a := range response.Analyzer {
		result = append(result, &analyzer{
			db:         d,
			definition: a,
		})
	}

	return result, nil
}
