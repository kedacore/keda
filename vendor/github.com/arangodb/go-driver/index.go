//
// DISCLAIMER
//
// Copyright 2017-2025 ArangoDB GmbH, Cologne, Germany
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

package driver

import "context"

// IndexType represents a index type as string
type IndexType string

// Symbolic constants for index types
const (
	PrimaryIndex     = IndexType("primary")
	FullTextIndex    = IndexType("fulltext") // Deprecated: since 3.10 version. Use ArangoSearch view instead.
	HashIndex        = IndexType("hash")     // Deprecated: use PersistentIndexType instead
	SkipListIndex    = IndexType("skiplist") // Deprecated: use PersistentIndexType instead
	PersistentIndex  = IndexType("persistent")
	GeoIndex         = IndexType("geo")
	EdgeIndex        = IndexType("edge")
	TTLIndex         = IndexType("ttl")
	ZKDIndex         = IndexType("zkd") // Deprecated: since 3.12 version use MDIIndexType instead.
	InvertedIndex    = IndexType("inverted")
	MDIIndex         = IndexType("mdi")
	MDIPrefixedIndex = IndexType("mdi-prefixed")
)

// Index provides access to a single index in a single collection.
type Index interface {
	// Name returns the collection specific ID of the index. This value should be used for all functions
	// the require a index _name_.
	Name() string

	// ID returns the ID of the index. Effectively this is `<collection-name>/<index.Name()>`.
	ID() string

	// UserName returns the user provided name of the index or empty string if non is provided. This _name_
	// is used in query to provide hints for the optimizer about preferred indexes.
	UserName() string

	// Type returns the type of the index
	Type() IndexType

	// Remove removes the entire index.
	// If the index does not exist, a NotFoundError is returned.
	Remove(ctx context.Context) error

	// Fields returns a list of attributes of this index.
	Fields() []string

	// Unique returns if this index is unique.
	Unique() bool

	// Deduplicate returns deduplicate setting of this index.
	Deduplicate() bool

	// Sparse returns if this is a sparse index or not.
	Sparse() bool

	// GeoJSON returns if geo json was set for this index or not.
	GeoJSON() bool

	// InBackground if true will not hold an exclusive collection lock for the entire index creation period (rocksdb only).
	InBackground() bool

	// Estimates  determines if the to-be-created index should maintain selectivity estimates or not.
	Estimates() bool

	// MinLength returns min length for this index if set.
	MinLength() int

	// ExpireAfter returns an expire after for this index if set.
	ExpireAfter() int

	// LegacyPolygons determines if the index uses legacy polygons or not - GeoIndex only
	LegacyPolygons() bool

	// CacheEnabled returns if the index is enabled for caching or not - PersistentIndex only
	CacheEnabled() bool

	// StoredValues returns a list of stored values for this index - PersistentIndex only
	StoredValues() []string

	// InvertedIndexOptions returns the inverted index options for this index - InvertedIndex only
	InvertedIndexOptions() InvertedIndexOptions
}
