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

package driver

import (
	"context"
)

// ArangoSearchView provides access to the information of a view.
// Views are only available in ArangoDB 3.4 and higher.
type ArangoSearchView interface {
	// View Includes generic View functions
	View

	// Properties fetches extended information about the view.
	Properties(ctx context.Context) (ArangoSearchViewProperties, error)

	// SetProperties changes properties of the view.
	SetProperties(ctx context.Context, options ArangoSearchViewProperties) error
}

// ArangoSearchAnalyzerType specifies type of analyzer
type ArangoSearchAnalyzerType string

const (
	// ArangoSearchAnalyzerTypeIdentity treat value as atom (no transformation)
	ArangoSearchAnalyzerTypeIdentity ArangoSearchAnalyzerType = "identity"
	// ArangoSearchAnalyzerTypeDelimiter split into tokens at user-defined character
	ArangoSearchAnalyzerTypeDelimiter ArangoSearchAnalyzerType = "delimiter"
	// ArangoSearchAnalyzerTypeStem apply stemming to the value as a whole
	ArangoSearchAnalyzerTypeStem ArangoSearchAnalyzerType = "stem"
	// ArangoSearchAnalyzerTypeNorm apply normalization to the value as a whole
	ArangoSearchAnalyzerTypeNorm ArangoSearchAnalyzerType = "norm"
	// ArangoSearchAnalyzerTypeNGram create n-grams from value with user-defined lengths
	ArangoSearchAnalyzerTypeNGram ArangoSearchAnalyzerType = "ngram"
	// ArangoSearchAnalyzerTypeText tokenize into words, optionally with stemming, normalization and stop-word filtering
	ArangoSearchAnalyzerTypeText ArangoSearchAnalyzerType = "text"
	// ArangoSearchAnalyzerTypeAQL an Analyzer capable of running a restricted AQL query to perform data manipulation / filtering.
	ArangoSearchAnalyzerTypeAQL ArangoSearchAnalyzerType = "aql"
	// ArangoSearchAnalyzerTypePipeline an Analyzer capable of chaining effects of multiple Analyzers into one. The pipeline is a list of Analyzers, where the output of an Analyzer is passed to the next for further processing. The final token value is determined by last Analyzer in the pipeline.
	ArangoSearchAnalyzerTypePipeline ArangoSearchAnalyzerType = "pipeline"
	// ArangoSearchAnalyzerTypeStopwords an Analyzer capable of removing specified tokens from the input.
	ArangoSearchAnalyzerTypeStopwords ArangoSearchAnalyzerType = "stopwords"
	// ArangoSearchAnalyzerTypeGeoJSON an Analyzer capable of breaking up a GeoJSON object into a set of indexable tokens for further usage with ArangoSearch Geo functions.
	ArangoSearchAnalyzerTypeGeoJSON ArangoSearchAnalyzerType = "geojson"
	// ArangoSearchAnalyzerTypeGeoS2 an Analyzer capable of index GeoJSON data with inverted indexes or Views similar
	// to the existing `geojson` Analyzer, but it internally uses a format for storing the geo-spatial data.
	// that is more efficient.
	ArangoSearchAnalyzerTypeGeoS2 ArangoSearchAnalyzerType = "geo_s2"
	// ArangoSearchAnalyzerTypeGeoPoint an Analyzer capable of breaking up JSON object describing a coordinate into a set of indexable tokens for further usage with ArangoSearch Geo functions.
	ArangoSearchAnalyzerTypeGeoPoint ArangoSearchAnalyzerType = "geopoint"
	// ArangoSearchAnalyzerTypeSegmentation an Analyzer capable of breaking up the input text into tokens in a language-agnostic manner
	ArangoSearchAnalyzerTypeSegmentation ArangoSearchAnalyzerType = "segmentation"
	// ArangoSearchAnalyzerTypeCollation an Analyzer capable of converting the input into a set of language-specific tokens
	ArangoSearchAnalyzerTypeCollation ArangoSearchAnalyzerType = "collation"
	// ArangoSearchAnalyzerTypeClassification An Analyzer capable of classifying tokens in the input text. (EE only)
	ArangoSearchAnalyzerTypeClassification ArangoSearchAnalyzerType = "classification"
	// ArangoSearchAnalyzerTypeNearestNeighbors An Analyzer capable of finding nearest neighbors of tokens in the input. (EE only)
	ArangoSearchAnalyzerTypeNearestNeighbors ArangoSearchAnalyzerType = "nearest_neighbors"
	// ArangoSearchAnalyzerTypeMinhash an analyzer which is capable of evaluating so called MinHash signatures as a stream of tokens. (EE only)
	ArangoSearchAnalyzerTypeMinhash ArangoSearchAnalyzerType = "minhash"
)

