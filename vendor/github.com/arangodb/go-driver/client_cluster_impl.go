//
// DISCLAIMER
//
// Copyright 2017 ArangoDB GmbH, Cologne, Germany
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Copyright holder is ArangoDB GmbH, Cologne, Germany
//
// Author Ewout Prangsma
//

package driver

import (
	"context"
)

// Cluster provides access to cluster wide specific operations.
// To use this interface, an ArangoDB cluster is required.
// If this method is a called without a cluster, a PreconditionFailed error is returned.
func (c *client) Cluster(ctx context.Context) (Cluster, error) {
	role, err := c.ServerRole(ctx)
	if err != nil {
		return nil, WithStack(err)
	}
	if role == ServerRoleSingle || role == ServerRoleSingleActive || role == ServerRoleSinglePassive {
		// Standalone server, this is wrong
		return nil, WithStack(newArangoError(412, 0, "Cluster expected, found SINGLE server"))
	}
	cl, err := newCluster(c.conn)
	if err != nil {
		return nil, WithStack(err)
	}
	return cl, nil
}
