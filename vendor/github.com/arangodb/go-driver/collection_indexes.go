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

// CollectionIndexes provides access to the indexes in a single collection.
type CollectionIndexes interface {
	// Index opens a connection to an existing index within the collection.
	// If no index with given name exists, an NotFoundError is returned.
	Index(ctx context.Context, name string) (Index, error)

	// IndexExists returns true if an index with given name exists within the collection.
	IndexExists(ctx context.Context, name string) (bool, error)

	// Indexes returns a list of all indexes in the collection.
	Indexes(ctx context.Context) ([]Index, error)

	// Deprecated: since 3.10 version. Use ArangoSearch view instead.
	//
	// EnsureFullTextIndex creates a fulltext index in the collection, if it does not already exist.
	// Fields is a slice of attribute names. Currently, the slice is limited to exactly one attribute.
	// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
	EnsureFullTextIndex(ctx context.Context, fields []string, options *EnsureFullTextIndexOptions) (Index, bool, error)

	// EnsureGeoIndex creates a hash index in the collection, if it does not already exist.
	// Fields is a slice with one or two attribute paths. If it is a slice with one attribute path location,
	// then a geo-spatial index on all documents is created using location as path to the coordinates.
	// The value of the attribute must be a slice with at least two double values. The slice must contain the latitude (first value)
	// and the longitude (second value). All documents, which do not have the attribute path or with value that are not suitable, are ignored.
	// If it is a slice with two attribute paths latitude and longitude, then a geo-spatial index on all documents is created
	// using latitude and longitude as paths the latitude and the longitude. The value of the attribute latitude and of the
	// attribute longitude must a double. All documents, which do not have the attribute paths or which values are not suitable, are ignored.
	// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
	EnsureGeoIndex(ctx context.Context, fields []string, options *EnsureGeoIndexOptions) (Index, bool, error)

	// EnsureHashIndex creates a hash index in the collection, if it does not already exist.
	// Fields is a slice of attribute paths.
	// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
	EnsureHashIndex(ctx context.Context, fields []string, options *EnsureHashIndexOptions) (Index, bool, error)

	// EnsurePersistentIndex creates a persistent index in the collection, if it does not already exist.
	// Fields is a slice of attribute paths.
	// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
	EnsurePersistentIndex(ctx context.Context, fields []string, options *EnsurePersistentIndexOptions) (Index, bool, error)

	// EnsureSkipListIndex creates a skiplist index in the collection, if it does not already exist.
	// Fields is a slice of attribute paths.
	// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
	EnsureSkipListIndex(ctx context.Context, fields []string, options *EnsureSkipListIndexOptions) (Index, bool, error)

	// EnsureTTLIndex creates a TLL collection, if it does not already exist.
	// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
	EnsureTTLIndex(ctx context.Context, field string, expireAfter int, options *EnsureTTLIndexOptions) (Index, bool, error)

	// EnsureZKDIndex creates a ZKD multi-dimensional index for the collection, if it does not already exist.
	// Note that zkd indexes are an experimental feature in ArangoDB 3.9.
	EnsureZKDIndex(ctx context.Context, fields []string, options *EnsureZKDIndexOptions) (Index, bool, error)

	// EnsureMDIIndex creates a multidimensional index for the collection, if it does not already exist.
	// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
	// Available in ArangoDB 3.12 and later.
	EnsureMDIIndex(ctx context.Context, fields []string, options *EnsureMDIIndexOptions) (Index, bool, error)

	// EnsureMDIPrefixedIndex creates is an additional index variant of mdi index that lets you specify additional
	// attributes for the index to narrow down the search space using equality checks.
	// Available in ArangoDB 3.12 and later.
	EnsureMDIPrefixedIndex(ctx context.Context, fields []string, options *EnsureMDIPrefixedIndexOptions) (Index, bool, error)

	// EnsureInvertedIndex creates an inverted index in the collection, if it does not already exist.
	// Available in ArangoDB 3.10 and later.
	EnsureInvertedIndex(ctx context.Context, options *InvertedIndexOptions) (Index, bool, error)
}

// Deprecated: since 3.10 version. Use ArangoSearch view instead.
//
// EnsureFullTextIndexOptions contains specific options for creating a full text index.
type EnsureFullTextIndexOptions struct {
	// MinLength is the minimum character length of words to index. Will default to a server-defined
	// value if unspecified (0). It is thus recommended to set this value explicitly when creating the index.
	MinLength int
	// InBackground if true will not hold an exclusive collection lock for the entire index creation period (rocksdb only).
	InBackground bool
	// Name optional user defined name used for hints in AQL queries
	Name string
	// Estimates  determines if the to-be-created index should maintain selectivity estimates or not.
	Estimates *bool
}

