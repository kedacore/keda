package kivik

import (
	"context"
	"net/http"

	"github.com/go-kivik/kivik/v3/driver"
)

// Rows is an iterator over a a multi-value query.
type Rows struct {
	*iter
	rowsi driver.Rows
}

// Next prepares the next result value for reading. It returns true on success
// or false if there are no more results or an error  occurs while preparing it.
// Err should be consulted to distinguish between the two.
func (r *Rows) Next() bool {
	return r.iter.Next()
}

// Err returns the error, if any, that was encountered during iteration. Err may
// be called after an explicit or implicit Close.
func (r *Rows) Err() error {
	return r.iter.Err()
}

// Close closes the Rows, preventing further enumeration, and freeing any
// resources (such as the http request body) of the underlying query. If Next is
// called and there are no further results, Rows is closed automatically and it
// will suffice to check the result of Err. Close is idempotent and does not
// affect the result of Err.
func (r *Rows) Close() error {
	return r.iter.Close()
}

type rowsIterator struct{ driver.Rows }

var _ iterator = &rowsIterator{}

func (r *rowsIterator) Next(i interface{}) error { return r.Rows.Next(i.(*driver.Row)) }

func newRows(ctx context.Context, rowsi driver.Rows) *Rows {
	return &Rows{
		iter:  newIterator(ctx, &rowsIterator{rowsi}, &driver.Row{}),
		rowsi: rowsi,
	}
}

var (
	errNilPtr = &Error{HTTPStatus: http.StatusBadRequest, Message: "kivik: destination pointer is nil"}
	errNonPtr = &Error{HTTPStatus: http.StatusBadRequest, Message: "kivik: destination is not a pointer"}
)

// ScanValue copies the data from the result value into the value pointed at by
// dest. Think of this as a json.Unmarshal into dest.
//
// If the dest argument has type *[]byte, Scan stores a copy of the input data.
// The copy is owned by the caller and can be modified and held indefinitely.
//
// The copy can be avoided by using an argument of type *json.RawMessage
// instead. After a ScanValue into a json.RawMessage, the slice is only valid
// until the next call to Next or Close.
//
// For all other types, refer to the documentation for json.Unmarshal for type
// conversion rules.
func (r *Rows) ScanValue(dest interface{}) error {
	runlock, err := r.rlock()
	if err != nil {
		return err
	}
	defer runlock()
	if err := r.curVal.(*driver.Row).Error; err != nil {
		return err
	}
	return scan(dest, r.curVal.(*driver.Row).Value)
}

// ScanDoc works the same as ScanValue, but on the doc field of the result. It
// will panic if the query does not include documents.
func (r *Rows) ScanDoc(dest interface{}) error {
	runlock, err := r.rlock()
	if err != nil {
		return err
	}
	defer runlock()
	if err := r.curVal.(*driver.Row).Error; err != nil {
		return err
	}
	doc := r.curVal.(*driver.Row).Doc
	if doc == nil {
		return &Error{HTTPStatus: http.StatusBadRequest, Message: "kivik: doc is nil; does the query include docs?"}
	}
	return scan(dest, doc)
}

// ScanKey works the same as ScanValue, but on the key field of the result. For
// simple keys, which are just strings, the Key() method may be easier to use.
func (r *Rows) ScanKey(dest interface{}) error {
	runlock, err := r.rlock()
	if err != nil {
		return err
	}
	if err := r.curVal.(*driver.Row).Error; err != nil {
		return err
	}
	defer runlock()
	return scan(dest, r.curVal.(*driver.Row).Key)
}

// ID returns the ID of the current result.
func (r *Rows) ID() string {
	runlock, err := r.rlock()
	if err != nil {
		return ""
	}
	defer runlock()
	return r.curVal.(*driver.Row).ID
}

// Key returns the Key of the current result as a raw JSON string. For
// compound keys, the ScanKey() method may be more convenient.
func (r *Rows) Key() string {
	runlock, err := r.rlock()
	if err != nil {
		return ""
	}
	defer runlock()
	return string(r.curVal.(*driver.Row).Key)
}

// Offset returns the starting offset where the result set started. It is
// only guaranteed to be set after all result rows have been enumerated through
// by Next, and thus should only be read after processing all rows in a result
// set. Calling Close before enumerating will render this value unreliable.
func (r *Rows) Offset() int64 {
	return r.rowsi.Offset()
}

// TotalRows returns the total number of rows in the view which would have been
// returned if no limiting were used. This value is only guaranteed to be set
// after all result rows have been enumerated through by Next, and thus should
// only be read after processing all rows in a result set. Calling Close before
// enumerating will render this value unreliable.
func (r *Rows) TotalRows() int64 {
	return r.rowsi.TotalRows()
}

// UpdateSeq returns the sequence id of the underlying database the view
// reflects, if requested in the query. This value is only guaranteed to be set
// after all result rows have been enumerated through by Next, and thus should
// only be read after processing all rows in a result set. Calling Close before
// enumerating will render this value unreliable.
func (r *Rows) UpdateSeq() string {
	return r.rowsi.UpdateSeq()
}

// Warning returns a warning generated by the query, if any. This value is only
// guaranteed to be set after all result rows have been enumeratd through by
// Next.
func (r *Rows) Warning() string {
	if w, ok := r.rowsi.(driver.RowsWarner); ok {
		return w.Warning()
	}
	return ""
}

// QueryIndex returns the 0-based index of the query. For standard queries,
// this is always 0. When multiple queries are passed to the view, this will
// represent the query currently being iterated
func (r *Rows) QueryIndex() int {
	if qi, ok := r.rowsi.(driver.QueryIndexer); ok {
		return qi.QueryIndex()
	}
	return 0
}

// Bookmark returns the paging bookmark, if one was provided with the result
// set. This is intended for use with the Mango /_find interface, with CouchDB
// 2.1.1 and later. Consult the official CouchDB documentation for detailed
// usage instructions. http://docs.couchdb.org/en/2.1.1/api/database/find.html#pagination
func (r *Rows) Bookmark() string {
	if b, ok := r.rowsi.(driver.Bookmarker); ok {
		return b.Bookmark()
	}
	return ""
}
