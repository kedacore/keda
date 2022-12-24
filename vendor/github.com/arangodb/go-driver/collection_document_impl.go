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
func (c *collection) DocumentExists(ctx context.Context, key string) (bool, error) {
	if err := validateKey(key); err != nil {
		return false, WithStack(err)
	}
	escapedKey := pathEscape(key)
	req, err := c.conn.NewRequest("HEAD", path.Join(c.relPath("document"), escapedKey))
	if err != nil {
		return false, WithStack(err)
	}
	applyContextSettings(ctx, req)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return false, WithStack(err)
	}
	found := resp.StatusCode() == 200
	return found, nil
}

// ReadDocument reads a single document with given key from the collection.
// The document data is stored into result, the document meta data is returned.
// If no document exists with given key, a NotFoundError is returned.
func (c *collection) ReadDocument(ctx context.Context, key string, result interface{}) (DocumentMeta, error) {
	if err := validateKey(key); err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	escapedKey := pathEscape(key)
	req, err := c.conn.NewRequest("GET", path.Join(c.relPath("document"), escapedKey))
	if err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	// This line introduces a lot of side effects. In particular If-Match headers are now set (which is a bugfix)
	// and invalid query parameters like waitForSync (which is potentially breaking change)
	cs := applyContextSettings(ctx, req)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	// Parse metadata
	var meta DocumentMeta
	if err := resp.ParseBody("", &meta); err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	// load context response values
	loadContextResponseValues(cs, resp)
	// Parse result
	if result != nil {
		if err := resp.ParseBody("", result); err != nil {
			return meta, WithStack(err)
		}
	}
	return meta, nil
}

