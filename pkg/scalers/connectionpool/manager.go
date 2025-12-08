package connectionpool

import (
	"sync"

	"go.uber.org/atomic"
)

type ResourcePool interface {
	close()
}

type poolEntry struct {
	pool ResourcePool
	ref  atomic.Int32
}

var poolMap sync.Map

func GetOrCreate(poolKey string, createFn func() (ResourcePool, error)) (ResourcePool, error) {
	if val, ok := poolMap.Load(poolKey); ok {
		entry := val.(*poolEntry)
		entry.ref.Inc()
		logger.V(1).Info("Reusing existing pool", "poolKey", poolKey, "refCount", entry.ref.Load())
		return entry.pool, nil
	}

	logger.Info("Creating new pool", "poolKey", poolKey)
	newPool, err := createFn()
	if err != nil {
		logger.Error(err, "Failed to create new pool", "poolKey", poolKey)
		return nil, err
	}

	e := &poolEntry{pool: newPool}
	e.ref.Store(1)

	actual, loaded := poolMap.LoadOrStore(poolKey, e)
	if loaded {
		logger.Info("Duplicate creation detected, closing redundant pool", "poolKey", poolKey)
		newPool.close()
		old := actual.(*poolEntry)
		old.ref.Inc()
		logger.V(1).Info("Reusing existing pool after race", "poolKey", poolKey, "refCount", old.ref.Load())
		return old.pool, nil
	}

	logger.Info("Pool created successfully", "poolKey", poolKey)
	return newPool, nil
}

func Release(poolKey string) {
	val, ok := poolMap.Load(poolKey)
	if !ok {
		logger.V(1).Info("Attempted to release non-existent pool", "poolKey", poolKey)
		return
	}
	entry := val.(*poolEntry)
	if entry.ref.Dec() <= 0 {
		logger.Info("Closing pool, no active references", "poolKey", poolKey)
		entry.pool.close()
		poolMap.Delete(poolKey)
	} else {
		logger.V(1).Info("Released pool reference", "poolKey", poolKey, "refCount", entry.ref.Load())
	}
}

func CloseAll() {
	poolMap.Range(func(key, val any) bool {
		entry := val.(*poolEntry)
		logger.Info("Closing pool", "poolKey", key)
		entry.pool.close()
		poolMap.Delete(key)
		return true
	})
}
