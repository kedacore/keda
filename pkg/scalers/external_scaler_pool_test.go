package scalers

import (
	"context"
	"testing"

	"google.golang.org/grpc/connectivity"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

func poolEntries() int {
	n := 0
	connectionPool.Range(func(_, _ any) bool {
		n++
		return true
	})
	return n
}

func newTestExternalScaler(t *testing.T, address string) Scaler {
	t.Helper()
	s, err := NewExternalScaler(&scalersconfig.ScalerConfig{
		ScalableObjectName:      "app",
		ScalableObjectNamespace: "namespace",
		TriggerMetadata:         map[string]string{"scalerAddress": address},
		ResolvedEnv:             map[string]string{},
	})
	if err != nil {
		t.Fatalf("NewExternalScaler: %v", err)
	}
	return s
}

// Scalers sharing connection properties share one pooled connection, and that
// connection is closed and dropped from the pool once the last of them is
// closed. Without the reference count the pool keeps the entry and the
// grpc.ClientConn stays open for the lifetime of the process.
func TestExternalScalerConnectionPoolReleasedOnClose(t *testing.T) {
	const address = "pool-release-test.default.svc.cluster.local:9090"

	before := poolEntries()

	first := newTestExternalScaler(t, address)
	second := newTestExternalScaler(t, address)

	if got := poolEntries(); got != before+1 {
		t.Fatalf("pool entries = %d, want %d, two scalers should share one connection", got, before+1)
	}

	var connGroup *connectionGroup
	connectionPool.Range(func(_, v any) bool {
		if cg, ok := v.(*connectionGroup); ok && cg.refCount == 2 {
			connGroup = cg
			return false
		}
		return true
	})
	if connGroup == nil {
		t.Fatal("expected a pooled connection with two users")
	}

	// The first scaler goes away. The connection is still in use.
	if err := first.Close(context.Background()); err != nil {
		t.Fatalf("closing the first scaler: %v", err)
	}
	if got := poolEntries(); got != before+1 {
		t.Errorf("pool entries = %d, want %d, the connection is still in use", got, before+1)
	}
	if got := connGroup.grpcConnection.GetState(); got == connectivity.Shutdown {
		t.Error("the connection was closed while another scaler was still using it")
	}

	// The last scaler goes away. Now the connection can be released.
	if err := second.Close(context.Background()); err != nil {
		t.Fatalf("closing the second scaler: %v", err)
	}
	if got := poolEntries(); got != before {
		t.Errorf("pool entries = %d, want %d, the connection was not dropped from the pool", got, before)
	}
	if got := connGroup.grpcConnection.GetState(); got != connectivity.Shutdown {
		t.Errorf("connection state = %v, want %v", got, connectivity.Shutdown)
	}
}

// Scalers pointing at different addresses do not share a connection, and
// releasing one leaves the other alone.
func TestExternalScalerConnectionPoolPerAddress(t *testing.T) {
	before := poolEntries()

	one := newTestExternalScaler(t, "pool-a.default.svc.cluster.local:9090")
	two := newTestExternalScaler(t, "pool-b.default.svc.cluster.local:9090")

	if got := poolEntries(); got != before+2 {
		t.Fatalf("pool entries = %d, want %d", got, before+2)
	}

	if err := one.Close(context.Background()); err != nil {
		t.Fatalf("closing the first scaler: %v", err)
	}
	if got := poolEntries(); got != before+1 {
		t.Errorf("pool entries = %d, want %d", got, before+1)
	}

	if err := two.Close(context.Background()); err != nil {
		t.Fatalf("closing the second scaler: %v", err)
	}
	if got := poolEntries(); got != before {
		t.Errorf("pool entries = %d, want %d", got, before)
	}
}
