//
// DISCLAIMER
//
// Copyright 2017 ArangoDB GmbH, Cologne, Germany
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
	"fmt"
	"path"
	"reflect"
)

// DocumentExists checks if a document with given key exists in the collection.
func (c *vertexCollection) DocumentExists(ctx context.Context, key string) (bool, error) {
	if result, err := c.rawCollection().DocumentExists(ctx, key); err != nil {
		return false, WithStack(err)
	} else {
		return result, nil
	}
}

// ReadDocument reads a single document with given key from the collection.
// The document data is stored into result, the document meta data is returned.
// If no document exists with given key, a NotFoundError is returned.
func (c *vertexCollection) ReadDocument(ctx context.Context, key string, result interface{}) (DocumentMeta, error) {
	meta, _, err := c.readDocument(ctx, key, result)
	if err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	return meta, nil
}

func (c *vertexCollection) readDocument(ctx context.Context, key string, result interface{}) (DocumentMeta, contextSettings, error) {
	if err := validateKey(key); err != nil {
		return DocumentMeta{}, contextSettings{}, WithStack(err)
	}
	escapedKey := pathEscape(key)
	req, err := c.conn.NewRequest("GET", path.Join(c.relPath(), escapedKey))
	if err != nil {
		return DocumentMeta{}, contextSettings{}, WithStack(err)
	}
	cs := applyContextSettings(ctx, req)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return DocumentMeta{}, contextSettings{}, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return DocumentMeta{}, contextSettings{}, WithStack(err)
	}
	// Concerns: ReadDocuments reads multiple documents via multiple calls to readDocument (this function).
	// Currently with AllowDirtyReads the wasDirtyFlag is only set according to the last read request.
	loadContextResponseValues(cs, resp)
	// Parse metadata
	var meta DocumentMeta
	if err := resp.ParseBody("vertex", &meta); err != nil {
		return DocumentMeta{}, contextSettings{}, WithStack(err)
	}
	// Parse result
	if result != nil {
		if err := resp.ParseBody("vertex", result); err != nil {
			return meta, contextSettings{}, WithStack(err)
		}
	}
	return meta, cs, nil
}

// ReadDocuments reads multiple documents with given keys from the collection.
// The documents data is stored into elements of the given results slice,
// the documents meta data is returned.
// If no document exists with a given key, a NotFoundError is returned at its errors index.
func (c *vertexCollection) ReadDocuments(ctx context.Context, keys []string, results interface{}) (DocumentMetaSlice, ErrorSlice, error) {
	resultsVal := reflect.ValueOf(results)
	switch resultsVal.Kind() {
	case reflect.Array, reflect.Slice:
		// OK
	default:
		return nil, nil, WithStack(InvalidArgumentError{Message: fmt.Sprintf("results data must be of kind Array, got %s", resultsVal.Kind())})
	}
	if keys == nil {
		return nil, nil, WithStack(InvalidArgumentError{Message: "keys nil"})
	}
	resultCount := resultsVal.Len()
	if len(keys) != resultCount {
		return nil, nil, WithStack(InvalidArgumentError{Message: fmt.Sprintf("expected %d keys, got %d", resultCount, len(keys))})
	}
	for _, key := range keys {
		if err := validateKey(key); err != nil {
			return nil, nil, WithStack(err)
		}
	}
	metas := make(DocumentMetaSlice, resultCount)
	errs := make(ErrorSlice, resultCount)
	silent := false
	for i := 0; i < resultCount; i++ {
		result := resultsVal.Index(i).Addr()
		ctx, err := withDocumentAt(ctx, i)
		if err != nil {
			return nil, nil, WithStack(err)
		}
		key := keys[i]
		meta, cs, err := c.readDocument(ctx, key, result.Interface())
		if cs.Silent {
			silent = true
		} else {
			metas[i], errs[i] = meta, err
		}
	}
	if silent {
		return nil, nil, nil
	}
	return metas, errs, nil
}

