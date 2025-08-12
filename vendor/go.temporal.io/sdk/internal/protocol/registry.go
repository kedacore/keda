package protocol

import (
	"sync"

	protocolpb "go.temporal.io/api/protocol/v1"
)

type (
	// Instance is the required interface for protocol objects.
	Instance interface {
		HandleMessage(*protocolpb.Message) error
		HasCompleted() bool
	}

	// Registry stores running protocols.
	Registry struct {
		mut       sync.Mutex
		instances map[string]Instance
	}
)

func NewRegistry() *Registry {
	return &Registry{instances: map[string]Instance{}}
}

// FindOrAdd looks up an existing protocol by instance ID or constructs a new
// one and registers it under the instance ID indicated.
func (r *Registry) FindOrAdd(instID string, ctor func() Instance) Instance {
	r.mut.Lock()
	defer r.mut.Unlock()
	p, ok := r.instances[instID]
	if !ok {
		p = ctor()
		r.instances[instID] = p
	}
	return p
}

// ClearCompleted walks the registered protocols and removes those that have
// completed.
func (r *Registry) ClearCompleted() {
	r.mut.Lock()
	defer r.mut.Unlock()
	for instID, inst := range r.instances {
		if inst.HasCompleted() {
			delete(r.instances, instID)
		}
	}
}
