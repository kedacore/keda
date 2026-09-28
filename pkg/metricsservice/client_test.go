/*
Copyright 2026 The KEDA Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metricsservice

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

// TestGetConnectionState is a regression test for issue #8211: the metrics
// adapter needs a way to observe the gRPC connection state so a readiness
// probe can report NotReady when the connection to the KEDA metrics service
// is down. GetConnectionState must reflect the underlying connection's state
// without blocking.
func TestGetConnectionState(t *testing.T) {
	// A freshly created client (grpc.NewClient does not dial until first use)
	// starts in the Idle state, which is not Ready. A readiness probe must be
	// able to observe this so it does not report a not-yet-connected replica
	// as Ready.
	conn, err := grpc.NewClient("passthrough:///127.0.0.1:0",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	c := &GrpcClient{connection: conn}

	// The getter must report exactly what the underlying connection reports,
	// and must not block.
	require.Equal(t, conn.GetState(), c.GetConnectionState())

	// A connection that has never dialed is not Ready, so a readiness check
	// built on this getter would correctly mark the replica NotReady.
	require.NotEqual(t, connectivity.Ready, c.GetConnectionState())
}