// CreateDocument creates a single document in the collection.
// The document data is loaded from the given document, the document meta data is returned.
// If the document data already contains a `_key` field, this will be used as key of the new document,
// otherwise a unique key is created.
// A ConflictError is returned when a `_key` field contains a duplicate key, other any other field violates an index constraint.
// To return the NEW document, prepare a context with `WithReturnNew`.
// To wait until document has been synced to disk, prepare a context with `WithWaitForSync`.
func (c *vertexCollection) CreateDocument(ctx context.Context, document interface{}) (DocumentMeta, error) {
	meta, _, err := c.createDocument(ctx, document)
	if err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	return meta, nil
}

func (c *vertexCollection) createDocument(ctx context.Context, document interface{}) (DocumentMeta, contextSettings, error) {
	if document == nil {
		return DocumentMeta{}, contextSettings{}, WithStack(InvalidArgumentError{Message: "document nil"})
	}
	req, err := c.conn.NewRequest("POST", c.relPath())
	if err != nil {
		return DocumentMeta{}, contextSettings{}, WithStack(err)
	}
	if _, err := req.SetBody(document); err != nil {
		return DocumentMeta{}, contextSettings{}, WithStack(err)
	}
	cs := applyContextSettings(ctx, req)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return DocumentMeta{}, cs, WithStack(err)
	}
	if err := resp.CheckStatus(201, 202); err != nil {
		return DocumentMeta{}, cs, WithStack(err)
	}
	if cs.Silent {
		// Empty response, we're done
		return DocumentMeta{}, cs, nil
	}
	// Parse metadata
	var meta DocumentMeta
	if err := resp.ParseBody("vertex", &meta); err != nil {
		return DocumentMeta{}, cs, WithStack(err)
	}
	// Parse returnNew (if needed)
	if cs.ReturnNew != nil {
		if err := resp.ParseBody("new", cs.ReturnNew); err != nil {
			return meta, cs, WithStack(err)
		}
	}
	return meta, cs, nil
}

// CreateDocuments creates multiple documents in the collection.
// The document data is loaded from the given documents slice, the documents meta data is returned.
// If a documents element already contains a `_key` field, this will be used as key of the new document,
// otherwise a unique key is created.
// If a documents element contains a `_key` field with a duplicate key, other any other field violates an index constraint,
// a ConflictError is returned in its inded in the errors slice.
// To return the NEW documents, prepare a context with `WithReturnNew`. The data argument passed to `WithReturnNew` must be
// a slice with the same number of entries as the `documents` slice.
// To wait until document has been synced to disk, prepare a context with `WithWaitForSync`.
// If the create request itself fails or one of the arguments is invalid, an error is returned.
func (c *vertexCollection) CreateDocuments(ctx context.Context, documents interface{}) (DocumentMetaSlice, ErrorSlice, error) {
	documentsVal := reflect.ValueOf(documents)
	switch documentsVal.Kind() {
	case reflect.Array, reflect.Slice:
		// OK
	default:
		return nil, nil, WithStack(InvalidArgumentError{Message: fmt.Sprintf("documents data must be of kind Array, got %s", documentsVal.Kind())})
	}
	documentCount := documentsVal.Len()
	metas := make(DocumentMetaSlice, documentCount)
	errs := make(ErrorSlice, documentCount)
	silent := false
	for i := 0; i < documentCount; i++ {
		doc := documentsVal.Index(i)
		ctx, err := withDocumentAt(ctx, i)
		if err != nil {
			return nil, nil, WithStack(err)
		}
		meta, cs, err := c.createDocument(ctx, doc.Interface())
		if cs.Silent {
			silent = true
		} else {
			metas[i], errs[i] = meta, err
		}
	}
	if silent {
		return nil, nil, nil
	}
	return metas, errs, nil
}

// UpdateDocument updates a single document with given key in the collection.
// The document meta data is returned.
// To return the NEW document, prepare a context with `WithReturnNew`.
// To return the OLD document, prepare a context with `WithReturnOld`.
// To wait until document has been synced to disk, prepare a context with `WithWaitForSync`.
// If no document exists with given key, a NotFoundError is returned.
func (c *vertexCollection) UpdateDocument(ctx context.Context, key string, update interface{}) (DocumentMeta, error) {
	meta, _, err := c.updateDocument(ctx, key, update)
	if err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	return meta, nil
}

