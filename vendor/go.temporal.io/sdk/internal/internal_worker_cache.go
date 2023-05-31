// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2020 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package internal

import (
	"runtime"
	"sync"

	"go.temporal.io/sdk/internal/common/cache"
)

// A WorkerCache instance is held by each worker to hold cached data. The contents of this struct should always be
// pointers for any data shared with other workers, and owned values for any instance-specific caches.
type WorkerCache struct {
	sharedCache *sharedWorkerCache
}

// A container for data workers in this process may want to share with eachother
type sharedWorkerCache struct {
	// Count of live workers
	workerRefcount int

	// A cache workers can use to store workflow state.
	workflowCache *cache.Cache
	// Max size for the cache
	maxWorkflowCacheSize int
}

// A shared cache workers can use to store state. The cache is expected to be initialized with the first worker to be
// instantiated. IE: All workers have a pointer to it. The pointer itself is never made nil, but when the refcount
// reaches zero, the shared caches inside of it will be nilled out. Do not manipulate without holding
// sharedWorkerCacheLock
var sharedWorkerCachePtr = &sharedWorkerCache{}
var sharedWorkerCacheLock sync.Mutex

// Must be set before spawning any workers
var desiredWorkflowCacheSize = defaultStickyCacheSize

// SetStickyWorkflowCacheSize sets the cache size for sticky workflow cache. Sticky workflow execution is the affinity
// between workflow tasks of a specific workflow execution to a specific worker. The benefit of sticky execution is that
// the workflow does not have to reconstruct state by replaying history from the beginning. The cache is shared between
// workers running within same process. This must be called before any worker is started. If not called, the default
// size of 10K (which may change) will be used.
func SetStickyWorkflowCacheSize(cacheSize int) {
	sharedWorkerCacheLock.Lock()
	defer sharedWorkerCacheLock.Unlock()
	desiredWorkflowCacheSize = cacheSize
}

// PurgeStickyWorkflowCache resets the sticky workflow cache. This must be called only when all workers are stopped.
func PurgeStickyWorkflowCache() {
	sharedWorkerCacheLock.Lock()
	defer sharedWorkerCacheLock.Unlock()

	if sharedWorkerCachePtr.workflowCache != nil {
		(*sharedWorkerCachePtr.workflowCache).Clear()
	}
}

// NewWorkerCache Creates a new WorkerCache, and increases workerRefcount by one. Instances of WorkerCache decrement the refcounter as
// a hook to runtime.SetFinalizer (ie: When they are freed by the GC). When there are no reachable instances of
// WorkerCache, shared caches will be cleared
func NewWorkerCache() *WorkerCache {
	return newWorkerCache(sharedWorkerCachePtr, &sharedWorkerCacheLock, desiredWorkflowCacheSize)
}

// This private version allows us to test functionality without affecting the global shared cache
func newWorkerCache(storeIn *sharedWorkerCache, lock *sync.Mutex, cacheSize int) *WorkerCache {
	lock.Lock()
	defer lock.Unlock()

	if storeIn == nil {
		panic("Provided sharedWorkerCache pointer must not be nil")
	}

	if storeIn.workerRefcount == 0 {
		newcache := cache.New(cacheSize-1, &cache.Options{
			RemovedFunc: func(cachedEntity interface{}) {
				wc := cachedEntity.(*workflowExecutionContextImpl)
				wc.onEviction()
			},
		})
		*storeIn = sharedWorkerCache{workflowCache: &newcache, workerRefcount: 0, maxWorkflowCacheSize: cacheSize}
	}
	storeIn.workerRefcount++
	newWorkerCache := WorkerCache{
		sharedCache: storeIn,
	}
	runtime.SetFinalizer(&newWorkerCache, func(wc *WorkerCache) {
		wc.close(lock)
	})
	return &newWorkerCache
}

func (wc *WorkerCache) getWorkflowCache() cache.Cache {
	return *wc.sharedCache.workflowCache
}

func (wc *WorkerCache) close(lock *sync.Mutex) {
	lock.Lock()
	defer lock.Unlock()

	wc.sharedCache.workerRefcount--
	if wc.sharedCache.workerRefcount == 0 {
		// Delete cache if no more outstanding references
		wc.sharedCache.workflowCache = nil
	}
}

func (wc *WorkerCache) getWorkflowContext(runID string) *workflowExecutionContextImpl {
	o := (*wc.sharedCache.workflowCache).Get(runID)
	if o == nil {
		return nil
	}
	wec := o.(*workflowExecutionContextImpl)
	return wec
}

func (wc *WorkerCache) putWorkflowContext(runID string, wec *workflowExecutionContextImpl) (*workflowExecutionContextImpl, error) {
	existing, err := (*wc.sharedCache.workflowCache).PutIfNotExist(runID, wec)
	if err != nil {
		return nil, err
	}
	return existing.(*workflowExecutionContextImpl), nil
}

func (wc *WorkerCache) removeWorkflowContext(runID string) {
	(*wc.sharedCache.workflowCache).Delete(runID)
}

// MaxWorkflowCacheSize returns the maximum allowed size of the sticky cache
func (wc *WorkerCache) MaxWorkflowCacheSize() int {
	if wc == nil {
		return desiredWorkflowCacheSize
	}
	return wc.sharedCache.maxWorkflowCacheSize
}
