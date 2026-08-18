//go:build e2e
// +build e2e

package azure_cosmosdb_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
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
	databaseID       = fmt.Sprintf("keda-test-db-%d-%d", time.Now().UnixNano(), GetRandomNumber())
	containerID      = "keda-test-container"
	leaseDatabaseID  = databaseID
	leaseContainerID = "keda-test-leases"
	processorName    = "keda-test-processor"
)

type templateData struct {
	TestNamespace    string
	SecretName       string
	Connection       string
	Endpoint         string
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
  podIdentity:
    provider: azure-workload
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
  replicas: 1
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
            - name: COSMOS_DATABASE_ID
              value: {{.DatabaseID}}
            - name: COSMOS_CONTAINER_ID
              value: {{.ContainerID}}
            - name: COSMOS_LEASE_DATABASE_ID
              value: {{.LeaseDatabaseID}}
            - name: COSMOS_LEASE_CONTAINER_ID
              value: {{.LeaseContainerID}}
            - name: COSMOS_PROCESSOR_NAME
              value: {{.ProcessorName}}
            - name: CosmosDbConfig__Connection
              valueFrom:
                secretKeyRef:
                  name: {{.SecretName}}
                  key: connection
            - name: CosmosDbConfig__DatabaseId
              value: {{.DatabaseID}}
            - name: CosmosDbConfig__ContainerId
              value: {{.ContainerID}}
            - name: CosmosDbConfig__LeaseConnection
              valueFrom:
                secretKeyRef:
                  name: {{.SecretName}}
                  key: connection
            - name: CosmosDbConfig__LeaseDatabaseId
              value: {{.LeaseDatabaseID}}
            - name: CosmosDbConfig__LeaseContainerId
              value: {{.LeaseContainerID}}
            - name: CosmosDbConfig__ProcessorName
              value: {{.ProcessorName}}
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
        endpoint: {{.Endpoint}}
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
	endpoint, _, err := parseConnString(connectionString)
	require.NoErrorf(t, err, "cannot parse connection string - %s", err)

	// Create Cosmos DB resources (database + containers)
	setupCosmosDB(ctx, t)

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData(endpoint)

	CreateKubernetesResources(t, kc, testNamespace, data, templates[:3])
	t.Cleanup(func() {
		DeleteNamespace(t, testNamespace)
		assert.Truef(t, WaitForNamespaceDeletion(t, testNamespace), "%s namespace not deleted", testNamespace)
	})
	require.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 1),
		"bootstrap processor pod should be ready after 1 minute")
	waitForLeaseDocuments(ctx, t, time.Minute)
	KubectlApplyWithTemplate(t, data, templates[3].Name, templates[3].Config)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")

	// test scaling
	testActivation(t, kc)
	testScaleOut(ctx, t, kc)
	testScaleIn(t, kc)

	// cleanup
	KubectlDeleteMultipleWithTemplate(t, data, templates)
}