func (c *vertexCollection) updateDocument(ctx context.Context, key string, update interface{}) (DocumentMeta, contextSettings, error) {
	if err := validateKey(key); err != nil {
		return DocumentMeta{}, contextSettings{}, WithStack(err)
	}
	if update == nil {
		return DocumentMeta{}, contextSettings{}, WithStack(InvalidArgumentError{Message: "update nil"})
	}
	escapedKey := pathEscape(key)
	req, err := c.conn.NewRequest("PATCH", path.Join(c.relPath(), escapedKey))
	if err != nil {
		return DocumentMeta{}, contextSettings{}, WithStack(err)
	}
	if _, err := req.SetBody(update); err != nil {
		return DocumentMeta{}, contextSettings{}, WithStack(err)
	}
	cs := applyContextSettings(ctx, req)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return DocumentMeta{}, cs, WithStack(err)
	}
	if err := resp.CheckStatus(200, 201, 202); err != nil {
		return DocumentMeta{}, cs, WithStack(err)
	}
	if cs.Silent {
		// Empty response, we're done
		return DocumentMeta{}, cs, nil
	}
	// Parse metadata
	var meta DocumentMeta
	if err := resp.ParseBody("vertex", &meta); err != nil {
		return DocumentMeta{}, cs, WithStack(err)
	}
	// Parse returnOld (if needed)
	if cs.ReturnOld != nil {
		if err := resp.ParseBody("old", cs.ReturnOld); err != nil {
			return meta, cs, WithStack(err)
		}
	}
	// Parse returnNew (if needed)
	if cs.ReturnNew != nil {
		if err := resp.ParseBody("new", cs.ReturnNew); err != nil {
			return meta, cs, WithStack(err)
		}
	}
	return meta, cs, nil
}

// UpdateDocuments updates multiple document with given keys in the collection.
// The updates are loaded from the given updates slice, the documents meta data are returned.
// To return the NEW documents, prepare a context with `WithReturnNew` with a slice of documents.
// To return the OLD documents, prepare a context with `WithReturnOld` with a slice of documents.
// To wait until documents has been synced to disk, prepare a context with `WithWaitForSync`.
// If no document exists with a given key, a NotFoundError is returned at its errors index.
func (c *vertexCollection) UpdateDocuments(ctx context.Context, keys []string, updates interface{}) (DocumentMetaSlice, ErrorSlice, error) {
	updatesVal := reflect.ValueOf(updates)
	switch updatesVal.Kind() {
	case reflect.Array, reflect.Slice:
		// OK
	default:
		return nil, nil, WithStack(InvalidArgumentError{Message: fmt.Sprintf("updates data must be of kind Array, got %s", updatesVal.Kind())})
	}
	updateCount := updatesVal.Len()
	if keys != nil {
		if len(keys) != updateCount {
			return nil, nil, WithStack(InvalidArgumentError{Message: fmt.Sprintf("expected %d keys, got %d", updateCount, len(keys))})
		}
		for _, key := range keys {
			if err := validateKey(key); err != nil {
				return nil, nil, WithStack(err)
			}
		}
	}
	metas := make(DocumentMetaSlice, updateCount)
	errs := make(ErrorSlice, updateCount)
	silent := false
	for i := 0; i < updateCount; i++ {
		update := updatesVal.Index(i)
		ctx, err := withDocumentAt(ctx, i)
		if err != nil {
			return nil, nil, WithStack(err)
		}
		var key string
		if keys != nil {
			key = keys[i]
		} else {
			var err error
			key, err = getKeyFromDocument(update)
			if err != nil {
				errs[i] = err
				continue
			}
		}
		meta, cs, err := c.updateDocument(ctx, key, update.Interface())
		if cs.Silent {
			silent = true
		} else {
			metas[i], errs[i] = meta, err
		}
	}
	if silent {
		return nil, nil, nil
	}
	return metas, errs, nil
}