// ArangoSearchAnalyzerFeature specifies a feature to an analyzer
type ArangoSearchAnalyzerFeature string

const (
	// ArangoSearchAnalyzerFeatureFrequency how often a term is seen, required for PHRASE()
	ArangoSearchAnalyzerFeatureFrequency ArangoSearchAnalyzerFeature = "frequency"
	// ArangoSearchAnalyzerFeatureNorm the field normalization factor
	ArangoSearchAnalyzerFeatureNorm ArangoSearchAnalyzerFeature = "norm"
	// ArangoSearchAnalyzerFeaturePosition sequentially increasing term position, required for PHRASE(). If present then the frequency feature is also required
	ArangoSearchAnalyzerFeaturePosition ArangoSearchAnalyzerFeature = "position"
	// ArangoSearchAnalyzerFeatureOffset can be specified if 'position' feature is set
	ArangoSearchAnalyzerFeatureOffset ArangoSearchAnalyzerFeature = "offset"
)

type ArangoSearchCaseType string

const (
	// ArangoSearchCaseUpper to convert to all lower-case characters
	ArangoSearchCaseUpper ArangoSearchCaseType = "upper"
	// ArangoSearchCaseLower to convert to all upper-case characters
	ArangoSearchCaseLower ArangoSearchCaseType = "lower"
	// ArangoSearchCaseNone to not change character case (default)
	ArangoSearchCaseNone ArangoSearchCaseType = "none"
)

type ArangoSearchBreakType string

const (
	// ArangoSearchBreakTypeAll to return all tokens
	ArangoSearchBreakTypeAll ArangoSearchBreakType = "all"
	// ArangoSearchBreakTypeAlpha to return tokens composed of alphanumeric characters only (default)
	ArangoSearchBreakTypeAlpha ArangoSearchBreakType = "alpha"
	// ArangoSearchBreakTypeGraphic to return tokens composed of non-whitespace characters only
	ArangoSearchBreakTypeGraphic ArangoSearchBreakType = "graphic"
)

type ArangoSearchNGramStreamType string

const (
	// ArangoSearchNGramStreamBinary used by NGram. Default value
	ArangoSearchNGramStreamBinary ArangoSearchNGramStreamType = "binary"
	// ArangoSearchNGramStreamUTF8 used by NGram
	ArangoSearchNGramStreamUTF8 ArangoSearchNGramStreamType = "utf8"
)

// ArangoSearchEdgeNGram specifies options for the edgeNGram text analyzer.
// More information can be found here: https://docs.arangodb.com/stable/index-and-search/analyzers/#text
type ArangoSearchEdgeNGram struct {
	// Min used by Text
	Min *int64 `json:"min,omitempty"`
	// Max used by Text
	Max *int64 `json:"max,omitempty"`
	// PreserveOriginal used by Text
	PreserveOriginal *bool `json:"preserveOriginal,omitempty"`
}

type ArangoSearchFormat string

