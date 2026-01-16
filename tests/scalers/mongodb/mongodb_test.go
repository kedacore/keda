//go:build e2e
// +build e2e

package mongodb_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "mongodb-test"
)

var (
	testNamespace   = fmt.Sprintf("%s-ns", testName)
	secretName      = fmt.Sprintf("%s-secret", testName)
	triggerAuthName = fmt.Sprintf("%s-ta", testName)
	scaledJobName   = fmt.Sprintf("%s-sj", testName)
	mongoNamespace  = "mongo-ns"
	mongoUser       = "test_user"
	mongoPassword   = "test_password"
	mongoDBName     = "test"
	mongoCollection = "test_collection"
)

type templateData struct {
	TestNamespace                string
	MongoNamespace               string
	SecretName                   string
	TriggerAuthName              string
	ScaledJobName                string
	Connection, Base64Connection string
	Database, Collection         string
}

const (
	secretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  connectionString: {{.Base64Connection}}
`

	triggerAuthTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthName}}
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
    - parameter: connectionString
      name: {{.SecretName}}
      key: connectionString
`

	scaledJobTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledJob
metadata:
  name: {{.ScaledJobName}}
  namespace: {{.TestNamespace}}
spec:
  jobTargetRef:
    template:
      spec:
        containers:
          - name: mongodb-update
            image: ghcr.io/kedacore/tests-mongodb:latest
            args:
            - --connectStr={{.Connection}}
            - --dataBase={{.Database}}
            - --collection={{.Collection}}
            imagePullPolicy: IfNotPresent
        restartPolicy: Never
    backoffLimit: 1
  pollingInterval: 20
  successfulJobsHistoryLimit: 0
  failedJobsHistoryLimit: 10
  triggers:
    - type: mongodb
      metadata:
        dbName: {{.Database}}
        collection: {{.Collection}}
        query: '{"region":"eu-1","state":"running","plan":"planA"}'
        queryValue: "1"
        activationQueryValue: "4"
      authenticationRef:
        name: {{.TriggerAuthName}}
`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	t.Cleanup(func() {
		cleanupMongo(t)
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})
	mongoPod := setupMongo(t, kc)
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForScaledJobCount(t, kc, scaledJobName, testNamespace, 0, 60, 1),
		"job count should be 0 after 1 minute")

	// test scaling
	testActivation(t, kc, mongoPod)
	testScaleOut(t, kc, mongoPod)
}

func getTemplateData() (templateData, []Template) {
	connectionString := fmt.Sprintf("mongodb://%s:%s@mongodb.%s.svc.cluster.local:27017/%s",
		mongoUser, mongoPassword, mongoNamespace, mongoDBName)
	base64ConnectionString := base64.StdEncoding.EncodeToString([]byte(connectionString))

	return templateData{
			TestNamespace:    testNamespace,
			SecretName:       secretName,
			TriggerAuthName:  triggerAuthName,
			ScaledJobName:    scaledJobName,
			MongoNamespace:   mongoNamespace,
			Database:         mongoDBName,
			Collection:       mongoCollection,
			Connection:       connectionString,
			Base64Connection: base64ConnectionString,
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "triggerAuthTemplate", Config: triggerAuthTemplate},
			{Name: "scaledJobTemplate", Config: scaledJobTemplate},
		}
}

func setupMongo(t *testing.T, kc *kubernetes.Clientset) string {
	CreateNamespace(t, kc, mongoNamespace)
	_, err := ExecuteCommand(fmt.Sprintf("helm install mongodb --set auth.enabled=false --set persistence.enabled=false --namespace %s oci://registry-1.docker.io/cloudpirates/mongodb --wait", mongoNamespace))
	require.NoErrorf(t, err, "cannot execute command - %s", err)

	podList, err := kc.CoreV1().Pods(mongoNamespace).List(context.Background(), metav1.ListOptions{})
	require.NoErrorf(t, err, "cannot get mongo pod - %s", err)

	if len(podList.Items) != 1 {
		t.Error("cannot get mongo pod name")
		return ""
	}

	mongoPod := podList.Items[0].Name

	createUserCmd := fmt.Sprintf("db.createUser({ user:\"%s\",pwd:\"%s\",roles:[{ role:\"readWrite\", db: \"%s\"}]})",
		mongoUser, mongoPassword, mongoDBName)
	_, err = ExecuteCommand(fmt.Sprintf("kubectl exec %s -n %s -- mongosh --eval '%s'", mongoPod, mongoNamespace, createUserCmd))
	require.NoErrorf(t, err, "cannot create user - %s", err)

	loginCmd := fmt.Sprintf("db.auth(\"%s\",\"%s\")", mongoUser, mongoPassword)
	_, err = ExecuteCommand(fmt.Sprintf("kubectl exec %s -n %s -- mongosh --eval '%s'", mongoPod, mongoNamespace, loginCmd))
	require.NoErrorf(t, err, "cannot login - %s", err)

	return mongoPod
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, mongoPod string) {
	t.Log("--- testing activation ---")

	insertCmd := fmt.Sprintf(`db.%s.insert([
		{"region":"eu-1","state":"running","plan":"planA","goods":"apple"},
		{"region":"eu-1","state":"running","plan":"planA","goods":"orange"}
		])`, mongoCollection)

	_, err := ExecuteCommand(fmt.Sprintf("kubectl exec %s -n %s -- mongosh --eval '%s'", mongoPod, mongoNamespace, insertCmd))
	assert.NoErrorf(t, err, "cannot insert mongo records - %s", err)
	time.Sleep(time.Second * 60)
	assert.True(t, WaitForScaledJobCount(t, kc, scaledJobName, testNamespace, 0, 60, 1),
		"job count should be 0 after 1 minute")
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, mongoPod string) {
	t.Log("--- testing scale out ---")

	insertCmd := fmt.Sprintf(`db.%s.insert([
		{"region":"eu-1","state":"running","plan":"planA","goods":"strawberry"},
		{"region":"eu-1","state":"running","plan":"planA","goods":"cherry"},
		{"region":"eu-1","state":"running","plan":"planA","goods":"pineapple"}
		])`, mongoCollection)

	_, err := ExecuteCommand(fmt.Sprintf("kubectl exec %s -n %s -- mongosh --eval '%s'", mongoPod, mongoNamespace, insertCmd))
	assert.NoErrorf(t, err, "cannot insert mongo records - %s", err)

	assert.True(t, WaitForScaledJobCount(t, kc, scaledJobName, testNamespace, 5, 60, 1),
		"job count should be 5 after 1 minute")
}

func cleanupMongo(t *testing.T) {
	_, err := ExecuteCommand(fmt.Sprintf("helm uninstall mongodb --namespace %s", mongoNamespace))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
	DeleteNamespace(t, mongoNamespace)
}
