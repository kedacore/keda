package kivik

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/go-kivik/kivik/v3/driver"
)

// BulkResults is an iterator over the results of a BulkDocs query.
type BulkResults struct {
	*iter
	bulki driver.BulkResults
}

// Next returns the next BulkResult from the feed. If an error occurs, it will
// be returned and the feed closed. io.EOF will be returned when there are no
// more results.
func (r *BulkResults) Next() bool {
	return r.iter.Next()
}

// Err returns the error, if any, that was encountered during iteration. Err
// may be called after an explicit or implicit Close.
func (r *BulkResults) Err() error {
	return r.iter.Err()
}

// Close closes the feed. Any unread updates will still be accessible via
// Next().
func (r *BulkResults) Close() error {
	return r.iter.Close()
}

type bulkIterator struct{ driver.BulkResults }

var _ iterator = &bulkIterator{}

func (r *bulkIterator) Next(i interface{}) error {
	return r.BulkResults.Next(i.(*driver.BulkResult))
}

func newBulkResults(ctx context.Context, bulki driver.BulkResults) *BulkResults {
	return &BulkResults{
		iter:  newIterator(ctx, &bulkIterator{bulki}, &driver.BulkResult{}),
		bulki: bulki,
	}
}

// ID returns the document ID name for the current result.
func (r *BulkResults) ID() string {
	runlock, err := r.rlock()
	if err != nil {
		return ""
	}
	defer runlock()
	return r.curVal.(*driver.BulkResult).ID
}

// Rev returns the revision of the current curResult.
func (r *BulkResults) Rev() string {
	runlock, err := r.rlock()
	if err != nil {
		return ""
	}
	defer runlock()
	return r.curVal.(*driver.BulkResult).Rev
}

// UpdateErr returns the error associated with the current result, or nil
// if none. Do not confuse this with Err, which returns an error for the
// iterator itself.
func (r *BulkResults) UpdateErr() error {
	runlock, err := r.rlock()
	if err != nil {
		return nil
	}
	defer runlock()
	return r.curVal.(*driver.BulkResult).Error
}

// BulkDocs allows you to create and update multiple documents at the same time
// within a single request. This function returns an iterator over the results
// of the bulk operation.
// See http://docs.couchdb.org/en/2.0.0/api/database/bulk-api.html#db-bulk-docs
//
// As with Put, each individual document may be a JSON-marshable object, or a
// raw JSON string in a []byte, json.RawMessage, or io.Reader.
func (db *DB) BulkDocs(ctx context.Context, docs []interface{}, options ...Options) (*BulkResults, error) {
	docsi, err := docsInterfaceSlice(docs)
	if err != nil {
		return nil, err
	}
	if len(docsi) == 0 {
		return nil, &Error{HTTPStatus: http.StatusBadRequest, Err: errors.New("kivik: no documents provided")}
	}
	opts := mergeOptions(options...)
	if bulkDocer, ok := db.driverDB.(driver.BulkDocer); ok {
		bulki, err := bulkDocer.BulkDocs(ctx, docsi, opts)
		if err != nil {
			return nil, err
		}
		return newBulkResults(ctx, bulki), nil
	}
	var results []driver.BulkResult
	for _, doc := range docsi {
		var err error
		var id, rev string
		if docID, ok := extractDocID(doc); ok {
			id = docID
			rev, err = db.Put(ctx, id, doc, opts)
		} else {
			id, rev, err = db.CreateDoc(ctx, doc, opts)
		}
		results = append(results, driver.BulkResult{
			ID:    id,
			Rev:   rev,
			Error: err,
		})
	}
	return newBulkResults(ctx, &emulatedBulkResults{results}), nil
}

type emulatedBulkResults struct {
	results []driver.BulkResult
}

var _ driver.BulkResults = &emulatedBulkResults{}

func (r *emulatedBulkResults) Close() error {
	r.results = nil
	return nil
}

func (r *emulatedBulkResults) Next(res *driver.BulkResult) error {
	if len(r.results) == 0 {
		return io.EOF
	}
	*res = r.results[0]
	r.results = r.results[1:]
	return nil
}

func docsInterfaceSlice(docsi []interface{}) ([]interface{}, error) {
	for i, doc := range docsi {
		x, err := normalizeFromJSON(doc)
		if err != nil {
			return nil, &Error{HTTPStatus: http.StatusBadRequest, Err: err}
		}
		docsi[i] = x
	}
	return docsi, nil
}
