package kivik

import (
	"context"
	"net/http"

	"github.com/go-kivik/kivik/v3/driver"
)

var clusterNotImplemented = &Error{HTTPStatus: http.StatusNotImplemented, Message: "kivik: driver does not support cluster operations"}

// ClusterStatus returns the current cluster status.
//
// See http://docs.couchdb.org/en/stable/api/server/common.html#cluster-setup
func (c *Client) ClusterStatus(ctx context.Context, options ...Options) (string, error) {
	cluster, ok := c.driverClient.(driver.Cluster)
	if !ok {
		return "", clusterNotImplemented
	}
	return cluster.ClusterStatus(ctx, mergeOptions(options...))
}

// ClusterSetup performs the requested cluster action. action should be
// an object understood by the driver. For the CouchDB driver, this means an
// object which is marshalable to a JSON object of the expected format.
//
// See http://docs.couchdb.org/en/stable/api/server/common.html#post--_cluster_setup
func (c *Client) ClusterSetup(ctx context.Context, action interface{}) error {
	cluster, ok := c.driverClient.(driver.Cluster)
	if !ok {
		return clusterNotImplemented
	}
	return cluster.ClusterSetup(ctx, action)
}

// ClusterMembership contains the list of known nodes, and cluster nodes, as returned
// by the /_membership endpoint.
// See https://docs.couchdb.org/en/latest/api/server/common.html#get--_membership
type ClusterMembership struct {
	AllNodes     []string `json:"all_nodes"`
	ClusterNodes []string `json:"cluster_nodes"`
}

// Membership returns a list of known CouchDB nodes.
// See https://docs.couchdb.org/en/latest/api/server/common.html#get--_membership
func (c *Client) Membership(ctx context.Context) (*ClusterMembership, error) {
	cluster, ok := c.driverClient.(driver.Cluster2)
	if !ok {
		return nil, &Error{HTTPStatus: http.StatusNotImplemented, Message: "kivik: driver does not support the /_membership endpoint"}
	}
	nodes, err := cluster.Membership(ctx)
	return (*ClusterMembership)(nodes), err
}
