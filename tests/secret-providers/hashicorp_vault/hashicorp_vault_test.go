//go:build e2e
// +build e2e

package hashicorp_vault_test

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "hashicorp-vault-test"
)

var (
	testNamespace              = fmt.Sprintf("%s-ns", testName)
	vaultNamespace             = "hashicorp-ns"
	deploymentName             = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName           = fmt.Sprintf("%s-so", testName)
	triggerAuthenticationName  = fmt.Sprintf("%s-ta", testName)
	secretName                 = fmt.Sprintf("%s-secret", testName)
	postgreSQLStatefulSetName  = "postgresql"
	postgresqlPodName          = fmt.Sprintf("%s-0", postgreSQLStatefulSetName)
	postgreSQLUsername         = "test-user"
	postgreSQLPassword         = "test-password"
	postgreSQLDatabase         = "test_db"
	postgreSQLConnectionString = fmt.Sprintf("postgresql://%s:%s@postgresql.%s.svc.cluster.local:5432/%s?sslmode=disable",
		postgreSQLUsername, postgreSQLPassword, testNamespace, postgreSQLDatabase)
	minReplicaCount = 0
	maxReplicaCount = 2
)

type templateData struct {
	TestNamespace                    string
	DeploymentName                   string
	VaultNamespace                   string
	ScaledObjectName                 string
	TriggerAuthenticationName        string
	VaultSecretPath                  string
	SecretName                       string
	HashiCorpToken                   string
	PostgreSQLStatefulSetName        string
	PostgreSQLConnectionStringBase64 string
	PostgreSQLUsername               string
	PostgreSQLPassword               string
	PostgreSQLDatabase               string
	MinReplicaCount                  int
	MaxReplicaCount                  int
}

const (
	deploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: postgresql-update-worker
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
spec:
  replicas: 0
  selector:
    matchLabels:
      app: postgresql-update-worker
  template:
    metadata:
      labels:
        app: postgresql-update-worker
    spec:
      containers:
      - image: ghcr.io/kedacore/tests-postgresql
        imagePullPolicy: Always
        name: postgresql-processor-test
        command:
          - /app
          - update
        env:
          - name: TASK_INSTANCES_COUNT
            value: "6000"
          - name: CONNECTION_STRING
            valueFrom:
              secretKeyRef:
                name: {{.SecretName}}
                key: postgresql_conn_str
`

	secretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
type: Opaque
data:
  postgresql_conn_str: {{.PostgreSQLConnectionStringBase64}}
`

	triggerAuthenticationTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthenticationName}}
  namespace: {{.TestNamespace}}
spec:
  hashiCorpVault:
    address: http://vault.{{.VaultNamespace}}:8200
    authentication: token
    credential:
      token: {{.HashiCorpToken}}
    secrets:
    - parameter: connection
      key: connectionString
      path: {{.VaultSecretPath}}
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
  cooldownPeriod:  10
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  triggers:
  - type: postgresql
    metadata:
      targetQueryValue: "4"
      activationTargetQueryValue: "5"
      query: "SELECT CEIL(COUNT(*) / 5) FROM task_instance WHERE state='running' OR state='queued'"
    authenticationRef:
      name: {{.TriggerAuthenticationName}}
`

	postgreSQLStatefulSetTemplate = `
apiVersion: apps/v1
kind: StatefulSet
metadata:
  labels:
    app: {{.PostgreSQLStatefulSetName}}
  name: {{.PostgreSQLStatefulSetName}}
  namespace: {{.TestNamespace}}
spec:
  replicas: 1
  serviceName: {{.PostgreSQLStatefulSetName}}
  selector:
    matchLabels:
      app: {{.PostgreSQLStatefulSetName}}
  template:
    metadata:
      labels:
        app: {{.PostgreSQLStatefulSetName}}
    spec:
      containers:
      - image: postgres:10.5
        name: postgresql
        env:
          - name: POSTGRES_USER
            value: {{.PostgreSQLUsername}}
          - name: POSTGRES_PASSWORD
            value: {{.PostgreSQLPassword}}
          - name: POSTGRES_DB
            value: {{.PostgreSQLDatabase}}
        ports:
          - name: postgresql
            protocol: TCP
            containerPort: 5432
