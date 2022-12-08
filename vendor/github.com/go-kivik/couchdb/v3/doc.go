/*
Package couchdb is a driver for connecting with a CouchDB server over HTTP.

General Usage

Use the `couch` driver name when using this driver. The DSN should be a full
URL, likely with login credentials:

    import (
        kivik "github.com/go-kivik/kivik/v3"
        _ "github.com/go-kivik/couchdb/v3" // The CouchDB driver
    )

    client, err := kivik.New("couch", "http://username:password@127.0.0.1:5984/")

Options

The CouchDB driver generally interprets kivik.Options keys and values as URL
query parameters. Values of the following types will be converted to their
appropriate string representation when URL-encoded:

 - bool
 - string
 - []string
 - int, uint, uint8, uint16, uint32, uint64, int8, int16, int32, int64

Passing any other type will return an error.

The only exceptions to the above rule are:

 - the special option keys defined by the package constants `OptionFullCommit`
   and `OptionIfNoneMatch`. These options set the appropriate HTTP request
   headers rather than setting a URL parameter.
 - the `keys` key, when passed to a view query, will result in a POST query
   being done, rather than a GET, to accommodate an arbitrary number of keys.
 - the 'NoMultipartPut' option is interpreted by the Kivik CouchDB driver to
   disable multipart/related PUT uploads of attachments.
 - the 'NoMultipartGet' option is interpreted by the Kivik CouchDB driver to
   disable multipart/related GET downloads of attachments.

Authentication

The CouchDB driver supports a number of authentication methods. For most uses,
you don't need to worry about authentication at all--just include authentication
credentials in your connection DSN:

    client, _ := kivik.New("couch", "http://user:password@localhost:5984/")

This will use Cookie authentication by default.

To use one of the explicit authentication mechanisms, you'll need to use kivik's
Authenticate method.  For example:

    client, _ := kivik.New("couch", "http://localhost:5984/")
    err := client.Authenticate(ctx, couchdb.BasicAuth("bob", "abc123"))

Multipart PUT

Normally, to include an attachment in a CouchDB document, it must be base-64
encoded, which leads to increased network traffic and higher CPU load. CouchDB
also supports the option to upload multiple attachments in a single request
using the 'multipart/related' content type. See
http://docs.couchdb.org/en/stable/api/document/common.html#creating-multiple-attachments

As an experimental feature, this is now supported by the Kivik CouchDB driver as
well. To take advantage of this capability, the `doc` argument to the Put()
method must be either:

    - a map of type `map[string]interface{}`, with a key called `_attachments',
      and value of type `kivik.Attachments` or `*kivik.Attachments`
    - a struct, with a field having the tag `json:"_attachment"`, and the field
      having the type `kivik.Attachments` or `*kivik.Attachments`.

With this in place, the CouchDB driver will switch to `multipart/related` mode,
sending each attachment in binary format, rather than base-64 encoding it.

To function properly, each attachment must have an accurate Size value. If the
Size value is unset, the entirely attachment may be read to determine its size,
prior to sending it over the network, leading to delays and unnecessary I/O and
CPU usage. The simplest way to ensure efficiency is to use the NewAttachment()
method, provided by this package. See the documentation on that method for
proper usage.

Example:

    file, _ := os.Open("/path/to/photo.jpg")
    atts := &kivik.Attachments{
        "photo.jpg": NewAttachment("photo.jpg", "image/jpeg", file),
    }
    doc := map[string]interface{}{
        "_id":          "user123",
        "_attachments": atts,
    }
    rev, err := db.Put(ctx, "user123", doc)

To disable the `multipart/related` capabilities entirely, you may pass the
`NoMultipartPut` option, with any value. This will fallback to the default of
inline base-64 encoding the attachments.  Example:

    rev, err := db.Put(ctx, "user123", doc", kivik.Options{couchdb.NoMultipartPut: "xxx"})

If you find yourself wanting to disable this feature, due to bugs or performance,
please consider filing a bug report against Kivik as well, so we can look for a
solution that will allow using this optimization.
*/
package couchdb
