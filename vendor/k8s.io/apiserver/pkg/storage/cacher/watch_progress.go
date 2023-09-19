/*
Copyright 2023 The Kubernetes Authors.

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

package cacher

import (
	"context"
	"sync"
	"time"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"

	"k8s.io/klog/v2"
	"k8s.io/utils/clock"
)

const (
	// progressRequestPeriod determines period of requesting progress
	// from etcd when there is a request waiting for watch cache to be fresh.
	progressRequestPeriod = 100 * time.Millisecond
)

func newConditionalProgressRequester(requestWatchProgress WatchProgressRequester, clock TickerFactory) *conditionalProgressRequester {
	pr := &conditionalProgressRequester{
		clock:                clock,
		requestWatchProgress: requestWatchProgress,
	}
	pr.cond = sync.NewCond(pr.mux.RLocker())
	return pr
}

type WatchProgressRequester func(ctx context.Context) error

type TickerFactory interface {
	NewTicker(time.Duration) clock.Ticker
}

// conditionalProgressRequester will request progress notification if there
// is a request waiting for watch cache to be fresh.
type conditionalProgressRequester struct {
	clock                TickerFactory
	requestWatchProgress WatchProgressRequester

	mux     sync.RWMutex
	cond    *sync.Cond
	waiting int
	stopped bool
}

func (pr *conditionalProgressRequester) Run(stopCh <-chan struct{}) {
	ctx := wait.ContextForChannel(stopCh)
	go func() {
		defer utilruntime.HandleCrash()
		<-stopCh
		pr.mux.Lock()
		defer pr.mux.Unlock()
		pr.stopped = true
		pr.cond.Signal()
	}()
	ticker := pr.clock.NewTicker(progressRequestPeriod)
	defer ticker.Stop()
	for {
		stopped := func() bool {
			pr.mux.RLock()
			defer pr.mux.RUnlock()
			for pr.waiting == 0 && !pr.stopped {
				pr.cond.Wait()
			}
			return pr.stopped
		}()
		if stopped {
			return
		}

		select {
		case <-ticker.C():
			shouldRequest := func() bool {
				pr.mux.RLock()
				defer pr.mux.RUnlock()
				return pr.waiting > 0 && !pr.stopped
			}()
			if !shouldRequest {
				continue
			}
			err := pr.requestWatchProgress(ctx)
			if err != nil {
				klog.V(4).InfoS("Error requesting bookmark", "err", err)
			}
		case <-stopCh:
			return
		}
	}
}

func (pr *conditionalProgressRequester) Add() {
	pr.mux.Lock()
	defer pr.mux.Unlock()
	pr.waiting += 1
	pr.cond.Signal()
}

func (pr *conditionalProgressRequester) Remove() {
	pr.mux.Lock()
	defer pr.mux.Unlock()
	pr.waiting -= 1
	pr.cond.Signal()
}
