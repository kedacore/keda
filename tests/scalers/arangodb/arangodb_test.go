//go:build e2e
// +build e2e

package arangodb_test

import (
	"fmt"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
	"github.com/kedacore/keda/v2/tests/scalers/arangodb"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "arangodb-test"
)

var (
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	secretName       = fmt.Sprintf("%s-secret", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
	triggerAuthName  = fmt.Sprintf("%s-ta", testName)

	arangoDBUsername   = "cm9vdA=="
	arangoDBName       = "test"
	arangoDBCollection = "test"

	minReplicaCount = 0
	maxReplicaCount = 2
)

type templateData struct {
	TestNamespace    string
	DeploymentName   string
	ScaledObjectName string
	Database         string
	Collection       string
	TriggerAuthName  string
	Username         string
	SecretName       string
	MinReplicaCount  int
	MaxReplicaCount  int
}

const (
	deploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: test-app
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
spec:
  replicas: 0
  selector:
    matchLabels:
      app: test-app
  template:
    metadata:
      labels:
        app: test-app
        type: keda-testing
    spec:
      containers:
      - name: nginx
        image: ghcr.io/nginx/nginx-unprivileged:1.26
        ports:
        - containerPort: 80
`

	secretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  username: {{.Username}}
`

	triggerAuthTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthName}}
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
  - parameter: username
    name: {{.SecretName}}
    key: username
`

	scaledObjectTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  pollingInterval: 3
  cooldownPeriod:  1
  triggers:
  - type: arangodb
    metadata:
      endpoints: https://example-arangodb-cluster-int.{{.TestNamespace}}.svc.cluster.local:8529
      queryValue: '3'
      activationQueryValue: '3'
      dbName: {{.Database}}
      collection: {{.Collection}}
      unsafeSsl: "true"
      query: FOR doc IN {{.Collection}} COLLECT WITH COUNT INTO length RETURN {"value":length}
      authModes: "basic"
    authenticationRef:
      name: {{.TriggerAuthName}}
`

	generateLowLevelDataJobTemplate = `apiVersion: batch/v1
kind: Job
metadata:
  name: generate-low-level-data-job
  namespace: {{.TestNamespace}}
spec:
  template:
    spec:
      containers:
      - image: ghcr.io/nginx/nginx-unprivileged:1.26
        name: test
        command: ["/bin/sh"]
        args: ["-c", "curl --location --request POST 'https://example-arangodb-cluster-ea.{{.TestNamespace}}.svc.cluster.local:8529/_db/{{.Database}}/_api/document/{{.Collection}}' --header 'Authorization: Basic cm9vdDo=' --data-raw '[{\"Hi\": \"Nathan\"}, {\"Hi\": \"Laura\"}]' -k"]
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
              - ALL
          seccompProfile:
            type: RuntimeDefault
      restartPolicy: Never
  activeDeadlineSeconds: 100
  backoffLimit: 2
`

	generateDataJobTemplate = `apiVersion: batch/v1
kind: Job
metadata:
  name: generate-data-job
  namespace: {{.TestNamespace}}
spec:
  template:
    spec:
      containers:
      - image: ghcr.io/nginx/nginx-unprivileged:1.26
        name: test
        command: ["/bin/sh"]
        args: ["-c", "curl --location --request POST 'https://example-arangodb-cluster-ea.{{.TestNamespace}}.svc.cluster.local:8529/_db/{{.Database}}/_api/document/{{.Collection}}' --header 'Authorization: Basic cm9vdDo=' --data-raw '[{\"Hi\": \"Harry\"}, {\"Hi\": \"Neha\"}]' -k"]
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
              - ALL
          seccompProfile:
            type: RuntimeDefault
      restartPolicy: Never
  activeDeadlineSeconds: 100
  backoffLimit: 2
`

	deleteDataJobTemplate = `apiVersion: batch/v1
kind: Job
metadata:
  name: delete-data-job
  namespace: {{.TestNamespace}}
spec:
  template:
    spec:
      containers:
      - image: ghcr.io/nginx/nginx-unprivileged:1.26
        name: test
        command: ["/bin/sh"]
        args: ["-c", "curl --location --request POST 'https://example-arangodb-cluster-ea.{{.TestNamespace}}.svc.cluster.local:8529/_db/{{.Database}}/_api/cursor' --header 'Authorization: Basic cm9vdDo=' --data-raw '{\"query\": \"FOR doc in {{.Collection}} REMOVE doc in {{.Collection}}\"}' -k"]
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
              - ALL
          seccompProfile:
            type: RuntimeDefault
      restartPolicy: Never
  activeDeadlineSeconds: 100
  backoffLimit: 2
`
)

func TestArangoDBScaler(t *testing.T) {
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	CreateNamespace(t, kc, testNamespace)
	t.Cleanup(func() {
		arangodb.UninstallArangoDB(t, testNamespace)
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})

	// Create kubernetes resources
	arangodb.InstallArangoDB(t, kc, testNamespace)
	arangodb.SetupArangoDB(t, kc, testNamespace, arangoDBName, arangoDBCollection)
	KubectlApplyMultipleWithTemplate(t, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)

	testActivation(t, kc, data)
	testScaleOut(t, kc, data)
	testScaleIn(t, kc, data)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:    testNamespace,
			DeploymentName:   deploymentName,
			ScaledObjectName: scaledObjectName,
			MinReplicaCount:  minReplicaCount,
			MaxReplicaCount:  maxReplicaCount,
			Database:         arangoDBName,
			Collection:       arangoDBCollection,
			TriggerAuthName:  triggerAuthName,
			SecretName:       secretName,
			Username:         arangoDBUsername,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "triggerAuthTemplate", Config: triggerAuthTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")

	KubectlReplaceWithTemplate(t, data, "generateLowLevelDataJobTemplate", generateLowLevelDataJobTemplate)
	assert.True(t, WaitForJobSuccess(t, kc, "generate-low-level-data-job", testNamespace, 5, 60), "test activation job failed")

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")

	KubectlReplaceWithTemplate(t, data, "generateDataJobTemplate", generateDataJobTemplate)
	assert.True(t, WaitForJobSuccess(t, kc, "generate-data-job", testNamespace, 5, 60), "test scale-out job failed")

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale in ---")

	KubectlReplaceWithTemplate(t, data, "deleteDataJobTemplate", deleteDataJobTemplate)
	assert.True(t, WaitForJobSuccess(t, kc, "delete-data-job", testNamespace, 5, 60), "test scale-in job failed")

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 5),
		"replica count should be %d after 5 minutes", minReplicaCount)
}
