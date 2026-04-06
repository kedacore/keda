//go:build e2e
// +build e2e

package azure_mssql_flex_server_aad_wi_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
	mssql "github.com/kedacore/keda/v2/tests/scalers/mssql/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "azure-mssql-aad-test"
)

type authTestCase struct {
	Name                        string
	TriggerAuthenticationName   string
	TriggerAuthenticationConfig string
}

var authTestCases = []authTestCase{
	{
		Name:                        "workload-identity",
		TriggerAuthenticationName:   "azureTriggerAuthenticationTemplate",
		TriggerAuthenticationConfig: azureTriggerAuthTemplate,
	},
	// Future extension point:
	// {
	// 	Name:                        "managed-identity",
	// 	TriggerAuthenticationName:   "azureManagedIdentityTriggerAuthenticationTemplate",
	// 	TriggerAuthenticationConfig: azureManagedIdentityTriggerAuthTemplate,
	// },
}

var (
	testNamespace              = fmt.Sprintf("%s-ns", testName)
	mssqlHelperStatefulSetName = "azure-mssql-helper"
	mssqlHelperPodName         = fmt.Sprintf("%s-0", mssqlHelperStatefulSetName)

	azureMssqlAdminUsername = os.Getenv("TF_AZURE_MSSQL_ADMIN_USERNAME")
	azureMssqlAdminPassword = os.Getenv("TF_AZURE_MSSQL_ADMIN_PASSWORD")
	azureMssqlFQDN          = os.Getenv("TF_AZURE_MSSQL_FQDN")
	azureMssqlDatabase      = os.Getenv("TF_AZURE_MSSQL_DB_NAME")
	azureMssqlUamiName      = os.Getenv("TF_AZURE_IDENTITY_1_NAME")

	azureMssqlConnectionString = GetAzureConnectionString(
		azureMssqlAdminUsername,
		azureMssqlAdminPassword,
		azureMssqlFQDN,
		azureMssqlDatabase,
	)

	localMssqlPassword = "Pass@word1"
	minReplicaCount    = 0
	maxReplicaCount    = 5
)

type templateData struct {
	TestNamespace             string
	DeploymentName            string
	ScaledObjectName          string
	TriggerAuthenticationName string
	SecretName                string

	MssqlServerName string
	MssqlPassword   string

	AzureMssqlAdminUsername    string
	AzureMssqlAdminPassword    string
	AzureMssqlFQDN             string
	AzureMssqlDatabase         string
	AzureMssqlUamiName         string
	AzureMssqlConnectionString string

	DriverName      string
	MinReplicaCount int
	MaxReplicaCount int
}

const (
	azureSecretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
type: Opaque
stringData:
  mssql-connection-string: {{.AzureMssqlConnectionString}}
`

	azureTriggerAuthTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthenticationName}}
  namespace: {{.TestNamespace}}
spec:
  podIdentity:
    provider: azure-workload
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
  - type: mssql
    metadata:
      host: {{.AzureMssqlFQDN}}
      port: "1433"
      database: {{.AzureMssqlDatabase}}
      username: {{.AzureMssqlUamiName}}
      driverName: {{.DriverName}}
      query: "SELECT COUNT(*) FROM tasks WHERE [status]='running' OR [status]='queued'"
      targetValue: "1"
      activationTargetValue: "15"
    authenticationRef:
      name: {{.TriggerAuthenticationName}}
`
)

func TestMssqlScaler(t *testing.T) {
	requireEnv(t,
		"TF_AZURE_MSSQL_ADMIN_USERNAME",
		"TF_AZURE_MSSQL_ADMIN_PASSWORD",
		"TF_AZURE_MSSQL_FQDN",
		"TF_AZURE_MSSQL_DB_NAME",
		"TF_AZURE_IDENTITY_1_NAME",
	)

	for _, authCase := range authTestCases {
		t.Run(authCase.Name, func(t *testing.T) {
			testMssqlScalerAuthPath(t, authCase)
		})
	}
}

