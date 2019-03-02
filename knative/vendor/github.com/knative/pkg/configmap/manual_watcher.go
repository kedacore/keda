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

package configmap

import (
	"sync"

	corev1 "k8s.io/api/core/v1"
)

// ManualWatcher will notify Observers when a ConfigMap is manually reported as changed
type ManualWatcher struct {
	Namespace string

	// Guards mutations to defaultImpl fields
	m sync.Mutex

	started   bool
	observers map[string][]Observer
}

var _ Watcher = (*ManualWatcher)(nil)

// Watch implements Watcher
func (w *ManualWatcher) Watch(name string, o Observer) {
	w.m.Lock()
	defer w.m.Unlock()

	if w.observers == nil {
		w.observers = make(map[string][]Observer)
	}

	wl, _ := w.observers[name]
	w.observers[name] = append(wl, o)
}

func (w *ManualWatcher) Start(<-chan struct{}) error {
	return nil
}

func (w *ManualWatcher) OnChange(configMap *corev1.ConfigMap) {
	if configMap.Namespace != w.Namespace {
		return
	}
	// Within our namespace, take the lock and see if there are any registered observers.
	w.m.Lock()
	defer w.m.Unlock()
	observers, ok := w.observers[configMap.Name]
	if !ok {
		return // No observers.
	}

	// Iterate over the observers and invoke their callbacks.
	for _, o := range observers {
		o(configMap)
	}
}
