package couchdb

import (
	"fmt"
	"net/http"

	kivik "github.com/go-kivik/kivik/v3"
)

func fullCommit(opts map[string]interface{}) (bool, error) {
	fc, ok := opts[OptionFullCommit]
	if !ok {
		return false, nil
	}
	fcBool, ok := fc.(bool)
	if !ok {
		return false, &kivik.Error{HTTPStatus: http.StatusBadRequest, Err: fmt.Errorf("kivik: option '%s' must be bool, not %T", OptionFullCommit, fc)}
	}
	delete(opts, OptionFullCommit)
	return fcBool, nil
}

func ifNoneMatch(opts map[string]interface{}) (string, error) {
	inm, ok := opts[OptionIfNoneMatch]
	if !ok {
		return "", nil
	}
	inmString, ok := inm.(string)
	if !ok {
		return "", &kivik.Error{HTTPStatus: http.StatusBadRequest, Err: fmt.Errorf("kivik: option '%s' must be string, not %T", OptionIfNoneMatch, inm)}
	}
	delete(opts, OptionIfNoneMatch)
	if inmString[0] != '"' {
		return `"` + inmString + `"`, nil
	}
	return inmString, nil
}
