//go:build e2e
// +build e2e

package cache_metrics_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName = "scaled-job-validation-test"
)

var (
	testNamespace       = fmt.Sprintf("%s-ns", testName)
	emptyTriggersSjName = fmt.Sprintf("%s-sj-empty-triggers", testName)
)

type templateData struct {
	TestNamespace       string
	EmptyTriggersSjName string
}

const (
	emptyTriggersSjTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledJob
metadata:
  name: {{.EmptyTriggersSjName}}
  namespace: {{.TestNamespace}}
spec:
  jobTargetRef:
    template:
      spec:
        containers:
        - name: demo-rabbitmq-client
          image: demo-rabbitmq-client:1
          imagePullPolicy: Always
          command: ["receive",  "amqp://user:PASSWORD@rabbitmq.default.svc.cluster.local:5672"]
          envFrom:
            - secretRef:
                name: rabbitmq-consumer-secrets
        restartPolicy: Never
  triggers: []
`
)

func TestScaledJobValidations(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	testTriggersWithEmptyArray(t, data)

	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func testTriggersWithEmptyArray(t *testing.T, data templateData) {
	t.Log("--- triggers with empty array ---")

	err := KubectlApplyWithErrors(t, data, "emptyTriggersSjTemplate", emptyTriggersSjTemplate)
	assert.Errorf(t, err, "can deploy the scaledJob - %s", err)
	assert.Contains(t, err.Error(), "no triggers defined in the ScaledObject/ScaledJob")
}

func getTemplateData() (templateData, []Template) {
	return templateData{
		TestNamespace:       testNamespace,
		EmptyTriggersSjName: emptyTriggersSjName,
	}, []Template{}
}
