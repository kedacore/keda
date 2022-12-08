package kivik

const (
	// KivikVersion is the version of the Kivik library.
	KivikVersion = "3.1.0"
	// KivikVendor is the vendor string reported by this library.
	KivikVendor = "Kivik"
)

// SessionCookieName is the name of the CouchDB session cookie.
const SessionCookieName = "AuthSession"

// UserPrefix is the mandatory CouchDB user prefix.
// See http://docs.couchdb.org/en/2.0.0/intro/security.html#org-couchdb-user
const UserPrefix = "org.couchdb.user:"

// EndKeySuffix is a high Unicode character (0xfff0) useful for appending to an
// endkey argument, when doing a ranged search, as described here:
// http://couchdb.readthedocs.io/en/latest/ddocs/views/collation.html#string-ranges
//
// Example, to return all results with keys beginning with "foo":
//
//    rows, err := db.Query(context.TODO(), "ddoc", "view", map[string]interface{}{
//        "startkey": "foo",
//        "endkey":   "foo" + kivik.EndKeySuffix,
//    })
const EndKeySuffix = string(rune(0xfff0))