const (
	// FormatLatLngDouble stores each latitude and longitude value as an 8-byte floating-point value (16 bytes per coordinate pair).
	// It is default value.
	FormatLatLngDouble ArangoSearchFormat = "latLngDouble"
	// FormatLatLngInt stores each latitude and longitude value as an 4-byte integer value (8 bytes per coordinate pair).
	// This is the most compact format but the precision is limited to approximately 1 to 10 centimeters.
	FormatLatLngInt ArangoSearchFormat = "latLngInt"
	// FormatS2Point store each longitude-latitude pair in the native format of Google S2 which is used for geo-spatial
	// calculations (24 bytes per coordinate pair).
	FormatS2Point ArangoSearchFormat = "s2Point"
)

func (a ArangoSearchFormat) New() *ArangoSearchFormat {
	return &a
}

// ArangoSearchAnalyzerProperties specifies options for the analyzer. Which fields are required and
// respected depends on the analyzer type.
// more information can be found here: https://docs.arangodb.com/stable/index-and-search/analyzers/#analyzer-properties
type ArangoSearchAnalyzerProperties struct {
	// Locale used by Stem, Norm, Text
	Locale string `json:"locale,omitempty"`
	// Delimiter used by Delimiter
	Delimiter string `json:"delimiter,omitempty"`
	// Accent used by Norm, Text
	Accent *bool `json:"accent,omitempty"`
	// Case used by Norm, Text, Segmentation
	Case ArangoSearchCaseType `json:"case,omitempty"`

	// EdgeNGram used by Text
	EdgeNGram *ArangoSearchEdgeNGram `json:"edgeNgram,omitempty"`

	// Min used by NGram
	Min *int64 `json:"min,omitempty"`
	// Max used by NGram
	Max *int64 `json:"max,omitempty"`
	// PreserveOriginal used by NGram
	PreserveOriginal *bool `json:"preserveOriginal,omitempty"`

	// StartMarker used by NGram
	StartMarker *string `json:"startMarker,omitempty"`
	// EndMarker used by NGram
	EndMarker *string `json:"endMarker,omitempty"`
	// StreamType used by NGram
	StreamType *ArangoSearchNGramStreamType `json:"streamType,omitempty"`

	// Stemming used by Text
	Stemming *bool `json:"stemming,omitempty"`
	// Stopword used by Text and Stopwords. This field is not mandatory since version 3.7 of arangod so it can not be omitted in 3.6.
	Stopwords []string `json:"stopwords"`
	// StopwordsPath used by Text
	StopwordsPath []string `json:"stopwordsPath,omitempty"`

	// QueryString used by AQL.
	QueryString string `json:"queryString,omitempty"`
	// CollapsePositions used by AQL.
	CollapsePositions *bool `json:"collapsePositions,omitempty"`
	// KeepNull used by AQL.
	KeepNull *bool `json:"keepNull,omitempty"`
	// BatchSize used by AQL.
	BatchSize *int `json:"batchSize,omitempty"`
	// MemoryLimit used by AQL.
	MemoryLimit *int `json:"memoryLimit,omitempty"`
	// ReturnType used by AQL.
	ReturnType *ArangoSearchAnalyzerAQLReturnType `json:"returnType,omitempty"`

	// Pipeline used by Pipeline.
	Pipeline []ArangoSearchAnalyzerPipeline `json:"pipeline,omitempty"`

	// Type used by GeoJSON.
	Type *ArangoSearchAnalyzerGeoJSONType `json:"type,omitempty"`

	// Options used by GeoJSON and GeoPoint
	Options *ArangoSearchAnalyzerGeoOptions `json:"options,omitempty"`

	// Latitude used by GetPoint.
	Latitude []string `json:"latitude,omitempty"`
	// Longitude used by GetPoint.
	Longitude []string `json:"longitude,omitempty"`

	// Break used by Segmentation
	Break ArangoSearchBreakType `json:"break,omitempty"`

	// Hex used by stopwords.
	// If false then each string in stopwords is used verbatim.
	// If true, then each string in stopwords needs to be hex-encoded.
	Hex *bool `json:"hex,omitempty"`

	// ModelLocation used by Classification, NearestNeighbors
	// The on-disk path to the trained fastText supervised model.
	// Note: if you are running this in an ArangoDB cluster, this model must exist on every machine in the cluster.
	ModelLocation string `json:"model_location,omitempty"`
	// TopK  used by Classification, NearestNeighbors
	// The number of class labels that will be produced per input (default: 1)
	TopK *uint64 `json:"top_k,omitempty"`
	// Threshold  used by Classification
	// The probability threshold for which a label will be assigned to an input.
	// A fastText model produces a probability per class label, and this is what will be filtered (default: 0.99).
	Threshold *float64 `json:"threshold,omitempty"`

	// Analyzer used by Minhash
	// Definition of inner analyzer to use for incoming data. In case if omitted field or empty object falls back to 'identity' analyzer.
	Analyzer *ArangoSearchAnalyzerDefinition `json:"analyzer,omitempty"`
	// NumHashes used by Minhash
	// Size of min hash signature. Must be greater or equal to 1.
	NumHashes *uint64 `json:"numHashes,omitempty"`

	// Format is the internal binary representation to use for storing the geo-spatial data in an index.
	Format *ArangoSearchFormat `json:"format,omitempty"`
}

