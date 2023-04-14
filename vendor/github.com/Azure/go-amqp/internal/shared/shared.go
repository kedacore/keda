package shared

import (
	"encoding/base64"
	"math/rand"
	"sync"
	"time"
)

// lockedRand provides a rand source that is safe for concurrent use.
type lockedRand struct {
	mu  sync.Mutex
	src *rand.Rand
}

func (r *lockedRand) Read(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.src.Read(p)
}

// package scoped rand source to avoid any issues with seeding
// of the global source.
var pkgRand = &lockedRand{
	src: rand.New(rand.NewSource(time.Now().UnixNano())),
}

// RandString returns a base64 encoded string of n bytes.
func RandString(n int) string {
	b := make([]byte, n)
	// from math/rand, cannot fail
	_, _ = pkgRand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