// ReplaceDocument replaces a single document with given key in the collection with the document given in the document argument.
// The document meta data is returned.
// To return the NEW document, prepare a context with `WithReturnNew`.
// To return the OLD document, prepare a context with `WithReturnOld`.
// To wait until document has been synced to disk, prepare a context with `WithWaitForSync`.
// If no document exists with given key, a NotFoundError is returned.
func (c *vertexCollection) ReplaceDocument(ctx context.Context, key string, document interface{}) (DocumentMeta, error) {
	meta, _, err := c.replaceDocument(ctx, key, document)
	if err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	return meta, nil
}

func (c *vertexCollection) replaceDocument(ctx context.Context, key string, document interface{}) (DocumentMeta, contextSettings, error) {
	if err := validateKey(key); err != nil {
		return DocumentMeta{}, contextSettings{}, WithStack(err)
	}
	if document == nil {
		return DocumentMeta{}, contextSettings{}, WithStack(InvalidArgumentError{Message: "document nil"})
	}
	escapedKey := pathEscape(key)
	req, err := c.conn.NewRequest("PUT", path.Join(c.relPath(), escapedKey))
	if err != nil {
		return DocumentMeta{}, contextSettings{}, WithStack(err)
	}
	if _, err := req.SetBody(document); err != nil {
		return DocumentMeta{}, contextSettings{}, WithStack(err)
	}
	cs := applyContextSettings(ctx, req)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return DocumentMeta{}, cs, WithStack(err)
	}
	if err := resp.CheckStatus(201, 202); err != nil {
		return DocumentMeta{}, cs, WithStack(err)
	}
	if cs.Silent {
		// Empty response, we're done
		return DocumentMeta{}, cs, nil
	}
	// Parse metadata
	var meta DocumentMeta
	if err := resp.ParseBody("vertex", &meta); err != nil {
		return DocumentMeta{}, cs, WithStack(err)
	}
	// Parse returnOld (if needed)
	if cs.ReturnOld != nil {
		if err := resp.ParseBody("old", cs.ReturnOld); err != nil {
			return meta, cs, WithStack(err)
		}
	}
	// Parse returnNew (if needed)
	if cs.ReturnNew != nil {
		if err := resp.ParseBody("new", cs.ReturnNew); err != nil {
			return meta, cs, WithStack(err)
		}
	}
	return meta, cs, nil
}

// ReplaceDocuments replaces multiple documents with given keys in the collection with the documents given in the documents argument.
// The replacements are loaded from the given documents slice, the documents meta data are returned.
// To return the NEW documents, prepare a context with `WithReturnNew` with a slice of documents.
// To return the OLD documents, prepare a context with `WithReturnOld` with a slice of documents.
// To wait until documents has been synced to disk, prepare a context with `WithWaitForSync`.
// If no document exists with a given key, a NotFoundError is returned at its errors index.
func (c *vertexCollection) ReplaceDocuments(ctx context.Context, keys []string, documents interface{}) (DocumentMetaSlice, ErrorSlice, error) {
	documentsVal := reflect.ValueOf(documents)
	switch documentsVal.Kind() {
	case reflect.Array, reflect.Slice:
		// OK
	default:
		return nil, nil, WithStack(InvalidArgumentError{Message: fmt.Sprintf("documents data must be of kind Array, got %s", documentsVal.Kind())})
	}
	documentCount := documentsVal.Len()
	if keys != nil {
		if len(keys) != documentCount {
			return nil, nil, WithStack(InvalidArgumentError{Message: fmt.Sprintf("expected %d keys, got %d", documentCount, len(keys))})
		}
		for _, key := range keys {
			if err := validateKey(key); err != nil {
				return nil, nil, WithStack(err)
			}
		}
	}
	metas := make(DocumentMetaSlice, documentCount)
	errs := make(ErrorSlice, documentCount)
	silent := false
	for i := 0; i < documentCount; i++ {
		doc := documentsVal.Index(i)
		ctx, err := withDocumentAt(ctx, i)
		if err != nil {
			return nil, nil, WithStack(err)
		}
		var key string
		if keys != nil {
			key = keys[i]
		} else {
			var err error
			key, err = getKeyFromDocument(doc)
			if err != nil {
				errs[i] = err
				continue
			}
		}
		meta, cs, err := c.replaceDocument(ctx, key, doc.Interface())
		if cs.Silent {
			silent = true
		} else {
			metas[i], errs[i] = meta, err
		}
	}
	if silent {
		return nil, nil, nil
	}
	return metas, errs, nil
}

