package scalers

import (
	"sync"

	kore_v1alpha1 "github.com/Azure/Kore/pkg/apis/kore/v1alpha1"
)

type scaleHandlerSharedCache struct {
	activeScaledObjects map[string]*kore_v1alpha1.ScaledObject
	opsLock             sync.RWMutex
}

func (c *scaleHandlerSharedCache) get(key string) *kore_v1alpha1.ScaledObject {
	c.opsLock.RLock()
	defer c.opsLock.RUnlock()

	return c.activeScaledObjects[key]
}

func (c *scaleHandlerSharedCache) set(key string, value *kore_v1alpha1.ScaledObject) {
	c.opsLock.Lock()
	defer c.opsLock.Unlock()

	c.activeScaledObjects[key] = value
}

func (c *scaleHandlerSharedCache) delete(key string) {
	c.opsLock.Lock()
	defer c.opsLock.Unlock()

	delete(c.activeScaledObjects, key)
}
