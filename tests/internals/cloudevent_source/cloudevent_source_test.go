//go:build e2e
// +build e2e

package trigger_update_so_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName = "eventsource-test"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

var (
	namespace                  = fmt.Sprintf("%s-ns", testName)
	scaledObjectName           = fmt.Sprintf("%s-so", testName)
	clientName                 = fmt.Sprintf("%s-client", testName)
	cloudeventSourceName       = fmt.Sprintf("%s-ce", testName)
	cloudEventHTTPReceiverName = fmt.Sprintf("%s-cloudevent-http-receiver", testName)
	cloudEventHTTPServiceName  = fmt.Sprintf("%s-cloudevent-http-service", testName)
	cloudEventHTTPServiceURL   = fmt.Sprintf("http://%s.%s.svc.cluster.local:8899", cloudEventHTTPServiceName, namespace)
	clusterName                = "test-cluster"
	expectedSubject            = fmt.Sprintf("/%s/%s/scaledobject/%s", clusterName, namespace, scaledObjectName)
	expectedSource             = fmt.Sprintf("/%s/keda/keda", clusterName)
)

type templateData struct {
	TestNamespace              string
	ScaledObject               string
	ClientName                 string
	CloudEventSourceName       string
	CloudEventHTTPReceiverName string
	CloudEventHTTPServiceName  string
	CloudEventHTTPServiceURL   string
	ClusterName                string
}

const (
	cloudEventSourceTemplate = `
  apiVersion: eventing.keda.sh/v1alpha1
  kind: CloudEventSource
  metadata:
    name: {{.CloudEventSourceName}}
    namespace: {{.TestNamespace}}
  spec:
    clusterName: {{.ClusterName}}
    destination:
      http:
        uri: {{.CloudEventHTTPServiceURL}}
  `

	cloudEventHTTPServiceTemplate = `
  apiVersion: v1
  kind: Service
  metadata:
    name: {{.CloudEventHTTPServiceName}}
    namespace: {{.TestNamespace}}
  spec:
    type: ClusterIP
    ports:
    - protocol: TCP
      port: 8899
      targetPort: 8899
    selector:
      app: {{.CloudEventHTTPReceiverName}}
  `

	cloudEventHTTPReceiverTemplate = `
  apiVersion: apps/v1
  kind: Deployment
  metadata:
    labels:
      deploy: {{.CloudEventHTTPReceiverName}}
    name: {{.CloudEventHTTPReceiverName}}
    namespace: {{.TestNamespace}}
  spec:
    selector:
      matchLabels:
        app: {{.CloudEventHTTPReceiverName}}
    replicas: 1
    template:
      metadata:
        labels:
          app: {{.CloudEventHTTPReceiverName}}
      spec:
        containers:
        - name: httpreceiver
          image: ghcr.io/kedacore/tests-cloudevents-http:latest
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
        podSelector: 'pod=testWorkloadDeploymentName'
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

	time.Sleep(15 * time.Second)
	assert.True(t, WaitForAllPodRunningInNamespace(t, kc, namespace, 5, 20), "all pods should be running")

	testErrEventSourceEmitValue(t, kc, data)

	DeleteKubernetesResources(t, namespace, data, templates)
}

// tests error events emitted
func testErrEventSourceEmitValue(t *testing.T, _ *kubernetes.Clientset, data templateData) {
	t.Log("--- test emitting eventsource about scaledobject err---")
	KubectlApplyWithTemplate(t, data, "scaledObjectErrTemplate", scaledObjectErrTemplate)

	// wait 15 seconds to ensure event propagation
	time.Sleep(15 * time.Second)

	out, outErr, err := ExecCommandOnSpecificPod(t, clientName, namespace, fmt.Sprintf("curl -X GET %s/getCloudEvent/%s", cloudEventHTTPServiceURL, "ScaledObjectCheckFailed"))
	assert.NotEmpty(t, out)
	assert.Empty(t, outErr)
	assert.NoError(t, err, "dont expect error requesting ")

	cloudEvents := []cloudevents.Event{}
	err = json.Unmarshal([]byte(out), &cloudEvents)

	assert.NoError(t, err, "dont expect error unmarshaling the cloudEvents")
	assert.Greater(t, len(cloudEvents), 0, "cloudEvents should have at least 1 item")

	foundEvents := []cloudevents.Event{}

	for _, cloudEvent := range cloudEvents {
		if cloudEvent.Subject() == expectedSubject {
			foundEvents = append(foundEvents, cloudEvent)
			data := map[string]string{}
			err := cloudEvent.DataAs(&data)
			assert.NoError(t, err)
			assert.Equal(t, data["message"], "ScaledObject doesn't have correct scaleTargetRef specification")
			assert.Equal(t, cloudEvent.Type(), "com.cloudeventsource.keda")
			assert.Equal(t, cloudEvent.Source(), expectedSource)
			assert.Equal(t, cloudEvent.DataContentType(), "application/json")
		}
	}
	assert.NotEmpty(t, foundEvents)
}

// help function to load template data
func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:              namespace,
			ScaledObject:               scaledObjectName,
			ClientName:                 clientName,
			CloudEventSourceName:       cloudeventSourceName,
			CloudEventHTTPReceiverName: cloudEventHTTPReceiverName,
			CloudEventHTTPServiceName:  cloudEventHTTPServiceName,
			CloudEventHTTPServiceURL:   cloudEventHTTPServiceURL,
			ClusterName:                clusterName,
		}, []Template{
			{Name: "cloudEventHTTPReceiverTemplate", Config: cloudEventHTTPReceiverTemplate},
			{Name: "cloudEventHTTPServiceTemplate", Config: cloudEventHTTPServiceTemplate},
			{Name: "clientTemplate", Config: clientTemplate},
			{Name: "cloudEventSourceTemplate", Config: cloudEventSourceTemplate},
		}
}
