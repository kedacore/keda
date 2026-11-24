//go:build e2e
// +build e2e

//nolint:goconst
package fallback

import (
	"testing"

	"github.com/stretchr/testify/assert"

	helper "github.com/kedacore/keda/v2/tests/helper"
)

func TestFallbackWithAverageValueMetrics(t *testing.T, s ScaleTargetType) {
	kc := helper.GetKubernetesClient(t)
	// setup
	t.Logf("--- running TestFallbackWithAverageValueMetrics for %s ---", s)
	data, templates := getTemplateData(s, "avg-value")

	helper.CreateKubernetesResources(t, kc, data.Namespace, data, templates)

	assert.True(t, scaleTargetMap[s].WaitForReplicaReadyCount(t, kc, scaleTargetName, data.Namespace, minReplicas, 180, 3),
		"replica count should be %d after 9 minutes", minReplicas)

	testScaleOut(t, kc, s, data)
	testFallback(t, kc, s, data)
	testRestoreAfterFallback(t, kc, s, data)

	helper.DeleteKubernetesResources(t, data.Namespace, data, templates)
}
