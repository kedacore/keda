//go:build e2e
// +build e2e

package azure_postgresql_flex_server_aad_wi_test

import (
	"encoding/base64"
	"fmt"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
	pg "github.com/kedacore/keda/v2/tests/scalers/postgresql/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "azure-postgresql-test"
)

var (
	testNamespace                   = fmt.Sprintf("%s-ns", testName)
	deploymentName                  = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName                = fmt.Sprintf("%s-so", testName)
	azureTriggerAuthenticationName  = fmt.Sprintf("%s-ta", testName)
	secretName                      = fmt.Sprintf("%s-secret", testName)
	secretKey                       = "postgresql_conn_str"
	postgreSQLStatefulSetName       = "azure-postgresql"
	postgresqlPodName               = fmt.Sprintf("%s-0", postgreSQLStatefulSetName)
	azurePostgreSQLAdminUsername    = os.Getenv("TF_AZURE_POSTGRES_ADMIN_USERNAME")
	azurePostgreSQLAdminPassword    = os.Getenv("TF_AZURE_POSTGRES_ADMIN_PASSWORD")
	azurePostgreSQLFQDN             = os.Getenv("TF_AZURE_POSTGRES_FQDN")
	azurePostgreSQLDatabase         = os.Getenv("TF_AZURE_POSTGRES_DB_NAME")
	azureADTenantID                 = os.Getenv("TF_AZURE_SP_TENANT")
	azurePostgreSQLUamiClientID     = os.Getenv("TF_AZURE_POSTGRES_IDENTITY_APP_ID")
	azurePostgreSQLUamiName         = os.Getenv("TF_AZURE_POSTGRES_IDENTITY_NAME")
	azurePostgreSQLConnectionString = GetAzureConnectionString(azurePostgreSQLAdminUsername, azurePostgreSQLAdminPassword, azurePostgreSQLFQDN, azurePostgreSQLDatabase)
	localPostgreSQLUsername         = "test-user"
	localPostgreSQLPassword         = "test-password"
	localPostgreSQLDatabase         = "test_db"
	minReplicaCount                 = 0
	maxReplicaCount                 = 2
)

type templateData struct {
	TestNamespace                         string
	DeploymentName                        string
	ScaledObjectName                      string
	AzureTriggerAuthenticationName        string
	SecretName                            string
	SecretKey                             string
	PostgreSQLImage                       string
	PostgreSQLStatefulSetName             string
	AzurePostgreSQLConnectionStringBase64 string
	AzurePostgreSQLAdminUsername          string
	AzurePostgreSQLAdminPassword          string
	AzurePostgreSQLFQDN                   string
	AzurePostgreSQLDatabase               string
	AzurePostgreSQLUamiClientID           string
	AzurePostgreSQLUamiName               string
	AzureADTenantID                       string
	PostgreSQLUsername                    string
	PostgreSQLPassword                    string
	PostgreSQLDatabase                    string
	MinReplicaCount                       int
	MaxReplicaCount                       int
}

const (
	azureSecretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
type: Opaque
data:
  postgresql_conn_str: {{.AzurePostgreSQLConnectionStringBase64}}
`

	azureTriggerAuthTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthenticationName}}
  namespace: {{.TestNamespace}}
spec:
  podIdentity:
    provider: azure-workload
	identityId: {{.AzurePostgreSQLUAMIClientID}}
    identityTenantId: {{.AzureADTenantID}}
`

	azureScaledObjectTemplate = `apiVersion: keda.sh/v1alpha1
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
	  host: {{.AzurePostgreSQLFQDN}}
	  port: "5432"
	  userName: {{.AzurePostgreSQLUamiName}}
	  dbName: {{.AzurePostgreSQLDatabase}}
      targetQueryValue: "4"
      activationTargetQueryValue: "5"
      query: "SELECT CEIL(COUNT(*) / 5) FROM task_instance WHERE state='running' OR state='queued'"
    authenticationRef:
      name: {{.TriggerAuthenticationName}}
`
)

