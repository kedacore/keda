//
// DISCLAIMER
//
// Copyright 2023 ArangoDB GmbH, Cologne, Germany
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Copyright holder is ArangoDB GmbH, Cologne, Germany
//

package driver

import "context"

// IsAsyncRequest returns true if the given context is an async request.
func IsAsyncRequest(ctx context.Context) bool {
	if ctx != nil {
		if v := ctx.Value(keyAsyncRequest); v != nil {
			if isAsync, ok := v.(bool); ok && isAsync {
				return true
			}
		}
	}

	return false
}

// HasAsyncID returns the async Job ID from the given context.
func HasAsyncID(ctx context.Context) (string, bool) {
	if ctx != nil {
		if q := ctx.Value(keyAsyncID); q != nil {
			if v, ok := q.(string); ok {
				return v, true
			}
		}
	}

	return "", false
}

// HasTransactionID	returns the transaction ID from the given context.
func HasTransactionID(ctx context.Context) (TransactionID, bool) {
	if ctx != nil {
		if q := ctx.Value(keyTransactionID); q != nil {
			if v, ok := q.(TransactionID); ok {
				return v, true
			}
		}
	}

	return "", false
}

// HasReturnNew is used to fetch the new document from the context.
func HasReturnNew(ctx context.Context) (interface{}, bool) {
	if ctx != nil {
		if q := ctx.Value(keyReturnNew); q != nil {
			return q, true
		}
	}
	return nil, false
}

// HasReturnOld is used to fetch the old document from the context.
func HasReturnOld(ctx context.Context) (interface{}, bool) {
	if ctx != nil {
		if q := ctx.Value(keyReturnOld); q != nil {
			return q, true
		}
	}
	return nil, false
}
