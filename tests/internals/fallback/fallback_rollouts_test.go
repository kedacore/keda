//go:build e2e
// +build e2e

package fallback_test

import (
	"testing"

	"github.com/joho/godotenv"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

func TestFallback_Rollout(t *testing.T) {
	testFallbackWithAverageValueMetrics(t, Rollout)
	testFallbackWithValueMetrics(t, Rollout)
	testFallbackWithoutMetricType(t, Rollout)
	testFallbackWithCurrentReplicasIfHigher(t, Rollout)
	testFallbackWithCurrentReplicasIfLower(t, Rollout)
	testFallbackWithCurrentReplicas(t, Rollout)
	testFallbackWithStatic(t, Rollout)
}
