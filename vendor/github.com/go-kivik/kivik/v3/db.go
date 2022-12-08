package kivik

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-kivik/kivik/v3/driver"
)

// DB is a handle to a specific database.
type DB struct {
	client   *Client
	name     string
	driverDB driver.DB
	err      error
}

// Client returns the Client used to connect to the database.
func (db *DB) Client() *Client {
	return db.client
}

// Name returns the database name as passed when creating the DB connection.
func (db *DB) Name() string {
	return db.name
}

// Err returns the error, if any, that occurred while connecting to or creating
// the database. This error will be deferred until the next call, normally, so
// using this method is only ever necessary if you need to directly check the
// error status, and intend to do nothing else with the DB object.
func (db *DB) Err() error {
	return db.err
}

// AllDocs returns a list of all documents in the database.
func (db *DB) AllDocs(ctx context.Context, options ...Options) (*Rows, error) {
	if db.err != nil {
		return nil, db.err
	}
	rowsi, err := db.driverDB.AllDocs(ctx, mergeOptions(options...))
	if err != nil {
		return nil, err
	}
	return newRows(ctx, rowsi), nil
}

// DesignDocs returns a list of all documents in the database.
func (db *DB) DesignDocs(ctx context.Context, options ...Options) (*Rows, error) {
	if db.err != nil {
		return nil, db.err
	}
	ddocer, ok := db.driverDB.(driver.DesignDocer)
	if !ok {
		return nil, &Error{HTTPStatus: http.StatusNotImplemented, Err: errors.New("kivik: design doc view not supported by driver")}
	}
	rowsi, err := ddocer.DesignDocs(ctx, mergeOptions(options...))
	if err != nil {
		return nil, err
	}
	return newRows(ctx, rowsi), nil
}

// LocalDocs returns a list of all documents in the database.
func (db *DB) LocalDocs(ctx context.Context, options ...Options) (*Rows, error) {
	if db.err != nil {
		return nil, db.err
	}
	ldocer, ok := db.driverDB.(driver.LocalDocer)
	if !ok {
		return nil, &Error{HTTPStatus: http.StatusNotImplemented, Err: errors.New("kivik: local doc view not supported by driver")}
	}
	rowsi, err := ldocer.LocalDocs(ctx, mergeOptions(options...))
	if err != nil {
		return nil, err
	}
	return newRows(ctx, rowsi), nil
}

// Query executes the specified view function from the specified design
// document. ddoc and view may or may not be be prefixed with '_design/'
// and '_view/' respectively. No other
func (db *DB) Query(ctx context.Context, ddoc, view string, options ...Options) (*Rows, error) {
	if db.err != nil {
		return nil, db.err
	}
	ddoc = strings.TrimPrefix(ddoc, "_design/")
	view = strings.TrimPrefix(view, "_view/")
	rowsi, err := db.driverDB.Query(ctx, ddoc, view, mergeOptions(options...))
	if err != nil {
		return nil, err
	}
	return newRows(ctx, rowsi), nil
}

// Row contains the result of calling Get for a single document. For most uses,
// it is sufficient just to call the ScanDoc method. For more advanced uses, the
// fields may be accessed directly.
type Row struct {
	// ContentLength records the size of the JSON representation of the document
	// as requestd. The value -1 indicates that the length is unknown. Values
	// >= 0 indicate that the given number of bytes may be read from Body.
	ContentLength int64

	// Rev is the revision ID of the returned document.
	Rev string

	// Body represents the document's content.
	//
	// Kivik will always return a non-nil Body, except when Err is non-nil. The
	// ScanDoc method will close Body. When not using ScanDoc, it is the
	// caller's responsibility to close Body
	Body io.ReadCloser

	// Err contains any error that occurred while fetching the document. It is
	// typically returned by ScanDoc.
	Err error

	// Attachments is experimental
	Attachments *AttachmentsIterator
}

