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

import (
	"context"
	"encoding/json"
	"path"
)

type indexData struct {
	ID                  string   `json:"id,omitempty"`
	Type                string   `json:"type"`
	Fields              []string `json:"fields,omitempty"`
	Unique              *bool    `json:"unique,omitempty"`
	Deduplicate         *bool    `json:"deduplicate,omitempty"`
	Sparse              *bool    `json:"sparse,omitempty"`
	GeoJSON             *bool    `json:"geoJson,omitempty"`
	InBackground        *bool    `json:"inBackground,omitempty"`
	Estimates           *bool    `json:"estimates,omitempty"`
	MaxNumCoverCells    int      `json:"maxNumCoverCells,omitempty"`
	MinLength           int      `json:"minLength,omitempty"`
	ExpireAfter         int      `json:"expireAfter"`
	Name                string   `json:"name,omitempty"`
	FieldValueTypes     string   `json:"fieldValueTypes,omitempty"`
	IsNewlyCreated      *bool    `json:"isNewlyCreated,omitempty"`
	SelectivityEstimate float64  `json:"selectivityEstimate,omitempty"`
	BestIndexedLevel    int      `json:"bestIndexedLevel,omitempty"`
	WorstIndexedLevel   int      `json:"worstIndexedLevel,omitempty"`
	LegacyPolygons      *bool    `json:"legacyPolygons,omitempty"`
	CacheEnabled        *bool    `json:"cacheEnabled,omitempty"`
	StoredValues        []string `json:"storedValues,omitempty"`
	PrefixFields        []string `json:"prefixFields,omitempty"`

	ArangoError `json:",inline"`
}

type indexListResponse struct {
	Indexes []json.RawMessage `json:"indexes,omitempty"`
	ArangoError
}

// Index opens a connection to an existing index within the collection.
// If no index with given name exists, an NotFoundError is returned.
func (c *collection) Index(ctx context.Context, name string) (Index, error) {
	req, err := c.conn.NewRequest("GET", path.Join(c.relPath("index"), name))
	if err != nil {
		return nil, WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return nil, WithStack(err)
	}
	var data map[string]interface{}
	if err := resp.ParseBody("", &data); err != nil {
		return nil, WithStack(err)
	}

	rawResponse, err := json.Marshal(data)
	if err != nil {
		return nil, WithStack(err)
	}

	idx, err := newIndexFromMap(rawResponse, c)
	if err != nil {
		return nil, WithStack(err)
	}
	return idx, nil
}

