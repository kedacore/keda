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

package scaling

import (
	"hash/fnv"
	"time"

	"k8s.io/apimachinery/pkg/types"
)

// jitterOffset returns a deterministic offset in [0, interval) derived
// from the given UID. It is used to stagger the first tick of each
// scale loop so that ScalableObjects created in the same reconcile
// batch do not all poll on the same 30-second boundary.
//
// The offset is a function of the UID only, so the phase is stable
// across operator restarts: when the operator comes back up and
// re-spawns scale loops for existing objects, each loop lands at the
// same phase it had before.
func jitterOffset(uid types.UID, interval time.Duration) time.Duration {
	if interval <= 0 || uid == "" {
		return 0
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(uid))
	return time.Duration(h.Sum64() % uint64(interval))
}
