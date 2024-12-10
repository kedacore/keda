//go:build e2e
// +build e2e

package cache_metrics_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	testName = "scaled-object-validation-test"
)

var (
	testNamespace                     = fmt.Sprintf("%s-ns", testName)
	deployment1Name                   = fmt.Sprintf("%s-deployment", testName)
	deployment2Name                   = fmt.Sprintf("%s-deployment2", testName)
	scaledObject1Name                 = fmt.Sprintf("%s-so1", testName)
	scaledObject2Name                 = fmt.Sprintf("%s-so2", testName)
	emptyTriggersSoName               = fmt.Sprintf("%s-so-empty-triggers", testName)
	hpaName                           = fmt.Sprintf("%s-hpa", testName)
	ownershipTransferScaledObjectName = fmt.Sprintf("%s-ownership-transfer-so", testName)
	ownershipTransferHpaName          = fmt.Sprintf("%s-ownership-transfer-hpa", testName)
)

type templateData struct {
	TestNamespace       string
	DeploymentName      string
	ScaledObjectName    string
	HpaName             string
	EmptyTriggersSoName string
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

	customHpaScaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
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

	emptyTriggersTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.EmptyTriggersSoName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  triggers: []
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

	testManagedHpaByOtherScaledObject(t, data)

	testScaledWorkloadByOtherHpa(t, data)

	testScaledWorkloadByOtherHpaWithOwnershipTransfer(t, data)

	testMissingCPU(t, data)

	testMissingMemory(t, data)

	testWorkloadWithOnlyLimits(t, data)

	testTriggersWithEmptyArray(t, data)

	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func testWithNotScaledWorkload(t *testing.T, data templateData) {
	t.Log("--- scaled workload ---")
	err := KubectlApplyWithErrors(t, data, "deploymentTemplate", deploymentTemplate)
	assert.Errorf(t, err, "cannot deploy the deployment - %s", err)

	data.ScaledObjectName = scaledObject1Name
	err := KubectlApplyWithErrors(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	assert.NoErrorf(t, err, "cannot deploy the scaledObject - %s", err)

	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	KubectlDeleteWithTemplate(t, data, "deploymentTemplate", deploymentTemplate)
}

func testScaledWorkloadByOtherScaledObject(t *testing.T, data templateData) {
	t.Log("--- already scaled workload by other scaledobject---")

	err := KubectlApplyWithErrors(t, data, "deploymentTemplate", deploymentTemplate)
	assert.Errorf(t, err, "cannot deploy the deployment - %s", err)

	data.ScaledObjectName = scaledObject1Name
	err := KubectlApplyWithErrors(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	assert.NoErrorf(t, err, "cannot deploy the scaledObject - %s", err)

	data.ScaledObjectName = scaledObject2Name
	err = KubectlApplyWithErrors(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	assert.Errorf(t, err, "can deploy the scaledObject - %s", err)
	assert.Contains(t, err.Error(), fmt.Sprintf("the workload '%s' of type 'apps/v1.Deployment' is already managed by the ScaledObject '%s", deployment1Name, scaledObject1Name))

	data.ScaledObjectName = scaledObject1Name
	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	KubectlDeleteWithTemplate(t, data, "deploymentTemplate", deploymentTemplate)
}

func testManagedHpaByOtherScaledObject(t *testing.T, data templateData) {
	t.Log("--- already managed hpa by other scaledobject---")

	data.HpaName = hpaName

	err := KubectlApplyWithErrors(t, data, "deploymentTemplate", deploymentTemplate)
	assert.Errorf(t, err, "cannot deploy the deployment - %s", err)

	data.ScaledObjectName = scaledObject1Name
	err := KubectlApplyWithErrors(t, data, "scaledObjectTemplate", customHpaScaledObjectTemplate)
	assert.NoErrorf(t, err, "cannot deploy the scaledObject - %s", err)

	data.ScaledObjectName = scaledObject2Name
	data.DeploymentName = deployment2Name

	err := KubectlApplyWithErrors(t, data, "deploymentTemplate", deploymentTemplate)
	assert.Errorf(t, err, "cannot deploy the deployment - %s", err)

	err = KubectlApplyWithErrors(t, data, "scaledObjectTemplate", customHpaScaledObjectTemplate)
	assert.Errorf(t, err, "can deploy the scaledObject - %s", err)
	assert.Contains(t, err.Error(), fmt.Sprintf("the HPA '%s' is already managed by the ScaledObject '%s", hpaName, scaledObject1Name))

	data.ScaledObjectName = scaledObject1Name
	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	KubectlDeleteWithTemplate(t, data, "deploymentTemplate", deploymentTemplate)
	data.DeploymentName = deployment1Name
	KubectlDeleteWithTemplate(t, data, "deploymentTemplate", deploymentTemplate)
}

func testScaledWorkloadByOtherHpa(t *testing.T, data templateData) {
	t.Log("--- already scaled workload by other hpa---")

	err := KubectlApplyWithErrors(t, data, "deploymentTemplate", deploymentTemplate)
	assert.Errorf(t, err, "cannot deploy the deployment - %s", err)

	data.HpaName = hpaName
	err := KubectlApplyWithErrors(t, data, "hpaTemplate", hpaTemplate)
	assert.NoErrorf(t, err, "cannot deploy the hpa - %s", err)

	data.ScaledObjectName = scaledObject1Name
	err = KubectlApplyWithErrors(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	assert.Errorf(t, err, "can deploy the scaledObject - %s", err)
	assert.Contains(t, err.Error(), fmt.Sprintf("the workload '%s' of type 'apps/v1.Deployment' is already managed by the hpa '%s", deployment1Name, hpaName))

	KubectlDeleteWithTemplate(t, data, "hpaTemplate", hpaTemplate)
	KubectlDeleteWithTemplate(t, data, "deploymentTemplate", deploymentTemplate)
}

func testScaledWorkloadByOtherHpaWithOwnershipTransfer(t *testing.T, data templateData) {
	t.Log("--- already scaled workload by other hpa ownership transfer ---")

	err := KubectlApplyWithErrors(t, data, "deploymentTemplate", deploymentTemplate)
	assert.Errorf(t, err, "cannot deploy the deployment - %s", err)

	data.HpaName = ownershipTransferHpaName
	err := KubectlApplyWithErrors(t, data, "hpaTemplate", hpaTemplate)
	assert.NoErrorf(t, err, "cannot deploy the hpa - %s", err)

	data.ScaledObjectName = ownershipTransferScaledObjectName
	err = KubectlApplyWithErrors(t, data, "ownershipTransferScaledObjectTemplate", ownershipTransferScaledObjectTemplate)
	assert.NoErrorf(t, err, "can deploy the scaledObject - %s", err)

	KubectlDeleteWithTemplate(t, data, "hpaTemplate", hpaTemplate)
	KubectlDeleteWithTemplate(t, data, "ownershipTransferScaledObjectTemplate", ownershipTransferScaledObjectTemplate)
	KubectlDeleteWithTemplate(t, data, "deploymentTemplate", deploymentTemplate)
}

func testMissingCPU(t *testing.T, data templateData) {
	t.Log("--- missing cpu resource ---")

	err := KubectlApplyWithErrors(t, data, "deploymentTemplate", deploymentTemplate)
	assert.Errorf(t, err, "cannot deploy the deployment - %s", err)

	data.ScaledObjectName = scaledObject1Name
	err := KubectlApplyWithErrors(t, data, "scaledObjectTemplate", cpuScaledObjectTemplate)
	assert.Errorf(t, err, "can deploy the scaledObject - %s", err)
	assert.Contains(t, err.Error(), fmt.Sprintf("the scaledobject has a cpu trigger but the container %s doesn't have the cpu request defined", deployment1Name))

	KubectlDeleteWithTemplate(t, data, "deploymentTemplate", deploymentTemplate)
}

func testMissingMemory(t *testing.T, data templateData) {
	t.Log("--- missing memory resource ---")

	err := KubectlApplyWithErrors(t, data, "deploymentTemplate", deploymentTemplate)
	assert.Errorf(t, err, "cannot deploy the deployment - %s", err)

	data.ScaledObjectName = scaledObject1Name
	err := KubectlApplyWithErrors(t, data, "scaledObjectTemplate", memoryScaledObjectTemplate)
	assert.Errorf(t, err, "can deploy the scaledObject - %s", err)
	assert.Contains(t, err.Error(), fmt.Sprintf("the scaledobject has a memory trigger but the container %s doesn't have the memory request defined", deployment1Name))

	KubectlDeleteWithTemplate(t, data, "deploymentTemplate", deploymentTemplate)
}

func testWorkloadWithOnlyLimits(t *testing.T, data templateData) {
	t.Log("--- workload with only resource limits set ---")

	data.DeploymentName = fmt.Sprintf("%s-deploy-only-limits", testName)

	customDeploymentTemplate := `
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
          resources:
            limits:
              cpu: 50m
`

	KubectlApplyWithTemplate(t, data, "deploymentTemplate", customDeploymentTemplate)
	WaitForDeploymentReplicaReadyCount(t, GetKubernetesClient(t), data.DeploymentName, data.TestNamespace, 1, 10, 5)

	t.Log("deployment was updated with resource limits")

	data.ScaledObjectName = fmt.Sprintf("%s-so-only-limits", testName)

	customScaledObjectTemplate := `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  minReplicaCount: 1
  maxReplicaCount: 1
  triggers:
    - type: cpu
      metadata:
        type: Utilization
        value: "50"
`

	err := KubectlApplyWithErrors(t, data, "scaledObjectTemplate", customScaledObjectTemplate)
	assert.NoError(t, err, "Deployment with only resource limits set should be validated")
}

func testTriggersWithEmptyArray(t *testing.T, data templateData) {
	t.Log("--- triggers with empty array ---")

	err := KubectlApplyWithErrors(t, data, "deploymentTemplate", deploymentTemplate)
	assert.Errorf(t, err, "cannot deploy the deployment - %s", err)

	err := KubectlApplyWithErrors(t, data, "emptyTriggersTemplate", emptyTriggersTemplate)
	assert.Errorf(t, err, "can deploy the scaledObject - %s", err)
	assert.Contains(t, err.Error(), "no triggers defined in the ScaledObject/ScaledJob")

	KubectlDeleteWithTemplate(t, data, "deploymentTemplate", deploymentTemplate)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:       testNamespace,
			DeploymentName:      deployment1Name,
			EmptyTriggersSoName: emptyTriggersSoName,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
		}
}
