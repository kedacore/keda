package connectionpool

import "sync"

// ResourcePool : generic interface for any DB pool
type ResourcePool interface {
	Close()
}

// poolEntry : tracks a ResourcePool with reference count
type poolEntry struct {
	pool ResourcePool
	ref  int32
}

var poolMap sync.Map

// GetOrCreate : reuse or create new pool
func GetOrCreate(poolKey string, createFn func() (ResourcePool, error)) (ResourcePool, error) {
	if val, ok := poolMap.Load(poolKey); ok {
		entry := val.(*poolEntry)
		entry.ref++
		return entry.pool, nil
	}

	newPool, err := createFn()
	if err != nil {
		return nil, err
	}

	entry := &poolEntry{pool: newPool, ref: 1}
	actual, loaded := poolMap.LoadOrStore(poolKey, entry)
	if loaded {
		newPool.Close()
		old := actual.(*poolEntry)
		old.ref++
		return old.pool, nil
	}
	return newPool, nil
}

// Release : decrease ref count, close when last user gone
func Release(poolKey string) {
	val, ok := poolMap.Load(poolKey)
	if !ok {
		return
	}
	entry := val.(*poolEntry)
	entry.ref--
	if entry.ref <= 0 {
		entry.pool.Close()
		poolMap.Delete(poolKey)
	}
}

// CloseAll : close all pools (on operator shutdown)
func CloseAll() {
	poolMap.Range(func(key, val any) bool {
		entry := val.(*poolEntry)
		entry.pool.Close()
		poolMap.Delete(key)
		return true
	})
}
