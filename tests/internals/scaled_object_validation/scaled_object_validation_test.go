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
	testName = "scaled-object-validation-test"
)

var (
	testNamespace                     = fmt.Sprintf("%s-ns", testName)
	deploymentName                    = fmt.Sprintf("%s-deployment", testName)
	scaledObject1Name                 = fmt.Sprintf("%s-so1", testName)
	scaledObject2Name                 = fmt.Sprintf("%s-so2", testName)
	hpaName                           = fmt.Sprintf("%s-hpa", testName)
	ownershipTransferScaledObjectName = fmt.Sprintf("%s-ownership-transfer-so", testName)
	ownershipTransferHpaName          = fmt.Sprintf("%s-ownership-transfer-hpa", testName)
)

type templateData struct {
	TestNamespace    string
	DeploymentName   string
	ScaledObjectName string
	HpaName          string
}

const (
	deploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: {{.DeploymentName}}
  template:
    metadata:
      labels:
        app: {{.DeploymentName}}
    spec:
      containers:
        - name: {{.DeploymentName}}
          image: nginxinc/nginx-unprivileged
`

	scaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  triggers:
  - type: cron
    metadata:
      timezone: Etc/UTC
      start: 0 * * * *
      end: 1 * * * *
      desiredReplicas: '1'
`

	cpuScaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  triggers:
    - type: cpu
      metadata:
        type: Utilization
        value: "50"
`
	memoryScaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  triggers:
    - type: memory
      metadata:
        type: Utilization
        value: "50"
`

	ownershipTransferScaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
  annotations:
    scaledobject.keda.sh/transfer-hpa-ownership: "true"
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  advanced:
    horizontalPodAutoscalerConfig:
      name: {{.HpaName}}
  triggers:
  - type: cron
    metadata:
      timezone: Etc/UTC
      start: 0 * * * *
      end: 1 * * * *
      desiredReplicas: '1'
`

	hpaTemplate = `
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: {{.HpaName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: {{.DeploymentName}}
  minReplicas: 1
  maxReplicas: 1
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 50
`
)

func TestScaledObjectValidations(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	testWithNotScaledWorkload(t, data)

	testScaledWorkloadByOtherScaledObject(t, data)

	testScaledWorkloadByOtherHpa(t, data)

	testScaledWorkloadByOtherHpaWithOwnershipTransfer(t, data)

	testMissingCPU(t, data)

	testMissingMemory(t, data)

	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func testWithNotScaledWorkload(t *testing.T, data templateData) {
	t.Log("--- unscaled workload ---")

	data.ScaledObjectName = scaledObject1Name
	err := KubectlApplyWithErrors(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	assert.NoErrorf(t, err, "cannot deploy the scaledObject - %s", err)

	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
}

func testScaledWorkloadByOtherScaledObject(t *testing.T, data templateData) {
	t.Log("--- already scaled workload by other scaledobject---")

	data.ScaledObjectName = scaledObject1Name
	err := KubectlApplyWithErrors(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	assert.NoErrorf(t, err, "cannot deploy the scaledObject - %s", err)

	data.ScaledObjectName = scaledObject2Name
	err = KubectlApplyWithErrors(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	assert.Errorf(t, err, "can deploy the scaledObject - %s", err)
	assert.Contains(t, err.Error(), fmt.Sprintf("the workload '%s' of type 'apps/v1.Deployment' is already managed by the ScaledObject '%s", deploymentName, scaledObject1Name))

	data.ScaledObjectName = scaledObject1Name
	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
}

func testScaledWorkloadByOtherHpa(t *testing.T, data templateData) {
	t.Log("--- already scaled workload by other hpa---")

	data.HpaName = hpaName
	err := KubectlApplyWithErrors(t, data, "hpaTemplate", hpaTemplate)
	assert.NoErrorf(t, err, "cannot deploy the hpa - %s", err)

	data.ScaledObjectName = scaledObject1Name
	err = KubectlApplyWithErrors(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	assert.Errorf(t, err, "can deploy the scaledObject - %s", err)
	assert.Contains(t, err.Error(), fmt.Sprintf("the workload '%s' of type 'apps/v1.Deployment' is already managed by the hpa '%s", deploymentName, hpaName))

	KubectlDeleteWithTemplate(t, data, "hpaTemplate", hpaTemplate)
}

func testScaledWorkloadByOtherHpaWithOwnershipTransfer(t *testing.T, data templateData) {
	t.Log("--- already scaled workload by other hpa ownership transfer ---")

	data.HpaName = ownershipTransferHpaName
	err := KubectlApplyWithErrors(t, data, "hpaTemplate", hpaTemplate)
	assert.NoErrorf(t, err, "cannot deploy the hpa - %s", err)

	data.ScaledObjectName = ownershipTransferScaledObjectName
	err = KubectlApplyWithErrors(t, data, "ownershipTransferScaledObjectTemplate", ownershipTransferScaledObjectTemplate)
	assert.NoErrorf(t, err, "can deploy the scaledObject - %s", err)

	KubectlDeleteWithTemplate(t, data, "hpaTemplate", hpaTemplate)
}

func testMissingCPU(t *testing.T, data templateData) {
	t.Log("--- missing cpu resource ---")

	data.ScaledObjectName = scaledObject1Name
	err := KubectlApplyWithErrors(t, data, "scaledObjectTemplate", cpuScaledObjectTemplate)
	assert.Errorf(t, err, "can deploy the scaledObject - %s", err)
	assert.Contains(t, err.Error(), fmt.Sprintf("the scaledobject has a cpu trigger but the container %s doesn't have the cpu request defined", deploymentName))
}

func testMissingMemory(t *testing.T, data templateData) {
	t.Log("--- missing memory resource ---")

	data.ScaledObjectName = scaledObject1Name
	err := KubectlApplyWithErrors(t, data, "scaledObjectTemplate", memoryScaledObjectTemplate)
	assert.Errorf(t, err, "can deploy the scaledObject - %s", err)
	assert.Contains(t, err.Error(), fmt.Sprintf("the scaledobject has a memory trigger but the container %s doesn't have the memory request defined", deploymentName))
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:  testNamespace,
			DeploymentName: deploymentName,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
		}
}
