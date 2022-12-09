package couchdb

import "bytes"

// sequenceID is a CouchDB update sequence ID. This is just a string, but has
// a special JSON unmarshaler to work with both CouchDB 2.0.0 (which uses
// normal) strings for sequence IDs, and earlier versions (which use integers)
type sequenceID string

// UnmarshalJSON satisfies the json.Unmarshaler interface.
func (id *sequenceID) UnmarshalJSON(data []byte) error {
	sid := sequenceID(bytes.Trim(data, `""`))
	*id = sid
	return nil
}
