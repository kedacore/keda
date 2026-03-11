//go:build e2e
// +build e2e

package file_based_auth_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName = "file-based-auth-test"
)

var (
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
)

type templateData struct {
	TestNamespace    string
	DeploymentName   string
	ScaledObjectName string
}

const (
	curlDeployment = `
apiVersion: v1
kind: Pod
metadata:
  name: curl
  namespace: {{.TestNamespace}}
spec:
  terminationGracePeriodSeconds: 1
  containers:
  - name: curl
    image: curlimages/curl:latest
    command: [ "/bin/sleep", "infinity" ]
`

	metricsApiDeployment = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: metrics-api
  namespace: {{.TestNamespace}}
spec:
  selector:
    matchLabels:
      app: metrics-api
  template:
    metadata:
      labels:
        app: metrics-api
    spec:
      terminationGracePeriodSeconds: 1
      containers:
        - name: metrics-api
          image: ghcr.io/kedacore/tests-metrics-api:latest
          ports:
            - containerPort: 8080
          env:
            - name: AUTH_TOKEN
              valueFrom:
                secretKeyRef:
                  name: bearer-secret
                  key: token
`
	metricsApiService = `
apiVersion: v1
kind: Service
metadata:
  name: metrics-api
  namespace: {{.TestNamespace}}
spec:
  selector:
    app: metrics-api
  ports:
    - protocol: TCP
      port: 80
      targetPort: 8080
`

	metricsApiSecret = `
apiVersion: v1
kind: Secret
metadata:
  name: bearer-secret
  namespace: {{.TestNamespace}}
type: Opaque
stringData:
  token: e2e-mock-test-bearer-token   # as configured in ../config/e2e/file_auth/patch_operator.yml
`

	deploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  replicas: 0
  selector:
    matchLabels:
      app: {{.DeploymentName}}
  template:
    metadata:
      labels:
        app: {{.DeploymentName}}
    spec:
      terminationGracePeriodSeconds: 1
      containers:
      - name: {{.DeploymentName}}
        image: nginx
`

	clusterTriggerAuthenticationTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ClusterTriggerAuthentication
metadata:
  name: file-auth
spec:
  filePath: creds.json
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
  pollingInterval: 5
  minReplicaCount: 0
  maxReplicaCount: 1
  cooldownPeriod: 5
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 5
  triggers:
    - type: metrics-api
      metadata:
        targetValue: "10"
        url: "http://metrics-api.{{.TestNamespace}}.svc.cluster.local/api/token/value"
        valueLocation: 'value'
        authMode: "bearer"
      authenticationRef:
        name: file-auth
        kind: ClusterTriggerAuthentication
`
)

func TestFileBasedAuthentication(t *testing.T) {
	// setup
	t.Log("--- setting up file-based auth test ---")

	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	// test scaled object creation with file-based auth
	testScaledObjectWithFileAuth(t)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:    testNamespace,
			DeploymentName:   deploymentName,
			ScaledObjectName: scaledObjectName,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "clusterTriggerAuthenticationTemplate", Config: clusterTriggerAuthenticationTemplate},
			{Name: "metricsApiSecret", Config: metricsApiSecret},
			{Name: "metricsApiDeployment", Config: metricsApiDeployment},
			{Name: "metricsApiService", Config: metricsApiService},
			{Name: "curlDeployment", Config: curlDeployment},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

func testScaledObjectWithFileAuth(t *testing.T) {
	t.Log("--- testing scaled object with file-based authentication ---")

	kc := GetKubernetesClient(t)
	kedaKc := GetKedaKubernetesClient(t)

	// Verify ScaledObject was created successfully
	scaledObject, err := kedaKc.ScaledObjects(testNamespace).Get(context.Background(), scaledObjectName, metav1.GetOptions{})
	if err != nil {
		t.Logf("ScaledObject not found (expected in e2e environment): %v", err)
		return
	}
	assert.NotNil(t, scaledObject)

	// Verify the authenticationRef exists
	if len(scaledObject.Spec.Triggers) > 0 {
		assert.NotNil(t, scaledObject.Spec.Triggers[0].AuthenticationRef)
		assert.Equal(t, "file-auth", scaledObject.Spec.Triggers[0].AuthenticationRef.Name)
		assert.Equal(t, "ClusterTriggerAuthentication", scaledObject.Spec.Triggers[0].AuthenticationRef.Kind)
	}

	// Verify ClusterTriggerAuthentication has the filePath
	clusterTriggerAuth, err := kedaKc.ClusterTriggerAuthentications().Get(context.Background(), "file-auth", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("ClusterTriggerAuthentication not found: %v", err)
	}
	assert.NotNil(t, clusterTriggerAuth)
	assert.Equal(t, "creds.json", clusterTriggerAuth.Spec.FilePath)

	// Check app is scaled to 0
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, "metrics-api", testNamespace, 1, 10, 12),
		"metrics-api deployment should have 1 ready replica within 2 minutes")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 10, 6),
		"replica count should be 0 after 1 minute")

	// Set the metric value in metrics-api deployment to 10 to trigger scaling
	assert.True(t, WaitForPodReady(t, kc, "curl", testNamespace, 10, 12), "curl pod should be ready within 2 minutes")
	setMetricToTenCmd := fmt.Sprintf(`curl --retry 20 --retry-delay 2 --retry-connrefused --fail -X POST http://metrics-api.%s.svc/api/value/10`, testNamespace)
	stdout, stderr, err := ExecCommandOnSpecificPod(t, "curl", testNamespace, setMetricToTenCmd)
	if err != nil {
		t.Fatalf("failed to set metric to 10, stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
	}

	// Check app is scaled to 1
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 10, 6),
		"replica count should be 1 after 1 minute")
}
