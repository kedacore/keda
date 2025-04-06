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

// Index opens a connection to an existing index within the collection.
// If no index with given name exists, an NotFoundError is returned.
func (c *vertexCollection) Index(ctx context.Context, name string) (Index, error) {
	result, err := c.rawCollection().Index(ctx, name)
	if err != nil {
		return nil, WithStack(err)
	}
	return result, nil
}

// IndexExists returns true if an index with given name exists within the collection.
func (c *vertexCollection) IndexExists(ctx context.Context, name string) (bool, error) {
	result, err := c.rawCollection().IndexExists(ctx, name)
	if err != nil {
		return false, WithStack(err)
	}
	return result, nil
}

// Indexes returns a list of all indexes in the collection.
func (c *vertexCollection) Indexes(ctx context.Context) ([]Index, error) {
	result, err := c.rawCollection().Indexes(ctx)
	if err != nil {
		return nil, WithStack(err)
	}
	return result, nil
}

// Deprecated: since 3.10 version. Use ArangoSearch view instead.
//
// EnsureFullTextIndex creates a fulltext index in the collection, if it does not already exist.
//
// Fields is a slice of attribute names. Currently, the slice is limited to exactly one attribute.
// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
func (c *vertexCollection) EnsureFullTextIndex(ctx context.Context, fields []string, options *EnsureFullTextIndexOptions) (Index, bool, error) {
	result, created, err := c.rawCollection().EnsureFullTextIndex(ctx, fields, options)
	if err != nil {
		return nil, false, WithStack(err)
	}
	return result, created, nil
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
func (c *vertexCollection) EnsureGeoIndex(ctx context.Context, fields []string, options *EnsureGeoIndexOptions) (Index, bool, error) {
	result, created, err := c.rawCollection().EnsureGeoIndex(ctx, fields, options)
	if err != nil {
		return nil, false, WithStack(err)
	}
	return result, created, nil
}

// EnsureHashIndex creates a hash index in the collection, if it does not already exist.
// Fields is a slice of attribute paths.
// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
func (c *vertexCollection) EnsureHashIndex(ctx context.Context, fields []string, options *EnsureHashIndexOptions) (Index, bool, error) {
	result, created, err := c.rawCollection().EnsureHashIndex(ctx, fields, options)
	if err != nil {
		return nil, false, WithStack(err)
	}
	return result, created, nil
}

// EnsurePersistentIndex creates a persistent index in the collection, if it does not already exist.
// Fields is a slice of attribute paths.
// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
func (c *vertexCollection) EnsurePersistentIndex(ctx context.Context, fields []string, options *EnsurePersistentIndexOptions) (Index, bool, error) {
	result, created, err := c.rawCollection().EnsurePersistentIndex(ctx, fields, options)
	if err != nil {
		return nil, false, WithStack(err)
	}
	return result, created, nil
}

// EnsureSkipListIndex creates a skiplist index in the collection, if it does not already exist.
// Fields is a slice of attribute paths.
// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
func (c *vertexCollection) EnsureSkipListIndex(ctx context.Context, fields []string, options *EnsureSkipListIndexOptions) (Index, bool, error) {
	result, created, err := c.rawCollection().EnsureSkipListIndex(ctx, fields, options)
	if err != nil {
		return nil, false, WithStack(err)
	}
	return result, created, nil
}

// EnsureTTLIndex creates a TLL collection, if it does not already exist.
// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
func (c *vertexCollection) EnsureTTLIndex(ctx context.Context, field string, expireAfter int, options *EnsureTTLIndexOptions) (Index, bool, error) {
	result, created, err := c.rawCollection().EnsureTTLIndex(ctx, field, expireAfter, options)
	if err != nil {
		return nil, false, WithStack(err)
	}
	return result, created, nil
}

// EnsureZKDIndex creates a ZKD index in the collection, if it does not already exist.
// Fields is a slice of attribute paths.
// The index is returned, together with a boolean indicating if the index was newly created (true) or pre-existing (false).
func (c *vertexCollection) EnsureZKDIndex(ctx context.Context, fields []string, options *EnsureZKDIndexOptions) (Index, bool, error) {
	result, created, err := c.rawCollection().EnsureZKDIndex(ctx, fields, options)
	if err != nil {
		return nil, false, WithStack(err)
	}
	return result, created, nil
}

func (c *vertexCollection) EnsureMDIIndex(ctx context.Context, fields []string, options *EnsureMDIIndexOptions) (Index, bool, error) {
	result, created, err := c.rawCollection().EnsureMDIIndex(ctx, fields, options)
	if err != nil {
		return nil, false, WithStack(err)
	}
	return result, created, nil
}

func (c *vertexCollection) EnsureMDIPrefixedIndex(ctx context.Context, fields []string, options *EnsureMDIPrefixedIndexOptions) (Index, bool, error) {
	result, created, err := c.rawCollection().EnsureMDIPrefixedIndex(ctx, fields, options)
	if err != nil {
		return nil, false, WithStack(err)
	}
	return result, created, nil
}

// EnsureInvertedIndex creates an inverted index in the collection, if it does not already exist.
// Available in ArangoDB 3.10 and later.
func (c *vertexCollection) EnsureInvertedIndex(ctx context.Context, options *InvertedIndexOptions) (Index, bool, error) {
	result, created, err := c.rawCollection().EnsureInvertedIndex(ctx, options)
	if err != nil {
		return nil, false, WithStack(err)
	}
	return result, created, nil
}
