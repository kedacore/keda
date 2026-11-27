//go:build e2e
// +build e2e

package aws_rds_iam_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

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
	testName = "aws-rds-iam-postgresql-test"

	// rdsCABundleURL serves the CA bundle for all AWS Regions
	// https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/UsingWithRDS.SSL.html
	rdsCABundleURL = "https://truststore.pki.rds.amazonaws.com/global/global-bundle.pem"
)

var (
	testNamespace             = fmt.Sprintf("%s-ns", testName)
	deploymentName            = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName          = fmt.Sprintf("%s-so", testName)
	triggerAuthenticationName = fmt.Sprintf("%s-ta", testName)
	secretName                = fmt.Sprintf("%s-secret", testName)
	caSecretName              = fmt.Sprintf("%s-ca", testName)
	postgreSQLStatefulSetName = "aws-rds-postgresql-client"
	postgresqlPodName         = fmt.Sprintf("%s-0", postgreSQLStatefulSetName)
	awsRegion                 = os.Getenv("TF_AWS_REGION")
	rdsHost                   = os.Getenv("TF_AWS_RDS_POSTGRES_HOST")
	rdsDatabase               = os.Getenv("TF_AWS_RDS_POSTGRES_DB_NAME")
	rdsAdminUsername          = os.Getenv("TF_AWS_RDS_POSTGRES_ADMIN_USERNAME")
	rdsAdminPassword          = os.Getenv("TF_AWS_RDS_POSTGRES_ADMIN_PASSWORD")
	rdsIAMUsername            = os.Getenv("TF_AWS_RDS_POSTGRES_IAM_USERNAME") // The IAM role must be allowed rds-db:connect for this user only
	rdsIAMRoleArn             = os.Getenv("TF_AWS_RDS_POSTGRES_IAM_ROLE")
	minReplicaCount           = 0
	maxReplicaCount           = 2
)

type templateData struct {
	TestNamespace             string
	DeploymentName            string
	ScaledObjectName          string
	TriggerAuthenticationName string
	SecretName                string
	CASecretName              string
	CABundleBase64            string
	PostgreSQLImage           string
	PostgreSQLStatefulSetName string
	PostgreSQLUsername        string
	PostgreSQLPassword        string
	PostgreSQLDatabase        string
	RDSConnectionStringBase64 string
	RDSHost                   string
	RDSDatabase               string
	RDSIAMUsername            string
	RDSIAMRoleArn             string
	AwsRegion                 string
	MinReplicaCount           int
	MaxReplicaCount           int
}

const (
	// The workload and the jobs that generate load connect with the admin user, only KEDA uses IAM authentication
	secretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
type: Opaque
data:
  postgresql_conn_str: {{.RDSConnectionStringBase64}}
`

	caSecretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.CASecretName}}
  namespace: {{.TestNamespace}}
type: Opaque
data:
  ca.pem: {{.CABundleBase64}}
`

	triggerAuthTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthenticationName}}
  namespace: {{.TestNamespace}}
spec:
  podIdentity:
    provider: aws
    roleArn: {{.RDSIAMRoleArn}}
  secretTargetRef:
  - parameter: ca
    name: {{.CASecretName}}
    key: ca.pem
`

	scaledObjectTemplate = `apiVersion: keda.sh/v1alpha1
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
      host: {{.RDSHost}}
      port: "5432"
      userName: {{.RDSIAMUsername}}
      dbName: {{.RDSDatabase}}
      sslmode: verify-full
      awsRegion: {{.AwsRegion}}
      targetQueryValue: "4"
      activationTargetQueryValue: "5"
      query: "SELECT CEIL(COUNT(*) / 5) FROM task_instance WHERE state IN ('running', 'queued');"
    authenticationRef:
      name: {{.TriggerAuthenticationName}}
