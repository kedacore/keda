/*
Copyright 2019 The Knative Authors

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

package health

import (
	"io"
	"net/http"
	"sync"
)

// State holds state about the current healthiness of the component.
type State struct {
	alive        bool
	shuttingDown bool
	mutex        sync.RWMutex
}

// IsAlive returns whether or not the health server is in a known
// working state currently.
func (h *State) IsAlive() bool {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	return h.alive
}

// IsShuttingDown returns whether or not the health server is currently
// shutting down.
func (h *State) IsShuttingDown() bool {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	return h.shuttingDown
}

// setAlive updates the state to declare the service alive.
func (h *State) setAlive() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.alive = true
	h.shuttingDown = false
}

// shutdown updates the state to declare the service shutting down.
func (h *State) shutdown() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.alive = false
	h.shuttingDown = true
}

// HealthHandler constructs a handler that returns the current state of
// the health server.
func (h *State) HealthHandler(prober func() bool) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		sendAlive := func() {
			io.WriteString(w, "alive: true")
		}

		sendNotAlive := func() {
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, "alive: false")
		}

		switch {
		case h.IsAlive():
			sendAlive()
		case h.IsShuttingDown():
			sendNotAlive()
		case prober != nil && !prober():
			sendNotAlive()
		default:
			h.setAlive()
			sendAlive()
		}
	}
}

// QuitHandler constructs a handler that shuts the current server down.
// Optional cleanup logic can be added via the given cleaner function.
func (h *State) QuitHandler(cleaner func()) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		h.shutdown()

		if cleaner != nil {
			cleaner()
		}

		io.WriteString(w, "alive: false")
	}
}