`

	postgreSQLServiceTemplate = `
apiVersion: v1
kind: Service
metadata:
  labels:
    app: {{.PostgreSQLStatefulSetName}}
  name: {{.PostgreSQLStatefulSetName}}
  namespace: {{.TestNamespace}}
spec:
  ports:
  - port: 5432
    protocol: TCP
    targetPort: 5432
  selector:
    app: {{.PostgreSQLStatefulSetName}}
  type: ClusterIP
`

	lowLevelRecordsJobTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  labels:
    app: postgresql-insert-low-level-job
  name: postgresql-insert-low-level-job
  namespace: {{.TestNamespace}}
spec:
  template:
    metadata:
      labels:
        app: postgresql-insert-low-level-job
    spec:
      containers:
      - image: ghcr.io/kedacore/tests-postgresql
        imagePullPolicy: Always
        name: postgresql-processor-test
        command:
          - /app
          - insert
        env:
          - name: TASK_INSTANCES_COUNT
            value: "20"
          - name: CONNECTION_STRING
            valueFrom:
              secretKeyRef:
                name: {{.SecretName}}
                key: postgresql_conn_str
      restartPolicy: Never
  backoffLimit: 4
`

	insertRecordsJobTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  labels:
    app: postgresql-insert-job
  name: postgresql-insert-job
  namespace: {{.TestNamespace}}
spec:
  template:
    metadata:
      labels:
        app: postgresql-insert-job
    spec:
      containers:
      - image: ghcr.io/kedacore/tests-postgresql
        imagePullPolicy: Always
        name: postgresql-processor-test
        command:
          - /app
          - insert
        env:
          - name: TASK_INSTANCES_COUNT
            value: "10000"
          - name: CONNECTION_STRING
            valueFrom:
              secretKeyRef:
                name: {{.SecretName}}
                key: postgresql_conn_str
      restartPolicy: Never
  backoffLimit: 4