// ScanDoc unmarshals the data from the fetched row into dest. It is an
// intelligent wrapper around json.Unmarshal which also handles
// multipart/related responses. When done, the underlying reader is closed.
func (r *Row) ScanDoc(dest interface{}) error {
	if r.Err != nil {
		return r.Err
	}
	if reflect.TypeOf(dest).Kind() != reflect.Ptr {
		return errNonPtr
	}
	defer r.Body.Close() // nolint: errcheck
	if err := json.NewDecoder(r.Body).Decode(dest); err != nil {
		return err
	}
	return nil
}

// Get fetches the requested document. Any errors are deferred until the
// row.ScanDoc call.
func (db *DB) Get(ctx context.Context, docID string, options ...Options) *Row {
	if db.err != nil {
		return &Row{Err: db.err}
	}
	doc, err := db.driverDB.Get(ctx, docID, mergeOptions(options...))
	if err != nil {
		return &Row{Err: err}
	}
	row := &Row{
		ContentLength: doc.ContentLength,
		Rev:           doc.Rev,
		Body:          doc.Body,
	}
	if doc.Attachments != nil {
		row.Attachments = &AttachmentsIterator{atti: doc.Attachments}
	}
	return row
}

// GetMeta returns the size and rev of the specified document. GetMeta accepts
// the same options as the Get method.
func (db *DB) GetMeta(ctx context.Context, docID string, options ...Options) (size int64, rev string, err error) {
	if db.err != nil {
		return 0, "", db.err
	}
	opts := mergeOptions(options...)
	if r, ok := db.driverDB.(driver.MetaGetter); ok {
		return r.GetMeta(ctx, docID, opts)
	}
	row := db.Get(ctx, docID, opts)
	if row.Err != nil {
		return 0, "", row.Err
	}
	if row.Rev != "" {
		_ = row.Body.Close()
		return row.ContentLength, row.Rev, nil
	}
	var doc struct {
		Rev string `json:"_rev"`
	}
	// These last two lines cannot be combined for GopherJS due to a bug.
	// See https://github.com/gopherjs/gopherjs/issues/608
	err = row.ScanDoc(&doc)
	return row.ContentLength, doc.Rev, err
}

// CreateDoc creates a new doc with an auto-generated unique ID. The generated
// docID and new rev are returned.
func (db *DB) CreateDoc(ctx context.Context, doc interface{}, options ...Options) (docID, rev string, err error) {
	if db.err != nil {
		return "", "", db.err
	}
	return db.driverDB.CreateDoc(ctx, doc, mergeOptions(options...))
}

// normalizeFromJSON unmarshals a []byte, json.RawMessage or io.Reader to a
// map[string]interface{}, or passed through any other types.
func normalizeFromJSON(i interface{}) (interface{}, error) {
	var body []byte
	switch t := i.(type) {
	case []byte:
		body = t
	case json.RawMessage:
		body = t
	default:
		r, ok := i.(io.Reader)
		if !ok {
			return i, nil
		}
		var err error
		body, err = ioutil.ReadAll(r)
		if err != nil {
			return nil, &Error{HTTPStatus: http.StatusBadRequest, Err: err}
		}
	}
	var x map[string]interface{}
	if err := json.Unmarshal(body, &x); err != nil {
		return nil, &Error{HTTPStatus: http.StatusBadRequest, Err: err}
	}
	return x, nil
}

func extractDocID(i interface{}) (string, bool) {
	if i == nil {
		return "", false
	}
	var id string
	var ok bool
	switch t := i.(type) {
	case map[string]interface{}:
		id, ok = t["_id"].(string)
	case map[string]string:
		id, ok = t["_id"]
	default:
		data, err := json.Marshal(i)
		if err != nil {
			return "", false
		}
		var result struct {
			ID string `json:"_id"`
		}
		if err := json.Unmarshal(data, &result); err != nil {
			return "", false
		}
		id = result.ID
		ok = result.ID != ""
	}
	if !ok {
		return "", false
	}
	return id, true
}

