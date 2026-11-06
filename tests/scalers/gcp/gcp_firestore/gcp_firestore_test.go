//go:build e2e
// +build e2e

package gcp_firestore_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "gcp-firestore-test"

	// activationThreshold is the activationValue in the ScaledObject: the
	// scaler stays inactive (0 replicas) while the pending document count is
	// not above this value.
	activationThreshold = 2
	// targetValue is the value per replica: one replica handles this many documents.
	targetValue     = 5
	maxReplicaCount = 4
	// pendingDocsForScaleOut must exceed activationThreshold and drive enough
	// load to reach maxReplicaCount: ceil(pendingDocsForScaleOut / targetValue).
	pendingDocsForScaleOut = 20
)

var (
	gcpKey         = os.Getenv("TF_GCP_SA_CREDENTIALS")
	creds          = make(map[string]interface{})
	errGcpKey      = json.Unmarshal([]byte(gcpKey), &creds)
	testNamespace  = fmt.Sprintf("%s-ns", testName)
	secretName     = fmt.Sprintf("%s-secret", testName)
	deploymentName = fmt.Sprintf("%s-deployment", testName)
	scaledObjName  = fmt.Sprintf("%s-so", testName)
	hpaName        = fmt.Sprintf("keda-hpa-%s", scaledObjName)
	// The project comes from the credentials. The documents live in the default
	// database, in a collection that only this run uses, so that runs do not
	// see each other's documents.
	projectID  = fmt.Sprintf("%v", creds["project_id"])
	collection = fmt.Sprintf("keda-e2e-firestore-%d", time.Now().UnixNano())
)

// ── Kubernetes manifest templates ────────────────────────────────────────────

