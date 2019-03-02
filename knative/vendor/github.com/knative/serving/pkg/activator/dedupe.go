/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package activator

import (
	"errors"
	"net/http"
	"sync"
)

var shuttingDownError = ActivationResult{
	Endpoint: Endpoint{},
	Status:   http.StatusInternalServerError,
	Error:    errors.New("activator shutting down"),
}

var _ Activator = (*dedupingActivator)(nil)

type dedupingActivator struct {
	mux             sync.Mutex
	pendingRequests map[RevisionID][]chan ActivationResult
	activator       Activator
	shutdown        bool
}

// NewDedupingActivator creates an Activator that deduplicates
// activations requests for the same revision id and namespace.
func NewDedupingActivator(a Activator) Activator {
	return &dedupingActivator{
		pendingRequests: make(map[RevisionID][]chan ActivationResult),
		activator:       a,
	}
}

func (a *dedupingActivator) ActiveEndpoint(namespace, name string) ActivationResult {
	id := RevisionID{Namespace: namespace, Name: name}
	ch := make(chan ActivationResult, 1)
	a.dedupe(id, ch)
	result := <-ch
	return result
}

func (a *dedupingActivator) Shutdown() {
	a.activator.Shutdown()
	a.mux.Lock()
	defer a.mux.Unlock()
	a.shutdown = true
	for _, reqs := range a.pendingRequests {
		for _, ch := range reqs {
			ch <- shuttingDownError
		}
	}
}

func (a *dedupingActivator) dedupe(id RevisionID, ch chan ActivationResult) {
	a.mux.Lock()
	defer a.mux.Unlock()
	if a.shutdown {
		ch <- shuttingDownError
		return
	}
	if reqs, ok := a.pendingRequests[id]; ok {
		a.pendingRequests[id] = append(reqs, ch)
	} else {
		a.pendingRequests[id] = []chan ActivationResult{ch}
		go a.activate(id)
	}
}

func (a *dedupingActivator) activate(id RevisionID) {
	result := a.activator.ActiveEndpoint(id.Namespace, id.Name)
	a.mux.Lock()
	defer a.mux.Unlock()
	if reqs, ok := a.pendingRequests[id]; ok {
		delete(a.pendingRequests, id)
		for _, ch := range reqs {
			ch <- result
		}
	}
}
