//go:build e2e
// +build e2e

package trigger_update_so_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName = "cloudevent-test"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

var (
	namespace                  = fmt.Sprintf("%s-ns", testName)
	deploymentName             = fmt.Sprintf("%s-deployment", testName)
	workloadDeploymentName     = fmt.Sprintf("%s-workload-deployment", testName)
	scaledObjectName           = fmt.Sprintf("%s-so", testName)
	clientName                 = fmt.Sprintf("%s-client", testName)
	cloudEventName             = fmt.Sprintf("%s-ce", testName)
	cloudEventHttpReceiverName = fmt.Sprintf("%s-cloudevent-http-receiver", testName)
	cloudEventHttpServiceName  = fmt.Sprintf("%s-cloudevent-http-service", testName)
	cloudEventHttpServiceURL   = fmt.Sprintf("http://%s.%s.svc.cluster.local:8899", cloudEventHttpServiceName, namespace)
)

type templateData struct {
	TestNamespace              string
	DeploymentName             string
	ScaledObject               string
	WorkloadDeploymentName     string
	ClientName                 string
	CloudEventName             string
	CloudEventHttpReceiverName string
	CloudEventHttpServiceName  string
	CloudEventHttpServiceURL   string
}

const (
	cloudEventTemplate = `
  apiVersion: keda.sh/v1alpha1
  kind: CloudEvent
  metadata:
    name: {{.CloudEventName}}
    namespace: {{.TestNamespace}}
  spec:
    clusterName: cluster-sample
    cloudEventHttp:
      endPoint: {{.CloudEventHttpServiceURL}}
  `

	cloudEventHttpServiceTemplate = `
  apiVersion: v1
  kind: Service
  metadata:
    name: {{.CloudEventHttpServiceName}}
    namespace: {{.TestNamespace}}
  spec:
    type: ClusterIP
    ports:
    - protocol: TCP
      port: 8899
      targetPort: 8899
    selector:
      app: {{.CloudEventHttpReceiverName}}
  `

	cloudEventHttpReceiverTemplate = `
  apiVersion: apps/v1
  kind: Deployment
  metadata:
    labels:
      deploy: {{.CloudEventHttpReceiverName}}
    name: {{.CloudEventHttpReceiverName}}
    namespace: {{.TestNamespace}}
  spec:
    selector:
      matchLabels:
        app: {{.CloudEventHttpReceiverName}}
    replicas: 1
    template:
      metadata:
        labels:
          app: {{.CloudEventHttpReceiverName}}
      spec:
        containers:
        - name: httpreceiver
          image: docker.io/spiritzhou/cloudeventhttp:v2
          ports:
          - containerPort: 8899
          resources:
            requests:
              cpu: "200m"
            limits:
              cpu: "500m"
  `

	scaledObjectErrTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObject}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: test
  triggers:
    - type: kubernetes-workload
      metadata:
        podSelector: 'pod={{.WorkloadDeploymentName}}'
        value: '1'
        activationValue: '3'
`

	clientTemplate = `
apiVersion: v1
kind: Pod
metadata:
  name: {{.ClientName}}
  namespace: {{.TestNamespace}}
spec:
  containers:
  - name: {{.ClientName}}
    image: curlimages/curl
    command:
      - sh
      - -c
      - "exec tail -f /dev/null"`
)

func TestScaledObjectGeneral(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	CreateKubernetesResources(t, kc, namespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, 1, 180, 3),
		"replica count should be %d after 3 minutes", 1)

	testErrCloudEventEmitValue(t, kc, data) // one trigger target changes

	// DeleteKubernetesResources(t, namespace, data, templates)
}

// tests basic scaling with one trigger based on metrics
func testErrCloudEventEmitValue(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- test emitting cloudevent about scaledobject err---")
	KubectlApplyWithTemplate(t, data, "scaledObjectErrTemplate", scaledObjectErrTemplate)

	// recreate database to clear it
	out, _, _ := ExecCommandOnSpecificPod(t, clientName, namespace, fmt.Sprintf("curl -X GET %s/getCloudEvent/%s", cloudEventHttpServiceURL, "ScaledObjectCheckFailed"))

	assert.NotNil(t, out)

	cloudEvent := make(map[string]interface{})
	err := json.Unmarshal([]byte(out), &cloudEvent)
	assert.Nil(t, err)
	assert.Equal(t, cloudEvent["data"].(map[string]interface{})["message"], "ScaledObject doesn't have correct scaleTargetRef specification")
}

// help function to load template data
func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:              namespace,
			ScaledObject:               scaledObjectName,
			DeploymentName:             deploymentName,
			ClientName:                 clientName,
			CloudEventName:             cloudEventName,
			CloudEventHttpReceiverName: cloudEventHttpReceiverName,
			CloudEventHttpServiceName:  cloudEventHttpServiceName,
			CloudEventHttpServiceURL:   cloudEventHttpServiceURL,
			WorkloadDeploymentName:     workloadDeploymentName,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "workloadDeploymentTemplate", Config: workloadDeploymentTemplate},
			{Name: "cloudEventTemplate", Config: cloudEventTemplate},
			{Name: "cloudEventHttpReceiverTemplate", Config: cloudEventHttpReceiverTemplate},
			{Name: "cloudEventHttpServiceTemplate", Config: cloudEventHttpServiceTemplate},
			{Name: "clientTemplate", Config: clientTemplate},
		}
}