// EnsureGeoIndexOptions contains specific options for creating a geo index.
type EnsureGeoIndexOptions struct {
	// If a geo-spatial index on a location is constructed and GeoJSON is true, then the order within the array
	// is longitude followed by latitude. This corresponds to the format described in http://geojson.org/geojson-spec.html#positions
	GeoJSON bool
	// InBackground if true will not hold an exclusive collection lock for the entire index creation period (rocksdb only).
	InBackground bool
	// Name optional user defined name used for hints in AQL queries
	Name string
	// Estimates  determines if the to-be-created index should maintain selectivity estimates or not.
	Estimates *bool
	// LegacyPolygons determines if the to-be-created index should use legacy polygons or not.
	// It is relevant for those that have geoJson set to true only.
	// Old geo indexes from versions from below 3.10 will always implicitly have the legacyPolygons option set to true.
	// Newly generated geo indexes from 3.10 on will have the legacyPolygons option by default set to false,
	// however, it can still be explicitly overwritten with true to create a legacy index but is not recommended.
	LegacyPolygons bool
}

// EnsureHashIndexOptions contains specific options for creating a hash index.
// Note: "hash" and "skiplist" are only aliases for "persistent" with the RocksDB storage engine which is only storage engine since 3.7
type EnsureHashIndexOptions struct {
	// If true, then create a unique index.
	Unique bool
	// If true, then create a sparse index.
	Sparse bool
	// If true, de-duplication of array-values, before being added to the index, will be turned off.
	// This flag requires ArangoDB 3.2.
	// Note: this setting is only relevant for indexes with array fields (e.g. "fieldName[*]")
	NoDeduplicate bool
	// InBackground if true will not hold an exclusive collection lock for the entire index creation period (rocksdb only).
	InBackground bool
	// Name optional user defined name used for hints in AQL queries
	Name string
	// Estimates  determines if the to-be-created index should maintain selectivity estimates or not.
	Estimates *bool
}

// EnsurePersistentIndexOptions contains specific options for creating a persistent index.
// Note: "hash" and "skiplist" are only aliases for "persistent" with the RocksDB storage engine which is only storage engine since 3.7
type EnsurePersistentIndexOptions struct {
	// If true, then create a unique index.
	Unique bool
	// If true, then create a sparse index.
	Sparse bool
	// If true, de-duplication of array-values, before being added to the index, will be turned off.
	// This flag requires ArangoDB 3.2.
	// Note: this setting is only relevant for indexes with array fields (e.g. "fieldName[*]")
	NoDeduplicate bool
	// InBackground if true will not hold an exclusive collection lock for the entire index creation period (rocksdb only).
	InBackground bool
	// Name optional user defined name used for hints in AQL queries
	Name string
	// Estimates  determines if the to-be-created index should maintain selectivity estimates or not.
	Estimates *bool
	// CacheEnabled if true, then the index will be cached in memory. Caching is turned off by default.
	CacheEnabled bool
	// StoreValues if true, then the additional attributes will be included.
	// These additional attributes cannot be used for index lookups or sorts, but they can be used for projections.
	// There must be no overlap of attribute paths between `fields` and `storedValues`. The maximum number of values is 32.
	StoredValues []string
}

// EnsureSkipListIndexOptions contains specific options for creating a skip-list index.
// Note: "hash" and "skiplist" are only aliases for "persistent" with the RocksDB storage engine which is only storage engine since 3.7
type EnsureSkipListIndexOptions struct {
	// If true, then create a unique index.
	Unique bool
	// If true, then create a sparse index.
	Sparse bool
	// If true, de-duplication of array-values, before being added to the index, will be turned off.
	// This flag requires ArangoDB 3.2.
	// Note: this setting is only relevant for indexes with array fields (e.g. "fieldName[*]")
	NoDeduplicate bool
	// InBackground if true will not hold an exclusive collection lock for the entire index creation period (rocksdb only).
	InBackground bool
	// Name optional user defined name used for hints in AQL queries
	Name string
	// Estimates  determines if the to-be-created index should maintain selectivity estimates or not.
	Estimates *bool
}

// EnsureTTLIndexOptions provides specific options for creating a TTL index
type EnsureTTLIndexOptions struct {
	// InBackground if true will not hold an exclusive collection lock for the entire index creation period (rocksdb only).
	InBackground bool
	// Name optional user defined name used for hints in AQL queries
	Name string
	// Estimates  determines if the to-be-created index should maintain selectivity estimates or not.
	Estimates *bool
}

