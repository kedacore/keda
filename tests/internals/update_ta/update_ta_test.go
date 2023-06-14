//go:build e2e
// +build e2e

package update_ta_so_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

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
	midReplicas            = 3
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
        image: nginxinc/nginx-unprivileged
        ports:
        - containerPort: 80
        resources:
          requests:
            cpu: "200m"
          limits:
            cpu: "500m"
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
        image: nginxinc/nginx-unprivileged
        ports:
        - containerPort: 80
        resources:
          requests:
            cpu: "200m"
          limits:
            cpu: "500m"
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
  - type: cpu
    metricType: Utilization
    metadata:
      value: "50"
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
  - type: cpu
    metricType: Utilization
    metadata:
      value: "50"
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
  - type: cpu
    metadata:
      type: Utilization
      value: "50"
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
  - type: cpu
    metadata:
      type: Utilization
      value: "50"
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

	testTriggerAuthenticationStatusValue(t, kc, data, triggerAuthKind)

	// Clean resources and then testing clustertriggerauthentication
	DeleteKubernetesResources(t, namespace, data, templates)

	t.Log("--- testing clustertriggerauthentication  ---")
	data, templates = getTemplateData(clusterTriggerAuthKind)
	CreateKubernetesResources(t, kc, namespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, minReplicas, 180, 3),
		"replica count should be %d after 3 minutes", minReplicas)

	testTriggerAuthenticationStatusValue(t, kc, data, clusterTriggerAuthKind)
	DeleteKubernetesResources(t, namespace, data, templates)
}

func checkScaledObjectStatusFromKubectl(t *testing.T, kind string, expected string) {
	time.Sleep(1 * time.Second)
	kctlGetCmd := fmt.Sprintf(`kubectl get %s/%s -n %s -o jsonpath="{.status.scaledobjects}"`, kind, triggerAuthName, namespace)
	output, err := ExecuteCommand(kctlGetCmd)
	assert.NoErrorf(t, err, "cannot get rollout info - %s", err)

	unqoutedOutput := strings.ReplaceAll(string(output), "\"", "")
	assert.Equal(t, expected, unqoutedOutput)
}

func checkScaleJobStatusFromKubectl(t *testing.T, kind string, expected string) {
	time.Sleep(1 * time.Second)
	kctlGetCmd := fmt.Sprintf(`kubectl get %s/%s -n %s -o jsonpath="{.status.scaledjobs}"`, kind, triggerAuthName, namespace)
	output, err := ExecuteCommand(kctlGetCmd)
	assert.NoErrorf(t, err, "cannot get rollout info - %s", err)

	unqoutedOutput := strings.ReplaceAll(string(output), "\"", "")
	assert.Equal(t, expected, unqoutedOutput)
}

// tests basic scaling with one trigger based on metrics
func testTriggerAuthenticationStatusValue(t *testing.T, kc *kubernetes.Clientset, data templateData, kind string) {
	KubectlApplyWithTemplate(t, data, "triggerAuthenticationTemplate", triggerAuthenticationTemplate)
	t.Log("--- test one scaledObject ---")
	KubectlApplyWithTemplate(t, data, "scaledObjectTriggerTemplate", scaledObjectTriggerTemplate)
	checkScaledObjectStatusFromKubectl(t, kind, scaledObjectName)

	t.Log("--- test two scaledObject ---")
	KubectlApplyWithTemplate(t, data, "scaledObjectTrigger2Template", scaledObjectTrigger2Template)
	checkScaledObjectStatusFromKubectl(t, kind, scaledObjectName+","+scaledObject2Name)

	t.Log("--- test reomve scaledObject ---")
	KubectlDeleteWithTemplate(t, data, "scaledObjectTriggerTemplate", scaledObjectTriggerTemplate)
	checkScaledObjectStatusFromKubectl(t, kind, scaledObject2Name)
	KubectlDeleteWithTemplate(t, data, "scaledObjectTrigger2Template", scaledObjectTrigger2Template)
	checkScaledObjectStatusFromKubectl(t, kind, "")

	t.Log("--- test one scaledJob ---")
	KubectlApplyWithTemplate(t, data, "scaledJobTemplate", scaledJobTemplate)
	checkScaleJobStatusFromKubectl(t, kind, scaledJobName)

	t.Log("--- test two scaledJob ---")
	KubectlApplyWithTemplate(t, data, "scaledJob2Template", scaledJob2Template)
	checkScaleJobStatusFromKubectl(t, kind, scaledJobName+","+scaledJob2Name)

	t.Log("--- test reomve scaledObject ---")
	KubectlDeleteWithTemplate(t, data, "scaledJobTemplate", scaledJobTemplate)
	checkScaleJobStatusFromKubectl(t, kind, scaledJob2Name)
	KubectlDeleteWithTemplate(t, data, "scaledJob2Template", scaledJob2Template)
	checkScaleJobStatusFromKubectl(t, kind, "")
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
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "deployment2Template", Config: deployment2Template},
		}
}