`
)

func TestPostgreSQLScalerWithAwsRDSIAM(t *testing.T) {
	require.NotEmpty(t, awsRegion, "TF_AWS_REGION env variable is required for AWS RDS IAM tests")
	require.NotEmpty(t, rdsHost, "TF_AWS_RDS_POSTGRES_HOST env variable is required for AWS RDS IAM tests")
	require.NotEmpty(t, rdsDatabase, "TF_AWS_RDS_POSTGRES_DB_NAME env variable is required for AWS RDS IAM tests")
	require.NotEmpty(t, rdsAdminUsername, "TF_AWS_RDS_POSTGRES_ADMIN_USERNAME env variable is required for AWS RDS IAM tests")
	require.NotEmpty(t, rdsAdminPassword, "TF_AWS_RDS_POSTGRES_ADMIN_PASSWORD env variable is required for AWS RDS IAM tests")
	require.NotEmpty(t, rdsIAMUsername, "TF_AWS_RDS_POSTGRES_IAM_USERNAME env variable is required for AWS RDS IAM tests")
	require.NotEmpty(t, rdsIAMRoleArn, "TF_AWS_RDS_POSTGRES_IAM_ROLE env variable is required for AWS RDS IAM tests")

	kc := GetKubernetesClient(t)
	data, clientTemplates, templates := getTemplateData(t)
	t.Cleanup(func() {
		// Deferred so the namespace is removed even if the SQL cleanup fails
		defer DeleteKubernetesResources(t, testNamespace, data, clientTemplates)

		// Remove the ScaledObject first so KEDA stops connecting as the IAM user before it's dropped
		KubectlDeleteMultipleWithTemplate(t, data, templates)
		execAdminSQL(t, fmt.Sprintf(`DROP TABLE IF EXISTS task_instance; DROP ROLE IF EXISTS \"%s\";`, rdsIAMUsername))
	})

	// The in-cluster PostgreSQL pod is only used as a psql client for the RDS instance
	CreateKubernetesResources(t, kc, testNamespace, data, clientTemplates)
	require.True(t, WaitForStatefulsetReplicaReadyCount(t, kc, postgreSQLStatefulSetName, testNamespace, 1, 60, 3),
		"replica count should be %d after 3 minutes", 1)

	// Create the table and a read-only database user that can only log in with IAM authentication
	execAdminSQL(t, fmt.Sprintf(`DROP TABLE IF EXISTS task_instance; DROP ROLE IF EXISTS \"%[1]s\"; `+
		`CREATE ROLE \"%[1]s\" WITH LOGIN; GRANT rds_iam TO \"%[1]s\"; `+
		`CREATE TABLE task_instance (id serial PRIMARY KEY, state VARCHAR(10)); `+
		`GRANT SELECT ON task_instance TO \"%[1]s\";`, rdsIAMUsername))

	KubectlApplyMultipleWithTemplate(t, data, templates)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)

	testActivation(t, kc, data)
	testScaleOut(t, kc, data)
	testScaleIn(t, kc, data)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")
	KubectlReplaceWithTemplate(t, data, "lowLevelRecordsJobTemplate", pg.LowLevelRecordsJobTemplate)
	// Wait for the records to be inserted, otherwise the check below could pass against an empty table
	require.True(t, WaitForJobSuccess(t, kc, "postgresql-insert-low-level-job", testNamespace, 60, 3),
		"low level records job should succeed after 3 minutes")

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	KubectlReplaceWithTemplate(t, data, "insertRecordsJobTemplate", pg.InsertRecordsJobTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale in ---")

	// Similar to the Azure test, an external database is slower than an in-cluster one
	// Remove the load and mark every record as processed instead of waiting for the workload
	KubectlDeleteWithTemplate(t, data, "lowLevelRecordsJobTemplate", pg.LowLevelRecordsJobTemplate)
	KubectlDeleteWithTemplate(t, data, "insertRecordsJobTemplate", pg.InsertRecordsJobTemplate)
	execAdminSQL(t, "UPDATE task_instance SET state = 'processed';")

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
}

// execAdminSQL runs SQL against the RDS instance as the admin user from the psql client pod
// Double quotes inside the SQL must be escaped
func execAdminSQL(t *testing.T, sql string) {
	t.Helper()
	ok, out, errOut, err := WaitForSuccessfulExecCommandOnSpecificPod(t, postgresqlPodName, testNamespace,
		fmt.Sprintf(`PGPASSWORD='%s' PGSSLMODE=require psql -v ON_ERROR_STOP=1 -h %s -p 5432 -U %s -d %s -c "%s"`,
			rdsAdminPassword, rdsHost, rdsAdminUsername, rdsDatabase, sql), 60, 3)
	require.True(t, ok, "executing a command on PostgreSQL Pod should work; Output: %s, ErrorOutput: %s, Error: %s", out, errOut, err)
}

// fetchRDSCABundle downloads the RDS CA bundle that KEDA uses to verify the server certificate
func fetchRDSCABundle(t *testing.T) []byte {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rdsCABundleURL, nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err, "downloading the RDS CA bundle should work")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "downloading the RDS CA bundle should work")

	bundle, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NotEmpty(t, bundle, "the RDS CA bundle should not be empty")
	return bundle
}

func getTemplateData(t *testing.T) (templateData, []Template, []Template) {
	connectionString := url.URL{
		Scheme:   "postgresql",
		User:     url.UserPassword(rdsAdminUsername, rdsAdminPassword),
		Host:     net.JoinHostPort(rdsHost, "5432"),
		Path:     rdsDatabase,
		RawQuery: "sslmode=require",
	}

	data := templateData{
		TestNamespace:             testNamespace,
		DeploymentName:            deploymentName,
		ScaledObjectName:          scaledObjectName,
		TriggerAuthenticationName: triggerAuthenticationName,
		SecretName:                secretName,
		CASecretName:              caSecretName,
		CABundleBase64:            base64.StdEncoding.EncodeToString(fetchRDSCABundle(t)),
		PostgreSQLImage:           pg.PostgresqlImage,
		PostgreSQLStatefulSetName: postgreSQLStatefulSetName,
		PostgreSQLUsername:        "test-user",
		PostgreSQLPassword:        "test-password",
		PostgreSQLDatabase:        "test_db",
		RDSConnectionStringBase64: base64.StdEncoding.EncodeToString([]byte(connectionString.String())),
		RDSHost:                   rdsHost,
		RDSDatabase:               rdsDatabase,
		RDSIAMUsername:            rdsIAMUsername,
		RDSIAMRoleArn:             rdsIAMRoleArn,
		AwsRegion:                 awsRegion,
		MinReplicaCount:           minReplicaCount,
		MaxReplicaCount:           maxReplicaCount,
	}

	clientTemplates := []Template{
		{Name: "postgreSQLStatefulSetTemplate", Config: pg.PostgreSQLStatefulSetTemplate},
	}
	templates := []Template{
		{Name: "secretTemplate", Config: secretTemplate},
		{Name: "caSecretTemplate", Config: caSecretTemplate},
		{Name: "deploymentTemplate", Config: pg.DeploymentTemplate},
		{Name: "triggerAuthTemplate", Config: triggerAuthTemplate},
		{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
	}
	return data, clientTemplates, templates
}
