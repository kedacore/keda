//go:build e2e
// +build e2e

package keda_add_on_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "keda-add-on-test"
)

var (
	testNamespace              = fmt.Sprintf("%s-ns", testName)
	serviceName                = fmt.Sprintf("%s-service-%d", testName, GetRandomNumber())
	deploymentName             = fmt.Sprintf("%s-deployment", testName)
	scalerName                 = fmt.Sprintf("%s-scaler", testName)
	scaledObjectName           = fmt.Sprintf("%s-so", testName)
	clientName                 = fmt.Sprintf("%s-client", testName)
	customResourceName         = fmt.Sprintf("%s-cr", testName)
	metricsServerValueEndpoint = fmt.Sprintf("http://%s.%s:8080/api/value", serviceName, testNamespace)
	metricsServerTypeEndpoint  = fmt.Sprintf("http://%s.%s:8080/api/type", serviceName, testNamespace)
)

type templateData struct {
	TestNamespace      string
	ServiceName        string
	DeploymentName     string
	ScalerName         string
	ScaledObjectName   string
	CustomResourceName string
	ClientName         string
	HPAName            string
}

const (
	addOnCRDTemplate = `
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: addons.e2e.com
spec:
  group: e2e.com
  names:
    plural: addons
    singular: addon
    shortNames:
      - ao
    kind: AddOn
    listKind: AddOnList
  scope: Namespaced
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              x-kubernetes-preserve-unknown-fields: true
            status:
              type: object
              properties:
                addOnMetadata:
                  type: object
                  properties:
                    serverAddress:
                      type: string
                    usePushScaler:
                      type: boolean
                    metadata:
                      type: object
                      additionalProperties:
                        type: string
      subresources:
        status: {}
  conversion:
    strategy: None
`

	serviceTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: {{.ServiceName}}
  namespace: {{.TestNamespace}}
spec:
  ports:
    - port: 6000
      name: grpc
      targetPort: 6000
    - port: 8080
      name: http
      targetPort: 8080
  selector:
    app: {{.ScalerName}}
`

	scalerTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.ScalerName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.ScalerName}}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: {{.ScalerName}}
  template:
    metadata:
      labels:
        app: {{.ScalerName}}
    spec:
      containers:
        - name: scaler
          image: ghcr.io/kedacore/tests-external-scaler:latest
          imagePullPolicy: Always
          ports:
          - containerPort: 6000
          - containerPort: 8080
          readinessProbe:
            httpGet:
              path: /ready
              port: 8080
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
      containers:
        - name: nginx
          image: ghcr.io/nginx/nginx-unprivileged:1.26
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
  advanced:
    horizontalPodAutoscalerConfig:
      name: {{.HPAName}}
  pollingInterval: 5
  cooldownPeriod: 10
  minReplicaCount: 0
  maxReplicaCount: 2
  triggers:
    - type: keda-add-on
      metadata:
        apiVersion: e2e.com/v1
        kind: AddOn
        name: {{.CustomResourceName}}
`
	clientTemplate = `apiVersion: v1
kind: Pod
metadata:
  name: {{.ClientName}}
  namespace: {{.TestNamespace}}
spec:
  terminationGracePeriodSeconds: 0
  containers:
  - name: curl-client
    image: docker.io/curlimages/curl
    command: ["sleep", "1800s"]
  restartPolicy: Never`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	dc := GetDynamicKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})

	require.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 15, 1),
		"replica count should be 0 after 15 seconds")

	require.True(t, WaitForDeploymentReplicaReadyCount(t, kc, scalerName, testNamespace, 1, 15, 1),
		"replica count should be 1 after 15 seconds")

	require.True(t, WaitForPodReady(t, kc, clientName, testNamespace, 15, 1),
		"client pod should be ready after 15 seconds")

	err := createAddOnCR(t.Context(), dc)
	require.NoError(t, err, "failed to create AddOn CR")

	metricTypes := []string{"AverageValue", "Value"}
	for _, metricType := range metricTypes {
		t.Logf("--- testing with metric type: %s ---", metricType)
		setMetricType(t, metricType)

		data.HPAName = fmt.Sprintf("keda-hpa-%s", strings.ToLower(metricType))
		err := KubectlApplyWithErrors(t, data, "scaledObjectTemplate", scaledObjectTemplate)
		require.NoError(t, err, "failed to apply ScaledObject")
		err = testHPAMetricType(t, kc, metricType, data.HPAName)
		require.NoError(t, err, "failed to validate HPA metric type")

		testScaleOut(t, kc)
		testScaleIn(t, kc)

		KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	}
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:      testNamespace,
			ServiceName:        serviceName,
			DeploymentName:     deploymentName,
			ScalerName:         scalerName,
			ScaledObjectName:   scaledObjectName,
			CustomResourceName: customResourceName,
			ClientName:         clientName,
		}, []Template{
			{Name: "scalerTemplate", Config: scalerTemplate},
			{Name: "serviceTemplate", Config: serviceTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "clientTemplate", Config: clientTemplate},
			{Name: "addOnCRDTemplate", Config: addOnCRDTemplate},
		}
}

