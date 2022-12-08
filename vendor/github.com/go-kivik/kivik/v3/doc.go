/*
Package kivik provides a generic interface to CouchDB or CouchDB-like databases.

The kivik package must be used in conjunction with a database driver. The
officially supported drivers are:

 - CouchDB: https://github.com/go-kivik/couchdb
 - PouchDB: https://github.com/go-kivik/pouchdb (requires GopherJS)
 - KivikMock: https://github.com/go-kivik/kivikmock

The Filesystem and Memory drivers are also available, but in early stages of
development, and so many features do not yet work:

 - Filesystem: https://github.com/go-kivik/fsdb
 - MemroyDB: https://github.com/go-kivik/memorydb

The kivik driver system is modeled after the standard library's `sql` and
`sql/driver` packages, although the client API is completely different due to
the different database models implemented by SQL and NoSQL databases such as
CouchDB.

Working with JSON

couchDB stores JSON, so Kivik translates Go data structures to and from JSON as
necessary. The conversion between Go data types and JSON, and vice versa, is
handled automatically according to the rules and behavior described in the
documentationf or the standard library's `encoding/json` package
(https://golang.org/pkg/encoding/json).

One would be well-advised to become familiar with using `json` struct field
tags (https://golang.org/pkg/encoding/json/#Marshal) when working with JSON
documents.

Using contexts

Most Kivik methods take `context.Context` as their first argument. This allows
the cancellation of blocking operations in the case that the result is no
longer needed. A typical use case for a web application would be to cancel a
Kivik request if the remote HTTP client ahs disconnected, rednering the results
of the query irrelevant.

To learn more about Go's contexts, read the `context` package documentation
(https://golang.org/pkg/context/) and read the Go blog post "Go Concurrency
Patterns: Context" (https://blog.golang.org/context) for example code.

If in doubt, you can pass `context.TODO()` as the context variable. Example:

    row := db.Get(context.TODO(), "some_doc_id")

Error Handling

Kivik returns errors that embed an HTTP status code. In most cases, this is the
HTTP status code returned by the server. The embedded HTTP status code may be
accessed easily using the StatusCode() method, or with a type assertion to
`interface { StatusCode() int }`. Example:

    if statusErr, ok := err.(interface{ StatusCode() int }); ok {
		status = statusErr.StatusCode()
	}

Any error that does not conform to this interface will be assumed to represent
a http.StatusInternalServerError status code.

Authentication

For common usage, authentication should be as simple as including the authentication credentials in the connection DSN. For example:

    client, err := kivik.New("couch", "http://admin:abc123@localhost:5984/")

This will connect to `localhost` on port 5984, using the username `admin` and
the password `abc123`. When connecting to CouchDB (as in the above example),
this will use cookie auth (https://docs.couchdb.org/en/stable/api/server/authn.html?highlight=cookie%20auth#cookie-authentication).

Depending on which driver you use, there may be other ways to authenticate, as
well. At the moment, the CouchDB driver is the only official driver which offers
additional authentication methods. Please refer to the CouchDB package
documentation for details (https://pkg.go.dev/github.com/go-kivik/couchdb/v3).

*/
package kivik // import "github.com/go-kivik/kivik/v3"