// RemoveDocument removes a single document with given key from the collection.
// The document meta data is returned.
// To return the OLD document, prepare a context with `WithReturnOld`.
// To wait until removal has been synced to disk, prepare a context with `WithWaitForSync`.
// If no document exists with given key, a NotFoundError is returned.
func (c *vertexCollection) RemoveDocument(ctx context.Context, key string) (DocumentMeta, error) {
	meta, _, err := c.removeDocument(ctx, key)
	if err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	return meta, nil
}

func (c *vertexCollection) removeDocument(ctx context.Context, key string) (DocumentMeta, contextSettings, error) {
	if err := validateKey(key); err != nil {
		return DocumentMeta{}, contextSettings{}, WithStack(err)
	}
	escapedKey := pathEscape(key)
	req, err := c.conn.NewRequest("DELETE", path.Join(c.relPath(), escapedKey))
	if err != nil {
		return DocumentMeta{}, contextSettings{}, WithStack(err)
	}
	cs := applyContextSettings(ctx, req)
	if cs.ReturnOld != nil {
		return DocumentMeta{}, contextSettings{}, WithStack(InvalidArgumentError{Message: "ReturnOld is not support when removing vertices"})
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return DocumentMeta{}, cs, WithStack(err)
	}
	if err := resp.CheckStatus(200, 202); err != nil {
		return DocumentMeta{}, cs, WithStack(err)
	}
	if cs.Silent {
		// Empty response, we're done
		return DocumentMeta{}, cs, nil
	}
	// Parse metadata
	var meta DocumentMeta
	if err := resp.ParseBody("vertex", &meta); err != nil {
		return DocumentMeta{}, cs, WithStack(err)
	}
	// Parse returnOld (if needed)
	if cs.ReturnOld != nil {
		if err := resp.ParseBody("old", cs.ReturnOld); err != nil {
			return meta, cs, WithStack(err)
		}
	}
	return meta, cs, nil
}

// RemoveDocuments removes multiple documents with given keys from the collection.
// The document meta data are returned.
// To return the OLD documents, prepare a context with `WithReturnOld` with a slice of documents.
// To wait until removal has been synced to disk, prepare a context with `WithWaitForSync`.
// If no document exists with a given key, a NotFoundError is returned at its errors index.
func (c *vertexCollection) RemoveDocuments(ctx context.Context, keys []string) (DocumentMetaSlice, ErrorSlice, error) {
	keyCount := len(keys)
	for _, key := range keys {
		if err := validateKey(key); err != nil {
			return nil, nil, WithStack(err)
		}
	}
	metas := make(DocumentMetaSlice, keyCount)
	errs := make(ErrorSlice, keyCount)
	silent := false
	for i := 0; i < keyCount; i++ {
		key := keys[i]
		ctx, err := withDocumentAt(ctx, i)
		if err != nil {
			return nil, nil, WithStack(err)
		}
		meta, cs, err := c.removeDocument(ctx, key)
		if cs.Silent {
			silent = true
		} else {
			metas[i], errs[i] = meta, err
		}
	}
	if silent {
		return nil, nil, nil
	}
	return metas, errs, nil
}

// ImportDocuments imports one or more documents into the collection.
// The document data is loaded from the given documents argument, statistics are returned.
// The documents argument can be one of the following:
// - An array of structs: All structs will be imported as individual documents.
// - An array of maps: All maps will be imported as individual documents.
// To wait until all documents have been synced to disk, prepare a context with `WithWaitForSync`.
// To return details about documents that could not be imported, prepare a context with `WithImportDetails`.
func (c *vertexCollection) ImportDocuments(ctx context.Context, documents interface{}, options *ImportDocumentOptions) (ImportDocumentStatistics, error) {
	stats, err := c.rawCollection().ImportDocuments(ctx, documents, options)
	if err != nil {
		return ImportDocumentStatistics{}, WithStack(err)
	}
	return stats, nil
}