// IndexExists returns true if an index with given name exists within the collection.
func (c *collection) IndexExists(ctx context.Context, name string) (bool, error) {
	req, err := c.conn.NewRequest("GET", path.Join(c.relPath("index"), name))
	if err != nil {
		return false, WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
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

// Indexes returns a list of all indexes in the collection.
func (c *collection) Indexes(ctx context.Context) ([]Index, error) {
	req, err := c.conn.NewRequest("GET", path.Join(c.db.relPath(), "_api", "index"))
	if err != nil {
		return nil, WithStack(err)
	}
	req.SetQuery("collection", c.name)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return nil, WithStack(err)
	}
	var data indexListResponse
	if err := resp.ParseBody("", &data); err != nil {
		return nil, WithStack(err)
	}
	result := make([]Index, 0, len(data.Indexes))
	for _, x := range data.Indexes {
		idx, err := newIndexFromMap(x, c)
		if err != nil {
			return nil, WithStack(err)
		}
		result = append(result, idx)
	}
	return result, nil
}

// Deprecated: since 3.10 version. Use ArangoSearch view instead.
//
// EnsureFullTextIndex creates a fulltext index in the collection, if it does not already exist.
//
// Fields is a slice of attribute names. Currently, the slice is limited to exactly one attribute.
// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
func (c *collection) EnsureFullTextIndex(ctx context.Context, fields []string, options *EnsureFullTextIndexOptions) (Index, bool, error) {
	input := indexData{
		Type:   string(FullTextIndex),
		Fields: fields,
	}
	if options != nil {
		input.InBackground = &options.InBackground
		input.Name = options.Name
		input.MinLength = options.MinLength
		input.Estimates = options.Estimates
	}
	idx, created, err := c.ensureIndex(ctx, input)
	if err != nil {
		return nil, false, WithStack(err)
	}
	return idx, created, nil
}

// EnsureGeoIndex creates a hash index in the collection, if it does not already exist.
//
// Fields is a slice with one or two attribute paths. If it is a slice with one attribute path location,
// then a geo-spatial index on all documents is created using location as path to the coordinates.
// The value of the attribute must be a slice with at least two double values. The slice must contain the latitude (first value)
// and the longitude (second value). All documents, which do not have the attribute path or with value that are not suitable, are ignored.
// If it is a slice with two attribute paths latitude and longitude, then a geo-spatial index on all documents is created
// using latitude and longitude as paths the latitude and the longitude. The value of the attribute latitude and of the
// attribute longitude must a double. All documents, which do not have the attribute paths or which values are not suitable, are ignored.
// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
func (c *collection) EnsureGeoIndex(ctx context.Context, fields []string, options *EnsureGeoIndexOptions) (Index, bool, error) {
	input := indexData{
		Type:   string(GeoIndex),
		Fields: fields,
	}
	if options != nil {
		input.InBackground = &options.InBackground
		input.Name = options.Name
		input.GeoJSON = &options.GeoJSON
		input.Estimates = options.Estimates
		if options.LegacyPolygons {
			input.LegacyPolygons = &options.LegacyPolygons
		}
	}
	idx, created, err := c.ensureIndex(ctx, input)
	if err != nil {
		return nil, false, WithStack(err)
	}
	return idx, created, nil
}

// EnsureHashIndex creates a hash index in the collection, if it does not already exist.
// Fields is a slice of attribute paths.
// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
func (c *collection) EnsureHashIndex(ctx context.Context, fields []string, options *EnsureHashIndexOptions) (Index, bool, error) {
	input := indexData{
		Type:   string(HashIndex),
		Fields: fields,
	}
	off := false
	if options != nil {
		input.InBackground = &options.InBackground
		input.Name = options.Name
		input.Unique = &options.Unique
		input.Sparse = &options.Sparse
		input.Estimates = options.Estimates
		if options.NoDeduplicate {
			input.Deduplicate = &off
		}
	}
	idx, created, err := c.ensureIndex(ctx, input)
	if err != nil {
		return nil, false, WithStack(err)
	}
	return idx, created, nil
}

// EnsurePersistentIndex creates a persistent index in the collection, if it does not already exist.
// Fields is a slice of attribute paths.
// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
func (c *collection) EnsurePersistentIndex(ctx context.Context, fields []string, options *EnsurePersistentIndexOptions) (Index, bool, error) {
	input := indexData{
		Type:   string(PersistentIndex),
		Fields: fields,
	}
	off := false
	on := true
	if options != nil {
		input.InBackground = &options.InBackground
		input.Name = options.Name
		input.Unique = &options.Unique
		input.Sparse = &options.Sparse
		input.Estimates = options.Estimates
		if options.NoDeduplicate {
			input.Deduplicate = &off
		}
		if options.CacheEnabled {
			input.CacheEnabled = &on
		}
		if options.StoredValues != nil {
			input.StoredValues = options.StoredValues
		}
	}
	idx, created, err := c.ensureIndex(ctx, input)
	if err != nil {
		return nil, false, WithStack(err)
	}
	return idx, created, nil
}

// EnsureSkipListIndex creates a skiplist index in the collection, if it does not already exist.
// Fields is a slice of attribute paths.
// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
func (c *collection) EnsureSkipListIndex(ctx context.Context, fields []string, options *EnsureSkipListIndexOptions) (Index, bool, error) {
	input := indexData{
		Type:   string(SkipListIndex),
		Fields: fields,
	}
	off := false
	if options != nil {
		input.InBackground = &options.InBackground
		input.Name = options.Name
		input.Unique = &options.Unique
		input.Sparse = &options.Sparse
		input.Estimates = options.Estimates
		if options.NoDeduplicate {
			input.Deduplicate = &off
		}
	}
	idx, created, err := c.ensureIndex(ctx, input)
	if err != nil {
		return nil, false, WithStack(err)
	}
	return idx, created, nil
}

// EnsureTTLIndex creates a TLL collection, if it does not already exist.
// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
func (c *collection) EnsureTTLIndex(ctx context.Context, field string, expireAfter int, options *EnsureTTLIndexOptions) (Index, bool, error) {
	input := indexData{
		Type:        string(TTLIndex),
		Fields:      []string{field},
		ExpireAfter: expireAfter,
	}
	if options != nil {
		input.InBackground = &options.InBackground
		input.Name = options.Name
		input.Estimates = options.Estimates
	}
	idx, created, err := c.ensureIndex(ctx, input)
	if err != nil {
		return nil, false, WithStack(err)
	}
	return idx, created, nil
}

// EnsureZKDIndex creates a ZKD index in the collection, if it does not already exist.
// Fields is a slice of attribute paths.
// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
func (c *collection) EnsureZKDIndex(ctx context.Context, fields []string, options *EnsureZKDIndexOptions) (Index, bool, error) {
	input := indexData{
		Type:   string(ZKDIndex),
		Fields: fields,
		// fieldValueTypes is required and the only allowed value is "double". Future extensions of the index will allow other types.
		FieldValueTypes: "double",
	}
	if options != nil {
		input.InBackground = &options.InBackground
		input.Name = options.Name
		input.Unique = &options.Unique
		//input.Sparse = &options.Sparse
	}
	idx, created, err := c.ensureIndex(ctx, input)
	if err != nil {
		return nil, false, WithStack(err)
	}
	return idx, created, nil
}

func (c *collection) EnsureMDIIndex(ctx context.Context, fields []string, options *EnsureMDIIndexOptions) (Index, bool, error) {
	input := indexData{
		Type:   string(MDIIndex),
		Fields: fields,
		// fieldValueTypes is required and the only allowed value is "double". Future extensions of the index will allow other types.
		FieldValueTypes: "double",
	}
	if options != nil {
		input.InBackground = &options.InBackground
		input.Name = options.Name
		input.Unique = &options.Unique
		input.Sparse = &options.Sparse
		input.StoredValues = options.StoredValues
	}
	idx, created, err := c.ensureIndex(ctx, input)
	if err != nil {
		return nil, false, WithStack(err)
	}
	return idx, created, nil
}

func (c *collection) EnsureMDIPrefixedIndex(ctx context.Context, fields []string, options *EnsureMDIPrefixedIndexOptions) (Index, bool, error) {
	input := indexData{
		Type:   string(MDIPrefixedIndex),
		Fields: fields,
		// fieldValueTypes is required and the only allowed value is "double". Future extensions of the index will allow other types.
		FieldValueTypes: "double",
	}
	if options != nil {
		input.InBackground = &options.InBackground
		input.Name = options.Name
		input.Unique = &options.Unique
		input.Sparse = &options.Sparse
		input.StoredValues = options.StoredValues
		input.PrefixFields = options.PrefixFields
	}
	idx, created, err := c.ensureIndex(ctx, input)
	if err != nil {
		return nil, false, WithStack(err)
	}
	return idx, created, nil
}

type invertedIndexData struct {
	InvertedIndexOptions
	Type string `json:"type"`
	ID   string `json:"id,omitempty"`

	ArangoError `json:",inline"`
}

// EnsureInvertedIndex creates an inverted index in the collection, if it does not already exist.
// Available in ArangoDB 3.10 and later.
func (c *collection) EnsureInvertedIndex(ctx context.Context, options *InvertedIndexOptions) (Index, bool, error) {
	req, err := c.conn.NewRequest("POST", path.Join(c.db.relPath(), "_api/index"))
	if err != nil {
		return nil, false, WithStack(err)
	}
	if options == nil {
		options = &InvertedIndexOptions{}
	}
	req.SetQuery("collection", c.name)
	if _, err := req.SetBody(invertedIndexData{InvertedIndexOptions: *options, Type: string(InvertedIndex)}); err != nil {
		return nil, false, WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return nil, false, WithStack(err)
	}
	if err := resp.CheckStatus(200, 201); err != nil {
		return nil, false, WithStack(err)
	}
	created := resp.StatusCode() == 201

	var data invertedIndexData
	if err := resp.ParseBody("", &data); err != nil {
		return nil, false, WithStack(err)
	}
	idx, err := newInvertedIndex(data, c)
	if err != nil {
		return nil, false, WithStack(err)
	}
	return idx, created, nil
}

// ensureIndex creates a persistent index in the collection, if it does not already exist.
// Fields is a slice of attribute paths.
// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
func (c *collection) ensureIndex(ctx context.Context, options indexData) (Index, bool, error) {
	req, err := c.conn.NewRequest("POST", path.Join(c.db.relPath(), "_api/index"))
	if err != nil {
		return nil, false, WithStack(err)
	}
	req.SetQuery("collection", c.name)
	if _, err := req.SetBody(options); err != nil {
		return nil, false, WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return nil, false, WithStack(err)
	}
	if err := resp.CheckStatus(200, 201); err != nil {
		return nil, false, WithStack(err)
	}
	created := resp.StatusCode() == 201
	var data indexData
	if err := resp.ParseBody("", &data); err != nil {
		return nil, false, WithStack(err)
	}
	idx, err := newIndex(data, c)
	if err != nil {
		return nil, false, WithStack(err)
	}
	return idx, created, nil
}
