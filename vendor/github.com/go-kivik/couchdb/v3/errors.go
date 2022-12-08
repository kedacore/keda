package couchdb

import (
	"fmt"
	"net/http"

	kivik "github.com/go-kivik/kivik/v3"
)

func missingArg(arg string) error {
	return &kivik.Error{HTTPStatus: http.StatusBadRequest, Err: fmt.Errorf("kivik: %s required", arg)}
}
