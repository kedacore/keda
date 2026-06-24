//go:build e2e
// +build e2e

package gcp_spanner_test

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

const (
	testName = "gcp-spanner-test"

	// activationThreshold is the activationValue in the ScaledObject — the
	// scaler stays inactive (0 replicas) while pending row count ≤ this value.
	activationThreshold = 2
	// targetValue is the value per replica — one replica handles this many rows.
	targetValue     = 5
	maxReplicaCount = 4
	// pendingRowsForScaleOut must exceed activationThreshold and drive enough
	// load to reach maxReplicaCount: ceil(pendingRowsForScaleOut / targetValue).
	pendingRowsForScaleOut = 20
)

var (
	gcpKey         = os.Getenv("TF_GCP_SA_CREDENTIALS")
	creds          = make(map[string]interface{})
	errGcpKey      = json.Unmarshal([]byte(gcpKey), &creds)
	testNamespace  = fmt.Sprintf("%s-ns", testName)
	secretName     = fmt.Sprintf("%s-secret", testName)
	deploymentName = fmt.Sprintf("%s-deployment", testName)
	scaledObjName  = fmt.Sprintf("%s-so", testName)
	// Spanner resource identifiers — derived from credentials / env.
	projectID      = fmt.Sprintf("%v", creds["project_id"])
	instanceID     = fmt.Sprintf("keda-e2e-spanner-%d", time.Now().UnixNano())
	databaseID     = "keda-e2e-db"
	instanceConfig = envOrDefault("TF_GCP_SPANNER_INSTANCE_CONFIG", "regional-us-central1")
	// SQL used by the ScaledObject trigger — must return a single INT64 value.
	scalerQuery = "SELECT COUNT(*) FROM keda_test_jobs WHERE status = 'pending'"
	gsPrefix    = fmt.Sprintf("kubectl exec --namespace %s deploy/gcp-sdk -- ", testNamespace)
)

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// ── Kubernetes manifest templates ────────────────────────────────────────────

