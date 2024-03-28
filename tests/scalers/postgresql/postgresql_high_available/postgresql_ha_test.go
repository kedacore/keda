//go:build e2e
// +build e2e

package postgresql_standalone_test

import (
	"encoding/base64"
	"fmt"
	"strings"
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
	testName = "postgresql-ha-test"
)

var (
	testNamespace                    = fmt.Sprintf("%s-ns", testName)
	deploymentName                   = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName                 = fmt.Sprintf("%s-so", testName)
	triggerAuthenticationName        = fmt.Sprintf("%s-ta", testName)
	secretName                       = fmt.Sprintf("%s-secret", testName)
	secretKey                        = "postgresql_multihost_conn_str"
	postgreSQLImage                  = pg.PostgresqlImage
	postgreSQLStatefulSetName        = "postgresql-master"
	postgreSQLReplicaStatefulSetName = "postgresql-replica"
	postgresqlMasterPodName          = fmt.Sprintf("%s-0", postgreSQLStatefulSetName)
	postgresqlReplicaPodName         = fmt.Sprintf("%s-0", postgreSQLReplicaStatefulSetName)
	postgreSQLUsername               = "test-user"
	postgreSQLPassword               = "test-password"
	postgreSQLDatabase               = "test_db"
	postgreSQLConnectionString       = pg.GetConnectionString(postgreSQLUsername, postgreSQLPassword,
		[]string{postgreSQLStatefulSetName}, testNamespace, postgreSQLDatabase)
	postgreSQLMultihostConnectionString = pg.GetConnectionString(postgreSQLUsername, postgreSQLPassword,
		[]string{postgreSQLStatefulSetName, postgreSQLReplicaStatefulSetName}, testNamespace, postgreSQLDatabase)
	minReplicaCount = 0
	maxReplicaCount = 2
)

type templateData struct {
	TestNamespace                             string
	DeploymentName                            string
	ScaledObjectName                          string
	TriggerAuthenticationName                 string
	SecretName                                string
	SecretKey                                 string
	PostgreSQLImage                           string
	PostgreSQLStatefulSetName                 string
	PostgreSQLReplicaStatefulSetName          string
	PostgreSQLConnectionStringBase64          string
	PostgreSQLMultihostConnectionStringBase64 string
	PostgreSQLUsername                        string
	PostgreSQLPassword                        string
	PostgreSQLDatabase                        string
	MinReplicaCount                           int
	MaxReplicaCount                           int
}

