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

package cache

import (
	"container/list"
	"errors"
	"sync"
	"time"
)

var (
	// ErrCacheFull is returned if Put fails due to cache being filled with pinned elements
	ErrCacheFull = errors.New("Cache capacity is fully occupied with pinned elements")
)

// lru is a concurrent fixed size cache that evicts elements in lru order
type lru struct {
	mut      sync.Mutex
	byAccess *list.List
	byKey    map[string]*list.Element
	maxSize  int
	ttl      time.Duration
	pin      bool
	rmFunc   RemovedFunc
}

// New creates a new cache with the given options
func New(maxSize int, opts *Options) Cache {
	if opts == nil {
		opts = &Options{}
	}

	return &lru{
		byAccess: list.New(),
		byKey:    make(map[string]*list.Element, opts.InitialCapacity),
		ttl:      opts.TTL,
		maxSize:  maxSize,
		pin:      opts.Pin,
		rmFunc:   opts.RemovedFunc,
	}
}

// NewLRU creates a new LRU cache of the given size, setting initial capacity
// to the max size
func NewLRU(maxSize int) Cache {
	return New(maxSize, nil)
}

// NewLRUWithInitialCapacity creates a new LRU cache with an initial capacity
// and a max size
func NewLRUWithInitialCapacity(initialCapacity, maxSize int) Cache {
	return New(maxSize, &Options{
		InitialCapacity: initialCapacity,
	})
}

// Exist checks if a given key exists in the cache
func (c *lru) Exist(key string) bool {
	c.mut.Lock()
	defer c.mut.Unlock()
	_, ok := c.byKey[key]
	return ok
}

// Get retrieves the value stored under the given key
func (c *lru) Get(key string) interface{} {
	c.mut.Lock()
	defer c.mut.Unlock()

	elt := c.byKey[key]
	if elt == nil {
		return nil
	}

	cacheEntry := elt.Value.(*cacheEntry)

	if c.pin {
		cacheEntry.refCount++
	}

	if cacheEntry.refCount == 0 && !cacheEntry.expiration.IsZero() && time.Now().After(cacheEntry.expiration) {
		// Entry has expired
		if c.rmFunc != nil {
			go c.rmFunc(cacheEntry.value)
		}
		c.byAccess.Remove(elt)
		delete(c.byKey, cacheEntry.key)
		return nil
	}

	c.byAccess.MoveToFront(elt)
	return cacheEntry.value
}

// Put puts a new value associated with a given key, returning the existing value (if present)
func (c *lru) Put(key string, value interface{}) interface{} {
	if c.pin {
		panic("Cannot use Put API in Pin mode. Use Delete and PutIfNotExist if necessary")
	}
	val, _ := c.putInternal(key, value, true)
	return val
}

// PutIfNotExist puts a value associated with a given key if it does not exist
func (c *lru) PutIfNotExist(key string, value interface{}) (interface{}, error) {
	existing, err := c.putInternal(key, value, false)
	if err != nil {
		return nil, err
	}

	if existing == nil {
		// This is a new value
		return value, err
	}

	return existing, err
}

// Delete deletes a key, value pair associated with a key
func (c *lru) Delete(key string) {
	c.mut.Lock()
	defer c.mut.Unlock()

	elt := c.byKey[key]
	if elt != nil {
		entry := c.byAccess.Remove(elt).(*cacheEntry)
		if c.rmFunc != nil {
			go c.rmFunc(entry.value)
		}
		delete(c.byKey, key)
	}
}

// Release decrements the ref count of a pinned element.
func (c *lru) Release(key string) {
	c.mut.Lock()
	defer c.mut.Unlock()

	elt := c.byKey[key]
	cacheEntry := elt.Value.(*cacheEntry)
	cacheEntry.refCount--
}

// Size returns the number of entries currently in the lru, useful if cache is not full
func (c *lru) Size() int {
	c.mut.Lock()
	defer c.mut.Unlock()

	return len(c.byKey)
}

// Clear clears the cache.
func (c *lru) Clear() {
	c.mut.Lock()
	defer c.mut.Unlock()

	for key, elt := range c.byKey {
		if elt != nil {
			entry := c.byAccess.Remove(elt).(*cacheEntry)
			if c.rmFunc != nil {
				go c.rmFunc(entry.value)
			}
			delete(c.byKey, key)
		}
	}
}

// Put puts a new value associated with a given key, returning the existing value (if present)
// allowUpdate flag is used to control overwrite behavior if the value exists
func (c *lru) putInternal(key string, value interface{}, allowUpdate bool) (interface{}, error) {
	c.mut.Lock()
	defer c.mut.Unlock()

	elt := c.byKey[key]
	if elt != nil {
		entry := elt.Value.(*cacheEntry)
		existing := entry.value
		if allowUpdate {
			entry.value = value
		}
		if c.ttl != 0 {
			entry.expiration = time.Now().Add(c.ttl)
		}
		c.byAccess.MoveToFront(elt)
		if c.pin {
			entry.refCount++
		}
		return existing, nil
	}

	entry := &cacheEntry{
		key:   key,
		value: value,
	}

	if c.pin {
		entry.refCount++
	}

	if c.ttl != 0 {
		entry.expiration = time.Now().Add(c.ttl)
	}

	c.byKey[key] = c.byAccess.PushFront(entry)
	// Only trigger eviction when we have exceeded the max
	if len(c.byKey) > c.maxSize {
		oldest := c.byAccess.Back().Value.(*cacheEntry)

		if oldest.refCount > 0 {
			// Cache is full with pinned elements
			// revert the insert and return
			c.byAccess.Remove(c.byAccess.Front())
			delete(c.byKey, key)
			return nil, ErrCacheFull
		}

		c.byAccess.Remove(c.byAccess.Back())
		if c.rmFunc != nil {
			go c.rmFunc(oldest.value)
		}
		delete(c.byKey, oldest.key)
	}

	return nil, nil
}

type cacheEntry struct {
	key        string
	expiration time.Time
	value      interface{}
	refCount   int
}
