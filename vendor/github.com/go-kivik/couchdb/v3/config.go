package couchdb

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/go-kivik/couchdb/v3/chttp"
	"github.com/go-kivik/kivik/v3/driver"
)

// Couch1ConfigNode can be passed to any of the Config-related methods as the
// node name, to query the /_config endpoint in a CouchDB 1.x-compatible way.
const Couch1ConfigNode = "<Couch1Config>"

var _ driver.Configer = &client{}

func configURL(node string, parts ...string) string {
	var components []string
	if node == Couch1ConfigNode {
		components = append(make([]string, 0, len(parts)+1),
			"_config")
	} else {
		components = append(make([]string, 0, len(parts)+3),
			"_node", node, "_config",
		)
	}
	components = append(components, parts...)
	return "/" + strings.Join(components, "/")
}

func (c *client) Config(ctx context.Context, node string) (driver.Config, error) {
	cf := driver.Config{}
	_, err := c.Client.DoJSON(ctx, http.MethodGet, configURL(node), nil, &cf)
	return cf, err
}

func (c *client) ConfigSection(ctx context.Context, node, section string) (driver.ConfigSection, error) {
	sec := driver.ConfigSection{}
	_, err := c.Client.DoJSON(ctx, http.MethodGet, configURL(node, section), nil, &sec)
	return sec, err
}

func (c *client) ConfigValue(ctx context.Context, node, section, key string) (string, error) {
	var value string
	_, err := c.Client.DoJSON(ctx, http.MethodGet, configURL(node, section, key), nil, &value)
	return value, err
}

func (c *client) SetConfigValue(ctx context.Context, node, section, key, value string) (string, error) {
	body, _ := json.Marshal(value) // Strings never cause JSON marshaling errors
	var old string
	opts := &chttp.Options{
		Body: ioutil.NopCloser(bytes.NewReader(body)),
	}
	_, err := c.Client.DoJSON(ctx, http.MethodPut, configURL(node, section, key), opts, &old)
	return old, err
}

func (c *client) DeleteConfigKey(ctx context.Context, node, section, key string) (string, error) {
	var value string
	_, err := c.Client.DoJSON(ctx, http.MethodDelete, configURL(node, section, key), nil, &value)
	return value, err
}
