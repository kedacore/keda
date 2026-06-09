package timezone

import (
	"sync"
	"time"
)

var cache = struct {
	mutex sync.Mutex
	items map[string]*time.Location
}{
	items: make(map[string]*time.Location),
}

func Load(name string) (*time.Location, error) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	if tz, found := cache.items[name]; found {
		return tz, nil
	}
	tz, err := time.LoadLocation(name)
	if err != nil {
		return nil, err
	}
	cache.items[name] = tz
	return tz, nil
}
