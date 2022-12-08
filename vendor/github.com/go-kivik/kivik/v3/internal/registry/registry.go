// Package registry handles driver registrations. It's in a separate package
// to facilitate testing.
package registry

import (
	"sync"

	"github.com/go-kivik/kivik/v3/driver"
)

var (
	driversMu sync.RWMutex
	drivers   = make(map[string]driver.Driver)
)

// Register makes a database driver available by the provided name. If Register
// is called twice with the same name or if driver is nil, it panics.
func Register(name string, driver driver.Driver) {
	driversMu.Lock()
	defer driversMu.Unlock()
	if driver == nil {
		panic("kivik: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("kivik: Register called twice for driver " + name)
	}
	drivers[name] = driver
}

// Driver returns the driver registered with the requested name, or nil if
// it has not been registered.
func Driver(name string) driver.Driver {
	driversMu.RLock()
	defer driversMu.RUnlock()
	return drivers[name]
}
