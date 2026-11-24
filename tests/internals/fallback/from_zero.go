//go:build e2e
// +build e2e

//nolint:goconst
package fallback

import (
	"testing"

	"github.com/stretchr/testify/assert"

	helper "github.com/kedacore/keda/v2/tests/helper"
)

func TestFallbackFromZero(t *testing.T, s ScaleTargetType) {
	kc := helper.GetKubernetesClient(t)
	t.Logf("--- running TestFallbackFromZero test for %s ---", s)
	data, templates := getTemplateData(s, "from-zero")

	// Replace the default ScaledObject template
	for i, tmpl := range templates {
		if tmpl.Name == "scaledObjectTemplate" {
			templates[i].Config = scaledObjectTemplateWithStatic
			break
		}
	}

	helper.CreateKubernetesResources(t, kc, data.Namespace, data, templates)

	assert.True(t, scaleTargetMap[s].WaitForReplicaReadyCount(t, kc, scaleTargetName, data.Namespace, minReplicas, 180, 3),
		"replica count should be %d after 9 minutes", minReplicas)

	// Stop metrics server to trigger fallback
	helper.KubectlApplyWithTemplate(t, data, "fallbackMSDeploymentTemplate", fallbackMSDeploymentTemplate)

	// Should go to fallback value (3) because of static
	assert.True(t, scaleTargetMap[s].WaitForReplicaReadyCount(t, kc, scaleTargetName, data.Namespace, 3, 30, 3),
		"replica count should be 3 after fallback")

	// Ensure the replica count remains stable
	scaleTargetMap[s].AssertReplicaCountNotChangeDuringTimePeriod(t, kc, scaleTargetName, data.Namespace, 3, 30)

	helper.DeleteKubernetesResources(t, data.Namespace, data, templates)
}