// Put creates a new doc or updates an existing one, with the specified docID.
// If the document already exists, the current revision must be included in doc,
// with JSON key '_rev', otherwise a conflict will occur. The new rev is
// returned.
//
// doc may be one of:
//
//  - An object to be marshaled to JSON. The resulting JSON structure must
//    conform to CouchDB standards.
//  - A []byte value, containing a valid JSON document
//  - A json.RawMessage value containing a valid JSON document
//  - An io.Reader, from which a valid JSON document may be read.
func (db *DB) Put(ctx context.Context, docID string, doc interface{}, options ...Options) (rev string, err error) {
	if db.err != nil {
		return "", db.err
	}
	if docID == "" {
		return "", missingArg("docID")
	}
	i, err := normalizeFromJSON(doc)
	if err != nil {
		return "", err
	}
	return db.driverDB.Put(ctx, docID, i, mergeOptions(options...))
}

// Delete marks the specified document as deleted. The revision may be provided
// via options, which takes priority over the rev argument.
func (db *DB) Delete(ctx context.Context, docID, rev string, options ...Options) (newRev string, err error) {
	if db.err != nil {
		return "", db.err
	}
	if docID == "" {
		return "", missingArg("docID")
	}
	opts := mergeOptions(options...)
	if rv, ok := opts["rev"].(string); ok && rv != "" {
		rev = rv
	}
	return db.driverDB.Delete(ctx, docID, rev, opts)
}

// Flush requests a flush of disk cache to disk or other permanent storage.
//
// See http://docs.couchdb.org/en/2.0.0/api/database/compact.html#db-ensure-full-commit
func (db *DB) Flush(ctx context.Context) error {
	if db.err != nil {
		return db.err
	}
	if flusher, ok := db.driverDB.(driver.Flusher); ok {
		return flusher.Flush(ctx)
	}
	return &Error{HTTPStatus: http.StatusNotImplemented, Err: errors.New("kivik: flush not supported by driver")}
}

// DBStats contains database statistics..
type DBStats struct {
	// Name is the name of the database.
	Name string `json:"db_name"`
	// CompactRunning is true if the database is currently being compacted.
	CompactRunning bool `json:"compact_running"`
	// DocCount is the number of documents are currently stored in the database.
	DocCount int64 `json:"doc_count"`
	// DeletedCount is a count of documents which have been deleted from the
	// database.
	DeletedCount int64 `json:"doc_del_count"`
	// UpdateSeq is the current update sequence for the database.
	UpdateSeq string `json:"update_seq"`
	// DiskSize is the number of bytes used on-disk to store the database.
	DiskSize int64 `json:"disk_size"`
	// ActiveSize is the number of bytes used on-disk to store active documents.
	// If this number is lower than DiskSize, then compaction would free disk
	// space.
	ActiveSize int64 `json:"data_size"`
	// ExternalSize is the size of the documents in the database, as represented
	// as JSON, before compression.
	ExternalSize int64 `json:"-"`
	// Cluster reports the cluster replication configuration variables.
	Cluster *ClusterConfig `json:"cluster,omitempty"`
	// RawResponse is the raw response body returned by the server, useful if
	// you need additional backend-specific information.
	//
	// For the format of this document, see
	// http://docs.couchdb.org/en/2.1.1/api/database/common.html#get--db
	RawResponse json.RawMessage `json:"-"`
}

// ClusterConfig contains the cluster configuration for the database.
type ClusterConfig struct {
	Replicas    int `json:"n"`
	Shards      int `json:"q"`
	ReadQuorum  int `json:"r"`
	WriteQuorum int `json:"w"`
}

// Stats returns database statistics.
//
// See https://docs.couchdb.org/en/stable/api/database/common.html#get--db
func (db *DB) Stats(ctx context.Context) (*DBStats, error) {
	if db.err != nil {
		return nil, db.err
	}
	i, err := db.driverDB.Stats(ctx)
	if err != nil {
		return nil, err
	}
	return driverStats2kivikStats(i), nil
}

func driverStats2kivikStats(i *driver.DBStats) *DBStats {
	var cluster *ClusterConfig
	if i.Cluster != nil {
		c := ClusterConfig(*i.Cluster)
		cluster = &c
	}
	return &DBStats{
		Name:           i.Name,
		CompactRunning: i.CompactRunning,
		DocCount:       i.DocCount,
		DeletedCount:   i.DeletedCount,
		UpdateSeq:      i.UpdateSeq,
		DiskSize:       i.DiskSize,
		ActiveSize:     i.ActiveSize,
		ExternalSize:   i.ExternalSize,
		Cluster:        cluster,
		RawResponse:    i.RawResponse,
	}
}

