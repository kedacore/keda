//go:build e2e
// +build e2e

package gcp_storage_test

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
	testName = "gcp-storage-test"
)

var (
	gcpKey              = os.Getenv("TF_GCP_SA_CREDENTIALS")
	testNamespace       = fmt.Sprintf("%s-ns", testName)
	secretName          = fmt.Sprintf("%s-secret", testName)
	deploymentName      = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName    = fmt.Sprintf("%s-so", testName)
	bucketName          = fmt.Sprintf("%s-bucket", testName)
	maxReplicaCount     = 3
	activationThreshold = 5
	gsPrefix            = fmt.Sprintf("kubectl exec --namespace %s deploy/gcp-sdk -- ", testNamespace)
	blobPrefix          = "test/"
)

type templateData struct {
	TestNamespace       string
	SecretName          string
	GcpCreds            string
	DeploymentName      string
	ScaledObjectName    string
	BucketName          string
	MaxReplicaCount     int
	ActivationThreshold int
	BlobPrefix          string
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
          args: ["-c", "sleep 60"]
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
    - type: gcp-storage
      metadata:
        bucketName: {{.BucketName}}
        targetObjectCount: '5'
        blobPrefix: {{.BlobPrefix}}
        activationTargetObjectCount: '{{.ActivationThreshold}}'
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
	require.NotEmpty(t, gcpKey, "TF_GCP_SA_CREDENTIALS env variable is required for GCP storage test")

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, "gcp-sdk", testNamespace, 1, 60, 1),
		"gcp-sdk deployment should be ready after 1 minute")

	if createBucket(t) == nil {
		// test scaling
		testActivation(t, kc)
		testScaleOut(t, kc)
		testScaleIn(t, kc)
	}

	// cleanup
	t.Log("--- cleanup ---")
	cleanupBucket(t)
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func createBucket(t *testing.T) error {
	// Authenticate to GCP
	creds := make(map[string]interface{})
	err := json.Unmarshal([]byte(gcpKey), &creds)
	assert.NoErrorf(t, err, "Failed to load credentials from gcpKey - %s", err)

	cmd := fmt.Sprintf("%sgcloud auth activate-service-account %s --key-file /etc/secret-volume/creds.json --project=%s", gsPrefix, creds["client_email"], creds["project_id"])
	_, err = ExecuteCommand(cmd)
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

func getTemplateData() (templateData, []Template) {
	base64GcpCreds := base64.StdEncoding.EncodeToString([]byte(gcpKey))

	return templateData{
			TestNamespace:       testNamespace,
			SecretName:          secretName,
			GcpCreds:            base64GcpCreds,
			DeploymentName:      deploymentName,
			ScaledObjectName:    scaledObjectName,
			BucketName:          bucketName,
			MaxReplicaCount:     maxReplicaCount,
			ActivationThreshold: activationThreshold,
			BlobPrefix:          blobPrefix,
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
			{Name: "gcpSdkTemplate", Config: gcpSdkTemplate},
		}
}

func uploadFiles(t *testing.T, prefix string, count int) {
	t.Logf("--- uploading %d files ---", count)

	for i := 0; i < count; i++ {
		cmd := fmt.Sprintf("%sgsutil cp -n /usr/lib/google-cloud-sdk/bin/gsutil gs://%s/%sgsutil-%s%d", gsPrefix, bucketName, blobPrefix, prefix, i)
		_, err := ExecuteCommand(cmd)
		assert.NoErrorf(t, err, "cannot upload file to bucket - %s", err)
	}
}

func testActivation(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing not scaling if below threshold ---")

	uploadFiles(t, "active", activationThreshold)

	t.Log("--- waiting to see replicas are not scaled up ---")
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 240)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale out ---")

	uploadFiles(t, "scaling", 30-activationThreshold)

	t.Log("--- waiting for replicas to scale out ---")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 5),
		fmt.Sprintf("replica count should be %d after five minutes", maxReplicaCount))
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")

	// Delete files so we are still left with activationThreshold number of files which should be enough
	// to scale in to 0.
	cmd := fmt.Sprintf("%sgsutil -m rm -a  gs://%s/%sgsutil*", gsPrefix, bucketName, blobPrefix)
	_, err := ExecuteCommand(cmd)
	assert.NoErrorf(t, err, "cannot clear bucket - %s", err)

	t.Log("--- waiting for replicas to scale in to zero")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 30, 10),
		"replica count should be 0 after 5 minutes")
}