// EnsureZKDIndexOptions provides specific options for creating a ZKD index
type EnsureZKDIndexOptions struct {
	// If true, then create a unique index.
	Unique bool
	// InBackground if true will not hold an exclusive collection lock for the entire index creation period (rocksdb only).
	InBackground bool
	// Name optional user defined name used for hints in AQL queries
	Name string
	// fieldValueTypes is required and the only allowed value is "double". Future extensions of the index will allow other types.
	FieldValueTypes string
}

// EnsureMDIIndexOptions provides specific options for creating a MDI index
type EnsureMDIIndexOptions struct {
	// If true, then create a unique index.
	Unique bool
	// InBackground if true will not hold an exclusive collection lock for the entire index creation period (rocksdb only).
	InBackground bool
	// Name optional user defined name used for hints in AQL queries
	Name string
	// fieldValueTypes is required and the only allowed value is "double". Future extensions of the index will allow other types.
	FieldValueTypes string
	// Sparse If `true`, then create a sparse index to exclude documents from the index that do not have the defined
	// attributes or are explicitly set to `null` values. If a non-value is set, it still needs to be numeric.
	Sparse bool
	// StoredValues The optional `storedValues` attribute can contain an array of paths to additional attributes to
	// store in the index.
	StoredValues []string
}

type EnsureMDIPrefixedIndexOptions struct {
	EnsureMDIIndexOptions

	// PrefixFields is required and contains nn array of attribute names used as search prefix.
	// Array expansions are not allowed.
	PrefixFields []string
}

// InvertedIndexOptions provides specific options for creating an inverted index
// Available since ArangoDB 3.10
type InvertedIndexOptions struct {
	// Name optional user defined name used for hints in AQL queries
	Name string `json:"name"`
	// InBackground This attribute can be set to true to create the index in the background,
	// not write-locking the underlying collection for as long as if the index is built in the foreground.
	// The default value is false.
	InBackground   bool `json:"inBackground,omitempty"`
	IsNewlyCreated bool `json:"isNewlyCreated,omitempty"`

	// The number of threads to use for indexing the fields. Default: 2
	Parallelism int `json:"parallelism,omitempty"`
	// PrimarySort You can define a primary sort order to enable an AQL optimization.
	// If a query iterates over all documents of a collection, wants to sort them by attribute values, and the (left-most) fields to sort by,
	// as well as their sorting direction, match with the primarySort definition, then the SORT operation is optimized away.
	PrimarySort InvertedIndexPrimarySort `json:"primarySort,omitempty"`
	// StoredValues The optional storedValues attribute can contain an array of paths to additional attributes to store in the index.
	// These additional attributes cannot be used for index lookups or for sorting, but they can be used for projections.
	// This allows an index to fully cover more queries and avoid extra document lookups.
	StoredValues []StoredValue `json:"storedValues,omitempty"`
	// Analyzer  The name of an Analyzer to use by default. This Analyzer is applied to the values of the indexed fields for which you don’t define Analyzers explicitly.
	Analyzer string `json:"analyzer,omitempty"`
	// Features list of analyzer features, default []
	Features []ArangoSearchAnalyzerFeature `json:"features,omitempty"`
	// IncludeAllFields If set to true, all fields of this element will be indexed. Defaults to false.
	IncludeAllFields bool `json:"includeAllFields,omitempty"`
	// TrackListPositions If set to true, values in a listed are treated as separate values. Defaults to false.
	TrackListPositions bool `json:"trackListPositions,omitempty"`
	// This option only applies if you use the inverted index in a search-alias Views.
	// You can set the option to true to get the same behavior as with arangosearch Views regarding the indexing of array values as the default.
	// If enabled, both, array and primitive values (strings, numbers, etc.) are accepted. Every element of an array is indexed according to the trackListPositions option.
	// If set to false, it depends on the attribute path. If it explicitly expand an array ([*]), then the elements are indexed separately.
	// Otherwise, the array is indexed as a whole, but only geopoint and aql Analyzers accept array inputs.
	// You cannot use an array expansion if searchField is enabled.
	SearchField bool `json:"searchField,omitempty"`
	// Fields contains the properties for individual fields of the element.
	// The key of the map are field names.
	Fields []InvertedIndexField `json:"fields,omitempty"`
	// ConsolidationIntervalMsec Wait at least this many milliseconds between applying ‘consolidationPolicy’ to consolidate View data store
	// and possibly release space on the filesystem (default: 1000, to disable use: 0).
	ConsolidationIntervalMsec *int64 `json:"consolidationIntervalMsec,omitempty"`
	// CommitIntervalMsec Wait at least this many milliseconds between committing View data store changes and making
	// documents visible to queries (default: 1000, to disable use: 0).
	CommitIntervalMsec *int64 `json:"commitIntervalMsec,omitempty"`
	// CleanupIntervalStep Wait at least this many commits between removing unused files in the ArangoSearch data directory
	// (default: 2, to disable use: 0).
	CleanupIntervalStep *int64 `json:"cleanupIntervalStep,omitempty"`
	// ConsolidationPolicy The consolidation policy to apply for selecting which segments should be merged (default: {}).
	ConsolidationPolicy *ArangoSearchConsolidationPolicy `json:"consolidationPolicy,omitempty"`
	// WriteBufferIdle Maximum number of writers (segments) cached in the pool (default: 64, use 0 to disable)
	WriteBufferIdle *int64 `json:"writebufferIdle,omitempty"`
	// WriteBufferActive Maximum number of concurrent active writers (segments) that perform a transaction.
	// Other writers (segments) wait till current active writers (segments) finish (default: 0, use 0 to disable)
	WriteBufferActive *int64 `json:"writebufferActive,omitempty"`
	// WriteBufferSizeMax Maximum memory byte size per writer (segment) before a writer (segment) flush is triggered.
	// 0 value turns off this limit for any writer (buffer) and data will be flushed periodically based on the value defined for the flush thread (ArangoDB server startup option).
	// 0 value should be used carefully due to high potential memory consumption (default: 33554432, use 0 to disable)
	WriteBufferSizeMax *int64 `json:"writebufferSizeMax,omitempty"`
	// OptimizeTopK is an array of strings defining optimized sort expressions.
	// Introduced in v3.11.0, Enterprise Edition only.
	OptimizeTopK []string `json:"optimizeTopK,omitempty"`
}

