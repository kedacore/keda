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
// Author Lars Maier
//

package driver

import "context"

type ArangoSearchAnalyzer interface {
	// Name returns the analyzer name
	Name() string

	// Type returns the analyzer type
	Type() ArangoSearchAnalyzerType

	// UniqueName returns the unique name: <database>::<analyzer-name>
	UniqueName() string

	// Definition returns the analyzer definition
	Definition() ArangoSearchAnalyzerDefinition

	// Properties returns the analyzer properties
	Properties() ArangoSearchAnalyzerProperties

	// Database returns the database of this analyzer
	Database() Database

	// Removes the analyzers
	Remove(ctx context.Context, force bool) error
}

type DatabaseArangoSearchAnalyzers interface {

	// Ensure ensures that the given analyzer exists. If it does not exist it is created.
	// The function returns whether the analyzer already existed or an error.
	EnsureAnalyzer(ctx context.Context, analyzer ArangoSearchAnalyzerDefinition) (bool, ArangoSearchAnalyzer, error)

	// Get returns the analyzer definition for the given analyzer or returns an error
	Analyzer(ctx context.Context, name string) (ArangoSearchAnalyzer, error)

	// List returns a list of all analyzers
	Analyzers(ctx context.Context) ([]ArangoSearchAnalyzer, error)
}