// ArangoSearchAnalyzerGeoJSONType GeoJSON Type parameter.
type ArangoSearchAnalyzerGeoJSONType string

// New returns pointer to selected return type
func (a ArangoSearchAnalyzerGeoJSONType) New() *ArangoSearchAnalyzerGeoJSONType {
	return &a
}

const (
	// ArangoSearchAnalyzerGeoJSONTypeShape define index all GeoJSON geometry types (Point, Polygon etc.). (default)
	ArangoSearchAnalyzerGeoJSONTypeShape ArangoSearchAnalyzerGeoJSONType = "shape"
	// ArangoSearchAnalyzerGeoJSONTypeCentroid define compute and only index the centroid of the input geometry.
	ArangoSearchAnalyzerGeoJSONTypeCentroid ArangoSearchAnalyzerGeoJSONType = "centroid"
	// ArangoSearchAnalyzerGeoJSONTypePoint define only index GeoJSON objects of type Point, ignore all other geometry types.
	ArangoSearchAnalyzerGeoJSONTypePoint ArangoSearchAnalyzerGeoJSONType = "point"
)

// ArangoSearchAnalyzerGeoOptions for fine-tuning geo queries. These options should generally remain unchanged.
type ArangoSearchAnalyzerGeoOptions struct {
	// MaxCells define maximum number of S2 cells.
	MaxCells *int `json:"maxCells,omitempty"`
	// MinLevel define the least precise S2 level.
	MinLevel *int `json:"minLevel,omitempty"`
	// MaxLevel define the most precise S2 level
	MaxLevel *int `json:"maxLevel,omitempty"`
}

type ArangoSearchAnalyzerAQLReturnType string

const (
	ArangoSearchAnalyzerAQLReturnTypeString ArangoSearchAnalyzerAQLReturnType = "string"
	ArangoSearchAnalyzerAQLReturnTypeNumber ArangoSearchAnalyzerAQLReturnType = "number"
	ArangoSearchAnalyzerAQLReturnTypeBool   ArangoSearchAnalyzerAQLReturnType = "bool"
)

// New returns pointer to selected return type
func (a ArangoSearchAnalyzerAQLReturnType) New() *ArangoSearchAnalyzerAQLReturnType {
	return &a
}

// ArangoSearchAnalyzerPipeline provides object definition for Pipeline array parameter
type ArangoSearchAnalyzerPipeline struct {
	// Type of the Pipeline Analyzer
	Type ArangoSearchAnalyzerType `json:"type"`
	// Properties of the Pipeline Analyzer
	Properties ArangoSearchAnalyzerProperties `json:"properties,omitempty"`
}

// ArangoSearchAnalyzerDefinition provides definition of an analyzer
type ArangoSearchAnalyzerDefinition struct {
	Name       string                         `json:"name,omitempty"`
	Type       ArangoSearchAnalyzerType       `json:"type,omitempty"`
	Properties ArangoSearchAnalyzerProperties `json:"properties,omitempty"`
	Features   []ArangoSearchAnalyzerFeature  `json:"features,omitempty"`
	ArangoError
}

