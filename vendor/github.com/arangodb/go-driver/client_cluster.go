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

import "context"

// ClientCluster provides methods needed to access cluster functionality from a client.
type ClientCluster interface {
	// Cluster provides access to cluster wide specific operations.
	// To use this interface, an ArangoDB cluster is required.
	// If this method is a called without a cluster, a PreconditionFailed error is returned.
	Cluster(ctx context.Context) (Cluster, error)
}