`
)

func TestPostreSQLScaler(t *testing.T) {
	tests := []struct {
		name               string
		vaultEngineVersion uint
		vaultSecretPath    string
	}{
		{
			name:               "vault kv engine v1",
			vaultEngineVersion: 1,
			vaultSecretPath:    "secret/keda",
		},
		{
			name:               "vault kv engine v2",
			vaultEngineVersion: 2,
			vaultSecretPath:    "secret/data/keda",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create kubernetes resources for PostgreSQL server
			kc := GetKubernetesClient(t)
			data, postgreSQLtemplates := getPostgreSQLTemplateData()

			CreateKubernetesResources(t, kc, testNamespace, data, postgreSQLtemplates)
			hashiCorpToken := setupHashiCorpVault(t, kc, test.vaultEngineVersion)

			assert.True(t, WaitForStatefulsetReplicaReadyCount(t, kc, postgreSQLStatefulSetName, testNamespace, 1, 60, 3),
				"replica count should be %d after 3 minutes", 1)

			createTableSQL := "CREATE TABLE task_instance (id serial PRIMARY KEY,state VARCHAR(10));"
			psqlCreateTableCmd := fmt.Sprintf("psql -U %s -d %s -c \"%s\"", postgreSQLUsername, postgreSQLDatabase, createTableSQL)

			ok, out, errOut, err := WaitForSuccessfulExecCommandOnSpecificPod(t, postgresqlPodName, testNamespace, psqlCreateTableCmd, 60, 3)
			assert.True(t, ok, "executing a command on PostreSQL Pod should work; Output: %s, ErrorOutput: %s, Error: %s", out, errOut, err)

			// Create kubernetes resources for testing
			data, templates := getTemplateData()
			data.HashiCorpToken = RemoveANSI(hashiCorpToken)
			data.VaultSecretPath = test.vaultSecretPath

			KubectlApplyMultipleWithTemplate(t, data, templates)
			assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
				"replica count should be %d after 3 minutes", minReplicaCount)

			testActivation(t, kc, data)
			testScaleOut(t, kc, data)
			testScaleIn(t, kc)

			// cleanup
			KubectlDeleteMultipleWithTemplate(t, data, templates)
			cleanupHashiCorpVault(t)
			DeleteKubernetesResources(t, testNamespace, data, postgreSQLtemplates)
		})
	}
}

func setupHashiCorpVault(t *testing.T, kc *kubernetes.Clientset, kvVersion uint) string {
	CreateNamespace(t, kc, vaultNamespace)

	_, err := ExecuteCommand("helm repo add hashicorp https://helm.releases.hashicorp.com")
	assert.NoErrorf(t, err, "cannot add hashicorp repo - %s", err)

	_, err = ExecuteCommand("helm repo update")
	assert.NoErrorf(t, err, "cannot update repos - %s", err)

	var helmValues strings.Builder
	helmValues.WriteString("--set server.dev.enabled=true")

	if kvVersion == 1 {
		helmValues.WriteString(" --set server.extraArgs=-dev-kv-v1")
	}

	_, err = ExecuteCommand(fmt.Sprintf(`helm upgrade --install %s --namespace %s --wait vault hashicorp/vault`, helmValues.String(), vaultNamespace))
	assert.NoErrorf(t, err, "cannot install hashicorp vault - %s", err)

	podName := "vault-0"

	_, _, err = ExecCommandOnSpecificPod(t, podName, vaultNamespace, fmt.Sprintf("vault kv put secret/keda connectionString=%s", postgreSQLConnectionString))
	assert.NoErrorf(t, err, "cannot put connection string in hashicorp vault - %s", err)

	out, _, err := ExecCommandOnSpecificPod(t, podName, vaultNamespace, "vault token create -field token")
	assert.NoErrorf(t, err, "cannot create hashicorp vault token - %s", err)

	return out
}

func cleanupHashiCorpVault(t *testing.T) {
	_, err := ExecuteCommand(fmt.Sprintf("helm uninstall vault --namespace %s", vaultNamespace))
	assert.NoErrorf(t, err, "cannot uninstall hashicorp vault - %s", err)

	_, err = ExecuteCommand("helm repo remove hashicorp")
	assert.NoErrorf(t, err, "cannot remove hashicorp repo - %s", err)

	DeleteNamespace(t, vaultNamespace)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")
	KubectlApplyWithTemplate(t, data, "lowLevelRecordsJobTemplate", lowLevelRecordsJobTemplate)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	KubectlApplyWithTemplate(t, data, "insertRecordsJobTemplate", insertRecordsJobTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
}

var data = templateData{
	TestNamespace:                    testNamespace,
	PostgreSQLStatefulSetName:        postgreSQLStatefulSetName,
	DeploymentName:                   deploymentName,
	ScaledObjectName:                 scaledObjectName,
	MinReplicaCount:                  minReplicaCount,
	MaxReplicaCount:                  maxReplicaCount,
	TriggerAuthenticationName:        triggerAuthenticationName,
	SecretName:                       secretName,
	PostgreSQLUsername:               postgreSQLUsername,
	PostgreSQLPassword:               postgreSQLPassword,
	PostgreSQLDatabase:               postgreSQLDatabase,
	PostgreSQLConnectionStringBase64: base64.StdEncoding.EncodeToString([]byte(postgreSQLConnectionString)),
	VaultNamespace:                   vaultNamespace,
}

func getPostgreSQLTemplateData() (templateData, []Template) {
	return data, []Template{
		{Name: "postgreSQLStatefulSetTemplate", Config: postgreSQLStatefulSetTemplate},
		{Name: "postgreSQLServiceTemplate", Config: postgreSQLServiceTemplate},
	}
}

func getTemplateData() (templateData, []Template) {
	return data, []Template{
		{Name: "secretTemplate", Config: secretTemplate},
		{Name: "deploymentTemplate", Config: deploymentTemplate},
		{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
		{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
	}
}
