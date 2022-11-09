// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package internal

import (
	"context"
	"time"
)

type timeoutKey struct{}

// MakeTimeoutContext returns a new context with Client-Side Operation Timeout (CSOT) feature-gated behavior
// and a Timeout set to the passed in Duration. Setting a Timeout on a single operation is not supported in
// public API.
//
// TODO(GODRIVER-2348) We may be able to remove this function once CSOT feature-gated behavior becomes the
// TODO default behavior.
func MakeTimeoutContext(ctx context.Context, to time.Duration) (context.Context, context.CancelFunc) {
	// Only use the passed in Duration as a timeout on the Context if it
	// is non-zero.
	cancelFunc := func() {}
	if to != 0 {
		ctx, cancelFunc = context.WithTimeout(ctx, to)
	}
	return context.WithValue(ctx, timeoutKey{}, true), cancelFunc
}

func IsTimeoutContext(ctx context.Context) bool {
	return ctx.Value(timeoutKey{}) != nil
}