type ArangoSearchViewBase struct {
	Type ViewType `json:"type,omitempty"`
	Name string   `json:"name,omitempty"`
	ArangoID
	ArangoError
}

// ArangoSearchViewProperties contains properties on an ArangoSearch view.
type ArangoSearchViewProperties struct {
	// CleanupIntervalStep specifies the minimum number of commits to wait between
	// removing unused files in the data directory.
	// Defaults to 10.
	// Use 0 to disable waiting.
	// For the case where the consolidation policies merge segments often
	// (i.e. a lot of commit+consolidate), a lower value will cause a lot of
	// disk space to be wasted.
	// For the case where the consolidation policies rarely merge segments
	// (i.e. few inserts/deletes), a higher value will impact performance
	// without any added benefits.
	CleanupIntervalStep *int64 `json:"cleanupIntervalStep,omitempty"`
	// ConsolidationInterval specifies the minimum number of milliseconds that must be waited
	// between committing index data changes and making them visible to queries.
	// Defaults to 60000.
	// Use 0 to disable.
	// For the case where there are a lot of inserts/updates, a lower value,
	// until commit, will cause the index not to account for them and memory usage
	// would continue to grow.
	// For the case where there are a few inserts/updates, a higher value will
	// impact performance and waste disk space for each commit call without
	// any added benefits.
	ConsolidationInterval *int64 `json:"consolidationIntervalMsec,omitempty"`
	// ConsolidationPolicy specifies thresholds for consolidation.
	ConsolidationPolicy *ArangoSearchConsolidationPolicy `json:"consolidationPolicy,omitempty"`

	// CommitInterval ArangoSearch waits at least this many milliseconds between committing view data store changes and making documents visible to queries
	CommitInterval *int64 `json:"commitIntervalMsec,omitempty"`

	// WriteBufferIdle specifies the maximum number of writers (segments) cached in the pool.
	// 0 value turns off caching, default value is 64.
	WriteBufferIdel *int64 `json:"writebufferIdle,omitempty"`

	// WriteBufferActive specifies the maximum number of concurrent active writers (segments) performs (a transaction).
	// Other writers (segments) are wait till current active writers (segments) finish.
	// 0 value turns off this limit and used by default.
	WriteBufferActive *int64 `json:"writebufferActive,omitempty"`

	// WriteBufferSizeMax specifies maximum memory byte size per writer (segment) before a writer (segment) flush is triggered.
	// 0 value turns off this limit fon any writer (buffer) and will be flushed only after a period defined for special thread during ArangoDB server startup.
	// 0 value should be used with carefully due to high potential memory consumption.
	WriteBufferSizeMax *int64 `json:"writebufferSizeMax,omitempty"`

	// Links contains the properties for how individual collections
	// are indexed in the view.
	// The key of the map are collection names.
	Links ArangoSearchLinks `json:"links,omitempty"`

	// OptimizeTopK is an array of strings defining optimized sort expressions.
	// Introduced in v3.11.0, Enterprise Edition only.
	OptimizeTopK []string `json:"optimizeTopK,omitempty"`

	// PrimarySort describes how individual fields are sorted
	PrimarySort []ArangoSearchPrimarySortEntry `json:"primarySort,omitempty"`

	// PrimarySortCompression Defines how to compress the primary sort data (introduced in v3.7.1).
	// ArangoDB v3.5 and v3.6 always compress the index using LZ4. This option is immutable.
	PrimarySortCompression PrimarySortCompression `json:"primarySortCompression,omitempty"`

	// PrimarySortCache If you enable this option, then the primary sort columns are always cached in memory.
	// Can't be changed after creating View.
	// Introduced in v3.9.5, Enterprise Edition only
	PrimarySortCache *bool `json:"primarySortCache,omitempty"`

	// PrimaryKeyCache If you enable this option, then the primary key columns are always cached in memory.
	// Introduced in v3.9.6, Enterprise Edition only
	// Can't be changed after creating View.
	PrimaryKeyCache *bool `json:"primaryKeyCache,omitempty"`

	// StoredValues An array of objects to describe which document attributes to store in the View index (introduced in v3.7.1).
	// It can then cover search queries, which means the data can be taken from the index directly and accessing the storage engine can be avoided.
	// This option is immutable.
	StoredValues []StoredValue `json:"storedValues,omitempty"`

	ArangoSearchViewBase
}

