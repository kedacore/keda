package couchdb

// Version is the current version of this package.
const Version = "3.4.1"

const (
	// OptionFullCommit is the option key used to set the `X-Couch-Full-Commit`
	// header in the request when set to true.
	//
	// Example:
	//
	//    db.Put(ctx, "doc_id", doc, kivik.Options{couchdb.OptionFullCommit: true})
	OptionFullCommit = "X-Couch-Full-Commit"

	// OptionIfNoneMatch is an option key to set the If-None-Match header on
	// the request.
	//
	// Example:
	//
	//    row, err := db.Get(ctx, "doc_id", kivik.Options{couchdb.OptionIfNoneMatch: "1-xxx"})
	OptionIfNoneMatch = "If-None-Match"

	// OptionPartition instructs supporting methods to limit the query to the
	// specified partition. Supported methods are: Query, AllDocs, Find, and
	// Explain. Only supported by CouchDB 3.0.0 and newer.
	//
	// See https://docs.couchdb.org/en/stable/api/partitioned-dbs.html
	OptionPartition = "kivik:partition"

	// NoMultipartPut instructs the Put() method not to use CouchDB's
	// multipart/related upload capabilities. This only affects PUT requests that
	// also include attachments.
	NoMultipartPut = "kivik:no-multipart-put"

	// NoMultipartGet instructs the Get() method not to use CouchDB's ability to
	// download attachments with the multipart/related media type. This only
	// affects GET requests that request attachments.
	NoMultipartGet = "kivik:no-multipart-get"
)

const (
	typeJSON      = "application/json"
	typeMPRelated = "multipart/related"
)