func testMssqlScalerAuthPath(t *testing.T, authCase authTestCase) {
	data := newTemplateData(authCase)

	kc := GetKubernetesClient(t)
	_, mssqlTemplates := getMssqlTemplateData(data)
	_, templates := getTemplateData(data, authCase)

	t.Cleanup(func() {
		deleteTableCommand := fmt.Sprintf(
			"/opt/mssql-tools18/bin/sqlcmd -S %s -C -U %s -P %q -d %s -Q %q",
			data.AzureMssqlFQDN,
			data.AzureMssqlAdminUsername,
			data.AzureMssqlAdminPassword,
			data.AzureMssqlDatabase,
			"IF OBJECT_ID('dbo.tasks', 'U') IS NOT NULL DROP TABLE dbo.tasks;",
		)
		_, _, _, _ = WaitForSuccessfulExecCommandOnSpecificPod(t, mssqlHelperPodName, data.TestNamespace, deleteTableCommand, 60, 3)

		KubectlDeleteMultipleWithTemplate(t, data, templates)
		DeleteKubernetesResources(t, data.TestNamespace, data, mssqlTemplates)
	})

	// Create kubernetes resources for local MSSQL helper server.
	CreateKubernetesResources(t, kc, data.TestNamespace, data, mssqlTemplates)

	require.True(t, WaitForStatefulsetReplicaReadyCount(t, kc, data.MssqlServerName, data.TestNamespace, 1, 60, 3),
		"replica count should be %d after 3 minutes", 1)

	// Delete table on remote Azure MSSQL server.
	deleteTableCommand := fmt.Sprintf(
		"/opt/mssql-tools18/bin/sqlcmd -S %s -C -U %s -P %q -d %s -Q %q",
		data.AzureMssqlFQDN,
		data.AzureMssqlAdminUsername,
		data.AzureMssqlAdminPassword,
		data.AzureMssqlDatabase,
		"IF OBJECT_ID('dbo.tasks', 'U') IS NOT NULL DROP TABLE dbo.tasks;",
	)
	ok, out, errOut, err := WaitForSuccessfulExecCommandOnSpecificPod(t, mssqlHelperPodName, data.TestNamespace, deleteTableCommand, 60, 3)
	require.True(t, ok, "executing a command on MSSQL helper pod should work; Output: %s, ErrorOutput: %s, Error: %s", out, errOut, err)

	// Create table on remote Azure MSSQL server.
	createTableCommand := fmt.Sprintf(
		"/opt/mssql-tools18/bin/sqlcmd -S %s -C -U %s -P %q -d %s -Q %q",
		data.AzureMssqlFQDN,
		data.AzureMssqlAdminUsername,
		data.AzureMssqlAdminPassword,
		data.AzureMssqlDatabase,
		"CREATE TABLE tasks ([id] int identity primary key, [status] varchar(10));",
	)
	ok, out, errOut, err = WaitForSuccessfulExecCommandOnSpecificPod(t, mssqlHelperPodName, data.TestNamespace, createTableCommand, 60, 3)
	require.True(t, ok, "executing a command on MSSQL helper pod should work; Output: %s, ErrorOutput: %s, Error: %s", out, errOut, err)

	// This may need adjustment depending on how the Azure identity is provisioned.
	grantAccessCommand := fmt.Sprintf(
		"/opt/mssql-tools18/bin/sqlcmd -S %s -C -U %s -P %q -d %s -Q %q",
		data.AzureMssqlFQDN,
		data.AzureMssqlAdminUsername,
		data.AzureMssqlAdminPassword,
		data.AzureMssqlDatabase,
		fmt.Sprintf(
			"IF NOT EXISTS (SELECT * FROM sys.database_principals WHERE name = N'%s') BEGIN CREATE USER [%s] FROM EXTERNAL PROVIDER; END; ALTER ROLE db_datareader ADD MEMBER [%s]; ALTER ROLE db_datawriter ADD MEMBER [%s];",
			data.AzureMssqlUamiName,
			data.AzureMssqlUamiName,
			data.AzureMssqlUamiName,
			data.AzureMssqlUamiName,
		),
	)
	ok, out, errOut, err = WaitForSuccessfulExecCommandOnSpecificPod(t, mssqlHelperPodName, data.TestNamespace, grantAccessCommand, 60, 3)
	require.True(t, ok, "executing a command on MSSQL helper pod should work; Output: %s, ErrorOutput: %s, Error: %s", out, errOut, err)

	// Create kubernetes resources for testing.
	KubectlApplyMultipleWithTemplate(t, data, templates)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, data.DeploymentName, data.TestNamespace, data.MinReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", data.MinReplicaCount)

	testActivation(t, kc, data)
	testScaleOut(t, kc, data)
	testScaleIn(t, kc, data)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")
	KubectlReplaceWithTemplate(t, data, "insertRecordsJobTemplate1", mssql.InsertRecordsJobTemplate1)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, data.DeploymentName, data.TestNamespace, data.MinReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	KubectlReplaceWithTemplate(t, data, "insertRecordsJobTemplate2", mssql.InsertRecordsJobTemplate2)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, data.DeploymentName, data.TestNamespace, data.MaxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", data.MaxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale in ---")

	updateRecordsCommand := fmt.Sprintf(
		"/opt/mssql-tools18/bin/sqlcmd -S %s -C -U %s -P %q -d %s -Q %q",
		data.AzureMssqlFQDN,
		data.AzureMssqlAdminUsername,
		data.AzureMssqlAdminPassword,
		data.AzureMssqlDatabase,
		"UPDATE tasks SET [status] = 'processed';",
	)
	ok, out, errOut, err := WaitForSuccessfulExecCommandOnSpecificPod(t, mssqlHelperPodName, data.TestNamespace, updateRecordsCommand, 60, 3)
	require.True(t, ok, "executing a command on MSSQL helper pod should work; Output: %s, ErrorOutput: %s, Error: %s", out, errOut, err)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, data.DeploymentName, data.TestNamespace, data.MinReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", data.MinReplicaCount)
}

