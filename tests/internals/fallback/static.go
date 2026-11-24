//go:build e2e
// +build e2e

//nolint:dupl,goconst
package fallback

import (
	"testing"

	"github.com/stretchr/testify/assert"

	helper "github.com/kedacore/keda/v2/tests/helper"
)

func TestFallbackWithStatic(t *testing.T, s ScaleTargetType) {
	kc := helper.GetKubernetesClient(t)
	t.Logf("--- running TestFallbackWithStatic test for %s ---", s)
	data, templates := getTemplateData(s, "static")

	// Replace the default scaledObject template
	for i, tmpl := range templates {
		if tmpl.Name == "scaledObjectTemplate" {
			templates[i].Config = scaledObjectTemplateWithStatic
			break
		}
	}

	helper.CreateKubernetesResources(t, kc, data.Namespace, data, templates)

	assert.True(t, scaleTargetMap[s].WaitForReplicaReadyCount(t, kc, scaleTargetName, data.Namespace, minReplicas, 180, 3),
		"replica count should be %d after 9 minutes", minReplicas)

	// Scale out to 4 replicas (20 / 5 = 4)
	data.MetricValue = 20
	helper.KubectlReplaceWithTemplate(t, data, "updateMetricsTemplate", updateMetricsTemplate)

	assert.True(t, scaleTargetMap[s].WaitForReplicaReadyCount(t, kc, scaleTargetName, data.Namespace, 4, 60, 3),
		"replica count should be 4 after 3 minutes")

	// Stop metrics server to trigger fallback
	helper.KubectlApplyWithTemplate(t, data, "fallbackMSDeploymentTemplate", fallbackMSDeploymentTemplate)

	// Should keep fallback value (3) because of static
	assert.True(t, scaleTargetMap[s].WaitForReplicaReadyCount(t, kc, scaleTargetName, data.Namespace, 3, 30, 3),
		"replica count should remain at 3 after fallback")

	// Ensure the replica count remains stable
	scaleTargetMap[s].AssertReplicaCountNotChangeDuringTimePeriod(t, kc, scaleTargetName, data.Namespace, 3, 30)

	helper.DeleteKubernetesResources(t, data.Namespace, data, templates)
}
