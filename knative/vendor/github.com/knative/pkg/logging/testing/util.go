/*
Copyright 2018 The Knative Authors.

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

package testing

import (
	"context"
	"sync"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"github.com/knative/pkg/logging"
)

var (
	loggers = make(map[string]*zap.SugaredLogger)
	m       sync.Mutex
)

// TestLogger gets a logger to use in unit and end to end tests
func TestLogger(t *testing.T) *zap.SugaredLogger {
	m.Lock()
	defer m.Unlock()

	logger, ok := loggers[t.Name()]

	if ok {
		return logger
	}

	opts := zaptest.WrapOptions(
		zap.AddCaller(),
		zap.Development(),
	)

	logger = zaptest.NewLogger(t, opts).Sugar().Named(t.Name())
	loggers[t.Name()] = logger

	return logger
}

// ClearAll removes all the testing loggers.
// `go test -count=X` executes runs in the same process, thus the map
// persists between the runs, but the `t` will no longer be valid and will
// cause a panic deep inside testing code.
func ClearAll() {
	loggers = make(map[string]*zap.SugaredLogger)
}

// TestContextWithLogger returns a context with a logger to be used in tests
func TestContextWithLogger(t *testing.T) context.Context {
	return logging.WithLogger(context.TODO(), TestLogger(t))
}