// Compact begins compaction of the database. Check the CompactRunning field
// returned by Info() to see if the compaction has completed.
// See http://docs.couchdb.org/en/2.0.0/api/database/compact.html#db-compact
//
// This method may return immediately, or may wait for the compaction to
// complete before returning, depending on the backend implementation. In
// particular, CouchDB triggers the compaction and returns immediately, whereas
// PouchDB waits until compaction has completed, before returning.
func (db *DB) Compact(ctx context.Context) error {
	if db.err != nil {
		return db.err
	}
	return db.driverDB.Compact(ctx)
}

// CompactView compats the view indexes associated with the specified design
// document.
// See http://docs.couchdb.org/en/2.0.0/api/database/compact.html#db-compact-design-doc
//
// This method may return immediately, or may wait for the compaction to
// complete before returning, depending on the backend implementation. In
// particular, CouchDB triggers the compaction and returns immediately, whereas
// PouchDB waits until compaction has completed, before returning.
func (db *DB) CompactView(ctx context.Context, ddocID string) error {
	return db.driverDB.CompactView(ctx, ddocID)
}

// ViewCleanup removes view index files that are no longer required as a result
// of changed views within design documents.
// See http://docs.couchdb.org/en/2.0.0/api/database/compact.html#db-view-cleanup
func (db *DB) ViewCleanup(ctx context.Context) error {
	if db.err != nil {
		return db.err
	}
	return db.driverDB.ViewCleanup(ctx)
}

// Security returns the database's security document.
// See http://couchdb.readthedocs.io/en/latest/api/database/security.html#get--db-_security
func (db *DB) Security(ctx context.Context) (*Security, error) {
	if db.err != nil {
		return nil, db.err
	}
	s, err := db.driverDB.Security(ctx)
	if err != nil {
		return nil, err
	}
	return &Security{
		Admins:  Members(s.Admins),
		Members: Members(s.Members),
	}, err
}

// SetSecurity sets the database's security document.
// See http://couchdb.readthedocs.io/en/latest/api/database/security.html#put--db-_security
func (db *DB) SetSecurity(ctx context.Context, security *Security) error {
	if db.err != nil {
		return db.err
	}
	if security == nil {
		return missingArg("security")
	}
	sec := &driver.Security{
		Admins:  driver.Members(security.Admins),
		Members: driver.Members(security.Members),
	}
	return db.driverDB.SetSecurity(ctx, sec)
}

// Copy copies the source document to a new document with an ID of targetID. If
// the database backend does not support COPY directly, the operation will be
// emulated with a Get followed by Put. The target will be an exact copy of the
// source, with only the ID and revision changed.
//
// See http://docs.couchdb.org/en/2.0.0/api/document/common.html#copy--db-docid
func (db *DB) Copy(ctx context.Context, targetID, sourceID string, options ...Options) (targetRev string, err error) {
	if db.err != nil {
		return "", db.err
	}
	if targetID == "" {
		return "", missingArg("targetID")
	}
	if sourceID == "" {
		return "", missingArg("sourceID")
	}
	opts := mergeOptions(options...)
	if copier, ok := db.driverDB.(driver.Copier); ok {
		return copier.Copy(ctx, targetID, sourceID, opts)
	}
	var doc map[string]interface{}
	if err = db.Get(ctx, sourceID, opts).ScanDoc(&doc); err != nil {
		return "", err
	}
	delete(doc, "_rev")
	doc["_id"] = targetID
	delete(opts, "rev") // rev has a completely different meaning for Copy and Put
	return db.Put(ctx, targetID, doc, opts)
}

