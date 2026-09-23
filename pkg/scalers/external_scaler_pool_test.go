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

// poolEntryFor returns the pooled connection for an address, or nil when the
// pool holds none. Looking the entry up by its key keeps each test pinned to
// the connection it created, so a test cannot pass by finding an entry another
// test left in the shared pool.
func poolEntryFor(t *testing.T, address string) *connectionGroup {
	t.Helper()
	key, err := getConnectionPoolKey(externalScalerMetadata{ScalerAddress: address})
	if err != nil {
		t.Fatalf("getConnectionPoolKey: %v", err)
	}
	i, ok := connectionPool.Load(key)
	if !ok {
		return nil
	}
	connGroup, ok := i.(*connectionGroup)
	if !ok {
		t.Fatalf("pool entry for %s is a %T, want *connectionGroup", address, i)
	}
	return connGroup
}

// refCountFor returns the number of scalers sharing the connection for an
// address, and 0 when the pool holds no connection for it.
func refCountFor(t *testing.T, address string) int {
	t.Helper()
	connGroup := poolEntryFor(t, address)
	if connGroup == nil {
		return 0
	}
	return connGroup.refCount
}

// closeOnCleanup releases the scaler's share when the test ends, so a test that
// fails part way through does not leave an entry in the pool for the tests that
// follow. Close is idempotent, so this is a no-op once a test has closed the
// scaler itself.
func closeOnCleanup(t *testing.T, s Scaler) {
	t.Helper()
	t.Cleanup(func() {
		_ = s.Close(context.Background())
	})
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
	closeOnCleanup(t, s)
	return s
}

func newTestExternalPushScaler(t *testing.T, address string) PushScaler {
	t.Helper()
	s, err := NewExternalPushScaler(&scalersconfig.ScalerConfig{
		ScalableObjectName:      "app",
		ScalableObjectNamespace: "namespace",
		TriggerMetadata:         map[string]string{"scalerAddress": address},
		ResolvedEnv:             map[string]string{},
	})
	if err != nil {
		t.Fatalf("NewExternalPushScaler: %v", err)
	}
	closeOnCleanup(t, s)
	return s
}

// Scalers sharing connection properties share one pooled connection, and that
// connection is closed and dropped from the pool once the last of them is
// closed. Without the reference count the pool keeps the entry and the
// grpc.ClientConn stays open for the lifetime of the process.
func TestExternalScalerConnectionPoolReleasedOnClose(t *testing.T) {
	const address = "pool-release-test.default.svc.cluster.local:9090"

	first := newTestExternalScaler(t, address)
	second := newTestExternalScaler(t, address)

	connGroup := poolEntryFor(t, address)
	if connGroup == nil {
		t.Fatal("no pooled connection for the address the scalers were built with")
	}
	if got := connGroup.refCount; got != 2 {
		t.Fatalf("refCount = %d, want 2, two scalers should share one connection", got)
	}

	// The first scaler goes away. The connection is still in use.
	if err := first.Close(context.Background()); err != nil {
		t.Fatalf("closing the first scaler: %v", err)
	}
	if got := refCountFor(t, address); got != 1 {
		t.Errorf("refCount = %d, want 1, the connection is still in use", got)
	}
	if got := connGroup.grpcConnection.GetState(); got == connectivity.Shutdown {
		t.Error("the connection was closed while another scaler was still using it")
	}

	// The last scaler goes away. Now the connection can be released.
	if err := second.Close(context.Background()); err != nil {
		t.Fatalf("closing the second scaler: %v", err)
	}
	if entry := poolEntryFor(t, address); entry != nil {
		t.Error("the connection was not dropped from the pool")
	}
	if got := connGroup.grpcConnection.GetState(); got != connectivity.Shutdown {
		t.Errorf("connection state = %v, want %v", got, connectivity.Shutdown)
	}
}

// NewExternalPushScaler acquires a share of the same pool, and externalPushScaler
// takes Close from the externalScaler it embeds. A push scaler must therefore
// release its share like any other, including when it shares the connection
// with a plain scaler on the same address.
func TestExternalPushScalerConnectionPoolReleasedOnClose(t *testing.T) {
	const address = "pool-push-release.default.svc.cluster.local:9090"

	push := newTestExternalPushScaler(t, address)

	connGroup := poolEntryFor(t, address)
	if connGroup == nil {
		t.Fatal("no pooled connection for the address the push scaler was built with")
	}
	if got := connGroup.refCount; got != 1 {
		t.Fatalf("refCount = %d, want 1", got)
	}

	// A plain scaler on the same address shares the push scaler's connection.
	plain := newTestExternalScaler(t, address)
	if got := refCountFor(t, address); got != 2 {
		t.Fatalf("refCount = %d, want 2, the two scalers should share one connection", got)
	}

	// Closing the push scaler releases only its own share.
	if err := push.Close(context.Background()); err != nil {
		t.Fatalf("closing the push scaler: %v", err)
	}
	if got := refCountFor(t, address); got != 1 {
		t.Errorf("refCount = %d, want 1, the plain scaler is still using the connection", got)
	}
	if got := connGroup.grpcConnection.GetState(); got == connectivity.Shutdown {
		t.Error("the connection was closed while the plain scaler was still using it")
	}

	if err := plain.Close(context.Background()); err != nil {
		t.Fatalf("closing the plain scaler: %v", err)
	}
	if entry := poolEntryFor(t, address); entry != nil {
		t.Error("the connection was not dropped from the pool")
	}
	if got := connGroup.grpcConnection.GetState(); got != connectivity.Shutdown {
		t.Errorf("connection state = %v, want %v", got, connectivity.Shutdown)
	}
}