const (
	postgresqlInitScriptsConfigMapTemplate = `apiVersion: v1
kind: ConfigMap
metadata:
  name: master-slave-config
  namespace: {{.TestNamespace}}
data:
  master-slave-config.sh: |-
    HOST=$(hostname -s)
    HOST_TEMPLATE=${HOST%-*}
    case $HOST_TEMPLATE in
      *master*)
      echo "host    replication     all     all     md5" >> /var/lib/postgresql/data/pg_hba.conf
      echo "archive_mode = on"  >> /etc/postgresql/postgresql.conf
      echo "archive_mode = on"  >> /etc/postgresql/postgresql.conf
      echo "archive_command = '/bin/true'"  >> /etc/postgresql/postgresql.conf
      echo "archive_timeout = 0"  >> /etc/postgresql/postgresql.conf
      echo "max_wal_senders = 8"  >> /etc/postgresql/postgresql.conf
      echo "wal_keep_segments = 32"  >> /etc/postgresql/postgresql.conf
      echo "wal_level = hot_standby"  >> /etc/postgresql/postgresql.conf
      ;;
      *replica*)
      # stop initial server to copy data
      pg_ctl -D /var/lib/postgresql/data/ -m fast -w stop
      rm -rf /var/lib/postgresql/data/*
      # add service name for DNS resolution
      PGPASSWORD=postgresql-ha pg_basebackup -h {{.PostgreSQLStatefulSetName}} -w -U replicator -p 5432 -D /var/lib/postgresql/data -Fp -Xs -P -R
      # start server to keep container's screep happy
      pg_ctl -D /var/lib/postgresql/data/ -w start
      ;;
    esac


  create-replication-role.sql: |-
    CREATE USER replicator WITH REPLICATION ENCRYPTED PASSWORD 'postgresql-ha';
`
	secretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
type: Opaque
data:
  postgresql_conn_str: {{.PostgreSQLConnectionStringBase64}}
  postgresql_multihost_conn_str: {{.PostgreSQLMultihostConnectionStringBase64}}
`
)

func TestPostreSQLScaler(t *testing.T) {
	kc := GetKubernetesClient(t)
	_, postgreSQLtemplates := getPostgreSQLTemplateData()
	_, templates := getTemplateData()
	t.Cleanup(func() {
		KubectlDeleteMultipleWithTemplate(t, data, templates)
		DeleteKubernetesResources(t, testNamespace, data, postgreSQLtemplates)
	})
	// Create kubernetes resources for PostgreSQL server
	CreateKubernetesResources(t, kc, testNamespace, data, postgreSQLtemplates)

	require.True(t, WaitForStatefulsetReplicaReadyCount(t, kc, postgreSQLStatefulSetName, testNamespace, 1, 60, 3),
		"replica count should be %d after 3 minutes", 1)
	require.True(t, WaitForStatefulsetReplicaReadyCount(t, kc, postgreSQLReplicaStatefulSetName, testNamespace, 1, 60, 3),
		"replica count should be %d after 3 minutes", 1)

	createTableSQL := "CREATE TABLE task_instance (id serial PRIMARY KEY,state VARCHAR(10));"
	ok, out, errOut, err := WaitForSuccessfulExecCommandOnSpecificPod(t, postgresqlMasterPodName, testNamespace,
		fmt.Sprintf("psql -U %s -d %s -c \"%s\"", postgreSQLUsername, postgreSQLDatabase, createTableSQL), 60, 3)
	require.True(t, ok, "executing a command on PostreSQL Master Pod should work; Output: %s, ErrorOutput: %s, Error: %s", out, errOut, err)

	checkTableExists := "SELECT * from task_instance;"
	ok, out, errOut, err = WaitForSuccessfulExecCommandOnSpecificPod(t, postgresqlReplicaPodName, testNamespace,
		fmt.Sprintf("psql -U %s -d %s -c \"%s\"", postgreSQLUsername, postgreSQLDatabase, checkTableExists), 60, 3)

	require.True(t, ok, "executing a command on PostreSQL Replica Pod should work; Output: %s, ErrorOutput: %s, Error: %s", out, errOut, err)

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
	TestNamespace:                             testNamespace,
	PostgreSQLStatefulSetName:                 postgreSQLStatefulSetName,
	PostgreSQLReplicaStatefulSetName:          postgreSQLReplicaStatefulSetName,
	DeploymentName:                            deploymentName,
	ScaledObjectName:                          scaledObjectName,
	MinReplicaCount:                           minReplicaCount,
	MaxReplicaCount:                           maxReplicaCount,
	TriggerAuthenticationName:                 triggerAuthenticationName,
	SecretName:                                secretName,
	SecretKey:                                 secretKey,
	PostgreSQLImage:                           postgreSQLImage,
	PostgreSQLUsername:                        postgreSQLUsername,
	PostgreSQLPassword:                        postgreSQLPassword,
	PostgreSQLDatabase:                        postgreSQLDatabase,
	PostgreSQLConnectionStringBase64:          base64.StdEncoding.EncodeToString([]byte(postgreSQLConnectionString)),
	PostgreSQLMultihostConnectionStringBase64: base64.StdEncoding.EncodeToString([]byte(postgreSQLMultihostConnectionString)),
}

func getPostgreSQLTemplateData() (templateData, []Template) {
	return data, []Template{
		{Name: "postgresqlInitScriptsConfigMapTemplate", Config: postgresqlInitScriptsConfigMapTemplate},
		{Name: "postgresqlMasterStatefulSetTemplate", Config: pg.PostgreSQLStatefulSetTemplate},
		{Name: "postgresqlReplicaStatefulSetTemplate", Config: postgreSQLReplicaStatefulSetTemplate()},
		{Name: "postgreSQLMasterServiceTemplate", Config: pg.PostgreSQLServiceTemplate},
		{Name: "postgreSQLReplicaServiceTemplate", Config: postgreSQLReplicaServiceTemplate()},
	}
}

func postgreSQLReplicaStatefulSetTemplate() string {
	return strings.ReplaceAll(pg.PostgreSQLStatefulSetTemplate, ".PostgreSQLStatefulSetName", ".PostgreSQLReplicaStatefulSetName")
}

func postgreSQLReplicaServiceTemplate() string {
	return strings.ReplaceAll(pg.PostgreSQLServiceTemplate, ".PostgreSQLStatefulSetName", ".PostgreSQLReplicaStatefulSetName")
}

func getTemplateData() (templateData, []Template) {
	return data, []Template{
		{Name: "secretTemplate", Config: secretTemplate},
		{Name: "deploymentTemplate", Config: pg.DeploymentTemplate},
		{Name: "triggerAuthenticationTemplate", Config: pg.TriggerAuthenticationTemplate},
		{Name: "scaledObjectTemplate", Config: pg.ScaledObjectTemplate},
	}
}