type templateData struct {
	TestNamespace   string
	SecretName      string
	GcpCreds        string
	DeploymentName  string
	ScaledObjName   string
	ProjectID       string
	InstanceID      string
	DatabaseID      string
	Query           string
	TargetValue     int
	ActivationValue int
	MaxReplicaCount int
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

	// The workload being scaled — a no-op container that simulates a job
	// processor.  In a real scenario this container would read rows from
	// Spanner and mark them done; here we just let KEDA scale it and verify
	// the replica count matches the expected value.
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
        - name: job-processor
          image: ubuntu:22.04
          command: ["/bin/bash", "-c", "while true; do sleep 30; done"]
          env:
            - name: GOOGLE_APPLICATION_CREDENTIALS_JSON
              valueFrom:
                secretKeyRef:
                  name: {{.SecretName}}
                  key: creds.json
`

	scaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  pollingInterval: 5
  cooldownPeriod: 10
  minReplicaCount: 0
  maxReplicaCount: {{.MaxReplicaCount}}
  triggers:
    - type: gcp-spanner
      metadata:
        projectId: {{.ProjectID}}
        instanceId: {{.InstanceID}}
        databaseId: {{.DatabaseID}}
        query: "{{.Query}}"
        targetValue: "{{.TargetValue}}"
        activationValue: "{{.ActivationValue}}"
        credentialsFromEnv: GOOGLE_APPLICATION_CREDENTIALS_JSON
`

	// A pod running the gcloud SDK — used to run spanner CLI commands against
	// the real Spanner instance to seed and clean up test data.
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
          command: ["/bin/bash", "-c", "--"]
          args: ["while true; do sleep 30; done"]
          volumeMounts:
            - name: secret-volume
              mountPath: /etc/secret-volume
      volumes:
        - name: secret-volume
          secret:
            secretName: {{.SecretName}}
`
)

// ── Test entry point ──────────────────────────────────────────────────────────

func TestScaler(t *testing.T) {
	t.Log("--- validating prerequisites ---")
	require.NotEmpty(t, gcpKey, "TF_GCP_SA_CREDENTIALS env variable is required for GCP Spanner test")
	assert.NoErrorf(t, errGcpKey, "failed to parse TF_GCP_SA_CREDENTIALS as JSON: %s", errGcpKey)

	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	CreateKubernetesResources(t, kc, testNamespace, data, templates)
	defer DeleteKubernetesResources(t, testNamespace, data, templates)

	assert.True(t,
		WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"initial replica count should be 0")

	sdkReady := WaitForDeploymentReplicaReadyCount(t, kc, "gcp-sdk", testNamespace, 1, 60, 1)
	require.True(t, sdkReady, "gcp-sdk pod must be ready before the test can proceed")

	if err := setupSpanner(t); err != nil {
		return
	}

	testActivation(t, kc)
	testScaleOut(t, kc)
	testScaleIn(t, kc)
}

// ── Spanner setup / teardown ──────────────────────────────────────────────────

func spannerCmd(format string, args ...interface{}) string {
	return gsPrefix + fmt.Sprintf(format, args...)
}

func setupSpanner(t *testing.T) error {
	t.Helper()
	t.Log("--- authenticating to GCP ---")
	cmd := spannerCmd(
		"gcloud auth activate-service-account %s --key-file /etc/secret-volume/creds.json --project=%s",
		creds["client_email"], projectID,
	)
	_, err := ExecuteCommand(cmd)
	if !assert.NoErrorf(t, err, "failed to authenticate to GCP: %s", err) {
		return err
	}

	t.Logf("--- creating Spanner instance %s ---", instanceID)
	cmd = spannerCmd(
		"gcloud spanner instances create %s --config=%s --description=keda-e2e-test --processing-units=100 --project=%s",
		instanceID, instanceConfig, projectID,
	)
	_, err = ExecuteCommand(cmd)
	if !assert.NoErrorf(t, err, "failed to create Spanner instance: %s", err) {
		return err
	}
	// Register cleanup immediately so the instance is deleted even if a later
	// setup step fails and setupSpanner returns an error.
	t.Cleanup(func() { cleanupSpanner(t) })

	t.Logf("--- creating Spanner database %s ---", databaseID)
	cmd = spannerCmd(
		"gcloud spanner databases create %s --instance=%s --project=%s",
		databaseID, instanceID, projectID,
	)
	_, err = ExecuteCommand(cmd)
	if !assert.NoErrorf(t, err, "failed to create Spanner database: %s", err) {
		return err
	}

	t.Log("--- creating test table ---")
	// ParseCommand (tests/helper) splits on spaces honouring single-quotes only.
	// The flag and its value must be separated by a space so that the
	// single-quoted value becomes a standalone token and its quotes are stripped.
	cmd = spannerCmd(
		"gcloud spanner databases ddl update %s --instance=%s --project=%s --ddl 'CREATE TABLE keda_test_jobs (id INT64 NOT NULL, status STRING(64) NOT NULL) PRIMARY KEY (id)'",
		databaseID, instanceID, projectID,
	)
	_, err = ExecuteCommand(cmd)
	if !assert.NoErrorf(t, err, "failed to create test table: %s", err) {
		return err
	}

	return nil
}

func cleanupSpanner(t *testing.T) {
	t.Helper()
	t.Logf("--- deleting Spanner instance %s ---", instanceID)
	// Deleting the instance also removes all databases and tables within it.
	_, _ = ExecuteCommand(spannerCmd(
		"gcloud spanner instances delete %s --project=%s --quiet",
		instanceID, projectID,
	))
}

// insertPendingRows inserts n rows with status='pending' starting at the given
// id offset (to avoid primary key conflicts across calls).
func insertPendingRows(t *testing.T, count, idOffset int) {
	t.Helper()
	t.Logf("--- inserting %d pending rows (offset %d) ---", count, idOffset)
	for i := 0; i < count; i++ {
		id := idOffset + i
		cmd := spannerCmd(
			"gcloud spanner rows insert --table=keda_test_jobs --instance=%s --database=%s --project=%s --data=id=%d,status=pending",
			instanceID, databaseID, projectID, id,
		)
		_, err := ExecuteCommand(cmd)
		assert.NoErrorf(t, err, "failed to insert row id=%d: %s", id, err)
	}
}

func clearAllRows(t *testing.T) {
	t.Helper()
	t.Log("--- clearing all rows ---")
	_, _ = ExecuteCommand(spannerCmd(
		"gcloud spanner databases execute-sql %s --instance=%s --project=%s --sql 'DELETE FROM keda_test_jobs WHERE true'",
		databaseID, instanceID, projectID,
	))
}

// ── Scaling scenarios ─────────────────────────────────────────────────────────

// testActivation verifies that the scaler does not trigger when the row count
// is at or below activationThreshold.
func testActivation(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing activation threshold: scaler should stay at 0 replicas ---")
	insertPendingRows(t, activationThreshold, 1000)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 60)
}

// testScaleOut verifies that adding enough rows triggers scale-out to
// maxReplicaCount.
func testScaleOut(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale out ---")
	// Insert enough additional rows to exceed activationThreshold and saturate
	// maxReplicaCount: total = pendingRowsForScaleOut rows.
	insertPendingRows(t, pendingRowsForScaleOut-activationThreshold, 2000)

	assert.True(t,
		WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 30, 10),
		fmt.Sprintf("replica count should reach %d after scale-out", maxReplicaCount),
	)
}

// testScaleIn verifies that removing all pending rows causes the deployment to
// scale back to zero.
func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in to zero ---")
	clearAllRows(t)

	assert.True(t,
		WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 30, 10),
		"replica count should return to 0 after all rows are removed",
	)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func getTemplateData() (templateData, []Template) {
	data := templateData{
		TestNamespace:   testNamespace,
		SecretName:      secretName,
		GcpCreds:        base64.StdEncoding.EncodeToString([]byte(gcpKey)),
		DeploymentName:  deploymentName,
		ScaledObjName:   scaledObjName,
		ProjectID:       projectID,
		InstanceID:      instanceID,
		DatabaseID:      databaseID,
		Query:           scalerQuery,
		TargetValue:     targetValue,
		ActivationValue: activationThreshold,
		MaxReplicaCount: maxReplicaCount,
	}
	templates := []Template{
		{Name: "secretTemplate", Config: secretTemplate},
		{Name: "deploymentTemplate", Config: deploymentTemplate},
		{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		{Name: "gcpSdkTemplate", Config: gcpSdkTemplate},
	}
	return data, templates
}
