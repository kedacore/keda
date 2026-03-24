//go:build e2e
// +build e2e

package azure_cosmosdb_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../../.env")

const (
	testName = "azure-cosmosdb-test"
)

var (
	connectionString = os.Getenv("TF_AZURE_COSMOSDB_CONNECTION_STRING")
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	secretName       = fmt.Sprintf("%s-secret", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
	databaseID       = "keda-test-db"
	containerID      = "keda-test-container"
	leaseDatabaseID  = "keda-test-db"
	leaseContainerID = "keda-test-leases"
	processorName    = "keda-test-processor"
)

type templateData struct {
	TestNamespace    string
	SecretName       string
	Connection       string
	DeploymentName   string
	ScaledObjectName string
	DatabaseID       string
	ContainerID      string
	LeaseDatabaseID  string
	LeaseContainerID string
	ProcessorName    string
}

const (
	secretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  connection: {{.Connection}}
`

	triggerAuthTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.SecretName}}-trigger-auth
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
    - parameter: connection
      name: {{.SecretName}}
      key: connection
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
        - name: {{.DeploymentName}}
          image: ghcr.io/kedacore/tests-azure-cosmosdb
          env:
            - name: COSMOS_CONNECTION
              valueFrom:
                secretKeyRef:
                  name: {{.SecretName}}
                  key: connection
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
  cooldownPeriod: 10
  triggers:
    - type: azure-cosmosdb
      metadata:
        databaseId: {{.DatabaseID}}
        containerId: {{.ContainerID}}
        leaseDatabaseId: {{.LeaseDatabaseID}}
        leaseContainerId: {{.LeaseContainerID}}
        processorName: {{.ProcessorName}}
        connectionFromEnv: COSMOS_CONNECTION
        activationChangeFeedLagThreshold: "0"
      authenticationRef:
        name: {{.SecretName}}-trigger-auth
`
)

func TestScaler(t *testing.T) {
	// setup
	ctx := context.Background()
	t.Log("--- setting up ---")
	require.NotEmpty(t, connectionString, "TF_AZURE_COSMOSDB_CONNECTION_STRING env variable is required for azure cosmosdb test")

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")

	// test scaling
	testActivation(t, kc)
	testScaleOut(ctx, t, kc)
	testScaleIn(t, kc)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func getTemplateData() (templateData, []Template) {
	base64ConnectionString := base64.StdEncoding.EncodeToString([]byte(connectionString))

	return templateData{
			TestNamespace:    testNamespace,
			SecretName:       secretName,
			Connection:       base64ConnectionString,
			DeploymentName:   deploymentName,
			ScaledObjectName: scaledObjectName,
			DatabaseID:       databaseID,
			ContainerID:      containerID,
			LeaseDatabaseID:  leaseDatabaseID,
			LeaseContainerID: leaseContainerID,
			ProcessorName:    processorName,
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "triggerAuthTemplate", Config: triggerAuthTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

func testActivation(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing activation ---")
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 60)
}

func testScaleOut(ctx context.Context, t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale out ---")
	addDocuments(ctx, t, 10)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 1),
		"replica count should be 1 after 1 minute")
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")
}

// addDocuments inserts documents into the Cosmos DB data container via the REST API
// to generate change feed lag for the scaler to detect.
func addDocuments(ctx context.Context, t *testing.T, count int) {
	t.Helper()

	endpoint, key, err := parseConnString(connectionString)
	require.NoErrorf(t, err, "cannot parse connection string - %s", err)

	for i := 0; i < count; i++ {
		docID := fmt.Sprintf("test-doc-%d-%d", GetRandomNumber(), i)
		body := fmt.Sprintf(`{"id":"%s","partitionKey":"%s","message":"Test document %d"}`, docID, docID, i)

		resourceLink := fmt.Sprintf("dbs/%s/colls/%s", databaseID, containerID)
		reqURL := fmt.Sprintf("%s/%s/docs", strings.TrimRight(endpoint, "/"), resourceLink)

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(body))
		require.NoErrorf(t, err, "cannot create request - %s", err)

		now := time.Now().UTC().Format(http.TimeFormat)
		req.Header.Set("Authorization", cosmosAuthToken("post", "docs", resourceLink, now, key))
		req.Header.Set("x-ms-date", now)
		req.Header.Set("x-ms-version", "2018-12-31")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-ms-documentdb-partitionkey", fmt.Sprintf(`["%s"]`, docID))

		resp, err := http.DefaultClient.Do(req)
		require.NoErrorf(t, err, "cannot send request - %s", err)

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		require.Truef(t, resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK,
			"unexpected status %d creating document: %s", resp.StatusCode, string(respBody))

		t.Logf("Document created: %s", docID)
	}
}

func parseConnString(conn string) (string, string, error) {
	var endpoint, key string
	for _, part := range strings.Split(conn, ";") {
		part = strings.TrimSpace(part)
		switch {
		case strings.HasPrefix(part, "AccountEndpoint="):
			endpoint = strings.TrimPrefix(part, "AccountEndpoint=")
		case strings.HasPrefix(part, "AccountKey="):
			key = strings.TrimPrefix(part, "AccountKey=")
		}
	}
	if endpoint == "" || key == "" {
		return "", "", fmt.Errorf("invalid connection string: missing AccountEndpoint or AccountKey")
	}
	return endpoint, key, nil
}

func cosmosAuthToken(verb, resourceType, resourceLink, date, key string) string {
	keyBytes, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return ""
	}
	text := fmt.Sprintf("%s\n%s\n%s\n%s\n\n",
		strings.ToLower(verb),
		strings.ToLower(resourceType),
		resourceLink,
		strings.ToLower(date))
	h := hmac.New(sha256.New, keyBytes)
	h.Write([]byte(text))
	sig := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return url.QueryEscape(fmt.Sprintf("type=master&ver=1.0&sig=%s", sig))
}
