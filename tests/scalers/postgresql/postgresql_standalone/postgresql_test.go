//go:build e2e
// +build e2e

package postgresql_standalone_test

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
	pg "github.com/kedacore/keda/v2/tests/scalers/postgresql/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "postgresql-test"
)

var (
	testNamespace              = fmt.Sprintf("%s-ns", testName)
	deploymentName             = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName           = fmt.Sprintf("%s-so", testName)
	triggerAuthenticationName  = fmt.Sprintf("%s-ta", testName)
	secretName                 = fmt.Sprintf("%s-secret", testName)
	secretKey                  = "postgresql_conn_str"
	postgreSQLStatefulSetName  = "postgresql"
	postgresqlPodName          = fmt.Sprintf("%s-0", postgreSQLStatefulSetName)
	postgreSQLUsername         = "test-user"
	postgreSQLPassword         = "test-password"
	postgreSQLDatabase         = "test_db"
	postgreSQLConnectionString = pg.GetConnectionString(postgreSQLUsername, postgreSQLPassword, []string{postgreSQLStatefulSetName}, testNamespace, postgreSQLDatabase)
	minReplicaCount            = 0
	maxReplicaCount            = 2
)

type templateData struct {
	TestNamespace                    string
	DeploymentName                   string
	ScaledObjectName                 string
	TriggerAuthenticationName        string
	SecretName                       string
	SecretKey                        string
	PostgreSQLImage                  string
	PostgreSQLStatefulSetName        string
	PostgreSQLConnectionStringBase64 string
	PostgreSQLUsername               string
	PostgreSQLPassword               string
	PostgreSQLDatabase               string
	MinReplicaCount                  int
	MaxReplicaCount                  int
}

const (
	secretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
type: Opaque
data:
  postgresql_conn_str: {{.PostgreSQLConnectionStringBase64}}
`
)

func TestPostreSQLScaler(t *testing.T) {
	// Create kubernetes resources for PostgreSQL server
	kc := GetKubernetesClient(t)
	data, postgreSQLtemplates := getPostgreSQLTemplateData()
	CreateKubernetesResources(t, kc, testNamespace, data, postgreSQLtemplates)

	assert.True(t, WaitForStatefulsetReplicaReadyCount(t, kc, postgreSQLStatefulSetName, testNamespace, 1, 60, 3),
		"replica count should be %d after 3 minutes", 1)

	createTableSQL := "CREATE TABLE task_instance (id serial PRIMARY KEY,state VARCHAR(10));"
	ok, out, errOut, err := WaitForSuccessfulExecCommandOnSpecificPod(t, postgresqlPodName, testNamespace,
		fmt.Sprintf("psql -U %s -d %s -c \"%s\"", postgreSQLUsername, postgreSQLDatabase, createTableSQL), 60, 3)
	assert.True(t, ok, "executing a command on PostreSQL Pod should work; Output: %s, ErrorOutput: %s, Error: %s", out, errOut, err)

	// Create kubernetes resources for testing
	data, templates := getTemplateData()
	KubectlApplyMultipleWithTemplate(t, data, templates)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)

	testActivation(t, kc, data)
	testScaleOut(t, kc, data)
	testScaleIn(t, kc)

	// cleanup
	KubectlDeleteMultipleWithTemplate(t, data, templates)
	DeleteKubernetesResources(t, testNamespace, data, postgreSQLtemplates)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")
	KubectlReplaceWithTemplate(t, data, "lowLevelRecordsJobTemplate", pg.LowLevelRecordsJobTemplate)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	KubectlReplaceWithTemplate(t, data, "insertRecordsJobTemplate", pg.InsertRecordsJobTemplate)

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
	SecretKey:                        secretKey,
	PostgreSQLImage:                  pg.PostgresqlImage,
	PostgreSQLUsername:               postgreSQLUsername,
	PostgreSQLPassword:               postgreSQLPassword,
	PostgreSQLDatabase:               postgreSQLDatabase,
	PostgreSQLConnectionStringBase64: base64.StdEncoding.EncodeToString([]byte(postgreSQLConnectionString)),
}

func getPostgreSQLTemplateData() (templateData, []Template) {
	return data, []Template{
		{Name: "postgreSQLStatefulSetTemplate", Config: pg.PostgreSQLStatefulSetTemplate},
		{Name: "postgreSQLServiceTemplate", Config: pg.PostgreSQLServiceTemplate},
	}
}

func getTemplateData() (templateData, []Template) {
	return data, []Template{
		{Name: "secretTemplate", Config: secretTemplate},
		{Name: "deploymentTemplate", Config: pg.DeploymentTemplate},
		{Name: "triggerAuthenticationTemplate", Config: pg.TriggerAuthenticationTemplate},
		{Name: "scaledObjectTemplate", Config: pg.ScaledObjectTemplate},
	}
}