// PrimarySortCompression Defines how to compress the primary sort data (introduced in v3.7.1)
type PrimarySortCompression string

const (
	// PrimarySortCompressionLz4 (default): use LZ4 fast compression.
	PrimarySortCompressionLz4 PrimarySortCompression = "lz4"
	// PrimarySortCompressionNone disable compression to trade space for speed.
	PrimarySortCompressionNone PrimarySortCompression = "none"
)

type StoredValue struct {
	Fields      []string               `json:"fields,omitempty"`
	Compression PrimarySortCompression `json:"compression,omitempty"`
	// Cache attribute allows you to always cache stored values in memory
	// Introduced in v3.9.5, Enterprise Edition only
	Cache *bool `json:"cache,omitempty"`
}

// ArangoSearchSortDirection describes the sorting direction
type ArangoSearchSortDirection string

const (
	// ArangoSearchSortDirectionAsc sort ascending
	ArangoSearchSortDirectionAsc ArangoSearchSortDirection = "ASC"
	// ArangoSearchSortDirectionDesc sort descending
	ArangoSearchSortDirectionDesc ArangoSearchSortDirection = "DESC"
)

// ArangoSearchPrimarySortEntry describes an entry for the primarySort list
type ArangoSearchPrimarySortEntry struct {
	Field     string `json:"field,omitempty"`
	Ascending *bool  `json:"asc,omitempty"`

	// Deprecated: please use Ascending instead
	Direction *ArangoSearchSortDirection `json:"direction,omitempty"`
}

// GetDirection returns the sort direction or empty string if not set
func (pse ArangoSearchPrimarySortEntry) GetDirection() ArangoSearchSortDirection {
	if pse.Direction != nil {
		return *pse.Direction
	}

	return ArangoSearchSortDirection("")
}

// GetAscending returns the value of Ascending or false if not set
func (pse ArangoSearchPrimarySortEntry) GetAscending() bool {
	if pse.Ascending != nil {
		return *pse.Ascending
	}

	return false
}

// ArangoSearchConsolidationPolicyType strings for consolidation types
type ArangoSearchConsolidationPolicyType string

const (
	// ArangoSearchConsolidationPolicyTypeTier consolidate based on segment byte size and live document count as dictated by the customization attributes.
	ArangoSearchConsolidationPolicyTypeTier ArangoSearchConsolidationPolicyType = "tier"
	// ArangoSearchConsolidationPolicyTypeBytesAccum consolidate if and only if ({threshold} range [0.0, 1.0])
	// {threshold} > (segment_bytes + sum_of_merge_candidate_segment_bytes) / all_segment_bytes,
	// i.e. the sum of all candidate segment's byte size is less than the total segment byte size multiplied by the {threshold}.
	ArangoSearchConsolidationPolicyTypeBytesAccum ArangoSearchConsolidationPolicyType = "bytes_accum"
)

// ArangoSearchConsolidationPolicy holds threshold values specifying when to
// consolidate view data.
// Semantics of the values depend on where they are used.
type ArangoSearchConsolidationPolicy struct {
	// Type returns the type of the ConsolidationPolicy. This interface can then be casted to the corresponding ArangoSearchConsolidationPolicy* struct.
	Type ArangoSearchConsolidationPolicyType `json:"type,omitempty"`

	ArangoSearchConsolidationPolicyBytesAccum
	ArangoSearchConsolidationPolicyTier
}

// ArangoSearchConsolidationPolicyBytesAccum contains fields used for ArangoSearchConsolidationPolicyTypeBytesAccum
type ArangoSearchConsolidationPolicyBytesAccum struct {
	// Threshold, see ArangoSearchConsolidationTypeBytesAccum
	Threshold *float64 `json:"threshold,omitempty"`
}

