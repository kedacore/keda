package connectionpool

import (
	"sync"
)

// ResourcePool : generic interface for any DB pool
type ResourcePool interface {
	close()
}

// poolEntry : tracks a ResourcePool with reference count
type poolEntry struct {
	pool ResourcePool
	ref  int32
}

var (
	poolMap sync.Map
)

// GetOrCreate : reuse or create new pool
func GetOrCreate(poolKey string, createFn func() (ResourcePool, error)) (ResourcePool, error) {
	if val, ok := poolMap.Load(poolKey); ok {
		entry := val.(*poolEntry)
		entry.ref++
		logger.V(1).Info("Reusing existing pool", "poolKey", poolKey, "refCount", entry.ref)
		return entry.pool, nil
	}
	logger.Info("Creating new pool", "poolKey", poolKey)
	newPool, err := createFn()
	if err != nil {
		logger.Error(err, "Failed to create new pool", "poolKey", poolKey)
		return nil, err
	}

	entry := &poolEntry{pool: newPool, ref: 1}
	actual, loaded := poolMap.LoadOrStore(poolKey, entry)
	if loaded {
		logger.Info("Duplicate creation detected, closing redundant pool", "poolKey", poolKey)
		newPool.close()
		old := actual.(*poolEntry)
		old.ref++
		logger.V(1).Info("Reusing existing pool after race", "poolKey", poolKey, "refCount", old.ref)
		return old.pool, nil
	}
	logger.Info("Pool created successfully", "poolKey", poolKey)
	return newPool, nil
}

// Release : decrease ref count, close when last user gone
func Release(poolKey string) {
	val, ok := poolMap.Load(poolKey)
	if !ok {
		logger.V(1).Info("Attempted to release non-existent pool", "poolKey", poolKey)
		return
	}
	entry := val.(*poolEntry)
	entry.ref--
	logger.V(1).Info("Released pool reference", "poolKey", poolKey, "refCount", entry.ref)
	if entry.ref <= 0 {
		logger.Info("Closing pool, no active references", "poolKey", poolKey)
		entry.pool.close()
		poolMap.Delete(poolKey)
	}
}

// CloseAll : close all pools (on operator shutdown)
func CloseAll() {
	poolMap.Range(func(key, val any) bool {
		entry := val.(*poolEntry)
		logger.Info("Closing pool", "poolKey", key)
		entry.pool.close()
		poolMap.Delete(key)
		return true
	})
}
