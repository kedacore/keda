// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.
//
// Modifications Copyright OpenSearch Contributors. See
// GitHub history for details.

// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package opensearchtransport

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Discoverable defines the interface for transports supporting node discovery.
type Discoverable interface {
	DiscoverNodes() error
}

// nodeInfo represents the information about node in a cluster.
type nodeInfo struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	URL        *url.URL               `json:"url"`
	Roles      []string               `json:"roles"`
	Attributes map[string]interface{} `json:"attributes"`
	HTTP       struct {
		PublishAddress string `json:"publish_address"`
	} `json:"http"`
}

// DiscoverNodes reloads the client connections by fetching information from the cluster.
func (c *Client) DiscoverNodes() error {
	conns := make([]*Connection, 0)

	nodes, err := c.getNodesInfo()
	if err != nil {
		if debugLogger != nil {
			debugLogger.Logf("Error getting nodes info: %s\n", err)
		}

		return fmt.Errorf("discovery: get nodes: %w", err)
	}

	for _, node := range nodes {
		var isClusterManagerOnlyNode bool

		if len(node.Roles) == 1 && (node.Roles[0] == "master" || node.Roles[0] == "cluster_manager") {
			isClusterManagerOnlyNode = true
		}

		if debugLogger != nil {
			var skip string
			if isClusterManagerOnlyNode {
				skip = "; [SKIP]"
			}

			debugLogger.Logf("Discovered node [%s]; %s; roles=%s%s\n", node.Name, node.URL, node.Roles, skip)
		}

		// Skip cluster_manager only nodes
		// TODO: Move logic to Selector?
		if isClusterManagerOnlyNode {
			continue
		}

		conns = append(conns, &Connection{
			URL:        node.URL,
			ID:         node.ID,
			Name:       node.Name,
			Roles:      node.Roles,
			Attributes: node.Attributes,
		})
	}

	c.Lock()
	defer c.Unlock()

	if lockable, ok := c.pool.(sync.Locker); ok {
		lockable.Lock()
		defer lockable.Unlock()
	}

	if c.poolFunc != nil {
		c.pool = c.poolFunc(conns, c.selector)
	} else {
		// TODO: Replace only live connections, leave dead scheduled for resurrect?
		c.pool = NewConnectionPool(conns, c.selector)
	}

	return nil
}

func (c *Client) getNodesInfo() ([]nodeInfo, error) {
	scheme := c.urls[0].Scheme

	req, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, "/_nodes/http", nil)
	if err != nil {
		return nil, err
	}

	c.Lock()
	conn, err := c.pool.Next()
	c.Unlock()
	// TODO: If no connection is returned, fallback to original URLs
	if err != nil {
		return nil, err
	}

	c.setReqURL(conn.URL, req)
	c.setReqAuth(conn.URL, req)
	c.setReqUserAgent(req)

	res, err := c.transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	if res.Body == nil {
		return nil, fmt.Errorf("unexpected empty body")
	}
	defer res.Body.Close()

	if res.StatusCode > http.StatusOK {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("server error: %s: %w", res.Status, err)
		}
		return nil, fmt.Errorf("server error: %s: %s", res.Status, body)
	}

	var env map[string]json.RawMessage
	if err := json.NewDecoder(res.Body).Decode(&env); err != nil {
		return nil, err
	}

	var nodes map[string]nodeInfo
	if err := json.Unmarshal(env["nodes"], &nodes); err != nil {
		return nil, err
	}

	out := make([]nodeInfo, len(nodes))
	idx := 0

	for id, node := range nodes {
		node.ID = id
		u := c.getNodeURL(node, scheme)
		node.URL = u
		out[idx] = node
		idx++
	}

	return out, nil
}

func (c *Client) getNodeURL(node nodeInfo, scheme string) *url.URL {
	var (
		host string
		port string
		err  error

		addrs = strings.Split(node.HTTP.PublishAddress, "/")
		ports = strings.Split(node.HTTP.PublishAddress, ":")
	)

	if len(addrs) > 1 {
		host = addrs[0]
	} else {
		host, _, err = net.SplitHostPort(addrs[0])
		if err != nil {
			host = strings.Split(addrs[0], ":")[0]
		}
	}
	port = ports[len(ports)-1]
	u := &url.URL{
		Scheme: scheme,
		Host:   net.JoinHostPort(host, port),
	}

	return u
}

func (c *Client) scheduleDiscoverNodes() {
	//nolint:errcheck // errors are logged inside the function
	go c.DiscoverNodes()

	c.Lock()
	defer c.Unlock()

	if c.discoverNodesTimer != nil {
		c.discoverNodesTimer.Stop()
	}

	c.discoverNodesTimer = time.AfterFunc(c.discoverNodesInterval, func() {
		c.scheduleDiscoverNodes()
	})
}
