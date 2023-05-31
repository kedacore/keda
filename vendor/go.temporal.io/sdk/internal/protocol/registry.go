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

// ClearCopmleted walks the registered protocols and removes those that have
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