// PutAttachment uploads the supplied content as an attachment to the specified
// document.
func (db *DB) PutAttachment(ctx context.Context, docID, rev string, att *Attachment, options ...Options) (newRev string, err error) {
	if db.err != nil {
		return "", db.err
	}
	if docID == "" {
		return "", missingArg("docID")
	}
	if e := att.validate(); e != nil {
		return "", e
	}
	a := driver.Attachment(*att)
	opts := mergeOptions(options...)
	if rv, ok := opts["rev"].(string); ok && rv != "" {
		rev = rv
	}
	return db.driverDB.PutAttachment(ctx, docID, rev, &a, mergeOptions(options...))
}

// GetAttachment returns a file attachment associated with the document.
func (db *DB) GetAttachment(ctx context.Context, docID, filename string, options ...Options) (*Attachment, error) {
	if db.err != nil {
		return nil, db.err
	}
	if docID == "" {
		return nil, missingArg("docID")
	}
	if filename == "" {
		return nil, missingArg("filename")
	}
	att, err := db.driverDB.GetAttachment(ctx, docID, filename, mergeOptions(options...))
	if err != nil {
		return nil, err
	}
	a := Attachment(*att)
	return &a, nil
}

type nilContentReader struct{}

var _ io.ReadCloser = &nilContentReader{}

func (c nilContentReader) Read(_ []byte) (int, error) { return 0, io.EOF }
func (c nilContentReader) Close() error               { return nil }

var nilContent = nilContentReader{}

// GetAttachmentMeta returns meta data about an attachment. The attachment
// content returned will be empty.
func (db *DB) GetAttachmentMeta(ctx context.Context, docID, filename string, options ...Options) (*Attachment, error) {
	if db.err != nil {
		return nil, db.err
	}
	if docID == "" {
		return nil, missingArg("docID")
	}
	if filename == "" {
		return nil, missingArg("filename")
	}
	var att *Attachment
	if metaer, ok := db.driverDB.(driver.AttachmentMetaGetter); ok {
		a, err := metaer.GetAttachmentMeta(ctx, docID, filename, mergeOptions(options...))
		if err != nil {
			return nil, err
		}
		att = new(Attachment)
		*att = Attachment(*a)
	} else {
		var err error
		att, err = db.GetAttachment(ctx, docID, filename, options...)
		if err != nil {
			return nil, err
		}
	}
	if att.Content != nil {
		_ = att.Content.Close() // Ensure this is closed
	}
	att.Content = nilContent
	return att, nil
}

// DeleteAttachment deletes an attachment from a document, returning the
// document's new revision. The revision may be provided via options, which
// takes priority over the rev argument.
func (db *DB) DeleteAttachment(ctx context.Context, docID, rev, filename string, options ...Options) (newRev string, err error) {
	if db.err != nil {
		return "", db.err
	}
	if docID == "" {
		return "", missingArg("docID")
	}
	if filename == "" {
		return "", missingArg("filename")
	}
	opts := mergeOptions(options...)
	if rv, ok := opts["rev"].(string); ok && rv != "" {
		rev = rv
	}
	return db.driverDB.DeleteAttachment(ctx, docID, rev, filename, opts)
}

// PurgeResult is the result of a purge request.
type PurgeResult struct {
	// Seq is the purge sequence number.
	Seq int64 `json:"purge_seq"`
	// Purged is a map of document ids to revisions, indicated the
	// document/revision pairs that were successfully purged.
	Purged map[string][]string `json:"purged"`
}

// Purge permanently removes the reference to deleted documents from the
// database. Normal deletion only marks the document with the key/value pair
// `_deleted=true`, to ensure proper replication of deleted documents. By
// using Purge, the document can be completely removed. But note that this
// operation is not replication safe, so great care must be taken when using
// Purge, and this should only be used as a last resort.
//
// Purge expects as input a map with document ID as key, and slice of
// revisions as value.
func (db *DB) Purge(ctx context.Context, docRevMap map[string][]string) (*PurgeResult, error) {
	if db.err != nil {
		return nil, db.err
	}
	if purger, ok := db.driverDB.(driver.Purger); ok {
		res, err := purger.Purge(ctx, docRevMap)
		if err != nil {
			return nil, err
		}
		r := PurgeResult(*res)
		return &r, nil
	}
	return nil, &Error{HTTPStatus: http.StatusNotImplemented, Message: "kivik: purge not supported by driver"}
}