func TestPostreSQLScaler(t *testing.T) {
	kc := GetKubernetesClient(t)
	_, postgreSQLtemplates := getPostgreSQLTemplateData()
	_, templates := getTemplateData()
	t.Cleanup(func() {
		// Delete table on remote Azure Postgres Flexible server
		deleteTableSQL := "DROP TABLE task_instance;"
		delOk, delOut, delErrOut, delErr := WaitForSuccessfulExecCommandOnSpecificPod(t, postgresqlPodName, testNamespace,
			fmt.Sprintf("PGPASSWORD=%s psql -h %s -p 5432 -U %s -d %s -c \"%s\"", azurePostgreSQLAdminPassword, azurePostgreSQLFQDN, azurePostgreSQLAdminUsername, azurePostgreSQLDatabase, deleteTableSQL), 60, 3)
		require.True(t, delOk, "executing a command on PostreSQL Pod should work; Output: %s, ErrorOutput: %s, Error: %s", delOut, delErrOut, delErr)

		KubectlDeleteMultipleWithTemplate(t, data, templates)
		DeleteKubernetesResources(t, testNamespace, data, postgreSQLtemplates)
	})

	// Create kubernetes resources for local PostgreSQL server
	CreateKubernetesResources(t, kc, testNamespace, data, postgreSQLtemplates)

	require.True(t, WaitForStatefulsetReplicaReadyCount(t, kc, postgreSQLStatefulSetName, testNamespace, 1, 60, 3),
		"replica count should be %d after 3 minutes", 1)

	// Create table on remote Azure Postgres Flexible server
	createTableSQL := "CREATE TABLE task_instance (id serial PRIMARY KEY,state VARCHAR(10));"
	ok, out, errOut, err := WaitForSuccessfulExecCommandOnSpecificPod(t, postgresqlPodName, testNamespace,
		fmt.Sprintf("PGPASSWORD=%s psql -h %s -p 5432 -U %s -d %s -c \"%s\"", azurePostgreSQLAdminPassword, azurePostgreSQLFQDN, azurePostgreSQLAdminUsername, azurePostgreSQLDatabase, createTableSQL), 60, 3)
	require.True(t, ok, "executing a command on PostreSQL Pod should work; Output: %s, ErrorOutput: %s, Error: %s", out, errOut, err)

	// Create kubernetes resources for testing
	KubectlApplyMultipleWithTemplate(t, data, templates)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)

	testActivation(t, kc, data)
	testScaleOut(t, kc, data)
	testScaleIn(t, kc)
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
	TestNamespace:                         testNamespace,
	PostgreSQLStatefulSetName:             postgreSQLStatefulSetName,
	DeploymentName:                        deploymentName,
	ScaledObjectName:                      scaledObjectName,
	MinReplicaCount:                       minReplicaCount,
	MaxReplicaCount:                       maxReplicaCount,
	AzureTriggerAuthenticationName:        azureTriggerAuthenticationName,
	SecretName:                            secretName,
	SecretKey:                             secretKey,
	PostgreSQLImage:                       pg.PostgresqlImage,
	AzurePostgreSQLAdminUsername:          azurePostgreSQLAdminUsername,
	AzurePostgreSQLAdminPassword:          azurePostgreSQLAdminPassword,
	AzurePostgreSQLDatabase:               azurePostgreSQLDatabase,
	AzureADTenantID:                       azureADTenantID,
	AzurePostgreSQLUamiClientID:           azurePostgreSQLUamiClientID,
	AzurePostgreSQLUamiName:               azurePostgreSQLUamiName,
	AzurePostgreSQLConnectionStringBase64: base64.StdEncoding.EncodeToString([]byte(azurePostgreSQLConnectionString)),
	PostgreSQLUsername:                    localPostgreSQLUsername,
	PostgreSQLPassword:                    localPostgreSQLPassword,
	PostgreSQLDatabase:                    localPostgreSQLDatabase,
}

func getPostgreSQLTemplateData() (templateData, []Template) {
	return data, []Template{
		{Name: "postgreSQLStatefulSetTemplate", Config: pg.PostgreSQLStatefulSetTemplate},
	}
}

func getTemplateData() (templateData, []Template) {
	return data, []Template{
		{Name: "azureSecretTemplate", Config: azureSecretTemplate},
		{Name: "deploymentTemplate", Config: pg.DeploymentTemplate},
		{Name: "azureTriggerAuthenticationTemplate", Config: azureTriggerAuthTemplate},
		{Name: "azureScaledObjectTemplate", Config: azureScaledObjectTemplate},
	}
}

func GetAzureConnectionString(username string, password string, fqdn string, database string) string {
	return fmt.Sprintf("postgresql://%s:%s@%s:5432/%s?sslmode=require", username, password, fqdn, database)
}