type templateData struct {
	TestNamespace   string
	SecretName      string
	GcpCreds        string
	DeploymentName  string
	ScaledObjName   string
	ProjectID       string
	Collection      string
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

	// The workload being scaled is a no-op container that simulates a job
	// processor: KEDA scales it and the test verifies the replica count.
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
    - type: gcp-firestore
      metadata:
        projectId: {{.ProjectID}}
        collection: {{.Collection}}
        filterField: status
        filterValue: pending
        targetValue: "{{.TargetValue}}"
        activationValue: "{{.ActivationValue}}"
        credentialsFromEnv: GOOGLE_APPLICATION_CREDENTIALS_JSON
`
)

// The test needs a Firestore database in Native mode with the ID (default) in the project of the
// credentials, and a service account that can write and delete documents in it (for example
// roles/datastore.user). It creates and deletes only a collection of its own.

// ── Test entry point ──────────────────────────────────────────────────────────

func TestScaler(t *testing.T) {
	t.Log("--- validating prerequisites ---")
	require.NotEmpty(t, gcpKey, "TF_GCP_SA_CREDENTIALS env variable is required for GCP Firestore test")
	require.NoErrorf(t, errGcpKey, "failed to parse TF_GCP_SA_CREDENTIALS as JSON: %s", errGcpKey)
	require.NotEqual(t, "<nil>", projectID, "project_id is missing from TF_GCP_SA_CREDENTIALS")
	require.NotEmpty(t, projectID, "project_id is empty in TF_GCP_SA_CREDENTIALS")

	// The documents are written and deleted by the test binary itself, with the service
	// account credentials. The cleanup is registered right away and does not depend on the
	// pods or on the namespace of the test, so no document is left behind if the test fails.
	client := newFirestoreClient(t)
	verifyFirestoreAccess(t, client)

	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	CreateKubernetesResources(t, kc, testNamespace, data, templates)
	defer DeleteKubernetesResources(t, testNamespace, data, templates)

	// Without the HPA the scaler did not start, and the activation check below would pass
	// without ever reaching Firestore.
	_, err := WaitForHpaCreation(t, kc, hpaName, testNamespace, 60, 1)
	require.NoError(t, err, "the HPA of the ScaledObject should be created")

	require.True(t,
		WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"initial replica count should be 0")

	testActivation(t, kc, client)
	testScaleOut(t, kc, client)
	testScaleIn(t, kc, client)
}

// ── Firestore setup / teardown ────────────────────────────────────────────────

func newFirestoreClient(t *testing.T) *firestore.Client {
	t.Helper()
	// The client is used until the end of the test, so it is not created with a context that ends here.
	client, err := firestore.NewClient(context.Background(), projectID,
		option.WithAuthCredentialsJSON(option.ServiceAccount, []byte(gcpKey)))
	require.NoError(t, err, "failed to create the Firestore client")

	// Registered before any document is written.
	t.Cleanup(func() {
		defer client.Close()
		t.Logf("--- deleting the documents of collection %s ---", collection)
		if err := deleteAllDocuments(client); err != nil {
			t.Errorf("failed to delete the documents of collection %s, they may need manual cleanup: %v", collection, err)
		}
	})
	return client
}

// verifyFirestoreAccess stops the test before any Kubernetes resource exists when the project has
// no reachable (default) database in Native mode, or the service account cannot use it. The client
// connects lazily, so without this the first failure would be the first document write.
func verifyFirestoreAccess(t *testing.T, client *firestore.Client) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	iter := client.Collection(collection).Limit(1).Documents(ctx)
	defer iter.Stop()
	if _, err := iter.Next(); err != nil && !errors.Is(err, iterator.Done) {
		require.NoErrorf(t, err, "cannot read from Firestore: firestore.googleapis.com must be enabled in project %s, "+
			"the (default) database must exist in Native mode, and the service account must be allowed to read and write documents", projectID)
	}
}

// deleteAllDocuments deletes every document of the collection of the test, and reports every
// document that could not be deleted (the bulk writer returns the result of each delete only
// after it is flushed).
func deleteAllDocuments(client *firestore.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	bulk := client.BulkWriter(ctx)
	defer bulk.End()

	var jobs []*firestore.BulkWriterJob
	iter := client.Collection(collection).Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		job, err := bulk.Delete(doc.Ref)
		if err != nil {
			return err
		}
		jobs = append(jobs, job)
	}
	bulk.Flush()

	var errs []error
	for _, job := range jobs {
		if _, err := job.Results(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// addDocuments writes count documents with the given status. The ids start at idOffset,
// so that calls do not overwrite each other's documents.
func addDocuments(t *testing.T, client *firestore.Client, status string, count, idOffset int) {
	t.Helper()
	t.Logf("--- adding %d documents with status=%s (offset %d) ---", count, status, idOffset)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	for i := 0; i < count; i++ {
		id := fmt.Sprintf("%s-%d", status, idOffset+i)
		_, err := client.Collection(collection).Doc(id).Set(ctx, map[string]interface{}{"status": status})
		require.NoErrorf(t, err, "failed to write document %s", id)
	}
}

// ── Scaling scenarios ─────────────────────────────────────────────────────────

// testActivation verifies that the scaler does not trigger when the document
// count is at or below activationThreshold.
func testActivation(t *testing.T, kc *kubernetes.Clientset, client *firestore.Client) {
	t.Log("--- testing activation threshold: scaler should stay at 0 replicas ---")
	addDocuments(t, client, "pending", activationThreshold, 1000)
	// Documents that fail the filter must not be counted: an unfiltered count of all the documents
	// is above activationThreshold, so a scaler that ignores the filter would activate here.
	addDocuments(t, client, "done", activationThreshold, 3000)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 60)
}

// testScaleOut verifies that adding enough documents triggers scale-out to maxReplicaCount.
func testScaleOut(t *testing.T, kc *kubernetes.Clientset, client *firestore.Client) {
	t.Log("--- testing scale out ---")
	// Add enough documents to exceed activationThreshold and saturate maxReplicaCount:
	// the total is pendingDocsForScaleOut pending documents.
	addDocuments(t, client, "pending", pendingDocsForScaleOut-activationThreshold, 2000)

	require.True(t,
		WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 30, 10),
		fmt.Sprintf("replica count should reach %d after scale-out", maxReplicaCount),
	)
}

// testScaleIn verifies that removing all pending documents scales the deployment back to zero.
func testScaleIn(t *testing.T, kc *kubernetes.Clientset, client *firestore.Client) {
	t.Log("--- testing scale in to zero ---")
	require.NoError(t, deleteAllDocuments(client), "failed to delete the documents")

	assert.True(t,
		WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 30, 10),
		"replica count should return to 0 after all documents are removed",
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
		Collection:      collection,
		TargetValue:     targetValue,
		ActivationValue: activationThreshold,
		MaxReplicaCount: maxReplicaCount,
	}
	templates := []Template{
		{Name: "secretTemplate", Config: secretTemplate},
		{Name: "deploymentTemplate", Config: deploymentTemplate},
		{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
	}
	return data, templates
}
