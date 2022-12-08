package couchdb

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/go-kivik/couchdb/v3/chttp"
	kivik "github.com/go-kivik/kivik/v3"
	"github.com/go-kivik/kivik/v3/driver"
)

// Couch represents the parent driver instance.
type Couch struct {
	// If provided, UserAgent is appended to the User-Agent header on all
	// outbound requests.
	UserAgent string

	// If provided, HTTPClient will be used for requests to the CouchDB server.
	HTTPClient *http.Client
}

var _ driver.Driver = &Couch{}

func init() {
	kivik.Register("couch", &Couch{})
}

// Known vendor strings
const (
	VendorCouchDB  = "The Apache Software Foundation"
	VendorCloudant = "IBM Cloudant"
)

type client struct {
	*chttp.Client

	// schedulerDetected will be set once the scheduler has been detected.
	// It should only be accessed through the schedulerSupported() method.
	schedulerDetected *bool
	sdMU              sync.Mutex
}

var (
	_ driver.Client    = &client{}
	_ driver.DBUpdater = &client{}
)

// NewClient establishes a new connection to a CouchDB server instance. If
// auth credentials are included in the URL, they are used to authenticate using
// CookieAuth (or BasicAuth if compiled with GopherJS). If you wish to use a
// different auth mechanism, do not specify credentials here, and instead call
// Authenticate() later.
func (d *Couch) NewClient(dsn string) (driver.Client, error) {
	httpClient := d.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	chttpClient, err := chttp.NewWithClient(httpClient, dsn)
	if err != nil {
		return nil, err
	}
	chttpClient.UserAgents = []string{
		fmt.Sprintf("Kivik/%s", kivik.KivikVersion),
		fmt.Sprintf("Kivik CouchDB driver/%s", Version),
	}
	if d.UserAgent != "" {
		chttpClient.UserAgents = append(chttpClient.UserAgents, d.UserAgent)
	}
	return &client{
		Client: chttpClient,
	}, nil
}

func (c *client) DB(_ context.Context, dbName string, _ map[string]interface{}) (driver.DB, error) {
	if dbName == "" {
		return nil, missingArg("dbName")
	}
	return &db{
		client: c,
		dbName: url.PathEscape(dbName),
	}, nil
}