// Close is part of the Scaler interface, so it can be called more than once.
// A repeated Close must not drop a share another scaler still holds.
func TestExternalScalerCloseIsIdempotent(t *testing.T) {
	const address = "pool-double-close.default.svc.cluster.local:9090"

	first := newTestExternalScaler(t, address)
	second := newTestExternalScaler(t, address)

	// Close the first scaler repeatedly. Only the first call may release.
	for range 3 {
		if err := first.Close(context.Background()); err != nil {
			t.Fatalf("closing the first scaler: %v", err)
		}
	}

	connGroup := poolEntryFor(t, address)
	if connGroup == nil {
		t.Fatal("a repeated Close released a share it did not hold")
	}
	if got := connGroup.refCount; got != 1 {
		t.Fatalf("refCount = %d, want 1, a repeated Close released a share it did not hold", got)
	}
	if got := connGroup.grpcConnection.GetState(); got == connectivity.Shutdown {
		t.Error("the connection was closed while a scaler was still using it")
	}

	if err := second.Close(context.Background()); err != nil {
		t.Fatalf("closing the second scaler: %v", err)
	}
	if entry := poolEntryFor(t, address); entry != nil {
		t.Error("the connection was not dropped from the pool")
	}
}

// A request can arrive after the owning scaler has closed, for instance from
// the retry loop in runStreamIsActive racing its context cancellation. Looking
// the connection up must report an error at that point. Creating one would
// leave a pool entry with no owner to release it, which is the leak this change
// exists to remove.
func TestGetClientForConnectionPoolDoesNotRecreateAfterClose(t *testing.T) {
	const address = "pool-no-recreate.default.svc.cluster.local:9090"

	s := newTestExternalScaler(t, address)
	md := externalScalerMetadata{ScalerAddress: address}

	if _, err := getClientForConnectionPool(md); err != nil {
		t.Fatalf("looking up the connection while the scaler is open: %v", err)
	}

	if err := s.Close(context.Background()); err != nil {
		t.Fatalf("closing the scaler: %v", err)
	}

	if _, err := getClientForConnectionPool(md); err == nil {
		t.Error("expected an error looking up the connection after the scaler closed")
	}
	if entry := poolEntryFor(t, address); entry != nil {
		t.Error("the lookup recreated an entry nothing owns")
	}
}

// Scalers pointing at different addresses do not share a connection, and
// releasing one leaves the other alone.
func TestExternalScalerConnectionPoolPerAddress(t *testing.T) {
	const (
		addressA = "pool-a.default.svc.cluster.local:9090"
		addressB = "pool-b.default.svc.cluster.local:9090"
	)

	before := poolEntries()

	one := newTestExternalScaler(t, addressA)
	two := newTestExternalScaler(t, addressB)

	if got := poolEntries(); got != before+2 {
		t.Fatalf("pool entries = %d, want %d, the two addresses should not share a connection", got, before+2)
	}

	if err := one.Close(context.Background()); err != nil {
		t.Fatalf("closing the first scaler: %v", err)
	}
	if entry := poolEntryFor(t, addressA); entry != nil {
		t.Error("the first connection was not dropped from the pool")
	}
	if got := refCountFor(t, addressB); got != 1 {
		t.Errorf("refCount for the second address = %d, want 1, releasing one address must leave the other alone", got)
	}

	if err := two.Close(context.Background()); err != nil {
		t.Fatalf("closing the second scaler: %v", err)
	}
	if entry := poolEntryFor(t, addressB); entry != nil {
		t.Error("the second connection was not dropped from the pool")
	}
	if got := poolEntries(); got != before {
		t.Errorf("pool entries = %d, want %d", got, before)
	}
}

// The pooled connection is now built from the scaler constructor, so a client
// certificate that cannot be parsed is reported there rather than on the first
// poll. buildScalers turns any constructor error into a failure for the whole
// ScaledObject, so this moves a malformed certificate from degrading one
// trigger at poll time to failing every trigger on the object at build time.
// This test pins that behaviour so a change to it is deliberate. Whichever way
// the error is reported, the failed construction must leave nothing in the
// pool.
func TestExternalScalerReportsTLSErrorFromTheConstructor(t *testing.T) {
	const address = "pool-bad-tls.default.svc.cluster.local:9090"

	before := poolEntries()

	_, err := NewExternalScaler(&scalersconfig.ScalerConfig{
		ScalableObjectName:      "app",
		ScalableObjectNamespace: "namespace",
		TriggerMetadata:         map[string]string{"scalerAddress": address, "enableTLS": "true"},
		AuthParams: map[string]string{
			"tlsClientCert": "not a certificate",
			"tlsClientKey":  "not a key",
		},
		ResolvedEnv: map[string]string{},
	})
	if err == nil {
		t.Fatal("expected an error for an unparseable client certificate")
	}

	// The pool key includes the TLS fields, so the entry a failed construction
	// could leave behind is found by counting rather than by looking it up.
	if got := poolEntries(); got != before {
		t.Errorf("pool entries = %d, want %d, the failed construction left a connection in the pool", got, before)
	}
}
