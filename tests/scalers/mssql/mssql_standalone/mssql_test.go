//go:build e2e
// +build e2e

package mssql_standalone_test

import (
	"fmt"
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

var mssqlDrivers = []string{"sqlserver", "azuresql"}

type templateData struct {
	TestNamespace             string
	DeploymentName            string
	ScaledObjectName          string
	TriggerAuthenticationName string
	SecretName                string
	MssqlServerName           string
	MssqlHostname             string
	MssqlPassword             string
	MssqlDatabase             string
	MssqlConnectionString     string
	DriverName                string
	MinReplicaCount           int
	MaxReplicaCount           int
}

func TestMssqlScaler(t *testing.T) {
	for _, driver := range mssqlDrivers {
		t.Run(driver, func(t *testing.T) {
			testMssqlScalerDriver(t, driver)
		})
	}
}

func testMssqlScalerDriver(t *testing.T, driverName string) {
	data := newTemplateData(driverName)

	// Create kubernetes resources for MS SQL server
	kc := GetKubernetesClient(t)
	_, mssqlTemplates := getMssqlTemplateData(data)
	_, templates := getTemplateData(data)

	allTemplates := append([]Template{}, mssqlTemplates...)
	allTemplates = append(allTemplates, templates...)

	t.Cleanup(func() {
		DeleteKubernetesResources(t, data.TestNamespace, data, allTemplates)
	})

	CreateKubernetesResources(t, kc, data.TestNamespace, data, mssqlTemplates)

	require.True(t, WaitForStatefulsetReplicaReadyCount(t, kc, data.MssqlServerName, data.TestNamespace, 1, 60, 3),
		"replica count should be %d after 3 minutes", 1)

	createDatabaseCommand := fmt.Sprintf("/opt/mssql-tools18/bin/sqlcmd -S . -C -U sa -P \"%s\" -Q \"CREATE DATABASE [%s]\"", data.MssqlPassword, data.MssqlDatabase)

	mssqlServerPodName := fmt.Sprintf("%s-0", data.MssqlServerName)

	ok, out, errOut, err := WaitForSuccessfulExecCommandOnSpecificPod(t, mssqlServerPodName, data.TestNamespace, createDatabaseCommand, 60, 3)
	require.True(t, ok, "executing a command on MS SQL Pod should work; Output: %s, ErrorOutput: %s, Error: %s", out, errOut, err)

	createTableCommand := fmt.Sprintf("/opt/mssql-tools18/bin/sqlcmd -S . -C -U sa -P \"%s\" -d %s -Q \"CREATE TABLE tasks ([id] int identity primary key, [status] varchar(10))\"",
		data.MssqlPassword, data.MssqlDatabase)
	ok, out, errOut, err = WaitForSuccessfulExecCommandOnSpecificPod(t, mssqlServerPodName, data.TestNamespace, createTableCommand, 60, 3)
	require.True(t, ok, "executing a command on MS SQL Pod should work; Output: %s, ErrorOutput: %s, Error: %s", out, errOut, err)

	// Create kubernetes resources for testing
	KubectlApplyMultipleWithTemplate(t, data, templates)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, data.DeploymentName, data.TestNamespace, data.MinReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", data.MinReplicaCount)

	testActivation(t, kc, data)
	testScaleOut(t, kc, data)
	testScaleIn(t, kc, data)
}

// insert 10 records in the table -> activation should not happen (activationTargetValue = 15)
func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")
	KubectlReplaceWithTemplate(t, data, "insertRecordsJobTemplate1", mssql.InsertRecordsJobTemplate1)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, data.DeploymentName, data.TestNamespace, data.MinReplicaCount, 60)
}

// insert another 10 records in the table, which in total is 20 -> should be scaled up
func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	KubectlReplaceWithTemplate(t, data, "insertRecordsJobTemplate2", mssql.InsertRecordsJobTemplate2)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, data.DeploymentName, data.TestNamespace, data.MaxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", data.MaxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale in ---")

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, data.DeploymentName, data.TestNamespace, data.MinReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", data.MinReplicaCount)
}

func newTemplateData(driverName string) templateData {
	testName := fmt.Sprintf("mssql-%s-test", driverName)

	testNamespace := fmt.Sprintf("%s-ns", testName)
	deploymentName := fmt.Sprintf("%s-deployment", testName)
	scaledObjectName := fmt.Sprintf("%s-so", testName)
	triggerAuthenticationName := fmt.Sprintf("%s-ta", testName)
	secretName := fmt.Sprintf("%s-secret", testName)
	mssqlServerName := fmt.Sprintf("%s-server", testName)
	mssqlPassword := "Pass@word1"
	mssqlDatabase := "TestDB"
	mssqlHostname := fmt.Sprintf("%s.%s.svc.cluster.local", mssqlServerName, testNamespace)
	mssqlConnectionString := fmt.Sprintf(
		"Server=%s;Database=%s;User ID=sa;Password=%s;",
		mssqlHostname, mssqlDatabase, mssqlPassword,
	)

	return templateData{
		TestNamespace:             testNamespace,
		DeploymentName:            deploymentName,
		ScaledObjectName:          scaledObjectName,
		TriggerAuthenticationName: triggerAuthenticationName,
		SecretName:                secretName,
		MssqlServerName:           mssqlServerName,
		MssqlHostname:             mssqlHostname,
		MssqlPassword:             mssqlPassword,
		MssqlDatabase:             mssqlDatabase,
		MssqlConnectionString:     mssqlConnectionString,
		DriverName:                driverName,
		MinReplicaCount:           0,
		MaxReplicaCount:           5,
	}
}

func getMssqlTemplateData(data templateData) (templateData, []Template) {
	return data, []Template{
		{Name: "mssqlStatefulSetTemplate", Config: mssql.MssqlStatefulSetTemplate},
		{Name: "mssqlServiceTemplate", Config: mssql.MssqlServiceTemplate},
	}
}

func getTemplateData(data templateData) (templateData, []Template) {
	return data, []Template{
		{Name: "secretTemplate", Config: mssql.SecretTemplate},
		{Name: "deploymentTemplate", Config: mssql.DeploymentTemplate},
		{Name: "triggerAuthenticationTemplate", Config: mssql.TriggerAuthenticationTemplate},
		{Name: "scaledObjectTemplate", Config: mssql.ScaledObjectTemplate},
	}
}