func getTemplateData(endpoint string) (templateData, []Template) {
	base64ConnectionString := base64.StdEncoding.EncodeToString([]byte(connectionString))

	return templateData{
			TestNamespace:    testNamespace,
			SecretName:       secretName,
			Connection:       base64ConnectionString,
			Endpoint:         endpoint,
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

func waitForLeaseDocuments(ctx context.Context, t *testing.T, timeout time.Duration) {
	t.Helper()

	endpoint, key, err := parseConnString(connectionString)
	require.NoErrorf(t, err, "cannot parse connection string - %s", err)

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	var lastErr error
	for {
		leaseCount, err := getProcessorLeaseCount(timeoutCtx, endpoint, key)
		if err == nil && leaseCount > 0 {
			t.Logf("Processor initialized %d lease documents", leaseCount)
			return
		}
		lastErr = err

		select {
		case <-timeoutCtx.Done():
			require.NoError(t, lastErr, "failed to query processor lease documents")
			t.Fatal("processor did not initialize lease documents within the timeout")
		case <-ticker.C:
		}
	}
}

func getProcessorLeaseCount(ctx context.Context, endpoint, key string) (int, error) {
	resourceLink := fmt.Sprintf("dbs/%s/colls/%s", leaseDatabaseID, leaseContainerID)
	reqURL := fmt.Sprintf("%s/%s/docs", strings.TrimRight(endpoint, "/"), resourceLink)
	// Real .NET/Java SDK lease document ids are built as
	// {processorName}{monitoredAccountHost}_{rid}..{partitionId} - the processor name is
	// directly concatenated with the monitored (data) account's hostname, with no separator.
	// Matching on processorName alone would let a processor named "app" also match leases
	// from "app-extended"; appending the account's short name (e.g. "myaccount" from
	// "myaccount.documents.azure.com") closes that gap. Since every Cosmos DB account
	// hostname has the form "{account}.<domain-suffix>", the "." immediately following the
	// short name in the real id lets us match without needing the domain suffix.
	prefix, err := json.Marshal(processorName + accountShortName(endpoint) + ".")
	if err != nil {
		return 0, fmt.Errorf("cannot marshal processor name prefix: %w", err)
	}
	body := fmt.Sprintf(`{"query":"SELECT * FROM c WHERE STARTSWITH(c.id, @prefix)","parameters":[{"name":"@prefix","value":%s}]}`, prefix)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("cannot create lease query request: %w", err)
	}

	now := time.Now().UTC().Format(http.TimeFormat)
	req.Header.Set("Authorization", cosmosAuthToken(http.MethodPost, "docs", resourceLink, now, key))
	req.Header.Set("x-ms-date", now)
	req.Header.Set("x-ms-version", "2018-12-31")
	req.Header.Set("Content-Type", "application/query+json")
	req.Header.Set("x-ms-documentdb-isquery", "true")
	req.Header.Set("x-ms-documentdb-query-enablecrosspartition", "true")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("cannot query processor leases: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("cannot read processor lease query response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status %d querying processor leases: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Documents []struct {
			LeaseToken        string `json:"LeaseToken"`
			ContinuationToken string `json:"ContinuationToken"`
		} `json:"Documents"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return 0, fmt.Errorf("cannot decode processor lease query response: %w", err)
	}

	leaseCount := 0
	for _, document := range result.Documents {
		if document.LeaseToken != "" && document.ContinuationToken != "" {
			leaseCount++
		}
	}
	return leaseCount, nil
}

// accountShortName extracts the short account name from a Cosmos DB endpoint,
// e.g. "https://myaccount.documents.azure.com:443/" -> "myaccount". Every Cosmos DB account
// hostname has the form "{account}.<domain-suffix>", so this is stable across clouds
// (public, sovereign, or private) without needing to know the exact domain suffix.
func accountShortName(endpoint string) string {
	host := endpoint
	if u, err := url.Parse(endpoint); err == nil && u.Hostname() != "" {
		host = u.Hostname()
	}
	if idx := strings.IndexByte(host, '.'); idx >= 0 {
		return host[:idx]
	}
	return host
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

// setupCosmosDB creates the database, data container, and lease container if they don't exist.
func setupCosmosDB(ctx context.Context, t *testing.T) {
	t.Helper()

	endpoint, key, err := parseConnString(connectionString)
	require.NoErrorf(t, err, "cannot parse connection string - %s", err)

	// Create database
	cosmosCreateResource(ctx, t, endpoint, key, "", "dbs", fmt.Sprintf(`{"id":"%s"}`, databaseID))
	t.Cleanup(func() {
		deleteCosmosDatabase(context.Background(), t, endpoint, key)
	})

	// Create data container with /id as partition key
	dbLink := fmt.Sprintf("dbs/%s", databaseID)
	cosmosCreateResource(ctx, t, endpoint, key, dbLink, "colls",
		fmt.Sprintf(`{"id":"%s","partitionKey":{"paths":["/id"],"kind":"Hash"}}`, containerID))

	// Create lease container with /id as partition key
	cosmosCreateResource(ctx, t, endpoint, key, dbLink, "colls",
		fmt.Sprintf(`{"id":"%s","partitionKey":{"paths":["/id"],"kind":"Hash"}}`, leaseContainerID))
}

func deleteCosmosDatabase(ctx context.Context, t *testing.T, endpoint, key string) {
	t.Helper()

	resourceLink := fmt.Sprintf("dbs/%s", databaseID)
	reqURL := fmt.Sprintf("%s/%s", strings.TrimRight(endpoint, "/"), resourceLink)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, reqURL, nil)
	require.NoErrorf(t, err, "cannot create database deletion request - %s", err)

	now := time.Now().UTC().Format(http.TimeFormat)
	req.Header.Set("Authorization", cosmosAuthToken(http.MethodDelete, "dbs", resourceLink, now, key))
	req.Header.Set("x-ms-date", now)
	req.Header.Set("x-ms-version", "2018-12-31")

	resp, err := http.DefaultClient.Do(req)
	require.NoErrorf(t, err, "cannot delete Cosmos DB test database - %s", err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoErrorf(t, err, "cannot read database deletion response - %s", err)
	require.Truef(t, resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound,
		"unexpected status %d deleting database: %s", resp.StatusCode, string(respBody))
}

// cosmosCreateResource creates a Cosmos DB resource via REST API, ignoring 409 Conflict (already exists).
func cosmosCreateResource(ctx context.Context, t *testing.T, endpoint, key, parentLink, resourceType, body string) {
	t.Helper()

	reqURL := fmt.Sprintf("%s/%s/%s", strings.TrimRight(endpoint, "/"), parentLink, resourceType)
	if parentLink == "" {
		reqURL = fmt.Sprintf("%s/%s", strings.TrimRight(endpoint, "/"), resourceType)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(body))
	require.NoErrorf(t, err, "cannot create request - %s", err)

	now := time.Now().UTC().Format(http.TimeFormat)
	req.Header.Set("Authorization", cosmosAuthToken("post", resourceType, parentLink, now, key))
	req.Header.Set("x-ms-date", now)
	req.Header.Set("x-ms-version", "2018-12-31")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoErrorf(t, err, "cannot send request - %s", err)
	respBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		t.Logf("Resource already exists (409), skipping: %s/%s", parentLink, resourceType)
		return
	}

	require.Truef(t, resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK,
		"unexpected status %d creating resource: %s", resp.StatusCode, string(respBody))
	t.Logf("Created resource: %s/%s", parentLink, resourceType)
}