func newTemplateData(authCase authTestCase) templateData {
	testCaseName := fmt.Sprintf("%s-%s", testName, authCase.Name)

	deploymentName := fmt.Sprintf("%s-deployment", testCaseName)
	scaledObjectName := fmt.Sprintf("%s-so", testCaseName)
	triggerAuthenticationName := fmt.Sprintf("%s-ta", testCaseName)
	secretName := fmt.Sprintf("%s-secret", testCaseName)

	return templateData{
		TestNamespace:              testNamespace,
		DeploymentName:             deploymentName,
		ScaledObjectName:           scaledObjectName,
		TriggerAuthenticationName:  triggerAuthenticationName,
		SecretName:                 secretName,
		MssqlServerName:            mssqlHelperStatefulSetName,
		MssqlPassword:              localMssqlPassword,
		AzureMssqlAdminUsername:    azureMssqlAdminUsername,
		AzureMssqlAdminPassword:    azureMssqlAdminPassword,
		AzureMssqlFQDN:             azureMssqlFQDN,
		AzureMssqlDatabase:         azureMssqlDatabase,
		AzureMssqlUamiName:         azureMssqlUamiName,
		AzureMssqlConnectionString: azureMssqlConnectionString,
		DriverName:                 "azuresql",
		MinReplicaCount:            minReplicaCount,
		MaxReplicaCount:            maxReplicaCount,
	}
}

func getMssqlTemplateData(data templateData) (templateData, []Template) {
	return data, []Template{
		{Name: "mssqlStatefulSetTemplate", Config: mssql.MssqlStatefulSetTemplate},
		{Name: "mssqlServiceTemplate", Config: mssql.MssqlServiceTemplate},
	}
}

func getTemplateData(data templateData, authCase authTestCase) (templateData, []Template) {
	return data, []Template{
		{Name: "azureSecretTemplate", Config: azureSecretTemplate},
		{Name: "deploymentTemplate", Config: mssql.DeploymentTemplate},
		{Name: authCase.TriggerAuthenticationName, Config: authCase.TriggerAuthenticationConfig},
		{Name: "azureScaledObjectTemplate", Config: azureScaledObjectTemplate},
	}
}

func GetAzureConnectionString(username string, password string, fqdn string, database string) string {
	return fmt.Sprintf(
		"Server=%s;Database=%s;User ID=%s;Password=%s;Encrypt=true;TrustServerCertificate=false;",
		fqdn, database, username, password,
	)
}

func requireEnv(t *testing.T, keys ...string) {
	t.Helper()

	for _, key := range keys {
		require.NotEmpty(t, os.Getenv(key), "environment variable %s must be set", key)
	}
}
