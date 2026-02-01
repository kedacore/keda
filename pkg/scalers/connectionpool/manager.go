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
		return entry.pool, nil
	}

	logger.Info("Creating new pool", "poolKey", poolKey)
	newPool, err := createFn()
	if err != nil {
		return nil, err
	}

	e := &poolEntry{pool: newPool}
	e.ref.Store(1)

	actual, loaded := poolMap.LoadOrStore(poolKey, e)
	if loaded {
		newPool.close()
		old := actual.(*poolEntry)
		old.ref.Inc()
		return old.pool, nil
	}

	return newPool, nil
}

func Release(poolKey string) {
	val, ok := poolMap.Load(poolKey)
	if !ok {
		logger.V(0).Error(nil, "Attempted to release non-existent pool", "poolKey", poolKey)
		return
	}
	entry := val.(*poolEntry)

	if entry.ref.Dec() <= 0 {
		logger.Info("Closing pool (no active references)", "poolKey", poolKey)
		entry.pool.close()
		poolMap.Delete(poolKey)
	}
}

func CloseAll() {
	poolMap.Range(func(key, val any) bool {
		entry := val.(*poolEntry)
		entry.pool.close()
		poolMap.Delete(key)
		return true
	})
}
