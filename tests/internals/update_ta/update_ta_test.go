//go:build e2e
// +build e2e

package update_ta_so_test

import (
	"fmt"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"

	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName = "update-ta-so-test"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

var (
	namespace              = fmt.Sprintf("%s-ns", testName)
	deploymentName         = fmt.Sprintf("%s-deployment", testName)
	deployment2Name        = fmt.Sprintf("%s-deployment-2", testName)
	scaledObjectName       = fmt.Sprintf("%s-so", testName)
	scaledObject2Name      = fmt.Sprintf("%s-so-2", testName)
	secretName             = fmt.Sprintf("%s-secret", testName)
	triggerAuthName        = fmt.Sprintf("%s-ta", testName)
	scaledJobName          = fmt.Sprintf("%s-sj", testName)
	scaledJob2Name         = fmt.Sprintf("%s-sj-2", testName)
	minReplicas            = 1
	maxReplicas            = 5
	triggerAuthKind        = "TriggerAuthentication"
	clusterTriggerAuthKind = "ClusterTriggerAuthentication"
)

type templateData struct {
	TestNamespace   string
	TriggerAuthKind string
	DeploymentName  string
	Deployment2Name string
	ScaledObject    string
	ScaledObject2   string
	TriggerAuthName string
	SecretName      string
	MinReplicas     string
	MaxReplicas     string
	ScaledJob       string
	ScaledJob2      string
}

const (
	secretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  AUTH_PASSWORD: U0VDUkVUCg==
  AUTH_USERNAME: VVNFUgo=
`

	triggerAuthenticationTemplate = `apiVersion: keda.sh/v1alpha1
kind: {{.TriggerAuthKind}}
metadata:
  name: {{.TriggerAuthName}}
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
    - parameter: username
      name: {{.SecretName}}
      key: AUTH_USERNAME
    - parameter: password
      name: {{.SecretName}}
      key: AUTH_PASSWORD
`

	deploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    deploy: {{.DeploymentName}}
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
spec:
  selector:
    matchLabels:
      app: {{.DeploymentName}}
  replicas: {{.MinReplicas}}
  template:
    metadata:
      labels:
        app: {{.DeploymentName}}
    spec:
      containers:
      - name: nginx
        image: ghcr.io/nginx/nginx-unprivileged:1.26
        ports:
        - containerPort: 80
`

	deployment2Template = `
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    deploy: {{.Deployment2Name}}
  name: {{.Deployment2Name}}
  namespace: {{.TestNamespace}}
spec:
  selector:
    matchLabels:
      app: {{.Deployment2Name}}
  replicas: {{.MinReplicas}}
  template:
    metadata:
      labels:
        app: {{.Deployment2Name}}
    spec:
      containers:
      - name: nginx
        image: ghcr.io/nginx/nginx-unprivileged:1.26
        ports:
        - containerPort: 80
`

	scaledObjectTriggerTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObject}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 10
  pollingInterval: 10
  scaleTargetRef:
    name: {{.DeploymentName}}
  minReplicaCount: {{.MinReplicas}}
  maxReplicaCount: {{.MaxReplicas}}
  cooldownPeriod: 1
  triggers:
  - type: metrics-api
    metadata:
      targetValue: "2"
      url: "invalid-invalid"
      valueLocation: 'value'
      method: "query"
    authenticationRef:
      name: {{.TriggerAuthName}}
      kind: {{.TriggerAuthKind}}
`

	scaledObjectTrigger2Template = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObject2}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 10
  pollingInterval: 10
  scaleTargetRef:
    name: {{.Deployment2Name}}
  minReplicaCount: {{.MinReplicas}}
  maxReplicaCount: {{.MaxReplicas}}
  cooldownPeriod: 1
  triggers:
  - type: metrics-api
    metadata:
      targetValue: "2"
      url: "invalid-invalid"
      valueLocation: 'value'
      method: "query"
    authenticationRef:
      name: {{.TriggerAuthName}}
      kind: {{.TriggerAuthKind}}
`

	scaledJobTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledJob
metadata:
  name: {{.ScaledJob}}
  namespace: {{.TestNamespace}}
spec:
  jobTargetRef:
    template:
      spec:
        containers:
          - name: external-executor
            image: busybox
            command:
            - sleep
            - "30"
            imagePullPolicy: IfNotPresent
        restartPolicy: Never
    backoffLimit: 1
  pollingInterval: 5
  minReplicaCount: {{.MinReplicas}}
  maxReplicaCount: {{.MaxReplicas}}
  successfulJobsHistoryLimit: 0
  failedJobsHistoryLimit: 0
  triggers:
  - type: metrics-api
    metadata:
      targetValue: "2"
      url: "invalid-invalid"
      valueLocation: 'value'
      method: "query"
    authenticationRef:
      name: {{.TriggerAuthName}}
      kind: {{.TriggerAuthKind}}
`

	scaledJob2Template = `
apiVersion: keda.sh/v1alpha1
kind: ScaledJob
metadata:
  name: {{.ScaledJob2}}
  namespace: {{.TestNamespace}}
spec:
  jobTargetRef:
    template:
      spec:
        containers:
          - name: external-executor
            image: busybox
            command:
            - sleep
            - "30"
            imagePullPolicy: IfNotPresent
        restartPolicy: Never
    backoffLimit: 1
  pollingInterval: 5
  minReplicaCount: {{.MinReplicas}}
  maxReplicaCount: {{.MaxReplicas}}
  successfulJobsHistoryLimit: 0
  failedJobsHistoryLimit: 0
  triggers:
  - type: metrics-api
    metadata:
      targetValue: "2"
      url: "invalid-invalid"
      valueLocation: 'value'
      method: "query"
    authenticationRef:
      name: {{.TriggerAuthName}}
      kind: {{.TriggerAuthKind}}
`
)

func TestTriggerAuthenticationGeneral(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	// Create kubernetes resources
	kc := GetKubernetesClient(t)

	t.Log("--- testing triggerauthentication  ---")
	data, templates := getTemplateData(triggerAuthKind)
	CreateKubernetesResources(t, kc, namespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, minReplicas, 180, 3),
		"replica count should be %d after 3 minutes", minReplicas)

	testTriggerAuthenticationStatusValue(t, data, triggerAuthKind)

	// Clean resources and then testing clustertriggerauthentication
	DeleteKubernetesResources(t, namespace, data, templates)

	t.Log("--- testing clustertriggerauthentication  ---")
	data, templates = getTemplateData(clusterTriggerAuthKind)
	CreateKubernetesResources(t, kc, namespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, minReplicas, 180, 3),
		"replica count should be %d after 3 minutes", minReplicas)

	testTriggerAuthenticationStatusValue(t, data, clusterTriggerAuthKind)
	DeleteKubernetesResources(t, namespace, data, templates)
}

// tests basic scaling with one trigger based on metrics
func testTriggerAuthenticationStatusValue(t *testing.T, data templateData, kind string) {
	KubectlApplyWithTemplate(t, data, "triggerAuthenticationTemplate", triggerAuthenticationTemplate)
	t.Log("--- test one scaledObject ---")
	KubectlApplyWithTemplate(t, data, "scaledObjectTriggerTemplate", scaledObjectTriggerTemplate)
	otherparameter := `-o jsonpath="{.status.scaledobjects}"`
	CheckKubectlGetResult(t, kind, triggerAuthName, namespace, otherparameter, scaledObjectName)

	t.Log("--- test two scaledObject ---")
	KubectlApplyWithTemplate(t, data, "scaledObjectTrigger2Template", scaledObjectTrigger2Template)
	CheckKubectlGetResult(t, kind, triggerAuthName, namespace, otherparameter, scaledObjectName+","+scaledObject2Name)

	t.Log("--- test reomve scaledObject ---")
	KubectlDeleteWithTemplate(t, data, "scaledObjectTriggerTemplate", scaledObjectTriggerTemplate)
	CheckKubectlGetResult(t, kind, triggerAuthName, namespace, otherparameter, scaledObject2Name)
	KubectlDeleteWithTemplate(t, data, "scaledObjectTrigger2Template", scaledObjectTrigger2Template)
	CheckKubectlGetResult(t, kind, triggerAuthName, namespace, otherparameter, "")

	t.Log("--- test one scaledJob ---")
	otherparameter = `-o jsonpath="{.status.scaledjobs}"`
	KubectlApplyWithTemplate(t, data, "scaledJobTemplate", scaledJobTemplate)
	CheckKubectlGetResult(t, kind, triggerAuthName, namespace, otherparameter, scaledJobName)

	t.Log("--- test two scaledJob ---")
	KubectlApplyWithTemplate(t, data, "scaledJob2Template", scaledJob2Template)
	CheckKubectlGetResult(t, kind, triggerAuthName, namespace, otherparameter, scaledJobName+","+scaledJob2Name)

	t.Log("--- test reomve scaledObject ---")
	KubectlDeleteWithTemplate(t, data, "scaledJobTemplate", scaledJobTemplate)
	CheckKubectlGetResult(t, kind, triggerAuthName, namespace, otherparameter, scaledJob2Name)
	KubectlDeleteWithTemplate(t, data, "scaledJob2Template", scaledJob2Template)
	CheckKubectlGetResult(t, kind, triggerAuthName, namespace, otherparameter, "")
}

// help function to load template data
func getTemplateData(triggerAuthKind string) (templateData, []Template) {
	return templateData{
			TestNamespace:   namespace,
			TriggerAuthKind: triggerAuthKind,
			DeploymentName:  deploymentName,
			Deployment2Name: deployment2Name,
			TriggerAuthName: triggerAuthName,
			ScaledObject:    scaledObjectName,
			ScaledObject2:   scaledObject2Name,
			ScaledJob:       scaledJobName,
			ScaledJob2:      scaledJob2Name,
			SecretName:      secretName,
			MinReplicas:     fmt.Sprintf("%v", minReplicas),
			MaxReplicas:     fmt.Sprintf("%v", maxReplicas),
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "deployment2Template", Config: deployment2Template},
		}
}
