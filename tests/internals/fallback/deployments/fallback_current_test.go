//go:build e2e
// +build e2e

package fallback_test

import (
	"testing"

	"github.com/joho/godotenv"

	. "github.com/kedacore/keda/v2/tests/internals/fallback"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../../.env")

func TestFallbackCurrentReplicasForDeployments(t *testing.T) {
	TestFallbackWithCurrentReplicas(t, Deployment)
}
