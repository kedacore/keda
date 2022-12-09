package couchdb

import (
	"encoding/json"
	"net/http"

	kivik "github.com/go-kivik/kivik/v3"
)

// encodeKey encodes a key to a view query, or similar, to be passed to CouchDB.
func encodeKey(i interface{}) (string, error) {
	if raw, ok := i.(json.RawMessage); ok {
		return string(raw), nil
	}
	raw, err := json.Marshal(i)
	if err != nil {
		err = &kivik.Error{HTTPStatus: http.StatusBadRequest, Err: err}
	}
	return string(raw), err
}

var jsonKeys = []string{"endkey", "end_key", "key", "startkey", "start_key", "keys", "doc_ids"}

func encodeKeys(opts map[string]interface{}) error {
	for _, key := range jsonKeys {
		if v, ok := opts[key]; ok {
			new, err := encodeKey(v)
			if err != nil {
				return err
			}
			opts[key] = new
		}
	}
	return nil
}