// BulkGetReference is a reference to a document given in a BulkGet query.
type BulkGetReference struct {
	ID        string `json:"id"`
	Rev       string `json:"rev,omitempty"`
	AttsSince string `json:"atts_since,omitempty"`
}

// BulkGet can be called to query several documents in bulk. It is well suited
// for fetching a specific revision of documents, as replicators do for example,
// or for getting revision history.
//
// See http://docs.couchdb.org/en/stable/api/database/bulk-api.html#db-bulk-get
func (db *DB) BulkGet(ctx context.Context, docs []BulkGetReference, options ...Options) (*Rows, error) {
	if db.err != nil {
		return nil, db.err
	}
	bulkGetter, ok := db.driverDB.(driver.BulkGetter)
	if !ok {
		return nil, &Error{HTTPStatus: http.StatusNotImplemented, Message: "kivik: bulk get not supported by driver"}
	}
	refs := make([]driver.BulkGetReference, len(docs))
	for i, ref := range docs {
		refs[i] = driver.BulkGetReference(ref)
	}
	rowsi, err := bulkGetter.BulkGet(ctx, refs, mergeOptions(options...))
	if err != nil {
		return nil, err
	}
	return newRows(ctx, rowsi), nil
}

// Close cleans up any resources used by the DB. The default CouchDB driver
// does not use this, the default PouchDB driver does.
func (db *DB) Close(ctx context.Context) error {
	if db.err != nil {
		return db.err
	}
	if closer, ok := db.driverDB.(driver.DBCloser); ok {
		return closer.Close(ctx)
	}
	return nil
}

// RevDiff represents a rev diff for a single document, as returned by the
// RevsDiff method.
type RevDiff struct {
	Missing           []string `json:"missing,omitempty"`
	PossibleAncestors []string `json:"possible_ancestors,omitempty"`
}

// Diffs is a collection of RevDiffs as returned by RevsDiff. The map key is
// the document ID.
type Diffs map[string]RevDiff

// RevsDiff the subset of document/revision IDs that do not correspond to
// revisions stored in the database. This is used by the replication protocol,
// and is normally never needed otherwise.  revMap must marshal to the expected
// format.
//
// Use ID() to return the current document ID, and ScanValue to access the full
// JSON value, which should be of the JSON format:
//
//     {
//         "missing": ["rev1",...],
//         "possible_ancestors": ["revA",...]
//     }
//
// See http://docs.couchdb.org/en/stable/api/database/misc.html#db-revs-diff
func (db *DB) RevsDiff(ctx context.Context, revMap interface{}) (*Rows, error) {
	if db.err != nil {
		return nil, db.err
	}
	if rd, ok := db.driverDB.(driver.RevsDiffer); ok {
		rowsi, err := rd.RevsDiff(ctx, revMap)
		if err != nil {
			return nil, err
		}
		return newRows(ctx, rowsi), nil
	}
	return nil, &Error{HTTPStatus: http.StatusNotImplemented, Message: "kivik: _revs_diff not supported by driver"}
}

// PartitionStats contains partition statistics.
type PartitionStats struct {
	DBName          string
	DocCount        int64
	DeletedDocCount int64
	Partition       string
	ActiveSize      int64
	ExternalSize    int64
	RawResponse     json.RawMessage
}

// PartitionStats returns statistics about the named partition.
//
// See https://docs.couchdb.org/en/stable/api/partitioned-dbs.html#db-partition-partition
func (db *DB) PartitionStats(ctx context.Context, name string) (*PartitionStats, error) {
	if db.err != nil {
		return nil, db.err
	}
	if pdb, ok := db.driverDB.(driver.PartitionedDB); ok {
		stats, err := pdb.PartitionStats(ctx, name)
		if err != nil {
			return nil, err
		}
		s := PartitionStats(*stats)
		return &s, nil
	}
	return nil, &Error{HTTPStatus: http.StatusNotImplemented, Message: "kivik: partitions not supported by driver"}
}
