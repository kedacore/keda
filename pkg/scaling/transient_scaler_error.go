/*
Copyright 2021 The KEDA Authors

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

package scaling

import (
	"errors"
	"strings"
)

// IsTransientScalerCacheRebuildError reports errors that commonly occur when the scaler cache
// is being rebuilt or closed during a ScaledJob spec/generation change (for example an old Redis
// client is closed while a scale-loop goroutine still holds a stale cache pointer).
// These are expected to self-heal on the next poll or reconcile and must not flip Ready=False.
func IsTransientScalerCacheRebuildError(err error) bool {
	for err != nil {
		msg := err.Error()
		if strings.Contains(msg, "redis: client is closed") {
			return true
		}
		if strings.Contains(msg, "scaler with id") && strings.Contains(msg, "not found") && strings.Contains(msg, "Len = 0") {
			return true
		}
		err = errors.Unwrap(err)
	}
	return false
}
