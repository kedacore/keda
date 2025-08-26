//go:build e2e
// +build e2e

package gcp_cloud_tasks_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

var now = time.Now().UnixNano()

const (
	testName = "gcp-cloud-tasks-test"
)

var (
	gcpKey              = os.Getenv("TF_GCP_SA_CREDENTIALS")
	creds               = make(map[string]interface{})
	errGcpKey           = json.Unmarshal([]byte(gcpKey), &creds)
	testNamespace       = fmt.Sprintf("%s-ns", testName)
	secretName          = fmt.Sprintf("%s-secret", testName)
	deploymentName      = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName    = fmt.Sprintf("%s-so", testName)
	projectID           = fmt.Sprintf("%s", creds["project_id"])
	queueID             = fmt.Sprintf("keda-test-queue-%d", now)
	maxReplicaCount     = 4
	activationThreshold = 5
	gsPrefix            = fmt.Sprintf("kubectl exec --namespace %s deploy/gcp-sdk -- ", testNamespace)
)

type templateData struct {
	TestNamespace       string
	SecretName          string
	GcpCreds            string
	DeploymentName      string
	ScaledObjectName    string
	QueueID             string
	ProjectID           string
	MaxReplicaCount     int
	ActivationThreshold int
}

const (
	secretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  creds.json: {{.GcpCreds}}
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
  maxReplicaCount: {{.MaxReplicaCount}}
  cooldownPeriod: 10
  triggers:
    - type: gcp-cloudtasks
      metadata:
        projectID: {{.ProjectID}}
        queueName: {{.QueueID}}
        value: "5"
        activationValue: "{{.ActivationThreshold}}"
        credentialsFromEnv: GOOGLE_APPLICATION_CREDENTIALS_JSON
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
        - name: {{.DeploymentName}}-processor
          image: google/cloud-sdk:slim
          # Consume a message
          command: [ "/bin/bash", "-c", "--" ]
          args: [ "gcloud auth activate-service-account --key-file /etc/secret-volume/creds.json && \
          while true; do gcloud tasks list --queue={{.QueueID}} --location=europe-west1; sleep 20; done" ]
          env:
            - name: GOOGLE_APPLICATION_CREDENTIALS_JSON
              valueFrom:
                secretKeyRef:
                  name: {{.SecretName}}
                  key: creds.json
          volumeMounts:
            - name: secret-volume
              mountPath: /etc/secret-volume
      volumes:
        - name: secret-volume
          secret:
            secretName: {{.SecretName}}
`

	gcpSdkTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gcp-sdk
  namespace: {{.TestNamespace}}
  labels:
    app: gcp-sdk
spec:
  replicas: 1
  selector:
    matchLabels:
      app: gcp-sdk
  template:
    metadata:
      labels:
        app: gcp-sdk
    spec:
      containers:
        - name: gcp-sdk-container
          image: google/cloud-sdk:slim
          # Just spin & wait forever
          command: [ "/bin/bash", "-c", "--" ]
          args: [ "ls /tmp && while true; do sleep 30; done;" ]
          volumeMounts:
            - name: secret-volume
              mountPath: /etc/secret-volume
      volumes:
        - name: secret-volume
          secret:
            secretName: {{.SecretName}}
`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	require.NotEmpty(t, gcpKey, "TF_GCP_SA_CREDENTIALS env variable is required for GCP storage test")
	assert.NoErrorf(t, errGcpKey, "Failed to load credentials from gcpKey - %s", errGcpKey)

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")

	sdkReady := WaitForDeploymentReplicaReadyCount(t, kc, "gcp-sdk", testNamespace, 1, 60, 1)
	assert.True(t, sdkReady, "gcp-sdk deployment should be ready after a minute")

	if sdkReady {
		if createGcpCloudTasks(t) == nil {
			// test scaling
			testActivation(t, kc)
			testScaleOut(t, kc)
			testScaleIn(t, kc)

			// cleanup
			t.Log("--- cleanup ---")
			cleanupGcpCloudTasks(t)
		}
	}

	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func createGcpCloudTasks(t *testing.T) error {
	// Authenticate to GCP
	t.Log("--- authenticate to GCP ---")
	cmd := fmt.Sprintf("%sgcloud auth activate-service-account %s --key-file /etc/secret-volume/creds.json --project=%s", gsPrefix, creds["client_email"], projectID)
	_, err := ExecuteCommand(cmd)
	assert.NoErrorf(t, err, "Failed to set GCP authentication on gcp-sdk - %s", err)
	if err != nil {
		return err
	}

	// Create queue
	t.Log("--- create queue ---")
	cmd = fmt.Sprintf("%sgcloud tasks queues create %s --location europe-west1 --max-concurrent-dispatches 1 --max-dispatches-per-second 1", gsPrefix, queueID)
	_, err = ExecuteCommand(cmd)
	assert.NoErrorf(t, err, "Failed to create Cloud Tasks queue %s: %s", queueID, err)
	if err != nil {
		return err
	}

	return err
}

func cleanupGcpCloudTasks(t *testing.T) {
	// Delete the queue
	t.Log("--- cleaning up the queue ---")
	_, _ = ExecuteCommand(fmt.Sprintf("%sgcloud tasks queues delete %s --location europe-west1 --quiet", gsPrefix, queueID))
}

func getTemplateData() (templateData, []Template) {
	base64GcpCreds := base64.StdEncoding.EncodeToString([]byte(gcpKey))

	return templateData{
			TestNamespace:       testNamespace,
			SecretName:          secretName,
			GcpCreds:            base64GcpCreds,
			DeploymentName:      deploymentName,
			ScaledObjectName:    scaledObjectName,
			QueueID:             queueID,
			ProjectID:           projectID,
			MaxReplicaCount:     maxReplicaCount,
			ActivationThreshold: activationThreshold,
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
			{Name: "gcpSdkTemplate", Config: gcpSdkTemplate},
		}
}

func createGcpTasks(t *testing.T, count int) {
	t.Logf("--- creating %d tasks ---", count)
	publish := fmt.Sprintf(
		"%s/bin/bash -c -- 'for i in {1..%d}; do gcloud tasks create-http-task --location europe-west1 --queue %s --url http://foo.bar;done'",
		gsPrefix,
		count,
		queueID)
	_, err := ExecuteCommand(publish)
	assert.NoErrorf(t, err, "cannot create tasks to queue - %s", err)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing not scaling if below threshold ---")

	createGcpTasks(t, activationThreshold)

	t.Log("--- waiting to see replicas are not scaled up ---")
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 240)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale out ---")

	createGcpTasks(t, 20-activationThreshold)

	t.Log("--- waiting for replicas to scale out ---")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 30, 10),
		fmt.Sprintf("replica count should be %d after 5 minutes", maxReplicaCount))
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")
	cmd := fmt.Sprintf("%sgcloud tasks queues purge %s --location europe-west1 --quiet", gsPrefix, queueID)
	_, err := ExecuteCommand(cmd)
	assert.NoErrorf(t, err, "cannot purge queue - %s", err)

	t.Log("--- waiting for replicas to scale in to zero ---")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 30, 10),
		"replica count should be 0 after 5 minute")
}
