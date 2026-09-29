package scalers

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/stats"

	pb "github.com/kedacore/keda/v2/pkg/scalers/externalscaler"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

// connCounter counts transport connections as the server sees them, so the
// assertions below rest on observed TCP lifecycle rather than on the client's
// own view of its connectivity state.
type connCounter struct {
	mu     sync.Mutex
	begun  int
	ended  int
	closed chan struct{}
	once   sync.Once
}

func (c *connCounter) TagRPC(ctx context.Context, _ *stats.RPCTagInfo) context.Context   { return ctx }
func (c *connCounter) HandleRPC(context.Context, stats.RPCStats)                         {}
func (c *connCounter) TagConn(ctx context.Context, _ *stats.ConnTagInfo) context.Context { return ctx }

func (c *connCounter) HandleConn(_ context.Context, s stats.ConnStats) {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch s.(type) {
	case *stats.ConnBegin:
		c.begun++
	case *stats.ConnEnd:
		c.ended++
		c.once.Do(func() { close(c.closed) })
	}
}

func (c *connCounter) counts() (int, int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.begun, c.ended
}

// A pooled connection is a real socket to the external scaler. Closing the last
// scaler that uses it must close that socket, which is the leak this change
// exists to remove. grpc.NewClient connects lazily, so each scaler issues one
// call to force the transport up before anything is asserted.
func TestExternalScalerReleasesTheTCPConnection(t *testing.T) {
	counter := &connCounter{closed: make(chan struct{})}

	// nosemgrep: go.grpc.security.grpc-server-insecure-connection.grpc-server-insecure-connection
	grpcServer := grpc.NewServer(grpc.StatsHandler(counter))
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	address := lis.Addr().String()
	pb.RegisterExternalScalerServer(grpcServer, &testExternalScaler{t: t})
	go func() {
		_ = grpcServer.Serve(lis)
	}()
	defer grpcServer.Stop()

	newScaler := func() Scaler {
		s, err := NewExternalScaler(&scalersconfig.ScalerConfig{
			ScalableObjectName:      "app",
			ScalableObjectNamespace: "namespace",
			TriggerMetadata:         map[string]string{"scalerAddress": address},
			ResolvedEnv:             map[string]string{},
		})
		if err != nil {
			t.Fatalf("NewExternalScaler: %v", err)
		}
		// grpc.NewClient dials lazily, so issue one call to bring the transport up.
		// The metric name carries the trigger index prefix the scaler strips before
		// the request, without which it returns early and never reaches the wire.
		// The server leaves the method unimplemented, so the call fails; the socket
		// it opens on the way is the point.
		_, _, _ = s.GetMetricsAndActivity(context.Background(), "s0-metric")
		return s
	}

	first := newScaler()
	second := newScaler()

	// Both scalers share one pooled connection, so the server sees one socket.
	// ConnBegin is delivered from the transport goroutine, so give it a moment.
	var begun, ended int
	for range 50 {
		begun, ended = counter.counts()
		if begun > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if begun != 1 {
		t.Fatalf("connections opened = %d, want 1, the two scalers should share one", begun)
	}
	if ended != 0 {
		t.Fatalf("connections closed = %d, want 0, both scalers are still open", ended)
	}

	// The first scaler goes away. The socket is still in use.
	if err := first.Close(context.Background()); err != nil {
		t.Fatalf("closing the first scaler: %v", err)
	}
	select {
	case <-counter.closed:
		t.Fatal("the connection was closed while the second scaler was still using it")
	case <-time.After(300 * time.Millisecond):
	}

	// The last scaler goes away. The socket must go with it.
	if err := second.Close(context.Background()); err != nil {
		t.Fatalf("closing the second scaler: %v", err)
	}
	select {
	case <-counter.closed:
	case <-time.After(5 * time.Second):
		begun, ended = counter.counts()
		t.Fatalf("the connection was never closed, opened=%d closed=%d", begun, ended)
	}

	begun, ended = counter.counts()
	t.Logf("server observed connections opened=%d closed=%d", begun, ended)
	if begun != ended {
		t.Errorf("opened=%d closed=%d, every connection the server accepted should be closed", begun, ended)
	}
}