// ReadDocuments reads multiple documents with given keys from the collection.
// The documents data is stored into elements of the given results slice,
// the documents meta data is returned.
// If no document exists with a given key, a NotFoundError is returned at its errors index.
func (c *collection) ReadDocuments(ctx context.Context, keys []string, results interface{}) (DocumentMetaSlice, ErrorSlice, error) {
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
	req, err := c.conn.NewRequest("PUT", c.relPath("document"))
	if err != nil {
		return nil, nil, WithStack(err)
	}
	req = req.SetQuery("onlyget", "1")
	cs := applyContextSettings(ctx, req)
	if _, err := req.SetBodyArray(keys, nil); err != nil {
		return nil, nil, WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return nil, nil, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return nil, nil, WithStack(err)
	}
	// load context response values
	loadContextResponseValues(cs, resp)
	// Parse response array
	metas, errs, err := parseResponseArray(resp, resultCount, cs, results)
	if err != nil {
		return nil, nil, WithStack(err)
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
func (c *collection) CreateDocument(ctx context.Context, document interface{}) (DocumentMeta, error) {
	if document == nil {
		return DocumentMeta{}, WithStack(InvalidArgumentError{Message: "document nil"})
	}
	req, err := c.conn.NewRequest("POST", c.relPath("document"))
	if err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	if _, err := req.SetBody(document); err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	cs := applyContextSettings(ctx, req)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	if err := resp.CheckStatus(201, 202); err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	if cs.Silent {
		// Empty response, we're done
		return DocumentMeta{}, nil
	}
	// Parse metadata
	var meta DocumentMeta
	if err := resp.ParseBody("", &meta); err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	// Parse returnNew (if needed)
	if cs.ReturnNew != nil {
		if err := resp.ParseBody("new", cs.ReturnNew); err != nil {
			return meta, WithStack(err)
		}
	}
	return meta, nil
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
func (c *collection) CreateDocuments(ctx context.Context, documents interface{}) (DocumentMetaSlice, ErrorSlice, error) {
	documentsVal := reflect.ValueOf(documents)
	switch documentsVal.Kind() {
	case reflect.Array, reflect.Slice:
		// OK
	default:
		return nil, nil, WithStack(InvalidArgumentError{Message: fmt.Sprintf("documents data must be of kind Array, got %s", documentsVal.Kind())})
	}
	documentCount := documentsVal.Len()
	req, err := c.conn.NewRequest("POST", c.relPath("document"))
	if err != nil {
		return nil, nil, WithStack(err)
	}
	if _, err := req.SetBody(documents); err != nil {
		return nil, nil, WithStack(err)
	}
	cs := applyContextSettings(ctx, req)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return nil, nil, WithStack(err)
	}
	if err := resp.CheckStatus(201, 202); err != nil {
		return nil, nil, WithStack(err)
	}
	if cs.Silent {
		// Empty response, we're done
		return nil, nil, nil
	}
	// Parse response array
	metas, errs, err := parseResponseArray(resp, documentCount, cs, nil)
	if err != nil {
		return nil, nil, WithStack(err)
	}
	return metas, errs, nil
}

// UpdateDocument updates a single document with given key in the collection.
// The document meta data is returned.
// To return the NEW document, prepare a context with `WithReturnNew`.
// To return the OLD document, prepare a context with `WithReturnOld`.
// To wait until document has been synced to disk, prepare a context with `WithWaitForSync`.
// If no document exists with given key, a NotFoundError is returned.
func (c *collection) UpdateDocument(ctx context.Context, key string, update interface{}) (DocumentMeta, error) {
	if err := validateKey(key); err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	if update == nil {
		return DocumentMeta{}, WithStack(InvalidArgumentError{Message: "update nil"})
	}
	escapedKey := pathEscape(key)
	req, err := c.conn.NewRequest("PATCH", path.Join(c.relPath("document"), escapedKey))
	if err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	if _, err := req.SetBody(update); err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	cs := applyContextSettings(ctx, req)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	if err := resp.CheckStatus(201, 202); err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	if cs.Silent {
		// Empty response, we're done
		return DocumentMeta{}, nil
	}
	// Parse metadata
	var meta DocumentMeta
	if err := resp.ParseBody("", &meta); err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	// Parse returnOld (if needed)
	if cs.ReturnOld != nil {
		if err := resp.ParseBody("old", cs.ReturnOld); err != nil {
			return meta, WithStack(err)
		}
	}
	// Parse returnNew (if needed)
	if cs.ReturnNew != nil {
		if err := resp.ParseBody("new", cs.ReturnNew); err != nil {
			return meta, WithStack(err)
		}
	}
	return meta, nil
}

// UpdateDocuments updates multiple document with given keys in the collection.
// The updates are loaded from the given updates slice, the documents meta data are returned.
// To return the NEW documents, prepare a context with `WithReturnNew` with a slice of documents.
// To return the OLD documents, prepare a context with `WithReturnOld` with a slice of documents.
// To wait until documents has been synced to disk, prepare a context with `WithWaitForSync`.
// If no document exists with a given key, a NotFoundError is returned at its errors index.
func (c *collection) UpdateDocuments(ctx context.Context, keys []string, updates interface{}) (DocumentMetaSlice, ErrorSlice, error) {
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
	req, err := c.conn.NewRequest("PATCH", c.relPath("document"))
	if err != nil {
		return nil, nil, WithStack(err)
	}
	cs := applyContextSettings(ctx, req)
	mergeArray, err := createMergeArray(keys, cs.Revisions)
	if err != nil {
		return nil, nil, WithStack(err)
	}
	if _, err := req.SetBodyArray(updates, mergeArray); err != nil {
		return nil, nil, WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return nil, nil, WithStack(err)
	}
	if err := resp.CheckStatus(201, 202); err != nil {
		return nil, nil, WithStack(err)
	}
	if cs.Silent {
		// Empty response, we're done
		return nil, nil, nil
	}
	// Parse response array
	metas, errs, err := parseResponseArray(resp, updateCount, cs, nil)
	if err != nil {
		return nil, nil, WithStack(err)
	}
	return metas, errs, nil
}

// ReplaceDocument replaces a single document with given key in the collection with the document given in the document argument.
// The document meta data is returned.
// To return the NEW document, prepare a context with `WithReturnNew`.
// To return the OLD document, prepare a context with `WithReturnOld`.
// To wait until document has been synced to disk, prepare a context with `WithWaitForSync`.
// If no document exists with given key, a NotFoundError is returned.
func (c *collection) ReplaceDocument(ctx context.Context, key string, document interface{}) (DocumentMeta, error) {
	if err := validateKey(key); err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	if document == nil {
		return DocumentMeta{}, WithStack(InvalidArgumentError{Message: "document nil"})
	}
	escapedKey := pathEscape(key)
	req, err := c.conn.NewRequest("PUT", path.Join(c.relPath("document"), escapedKey))
	if err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	if _, err := req.SetBody(document); err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	cs := applyContextSettings(ctx, req)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	if err := resp.CheckStatus(201, 202); err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	if cs.Silent {
		// Empty response, we're done
		return DocumentMeta{}, nil
	}
	// Parse metadata
	var meta DocumentMeta
	if err := resp.ParseBody("", &meta); err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	// Parse returnOld (if needed)
	if cs.ReturnOld != nil {
		if err := resp.ParseBody("old", cs.ReturnOld); err != nil {
			return meta, WithStack(err)
		}
	}
	// Parse returnNew (if needed)
	if cs.ReturnNew != nil {
		if err := resp.ParseBody("new", cs.ReturnNew); err != nil {
			return meta, WithStack(err)
		}
	}
	return meta, nil
}

// ReplaceDocuments replaces multiple documents with given keys in the collection with the documents given in the documents argument.
// The replacements are loaded from the given documents slice, the documents meta data are returned.
// To return the NEW documents, prepare a context with `WithReturnNew` with a slice of documents.
// To return the OLD documents, prepare a context with `WithReturnOld` with a slice of documents.
// To wait until documents has been synced to disk, prepare a context with `WithWaitForSync`.
// If no document exists with a given key, a NotFoundError is returned at its errors index.
func (c *collection) ReplaceDocuments(ctx context.Context, keys []string, documents interface{}) (DocumentMetaSlice, ErrorSlice, error) {
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
	req, err := c.conn.NewRequest("PUT", c.relPath("document"))
	if err != nil {
		return nil, nil, WithStack(err)
	}
	cs := applyContextSettings(ctx, req)
	mergeArray, err := createMergeArray(keys, cs.Revisions)
	if err != nil {
		return nil, nil, WithStack(err)
	}
	if _, err := req.SetBodyArray(documents, mergeArray); err != nil {
		return nil, nil, WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return nil, nil, WithStack(err)
	}
	if err := resp.CheckStatus(201, 202); err != nil {
		return nil, nil, WithStack(err)
	}
	if cs.Silent {
		// Empty response, we're done
		return nil, nil, nil
	}
	// Parse response array
	metas, errs, err := parseResponseArray(resp, documentCount, cs, nil)
	if err != nil {
		return nil, nil, WithStack(err)
	}
	return metas, errs, nil
}

// RemoveDocument removes a single document with given key from the collection.
// The document meta data is returned.
// To return the OLD document, prepare a context with `WithReturnOld`.
// To wait until removal has been synced to disk, prepare a context with `WithWaitForSync`.
// If no document exists with given key, a NotFoundError is returned.
func (c *collection) RemoveDocument(ctx context.Context, key string) (DocumentMeta, error) {
	if err := validateKey(key); err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	escapedKey := pathEscape(key)
	req, err := c.conn.NewRequest("DELETE", path.Join(c.relPath("document"), escapedKey))
	if err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	cs := applyContextSettings(ctx, req)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	if err := resp.CheckStatus(200, 202); err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	if cs.Silent {
		// Empty response, we're done
		return DocumentMeta{}, nil
	}
	// Parse metadata
	var meta DocumentMeta
	if err := resp.ParseBody("", &meta); err != nil {
		return DocumentMeta{}, WithStack(err)
	}
	// Parse returnOld (if needed)
	if cs.ReturnOld != nil {
		if err := resp.ParseBody("old", cs.ReturnOld); err != nil {
			return meta, WithStack(err)
		}
	}
	return meta, nil
}

// RemoveDocuments removes multiple documents with given keys from the collection.
// The document meta data are returned.
// To return the OLD documents, prepare a context with `WithReturnOld` with a slice of documents.
// To wait until removal has been synced to disk, prepare a context with `WithWaitForSync`.
// If no document exists with a given key, a NotFoundError is returned at its errors index.
func (c *collection) RemoveDocuments(ctx context.Context, keys []string) (DocumentMetaSlice, ErrorSlice, error) {
	for _, key := range keys {
		if err := validateKey(key); err != nil {
			return nil, nil, WithStack(err)
		}
	}
	keyCount := len(keys)
	req, err := c.conn.NewRequest("DELETE", c.relPath("document"))
	if err != nil {
		return nil, nil, WithStack(err)
	}
	cs := applyContextSettings(ctx, req)
	metaArray, err := createMergeArray(keys, cs.Revisions)
	if err != nil {
		return nil, nil, WithStack(err)
	}
	if _, err := req.SetBodyArray(metaArray, nil); err != nil {
		return nil, nil, WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return nil, nil, WithStack(err)
	}
	if err := resp.CheckStatus(200, 202); err != nil {
		return nil, nil, WithStack(err)
	}
	if cs.Silent {
		// Empty response, we're done
		return nil, nil, nil
	}
	// Parse response array
	metas, errs, err := parseResponseArray(resp, keyCount, cs, nil)
	if err != nil {
		return nil, nil, WithStack(err)
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
func (c *collection) ImportDocuments(ctx context.Context, documents interface{}, options *ImportDocumentOptions) (ImportDocumentStatistics, error) {
	documentsVal := reflect.ValueOf(documents)
	switch documentsVal.Kind() {
	case reflect.Array, reflect.Slice:
		// OK
	default:
		return ImportDocumentStatistics{}, WithStack(InvalidArgumentError{Message: fmt.Sprintf("documents data must be of kind Array, got %s", documentsVal.Kind())})
	}
	req, err := c.conn.NewRequest("POST", path.Join(c.db.relPath(), "_api/import"))
	if err != nil {
		return ImportDocumentStatistics{}, WithStack(err)
	}
	req.SetQuery("collection", c.name)
	req.SetQuery("type", "documents")
	if options != nil {
		if v := options.FromPrefix; v != "" {
			req.SetQuery("fromPrefix", v)
		}
		if v := options.ToPrefix; v != "" {
			req.SetQuery("toPrefix", v)
		}
		if v := options.Overwrite; v {
			req.SetQuery("overwrite", "true")
		}
		if v := options.OnDuplicate; v != "" {
			req.SetQuery("onDuplicate", string(v))
		}
		if v := options.Complete; v {
			req.SetQuery("complete", "true")
		}
	}
	if _, err := req.SetBodyImportArray(documents); err != nil {
		return ImportDocumentStatistics{}, WithStack(err)
	}
	cs := applyContextSettings(ctx, req)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return ImportDocumentStatistics{}, WithStack(err)
	}
	if err := resp.CheckStatus(201); err != nil {
		return ImportDocumentStatistics{}, WithStack(err)
	}
	// Parse response
	var data ImportDocumentStatistics
	if err := resp.ParseBody("", &data); err != nil {
		return ImportDocumentStatistics{}, WithStack(err)
	}
	// Import details (if needed)
	if details := cs.ImportDetails; details != nil {
		if err := resp.ParseBody("details", details); err != nil {
			return ImportDocumentStatistics{}, WithStack(err)
		}
	}
	return data, nil
}

// createMergeArray returns an array of metadata maps with `_key` and/or `_rev` elements.
func createMergeArray(keys, revs []string) ([]map[string]interface{}, error) {
	if keys == nil && revs == nil {
		return nil, nil
	}
	if revs == nil {
		mergeArray := make([]map[string]interface{}, len(keys))
		for i, k := range keys {
			mergeArray[i] = map[string]interface{}{
				"_key": k,
			}
		}
		return mergeArray, nil
	}
	if keys == nil {
		mergeArray := make([]map[string]interface{}, len(revs))
		for i, r := range revs {
			mergeArray[i] = map[string]interface{}{
				"_rev": r,
			}
		}
		return mergeArray, nil
	}
	if len(keys) != len(revs) {
		return nil, WithStack(InvalidArgumentError{Message: fmt.Sprintf("#keys must be equal to #revs, got %d, %d", len(keys), len(revs))})
	}
	mergeArray := make([]map[string]interface{}, len(keys))
	for i, k := range keys {
		mergeArray[i] = map[string]interface{}{
			"_key": k,
			"_rev": revs[i],
		}
	}
	return mergeArray, nil

}

// parseResponseArray parses an array response in the given response
func parseResponseArray(resp Response, count int, cs contextSettings, results interface{}) (DocumentMetaSlice, ErrorSlice, error) {
	resps, err := resp.ParseArrayBody()
	if err != nil {
		return nil, nil, WithStack(err)
	}
	metas := make(DocumentMetaSlice, count)
	errs := make(ErrorSlice, count)
	returnOldVal := reflect.ValueOf(cs.ReturnOld)
	returnNewVal := reflect.ValueOf(cs.ReturnNew)
	resultsVal := reflect.ValueOf(results)
	for i := 0; i < count; i++ {
		resp := resps[i]
		var meta DocumentMeta
		if err := resp.CheckStatus(200, 201, 202); err != nil {
			errs[i] = err
		} else {
			if err := resp.ParseBody("", &meta); err != nil {
				errs[i] = err
			} else {
				metas[i] = meta
				// Parse returnOld (if needed)
				if cs.ReturnOld != nil {
					returnOldEntryVal := returnOldVal.Index(i).Addr()
					if err := resp.ParseBody("old", returnOldEntryVal.Interface()); err != nil {
						errs[i] = err
					}
				}
				// Parse returnNew (if needed)
				if cs.ReturnNew != nil {
					returnNewEntryVal := returnNewVal.Index(i).Addr()
					if err := resp.ParseBody("new", returnNewEntryVal.Interface()); err != nil {
						errs[i] = err
					}
				}
			}
			if results != nil {
				// Parse compare result document
				resultsEntryVal := resultsVal.Index(i).Addr()
				if err := resp.ParseBody("", resultsEntryVal.Interface()); err != nil {
					errs[i] = err
				}
			}
		}
	}
	return metas, errs, nil
}
