package util

//nolint:depguard // sync/atomic
import (
	"fmt"
	"sync"
	"sync/atomic"
)

// refCountedValue manages a reference-counted value with a cleanup function.
type refCountedValue[V any] struct {
	value     V
	refCount  atomic.Int64
	closeFunc func(V) error // Cleanup function to call when count reaches zero
}

// Add increments the reference count.
func (r *refCountedValue[V]) Add() {
	r.refCount.Add(1)
}

// Remove decrements the reference count and invokes closeFunc if the count
// reaches zero.
func (r *refCountedValue[V]) Remove() error {
	if r.refCount.Add(-1) == 0 {
		return r.closeFunc(r.value)
	}

	return nil
}

// Value returns the underlying value.
func (r *refCountedValue[V]) Value() V {
	return r.value
}

// RefMap manages reference-counted items in a concurrent-safe map.
type RefMap[K comparable, V any] struct {
	data map[K]*refCountedValue[V]
	mu   sync.RWMutex
}

// NewRefMap initializes a new RefMap. A RefMap is an atomic reference-counted
// concurrent hashmap. The general usage pattern is to Store a value with a
// close function, once the value is contained within the RefMap, it can be
// accessed via the Load method. The AddRef method signals ownership of the
// value and increments the reference count. The RemoveRef method decrements
// the reference count. When the reference count reaches zero, the close
// function is called and the value is removed from the map.
func NewRefMap[K comparable, V any]() *RefMap[K, V] {
	return &RefMap[K, V]{
		data: make(map[K]*refCountedValue[V]),
	}
}

// Store adds a new item with an initial reference count of 1 and a close
// function.
func (r *RefMap[K, V]) Store(key K, value V, closeFunc func(V) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.data[key]; exists {
		return fmt.Errorf("key already exists: %v", key)
	}

	r.data[key] = &refCountedValue[V]{value: value, refCount: atomic.Int64{}, closeFunc: closeFunc}
	r.data[key].Add() // Set initial reference count to 1

	return nil
}

// Load retrieves a value by key without modifying the reference count,
// returning the value and a boolean indicating if it was found. The reference
// count not being modified means that a check for the existence of a key
// can be performed without signalling ownership of the value. If the value is
// used after this method, it is recommended to call AddRef to increment the
// reference
func (r *RefMap[K, V]) Load(key K) (V, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if refValue, found := r.data[key]; found {
		return refValue.Value(), true
	}
	var zero V

	return zero, false
}

// AddRef increments the reference count for a key if it exists. Ensure
// to call RemoveRef when done with the value to prevent memory leaks.
func (r *RefMap[K, V]) AddRef(key K) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	refValue, found := r.data[key]
	if !found {
		return fmt.Errorf("key not found: %v", key)
	}

	refValue.Add()
	return nil
}

// RemoveRef decrements the reference count and deletes the entry if count
// reaches zero.
func (r *RefMap[K, V]) RemoveRef(key K) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	refValue, found := r.data[key]
	if !found {
		return fmt.Errorf("key not found: %v", key)
	}

	err := refValue.Remove()

	if refValue.refCount.Load() == 0 {
		delete(r.data, key)
	}

	return err // returns the error from closeFunc
}
