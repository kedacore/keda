/*
Copyright 2026 The KEDA Authors

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

package util

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKubernetesAPIContext(t *testing.T) {
	t.Run("adds configured timeout", func(t *testing.T) {
		const timeout = time.Second
		ctx, cancel := KubernetesAPIContext(context.Background(), timeout)
		defer cancel()

		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		assert.WithinDuration(t, time.Now().Add(timeout), deadline, 100*time.Millisecond)
	})

	t.Run("preserves parent deadline when disabled", func(t *testing.T) {
		parentDeadline := time.Now().Add(time.Minute)
		parent, parentCancel := context.WithDeadline(context.Background(), parentDeadline)
		defer parentCancel()

		ctx, cancel := KubernetesAPIContext(parent, 0)
		defer cancel()
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		assert.Equal(t, parentDeadline, deadline)
	})

	t.Run("uses earlier parent deadline", func(t *testing.T) {
		parentDeadline := time.Now().Add(time.Second)
		parent, parentCancel := context.WithDeadline(context.Background(), parentDeadline)
		defer parentCancel()

		ctx, cancel := KubernetesAPIContext(parent, time.Minute)
		defer cancel()
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		assert.Equal(t, parentDeadline, deadline)
	})
}
