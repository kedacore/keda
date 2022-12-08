package kivik

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-kivik/kivik/v3/driver"
	"github.com/go-kivik/kivik/v3/internal/registry"
)

// Client is a client connection handle to a CouchDB-like server.
type Client struct {
	dsn          string
	driverName   string
	driverClient driver.Client
}

// Options is a collection of options. The keys and values are backend specific.
type Options map[string]interface{}

func mergeOptions(otherOpts ...Options) Options {
	if len(otherOpts) == 0 {
		return nil
	}
	options := make(Options)
	for _, opts := range otherOpts {
		for k, v := range opts {
			options[k] = v
		}
	}
	if len(options) == 0 {
		return nil
	}
	return options
}

// Register makes a database driver available by the provided name. If Register
// is called twice with the same name or if driver is nil, it panics.
func Register(name string, driver driver.Driver) {
	registry.Register(name, driver)
}

// New creates a new client object specified by its database driver name
// and a driver-specific data source name.
func New(driverName, dataSourceName string) (*Client, error) {
	driveri := registry.Driver(driverName)
	if driveri == nil {
		return nil, &Error{HTTPStatus: http.StatusBadRequest, Message: fmt.Sprintf("kivik: unknown driver %q (forgotten import?)", driverName)}
	}
	client, err := driveri.NewClient(dataSourceName)
	if err != nil {
		return nil, err
	}
	return &Client{
		dsn:          dataSourceName,
		driverName:   driverName,
		driverClient: client,
	}, nil
}

// Driver returns the name of the driver string used to connect this client.
func (c *Client) Driver() string {
	return c.driverName
}

// DSN returns the data source name used to connect this client.
func (c *Client) DSN() string {
	return c.dsn
}

// Version represents a server version response.
type Version struct {
	// Version is the version number reported by the server or backend.
	Version string
	// Vendor is the vendor string reported by the server or backend.
	Vendor string
	// Features is a list of enabled, optional features.  This was added in
	// CouchDB 2.1.0, and can be expected to be empty for older versions.
	Features []string
	// RawResponse is the raw response body returned by the server, useful if
	// you need additional backend-specific information.
	//
	// For the format of this document, see
	// http://docs.couchdb.org/en/2.0.0/api/server/common.html#get
	RawResponse json.RawMessage
}

// Version returns version and vendor info about the backend.
func (c *Client) Version(ctx context.Context) (*Version, error) {
	ver, err := c.driverClient.Version(ctx)
	if err != nil {
		return nil, err
	}
	v := &Version{}
	*v = Version(*ver)
	return v, nil
}

// DB returns a handle to the requested database. Any options parameters
// passed are merged, with later values taking precidence. If any errors occur
// at this stage, they are deferred, or may be checked directly with Err()
func (c *Client) DB(ctx context.Context, dbName string, options ...Options) *DB {
	db, err := c.driverClient.DB(ctx, dbName, mergeOptions(options...))
	return &DB{
		client:   c,
		name:     dbName,
		driverDB: db,
		err:      err,
	}
}

// AllDBs returns a list of all databases.
func (c *Client) AllDBs(ctx context.Context, options ...Options) ([]string, error) {
	return c.driverClient.AllDBs(ctx, mergeOptions(options...))
}

// DBExists returns true if the specified database exists.
func (c *Client) DBExists(ctx context.Context, dbName string, options ...Options) (bool, error) {
	return c.driverClient.DBExists(ctx, dbName, mergeOptions(options...))
}

// CreateDB creates a DB of the requested name. Any errors are deferred, or may
// be checked with Err().
func (c *Client) CreateDB(ctx context.Context, dbName string, options ...Options) error {
	return c.driverClient.CreateDB(ctx, dbName, mergeOptions(options...))
}

// DestroyDB deletes the requested DB.
func (c *Client) DestroyDB(ctx context.Context, dbName string, options ...Options) error {
	return c.driverClient.DestroyDB(ctx, dbName, mergeOptions(options...))
}

// Authenticate authenticates the client with the passed authenticator, which
// is driver-specific. If the driver does not understand the authenticator, an
// error will be returned.
func (c *Client) Authenticate(ctx context.Context, a interface{}) error {
	if auth, ok := c.driverClient.(driver.Authenticator); ok {
		return auth.Authenticate(ctx, a)
	}
	return &Error{HTTPStatus: http.StatusNotImplemented, Message: "kivik: driver does not support authentication"}
}

func missingArg(arg string) error {
	return &Error{HTTPStatus: http.StatusBadRequest, Message: fmt.Sprintf("kivik: %s required", arg)}
}

// DBsStats returns database statistics about one or more databases.
func (c *Client) DBsStats(ctx context.Context, dbnames []string) ([]*DBStats, error) {
	dbstats, err := c.nativeDBsStats(ctx, dbnames)
	switch StatusCode(err) {
	case http.StatusNotFound, http.StatusNotImplemented:
		return c.fallbackDBsStats(ctx, dbnames)
	}
	return dbstats, err
}

func (c *Client) fallbackDBsStats(ctx context.Context, dbnames []string) ([]*DBStats, error) {
	dbstats := make([]*DBStats, len(dbnames))
	for i, dbname := range dbnames {
		db := c.DB(ctx, dbname)
		stat, err := db.Stats(ctx)
		if err != nil {
			return nil, err
		}
		dbstats[i] = stat
	}
	return dbstats, nil
}

func (c *Client) nativeDBsStats(ctx context.Context, dbnames []string) ([]*DBStats, error) {
	statser, ok := c.driverClient.(driver.DBsStatser)
	if !ok {
		return nil, &Error{HTTPStatus: http.StatusNotImplemented, Message: "kivik: not supported by driver"}
	}
	stats, err := statser.DBsStats(ctx, dbnames)
	if err != nil {
		return nil, err
	}
	dbstats := make([]*DBStats, len(stats))
	for i, stat := range stats {
		dbstats[i] = driverStats2kivikStats(stat)
	}
	return dbstats, nil
}

// Ping returns true if the database is online and available for requests,
// for instance by querying the /_up endpoint. If the underlying driver
// supports the Pinger interface, it will be used. Otherwise, a fallback is
// made to calling Version.
func (c *Client) Ping(ctx context.Context) (bool, error) {
	if pinger, ok := c.driverClient.(driver.Pinger); ok {
		return pinger.Ping(ctx)
	}
	_, err := c.driverClient.Version(ctx)
	return err == nil, err
}

// Close cleans up any resources used by Client.
func (c *Client) Close(ctx context.Context) error {
	if closer, ok := c.driverClient.(driver.ClientCloser); ok {
		return closer.Close(ctx)
	}
	return nil
}