// InvertedIndexPrimarySort defines compression and list of fields to be sorted.
type InvertedIndexPrimarySort struct {
	Fields []ArangoSearchPrimarySortEntry `json:"fields,omitempty"`
	// Compression optional
	Compression PrimarySortCompression `json:"compression,omitempty"`
}

// InvertedIndexField contains configuration for indexing of the field
type InvertedIndexField struct {
	// Name An attribute path. The . character denotes sub-attributes.
	Name string `json:"name"`
	// Analyzer indicating the name of an analyzer instance
	// Default: the value defined by the top-level analyzer option, or if not set, the default identity Analyzer.
	Analyzer string `json:"analyzer,omitempty"`
	// IncludeAllFields This option only applies if you use the inverted index in a search-alias Views.
	// If set to true, then all sub-attributes of this field are indexed, excluding any sub-attributes that are configured separately by other elements in the fields array (and their sub-attributes). The analyzer and features properties apply to the sub-attributes.
	// If set to false, then sub-attributes are ignored. The default value is defined by the top-level includeAllFields option, or false if not set.
	IncludeAllFields bool `json:"includeAllFields,omitempty"`
	// SearchField This option only applies if you use the inverted index in a search-alias Views.
	// You can set the option to true to get the same behavior as with arangosearch Views regarding the indexing of array values for this field. If enabled, both, array and primitive values (strings, numbers, etc.) are accepted. Every element of an array is indexed according to the trackListPositions option.
	// If set to false, it depends on the attribute path. If it explicitly expand an array ([*]), then the elements are indexed separately. Otherwise, the array is indexed as a whole, but only geopoint and aql Analyzers accept array inputs. You cannot use an array expansion if searchField is enabled.
	// Default: the value defined by the top-level searchField option, or false if not set.
	SearchField bool `json:"searchField,omitempty"`
	// TrackListPositions This option only applies if you use the inverted index in a search-alias Views.
	// If set to true, then track the value position in arrays for array values. For example, when querying a document like { attr: [ "valueX", "valueY", "valueZ" ] }, you need to specify the array element, e.g. doc.attr[1] == "valueY".
	// If set to false, all values in an array are treated as equal alternatives. You don’t specify an array element in queries, e.g. doc.attr == "valueY", and all elements are searched for a match.
	// Default: the value defined by the top-level trackListPositions option, or false if not set.
	TrackListPositions bool `json:"trackListPositions,omitempty"`
	// A list of Analyzer features to use for this field. They define what features are enabled for the analyzer
	Features []ArangoSearchAnalyzerFeature `json:"features,omitempty"`
	// Nested - Index the specified sub-objects that are stored in an array.
	// Other than with the fields property, the values get indexed in a way that lets you query for co-occurring values.
	// For example, you can search the sub-objects and all the conditions need to be met by a single sub-object instead of across all of them.
	// Enterprise-only feature
	Nested []InvertedIndexField `json:"nested,omitempty"`
}
