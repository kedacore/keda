package connectionpool

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"go.uber.org/atomic"
)

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

func mockCreateFn() func() (ResourcePool, error) {
	return func() (ResourcePool, error) {
		return &mockPool{}, nil
	}
}

func TestGetOrCreate_ReusesPool(t *testing.T) {
	t.Cleanup(CloseAll)
	poolKey := "postgres.db1.analytics"

	first, err := GetOrCreate(poolKey, mockCreateFn())
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	second, err := GetOrCreate(poolKey, mockCreateFn())
	if err != nil {
		t.Fatalf("failed to reuse pool: %v", err)
	}

	if first != second {
		t.Fatalf("expected same pool instance, got different ones")
	}
}

func TestGetOrCreate_DifferentKeys(t *testing.T) {
	t.Cleanup(CloseAll)

	pool1, _ := GetOrCreate("postgres.db1.analytics", mockCreateFn())
	pool2, _ := GetOrCreate("postgres.db2.reporting", mockCreateFn())

	if pool1 == pool2 {
		t.Fatalf("expected different pools for different keys")
	}
}

func TestRelease_RefCount(t *testing.T) {
	t.Cleanup(CloseAll)
	key := "postgres.db1.analytics"

	p1, _ := GetOrCreate(key, mockCreateFn())
	p2, _ := GetOrCreate(key, mockCreateFn())

	if p1 != p2 {
		t.Fatalf("expected pool reuse")
	}

	Release(key) // decrements (count: 1)
	if isClosed(p1) {
		t.Fatalf("pool should not be closed with remaining references")
	}

	Release(key) // closes and deletes (count: 0)

	if _, found := poolMap.Load(key); found {
		t.Fatalf("expected pool to be removed after final release")
	}
}

func TestConcurrentAccess(t *testing.T) {
	t.Cleanup(CloseAll)
	key := "postgres.db1.analytics"

	var wg sync.WaitGroup
	errChan := make(chan error, 1000)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				p, err := GetOrCreate(key, mockCreateFn())
				if err != nil {
					errChan <- fmt.Errorf("get or create failed: %w", err)
					return
				}
				if p == nil {
					errChan <- errors.New("nil pool returned")
					return
				}
				Release(key)
			}
		}()
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Error(err)
	}
}

func TestInvalidConnectionHandledGracefully(t *testing.T) {
	t.Cleanup(CloseAll)
	key := "postgres.invalid"
	var ErrPoolCreationFailed = errors.New("pool creation failed")

	createFn := func() (ResourcePool, error) {
		return nil, ErrPoolCreationFailed
	}

	_, err := GetOrCreate(key, createFn)
	if err == nil {
		t.Fatal("expected error during pool creation, but got none")
	}
}
