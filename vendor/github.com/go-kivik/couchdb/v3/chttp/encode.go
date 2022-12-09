package chttp

import (
	"net/url"
	"strings"
)

const (
	prefixDesign = "_design/"
	prefixLocal  = "_local/"
)

// EncodeDocID encodes a document ID according to CouchDB's path encoding rules.
//
// In particular:
// -  '_design/' and '_local/' prefixes are unaltered.
// - The rest of the docID is Query-URL encoded, except that spaces are converted to %20. See https://github.com/apache/couchdb/issues/3565 for an
// explanation.
func EncodeDocID(docID string) string {
	for _, prefix := range []string{prefixDesign, prefixLocal} {
		if strings.HasPrefix(docID, prefix) {
			return prefix + encodeDocID(strings.TrimPrefix(docID, prefix))
		}
	}
	return encodeDocID(docID)
}

func encodeDocID(docID string) string {
	docID = url.QueryEscape(docID)
	return strings.Replace(docID, "+", "%20", -1) // Ensure space is encoded as %20, not '+', so that if CouchDB ever fixes the encoding, we won't break
}
