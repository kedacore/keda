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
	"fmt"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/types"
)

func TestJitterOffset_InRange(t *testing.T) {
	interval := 30 * time.Second
	for i := 0; i < 1000; i++ {
		uid := types.UID(fmt.Sprintf("fake-uid-%d-%d-%d-%d", i, i*7, i*13, i*17))
		got := jitterOffset(uid, interval)
		if got < 0 || got >= interval {
			t.Fatalf("offset %s for uid %q out of range [0, %s)", got, uid, interval)
		}
	}
}

func TestJitterOffset_Deterministic(t *testing.T) {
	interval := 30 * time.Second
	uid := types.UID("f3c2e9b0-1a7d-4c21-b0e4-dd1c7f5e9a00")

	first := jitterOffset(uid, interval)
	for i := 0; i < 100; i++ {
		if got := jitterOffset(uid, interval); got != first {
			t.Fatalf("jitterOffset not deterministic: iteration %d got %s, want %s", i, got, first)
		}
	}
}

func TestJitterOffset_ZeroInterval(t *testing.T) {
	if got := jitterOffset("some-uid", 0); got != 0 {
		t.Fatalf("zero interval should return 0, got %s", got)
	}
	if got := jitterOffset("some-uid", -time.Second); got != 0 {
		t.Fatalf("negative interval should return 0, got %s", got)
	}
}

func TestJitterOffset_EmptyUID(t *testing.T) {
	if got := jitterOffset("", 30*time.Second); got != 0 {
		t.Fatalf("empty UID should return 0, got %s", got)
	}
}

// TestJitterOffset_Distribution checks that offsets from a large set
// of UIDs are reasonably spread across the interval. The guard is
// deliberately loose: we only fail on an obviously degenerate hash
// (e.g. everything bunched into one decile), not on statistical
// outliers of a fair hash.
func TestJitterOffset_Distribution(t *testing.T) {
	const (
		count    = 10000
		buckets  = 10
		interval = 30 * time.Second
		// With 10000 samples across 10 buckets, the expected count per
		// bucket is 1000. 2σ for a binomial(10000, 0.1) is ~60; we
		// allow much wider tolerance to avoid flakiness.
		minPerBucket = 700
		maxPerBucket = 1300
	)
	counts := make([]int, buckets)
	bucketWidth := interval / time.Duration(buckets)
	for i := 0; i < count; i++ {
		uid := types.UID(fmt.Sprintf("scaledobject-uid-%d", i))
		offset := jitterOffset(uid, interval)
		b := int(offset / bucketWidth)
		if b >= buckets {
			b = buckets - 1
		}
		counts[b]++
	}
	for i, c := range counts {
		if c < minPerBucket || c > maxPerBucket {
			t.Errorf("bucket %d has count %d, outside tolerance [%d, %d] -- distribution may be degenerate. Full counts: %v",
				i, c, minPerBucket, maxPerBucket, counts)
		}
	}
}