// ArangoSearchConsolidationPolicyTier contains fields used for ArangoSearchConsolidationPolicyTypeTier
type ArangoSearchConsolidationPolicyTier struct {
	MinScore *int64 `json:"minScore,omitempty"`
	// MinSegments specifies the minimum number of segments that will be evaluated as candidates for consolidation.
	MinSegments *int64 `json:"segmentsMin,omitempty"`
	// MaxSegments specifies the maximum number of segments that will be evaluated as candidates for consolidation.
	MaxSegments *int64 `json:"segmentsMax,omitempty"`
	// SegmentsBytesMax specifies the maxinum allowed size of all consolidated segments in bytes.
	SegmentsBytesMax *int64 `json:"segmentsBytesMax,omitempty"`
	// SegmentsBytesFloor defines the value (in bytes) to treat all smaller segments as equal for consolidation selection.
	SegmentsBytesFloor *int64 `json:"segmentsBytesFloor,omitempty"`
	// Lookahead specifies the number of additionally searched tiers except initially chosen candidated based on min_segments,
	// max_segments, segments_bytes_max, segments_bytes_floor with respect to defined values.
	// Default value falls to integer_traits<size_t>::const_max (in C++ source code).
	Lookahead *int64 `json:"lookahead,omitempty"`
}

// ArangoSearchLinks is a strongly typed map containing links between a
// collection and a view.
// The keys in the map are collection names.
type ArangoSearchLinks map[string]ArangoSearchElementProperties

// ArangoSearchFields is a strongly typed map containing properties per field.
// The keys in the map are field names.
type ArangoSearchFields map[string]ArangoSearchElementProperties

// ArangoSearchElementProperties contains properties that specify how an element
// is indexed in an ArangoSearch view.
// Note that this structure is recursive. Settings not specified (nil)
// at a given level will inherit their setting from a lower level.
type ArangoSearchElementProperties struct {
	AnalyzerDefinitions []ArangoSearchAnalyzerDefinition `json:"analyzerDefinitions,omitempty"`
	// The list of analyzers to be used for indexing of string values. Defaults to ["identify"].
	Analyzers []string `json:"analyzers,omitempty"`
	// If set to true, all fields of this element will be indexed. Defaults to false.
	IncludeAllFields *bool `json:"includeAllFields,omitempty"`
	// If set to true, values in a listed are treated as separate values. Defaults to false.
	TrackListPositions *bool `json:"trackListPositions,omitempty"`
	// This values specifies how the view should track values.
	StoreValues ArangoSearchStoreValues `json:"storeValues,omitempty"`
	// Fields contains the properties for individual fields of the element.
	// The key of the map are field names.
	Fields ArangoSearchFields `json:"fields,omitempty"`
	// If set to true, then no exclusive lock is used on the source collection during View index creation,
	// so that it remains basically available. inBackground is an option that can be set when adding links.
	// It does not get persisted as it is not a View property, but only a one-off option
	InBackground *bool `json:"inBackground,omitempty"`
	// Nested contains the properties for nested fields (sub-objects) of the element
	// Enterprise Edition only
	Nested ArangoSearchFields `json:"nested,omitempty"`
	// Cache If you enable this option, then field normalization values are always cached in memory.
	// Introduced in v3.9.5, Enterprise Edition only
	Cache *bool `json:"cache,omitempty"`
}

// ArangoSearchStoreValues is the type of the StoreValues option of an ArangoSearch element.
type ArangoSearchStoreValues string

const (
	// ArangoSearchStoreValuesNone specifies that a view should not store values.
	ArangoSearchStoreValuesNone ArangoSearchStoreValues = "none"
	// ArangoSearchStoreValuesID specifies that a view should only store
	// information about value presence, to allow use of the EXISTS() function.
	ArangoSearchStoreValuesID ArangoSearchStoreValues = "id"
)
