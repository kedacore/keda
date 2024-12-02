package util

//nolint:depguard // sync/atomic
import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Test the initial storage and retrieval of a value
func TestRefMapStoreAndLoad(t *testing.T) {
	refMap := NewRefMap[string, int]()
	cleanupCalled := atomic.Bool{}

	closeFunc := func(value int) error {
		cleanupCalled.Store(true)
		return nil
	}

	if err := refMap.Store("testKey", 42, closeFunc); err != nil {
		t.Errorf("unexpected error on Store: %v", err)
	}

	val, ok := refMap.Load("testKey")
	if !ok || val != 42 {
		t.Errorf("expected to load value 42 for key 'testKey', got %v", val)
	}

	if cleanupCalled.Load() {
		t.Error("expected cleanup function not to be called initially")
	}
}

// Test adding a reference and removing it, triggering cleanup on zero count
func TestRefMapAddAndRemoveRef(t *testing.T) {
	refMap := NewRefMap[string, int]()
	cleanupCalled := atomic.Bool{}

	closeFunc := func(value int) error {
		cleanupCalled.Store(true)
		return nil
	}

	if err := refMap.Store("testKey", 42, closeFunc); err != nil {
		t.Errorf("unexpected error on Store: %v", err)
	}

	// Add a reference
	if err := refMap.AddRef("testKey"); err != nil {
		t.Errorf("unexpected error on AddRef: %v", err)
	}

	// Remove references
	if err := refMap.RemoveRef("testKey"); err != nil {
		t.Errorf("unexpected error on first RemoveRef: %v", err)
	}

	if err := refMap.RemoveRef("testKey"); err != nil {
		t.Errorf("unexpected error on second RemoveRef: %v", err)
	}

	// Check that cleanup was called
	if !cleanupCalled.Load() {
		t.Error("expected cleanup function to be called")
	}

	// Check that key no longer exists
	_, ok := refMap.Load("testKey")
	if ok {
		t.Error("expected key 'testKey' to be deleted after reference count reached zero")
	}
}

// Test removing reference from a non-existent key
func TestRefMapRemoveRefNonExistentKey(t *testing.T) {
	refMap := NewRefMap[string, int]()

	if err := refMap.RemoveRef("nonExistentKey"); err == nil {
		t.Error("expected error when removing reference from a non-existent key")
	}
}

// Test that multiple calls to AddRef and RemoveRef are handled concurrently and correctly
func TestRefMapConcurrentAddAndRemove(t *testing.T) {
	refMap := NewRefMap[string, int]()
	cleanupCalled := atomic.Bool{}
	wg := sync.WaitGroup{}

	closeFunc := func(value int) error {
		cleanupCalled.Store(true)
		return nil
	}

	// Store a value with an initial reference count of 1
	if err := refMap.Store("testKey", 42, closeFunc); err != nil {
		t.Errorf("unexpected error on Store: %v", err)
	}

	// Confirm initial state before any increments
	val, ok := refMap.Load("testKey")
	if !ok || val != 42 {
		t.Fatalf("expected to find 'testKey' with initial value 42, found %v", val)
	}

	// Concurrently increment references
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := refMap.AddRef("testKey"); err != nil {
				t.Errorf("unexpected error on AddRef: %v", err)
			}
		}()
	}

	wg.Wait() // Wait for all AddRef operations to complete

	// Concurrently remove references
	for i := 0; i < 101; i++ { // Including the initial count
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := refMap.RemoveRef("testKey"); err != nil {
				t.Errorf("unexpected error on RemoveRef: %v", err)
			}
		}()
	}

	wg.Wait() // Wait for all RemoveRef operations to complete

	// Verify that cleanup was called after all references are removed
	if !cleanupCalled.Load() {
		t.Error("expected cleanup function to be called after all references are removed")
	}

	// Verify that the key no longer exists
	if _, ok := refMap.Load("testKey"); ok {
		t.Error("expected key 'testKey' to be deleted after reference count reached zero")
	}
}

// Test that an error in closeFunc is handled properly and does not prevent map deletion
func TestRefMapCleanupErrorHandling(t *testing.T) {
	refMap := NewRefMap[string, int]()
	cleanupCalled := atomic.Bool{}

	closeFunc := func(value int) error {
		cleanupCalled.Store(true)
		return errors.New("close error")
	}

	if err := refMap.Store("testKey", 42, closeFunc); err != nil {
		t.Errorf("unexpected error on Store: %v", err)
	}

	// initial store should increment reference count, so this should trigger cleanup
	// the closeFunc then throws an error, but the key should still be deleted
	if err := refMap.RemoveRef("testKey"); err == nil {
		t.Error("expected error when removing reference from a key with cleanup error")
	}

	if !cleanupCalled.Load() {
		t.Error("expected cleanup function to be called even if it returns an error")
	}

	// Check that key no longer exists
	_, ok := refMap.Load("testKey")
	if ok {
		t.Error("expected key 'testKey' to be deleted after reference count reached zero")
	}
}

// Test to ensure references are counted correctly
func TestRefMapReferenceCounting(t *testing.T) {
	refMap := NewRefMap[string, int]()
	closeFunc := func(value int) error { return nil }

	if err := refMap.Store("testKey", 100, closeFunc); err != nil {
		t.Errorf("unexpected error on Store: %v", err)
	}

	if err := refMap.AddRef("testKey"); err != nil {
		t.Errorf("unexpected error on first AddRef: %v", err)
	}

	if err := refMap.AddRef("testKey"); err != nil {
		t.Errorf("unexpected error on second AddRef: %v", err)
	}

	val, ok := refMap.Load("testKey")
	if !ok || val != 100 {
		t.Errorf("expected to load value 100 for key 'testKey', got %v", val)
	}

	// Remove references one by one, expecting the item to persist until the last removal
	if err := refMap.RemoveRef("testKey"); err != nil {
		t.Errorf("unexpected error on RemoveRef: %v", err)
	}

	if _, ok := refMap.Load("testKey"); !ok {
		t.Error("expected key 'testKey' to still exist after one RemoveRef")
	}

	if err := refMap.RemoveRef("testKey"); err != nil {
		t.Errorf("unexpected error on second RemoveRef: %v", err)
	}

	if _, ok := refMap.Load("testKey"); !ok {
		t.Error("expected key 'testKey' to still exist after second RemoveRef")
	}

	// Final removal should delete the key
	if err := refMap.RemoveRef("testKey"); err != nil {
		t.Errorf("unexpected error on final RemoveRef: %v", err)
	}

	if _, ok := refMap.Load("testKey"); ok {
		t.Error("expected key 'testKey' to be deleted after all references were removed")
	}
}

// Test to check if cleanup function handles delay in cleanup process.
func TestRefMapDelayedCleanup(t *testing.T) {
	refMap := NewRefMap[string, int]()
	cleanupCalled := atomic.Bool{}
	closeFunc := func(value int) error {
		time.Sleep(50 * time.Millisecond)
		cleanupCalled.Store(true)
		return nil
	}

	if err := refMap.Store("testKey", 100, closeFunc); err != nil {
		t.Errorf("unexpected error on Store: %v", err)
	}

	if err := refMap.RemoveRef("testKey"); err != nil {
		t.Errorf("unexpected error on RemoveRef: %v", err)
	}

	if !cleanupCalled.Load() {
		t.Error("expected cleanup function to be called after delay")
	}

	// Verify key deletion after cleanup completes
	_, ok := refMap.Load("testKey")
	if ok {
		t.Error("expected key 'testKey' to be deleted after cleanup")
	}
}
