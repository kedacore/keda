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
	"time"

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

	// A connection that has never dialed is not Ready, so a readiness check
	// built on this getter would correctly mark the replica NotReady.
	require.NotEqual(t, connectivity.Ready, c.GetConnectionState())
}

// TestGetConnectionStateReconnectsWhenIdle guards against the failure mode
// raised in review on PR #8226: once a non-Ready state removes the replica
// from the APIService endpoints, no RPCs flow through the client, so an Idle
// connection would never be woken and the replica could stay NotReady forever.
// GetConnectionState must therefore trigger a non-blocking reconnect when it
// observes an Idle connection, moving it off Idle so it can recover on its own.
func TestGetConnectionStateReconnectsWhenIdle(t *testing.T) {
	conn, err := grpc.NewClient("passthrough:///127.0.0.1:0",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	c := &GrpcClient{connection: conn}

	// A fresh client is parked in Idle until something drives it to connect.
	require.Equal(t, connectivity.Idle, conn.GetState())

	// Calling the getter on an Idle connection must kick a reconnection
	// attempt. The reported state is still the pre-nudge Idle, but the
	// underlying connection then leaves Idle without any RPC being issued.
	require.Equal(t, connectivity.Idle, c.GetConnectionState())

	require.Eventually(t, func() bool {
		return conn.GetState() != connectivity.Idle
	}, 2*time.Second, 10*time.Millisecond,
		"GetConnectionState should trigger a reconnect that moves the connection off Idle")
}