func createAddOnCR(ctx context.Context, dc dynamic.Interface) error {
	gvr := schema.GroupVersionResource{Group: "e2e.com", Version: "v1", Resource: "addons"}

	addOnCr := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "e2e.com/v1",
			"kind":       "AddOn",
			"metadata": map[string]interface{}{
				"name": customResourceName,
			},
		},
	}

	obj, err := dc.Resource(gvr).Namespace(testNamespace).Create(ctx, addOnCr, v1.CreateOptions{})
	if err != nil {
		return err
	}

	newStatus := map[string]interface{}{
		"addOnMetadata": map[string]interface{}{
			"serverAddress": fmt.Sprintf("%s.%s:6000", serviceName, testNamespace),
			"usePushScaler": true,
			"metadata": map[string]any{
				"metricThreshold": "10",
			},
		},
	}

	err = unstructured.SetNestedMap(obj.Object, newStatus, "status")
	if err != nil {
		return err
	}

	_, err = dc.Resource(gvr).Namespace(testNamespace).UpdateStatus(ctx, obj, v1.UpdateOptions{})
	return err
}

func setMetricType(t *testing.T, metricType string) {
	message, errMessage, err := ExecCommandOnSpecificPod(t, clientName, testNamespace, fmt.Sprintf(`curl -X POST "%s/%s"`, metricsServerTypeEndpoint, metricType))
	if err != nil {
		t.Logf("stdout: %s -- stderr %s", message, errMessage)
	}
	require.NoError(t, err, "failed to set metric type")
}

func setMetricValue(t *testing.T, metricValue int) {
	message, errMessage, err := ExecCommandOnSpecificPod(t, clientName, testNamespace, fmt.Sprintf(`curl -X POST "%s/%d"`, metricsServerValueEndpoint, metricValue))
	if err != nil {
		t.Logf("stdout: %s -- stderr %s", message, errMessage)
	}
	require.NoError(t, err, "failed to set metric value")
}

func testHPAMetricType(t *testing.T, kc *kubernetes.Clientset, metricType, hpaName string) error {
	t.Log("--- testing hpa metric type ---")
	hpa, err := WaitForHpaCreation(t, kc, hpaName, testNamespace, 15, 2)
	assert.NoError(t, err, "failed to get HPA")
	assert.NotNil(t, hpa, "HPA should not be nil")
	if hpa != nil {
		if len(hpa.Spec.Metrics) != 1 || hpa.Spec.Metrics[0].External == nil {
			return fmt.Errorf("unexpected HPA metrics configuration: %v", hpa.Spec.Metrics)
		}
		assert.Equal(t, metricType, string(hpa.Spec.Metrics[0].External.Target.Type), "metric type in HPA does not match expected")
		return nil
	}
	return fmt.Errorf("HPA not found")
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale out ---")

	setMetricValue(t, 100)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 2, 60, 2),
		"replica count should be 2 after 2 minutes")
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")

	t.Log("scaling to idle replicas")
	setMetricValue(t, 0)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 2),
		"replica count should be 0 after 2 minutes")
}
