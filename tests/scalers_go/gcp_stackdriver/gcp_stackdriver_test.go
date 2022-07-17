//go:build e2e
// +build e2e

package gcp_stackdriver_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "gcp-stackdriver-test"
)

var (
	gcpKey           = os.Getenv("GCP_SP_KEY")
	creds            = make(map[string]interface{})
	errGcpKey        = json.Unmarshal([]byte(gcpKey), &creds)
	projectId        = fmt.Sprintf("%s", creds["project_id"])
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	secretName       = fmt.Sprintf("%s-secret", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
	bucketName       = fmt.Sprintf("%s-bucket", testName)
	maxReplicaCount  = 3
	gsPrefix         = fmt.Sprintf("kubectl exec --namespace %s deploy/gcp-sdk -- ", testNamespace)
)

type templateData struct {
	TestNamespace    string
	SecretName       string
	GcpCreds         string
	DeploymentName   string
	ScaledObjectName string
	BucketName       string
	ProjectId        string
	MaxReplicaCount  int
}

type templateValues map[string]string

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
        - name: noop-processor
          image: ubuntu:20.04
          command: ["/bin/bash"]
          args: ["-c", "sleep 30"]
          env:
            - name: GOOGLE_APPLICATION_CREDENTIALS_JSON
              valueFrom:
                secretKeyRef:
                  name: {{.SecretName}}
                  key:  creds.json
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
    - type: gcp-stackdriver
      metadata:
        projectId: {{.ProjectId}}
        filter: 'metric.type="storage.googleapis.com/network/received_bytes_count" AND resource.type="gcs_bucket" AND metric.label.method="WriteObject" AND resource.label.bucket_name="{{.BucketName}}"'
        metricName: {{.BucketName}}
        targetValue: "5"
        alignmentPeriodSeconds: "60"
        alignmentAligner: max
        credentialsFromEnv: GOOGLE_APPLICATION_CREDENTIALS_JSON
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
	require.NotEmpty(t, gcpKey, "GCP_KEY env variable is required for GCP storage test")
	assert.NoErrorf(t, errGcpKey, "Failed to load credentials from gcpKey - %s", errGcpKey)

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after a minute")

	sdkReady := WaitForDeploymentReplicaCount(t, kc, "gcp-sdk", testNamespace, 1, 60, 1)
	assert.True(t, sdkReady, "gcp-sdk deployment should be ready after a minute")

	if sdkReady {
		if createBucket(t) == nil {
			// test scaling
			testScaleUp(t, kc)
			testScaleDown(t, kc)
		}

		// cleanup
		t.Log("--- cleanup ---")
		cleanupBucket(t)
	}

	DeleteKubernetesResources(t, kc, testNamespace, data, templates)
}

func createBucket(t *testing.T) error {
	// Authenticate to GCP

	cmd := fmt.Sprintf("%sgcloud auth activate-service-account %s --key-file /etc/secret-volume/creds.json --project=%s", gsPrefix, creds["client_email"], creds["project_id"])
	_, err := ExecuteCommand(cmd)
	assert.NoErrorf(t, err, "Failed to set GCP authentication on gcp-sdk - %s", err)

	cleanupBucket(t)

	// Create bucket
	cmd = fmt.Sprintf("%sgsutil mb gs://%s", gsPrefix, bucketName)
	_, err = ExecuteCommand(cmd)
	assert.NoErrorf(t, err, "Failed to create GCS bucket - %s", err)
	return err
}

func cleanupBucket(t *testing.T) {
	// Cleanup the bucket
	t.Log("--- cleaning up the bucket ---")
	_, _ = ExecuteCommand(fmt.Sprintf("%sgsutil -m rm -r gs://%s", gsPrefix, bucketName))
}

func getTemplateData() (templateData, templateValues) {
	base64GcpCreds := base64.StdEncoding.EncodeToString([]byte(gcpKey))

	return templateData{
			TestNamespace:    testNamespace,
			SecretName:       secretName,
			GcpCreds:         base64GcpCreds,
			DeploymentName:   deploymentName,
			ScaledObjectName: scaledObjectName,
			ProjectId:        projectId,
			BucketName:       bucketName,
			MaxReplicaCount:  maxReplicaCount,
		}, templateValues{
			"secretTemplate":       secretTemplate,
			"deploymentTemplate":   deploymentTemplate,
			"scaledObjectTemplate": scaledObjectTemplate,
			"gcpSdkTemplate":       gcpSdkTemplate}
}

func testScaleUp(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale up ---")

	cmd := fmt.Sprintf("%sgsutil cp /usr/lib/google-cloud-sdk/bin/gsutil gs://%s", gsPrefix, bucketName)
	haveAllReplicas := false
	for i := 0; i < 60 && !haveAllReplicas; i++ {
		t.Log("--- upload a file to generate traffic ---")
		_, err := ExecuteCommand(cmd)
		assert.NoErrorf(t, err, "cannot upload file to bucket - %s", err)
		haveAllReplicas = WaitForDeploymentReplicaCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 1, 5)
	}

	assert.True(t, haveAllReplicas, fmt.Sprintf("replica count should be %d after five minutes", maxReplicaCount))
}

func testScaleDown(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale down ---")

	t.Log("--- waiting for replicas to scale down to zero ---")
	assert.True(t, WaitForDeploymentReplicaCount(t, kc, deploymentName, testNamespace, 0, 30, 10),
		"replica count should be 0 after five minute")
}
