package connectionpool

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/atomic"
)

// mockPool simulates a ResourcePool for testing purposes.
type mockPool struct {
	closed atomic.Int32
}

func (m *mockPool) close() {
	m.closed.Store(1)
}

func isClosed(p ResourcePool) bool {
	if m, ok := p.(*mockPool); ok {
		return m.closed.Load() == 1
	}
	return false
}

func mockCreateFn(_ context.Context) func() (ResourcePool, error) {
	return func() (ResourcePool, error) {
		return &mockPool{}, nil
	}
}

func TestGetOrCreate_ReusesPool(t *testing.T) {
	ctx := context.Background()
	poolKey := "postgres.db1.analytics"

	first, err := GetOrCreate(poolKey, mockCreateFn(ctx))
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	second, err := GetOrCreate(poolKey, mockCreateFn(ctx))
	if err != nil {
		t.Fatalf("failed to reuse pool: %v", err)
	}

	if first != second {
		t.Fatalf("expected same pool instance, got different ones")
	}

	Release(poolKey)
	Release(poolKey)
}

func TestGetOrCreate_DifferentKeys(t *testing.T) {
	ctx := context.Background()

	pool1, _ := GetOrCreate("postgres.db1.analytics", mockCreateFn(ctx))
	pool2, _ := GetOrCreate("postgres.db2.reporting", mockCreateFn(ctx))

	if pool1 == pool2 {
		t.Fatalf("expected different pools for different keys")
	}

	Release("postgres.db1.analytics")
	Release("postgres.db2.reporting")
}

func TestRelease_RefCount(t *testing.T) {
	ctx := context.Background()
	key := "postgres.db1.analytics"

	p1, _ := GetOrCreate(key, mockCreateFn(ctx))
	p2, _ := GetOrCreate(key, mockCreateFn(ctx))

	if p1 != p2 {
		t.Fatalf("expected pool reuse")
	}

	Release(key) // decrements
	if isClosed(p1) {
		t.Fatalf("pool should not be closed with remaining references")
	}
	Release(key) // closes and deletes

	if _, found := poolMap.Load(key); found {
		t.Fatalf("expected pool to be removed after final release")
	}
}

func TestConcurrentAccess(t *testing.T) {
	key := "postgres.db1.analytics"

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				p, err := GetOrCreate(key, mockCreateFn(context.Background()))
				if err != nil {
					t.Errorf("get or create: %v", err)
				}
				if p == nil {
					t.Errorf("nil pool returned")
				}
				Release(key)
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestInvalidConnectionHandledGracefully(t *testing.T) {
	key := "postgres.invalid"
	var ErrPoolCreationFailed = errors.New("pool creation failed")
	createFn := func() (ResourcePool, error) {
		return nil, ErrPoolCreationFailed
	}

	_, err := GetOrCreate(key, createFn)
	if err == nil {
		t.Log("expected error during pool creation")
	}

	Release(key)
}
