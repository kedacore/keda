package aksk

import "sync"

// MemoryCache presents a thread safe memory cache
type MemoryCache struct {
	sync.Mutex                    // handling r/w for cache
	cacheHolder map[string]string // cache holder
	cacheKeys   []string          // cache keys
	MaxCount    int               // max cache entry count
}

// NewCache inits an new MemoryCache
func NewCache(maxCount int) *MemoryCache {
	return &MemoryCache{
		cacheHolder: make(map[string]string, maxCount),
		MaxCount:    maxCount,
	}
}

// Add an new cache item
func (cache *MemoryCache) Add(cacheKey string, cacheData string) {
	cache.Lock()
	defer cache.Unlock()

	if len(cache.cacheKeys) >= cache.MaxCount && len(cache.cacheKeys) > 1 {
		delete(cache.cacheHolder, cache.cacheKeys[0]) // delete first item
		cache.cacheKeys = append(cache.cacheKeys[1:]) // pop first one
	}

	cache.cacheHolder[cacheKey] = cacheData
	cache.cacheKeys = append(cache.cacheKeys, cacheKey)
}

// Get a cache item by its key
func (cache *MemoryCache) Get(cacheKey string) string {
	cache.Lock()
	defer cache.Unlock()

	return cache.cacheHolder[cacheKey]
}
