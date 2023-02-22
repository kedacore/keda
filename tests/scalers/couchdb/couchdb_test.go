//go:build e2e
// +build e2e

package couchdb_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "couchdb-test"
)

var (
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	secretName       = fmt.Sprintf("%s-secret", testName)
	triggerAuthName  = fmt.Sprintf("%s-ta", testName)
	scaledObjectName = fmt.Sprintf("%s-sj", testName)
	couchdbNamespace = "couchdb-ns"
	couchdbUser      = "admin"
	couchdbHelmRepo  = "https://apache.github.io/couchdb-helm"
	couchdbDBName    = "animals"
	minReplicaCount  = 1
	maxReplicaCount  = 2
)

type templateData struct {
	TestNamespace                string
	DeploymentName               string
	HostName                     string
	Port                         string
	Username                     string
	Password                     string
	CouchDBNamespace             string
	SecretName                   string
	TriggerAuthName              string
	ScaledObjectName             string
	Connection, Base64Connection string
	Database                     string
}

const (
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
      - name: nginx
        image: nginxinc/nginx-unprivileged
`

	secretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  connectionString: {{.Base64Connection}}
  username: {{.Username}}
  password: {{.Password}}
`
	triggerAuthTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthName}}
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
  - parameter: password
    name: {{.SecretName}}
    key: password
  - parameter: username
    name: {{.SecretName}}
    key: username
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
  minReplicaCount: 1
  maxReplicaCount: 2
  triggers:
  - type: couchdb
    metadata:
      host: {{.HostName}}
      port: "5984"
      dbName: "animals"
      queryValue: "1"
      query: '{ "selector": { "feet": { "$gt": 0 } }, "fields": ["_id", "feet", "greeting"] }'
      activationQueryValue: "1"
      metricName: "global-metric"
    authenticationRef:
      name: {{.TriggerAuthName}}
`
)

func CreateKubernetesResourcesCouchDB(t *testing.T, kc *kubernetes.Clientset, nsName string, data interface{}, templates []Template) {
	KubectlApplyMultipleWithTemplate(t, data, templates)
}

func TestCouchDBScaler(t *testing.T) {
	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	// setup couchdb
	CreateNamespace(t, kc, testNamespace)
	installCouchDB(t)

	data, templates := getTemplateData(kc)
	CreateKubernetesResourcesCouchDB(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)

	// test scaling
	testActivation(t, kc)
	testScaleUp(t, kc)
	testScaleDown(t, kc)

	// cleanup
	DeleteKubernetesResources(t, kc, testNamespace, data, templates)
}

func generateUUID() string {
	id := strings.ReplaceAll(uuid.New().String(), "-", "")
	return id
}

func installCouchDB(t *testing.T) {
	_, err := ExecuteCommand(fmt.Sprintf("helm repo add couchdb %s", couchdbHelmRepo))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
	_, err = ExecuteCommand("helm repo update")
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
	uuid := generateUUID()
	_, err = ExecuteCommand(fmt.Sprintf("helm install test-release  --set couchdbConfig.couchdb.uuid=%s --namespace %s couchdb/couchdb", uuid, testNamespace))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
}

func getPassword(kc *kubernetes.Clientset, namespace string) string {
	secret, _ := kc.CoreV1().Secrets(namespace).Get(context.Background(), "test-release-couchdb", metav1.GetOptions{})
	encodedPassword := secret.Data["adminPassword"]
	password := string(encodedPassword)
	return password
}

func WaitForDeploymentReplicaCountChangeCouchDB(t *testing.T, kc *kubernetes.Clientset, name, namespace string, iterations, intervalSeconds int) bool {
	t.Log("Waiting for some time to see if deployment replica count changes")
	var replicas, prevReplicas int32
	prevReplicas = -1

	for i := 0; i < iterations; i++ {
		deployment, _ := kc.AppsV1().Deployments(namespace).Get(context.Background(), name, metav1.GetOptions{})
		replicas = deployment.Status.Replicas

		t.Logf("Deployment - %s, Current  - %d", name, replicas)

		if replicas != prevReplicas && prevReplicas != -1 {
			break
		}

		prevReplicas = replicas
		time.Sleep(time.Duration(intervalSeconds) * time.Second)
	}

	return int(replicas) > 1
}

func deployPod(t *testing.T, kc *kubernetes.Clientset, podName, args string) {
	podSpec := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod-couchdb-" + podName,
			Namespace: testNamespace,
		},
		Spec: corev1.PodSpec{
			RestartPolicy: "OnFailure",
			Containers: []corev1.Container{
				{
					Name:    "test-couchdb-container",
					Image:   "nginxinc/nginx-unprivileged",
					Command: []string{"curl"},
					Args:    strings.Split(args, " "),
				},
			},
		},
	}

	_, err := kc.CoreV1().Pods(testNamespace).Create(context.Background(), podSpec, metav1.CreateOptions{})
	if err != nil {
		assert.NoErrorf(t, err, "cannot create pod - %s", err)
	}
}

func testActivation(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing activation ---")
	password := getPassword(kc, testNamespace)
	if WaitForStatefulsetReplicaReadyCount(t, kc, "test-release-couchdb", testNamespace, 3, 15, 1) {
		dsn := fmt.Sprintf("http://admin:%s@test-release-svc-couchdb.%s.svc.cluster.local:5984/animals", password, testNamespace)
		cmd := "-vX PUT " + dsn
		deployPod(t, kc, "activation", cmd)
		AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 5)
	}
}

func insertRecord(t *testing.T, kc *kubernetes.Clientset, password, record, name, uuid string) {
	dsn := fmt.Sprintf(`http://admin:%s@test-release-svc-couchdb.%s.svc.cluster.local:5984/animals/%s`, password, testNamespace, uuid)
	cmd := ` -X PUT ` + dsn + ` -d ` + record
	deployPod(t, kc, name, cmd)
}

func deployPodDelete(t *testing.T, kc *kubernetes.Clientset, podName, args string) {
	podSpec := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod-couchdb-" + podName,
			Namespace: testNamespace,
		},
		Spec: corev1.PodSpec{
			RestartPolicy: "OnFailure",
			Containers: []corev1.Container{
				{
					Name:    "test-couchdb-container",
					Image:   "nginxinc/nginx-unprivileged",
					Command: []string{"curl"},
					Args:    strings.Split(args, " "),
				},
			},
		},
	}
	_, err := kc.CoreV1().Pods(testNamespace).Create(context.Background(), podSpec, metav1.CreateOptions{})
	if err != nil {
		assert.NoErrorf(t, err, "couldn't create pod - %s", err)
	}
}

func deleteRecord(t *testing.T, kc *kubernetes.Clientset, password, name, uuid, rev string) {
	dsn := fmt.Sprintf("http://admin:%s@test-release-svc-couchdb.%s.svc.cluster.local:5984/animals/%s?rev=%s", password, testNamespace, uuid, rev)
	cmd := " -X DELETE " + dsn
	deployPodDelete(t, kc, name, cmd)
}

type PodLog struct {
	OK  bool   `json:"ok"`
	ID  string `json:"id"`
	Rev string `json:"rev"`
}

func getPodLogs(kc *kubernetes.Clientset, podName string) (string, error) {
	podLogOpts := corev1.PodLogOptions{}
	req := kc.CoreV1().Pods(testNamespace).GetLogs(podName, &podLogOpts)
	podLogs, err := req.Stream(context.Background())
	if err != nil {
		return "", err
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", err
	}
	str := buf.String()
	chars := str[strings.Index(str, `"rev":`)+7 : strings.Index(str, `}`)-1]
	return chars, nil
}

func testScaleUp(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale up ---")
	password := getPassword(kc, testNamespace)
	if WaitForStatefulsetReplicaReadyCount(t, kc, "test-release-couchdb", testNamespace, 3, 1, 1) {
		record := `{
			"_id":"Cow",
			"feet":4,
			"greeting":"moo"
		}`
		insertRecord(t, kc, password, record, "scaleup-1", "e488df68180d4c759d38bcf0881faca1")
		time.Sleep(time.Second * 60)
		_, err := getPodLogs(kc, "test-pod-couchdb-scaleup-1")
		if err != nil {
			assert.NoErrorf(t, err, "couldn't get pods logs for scaleup-1 - %s", err)
		}
		record = `{
			"_id":"Cat",
			"feet":4,
			"greeting":"meow"
		}`
		insertRecord(t, kc, password, record, "scaleup-2", "818bba66323b40bf83f42c04374cab23")
		time.Sleep(time.Second * 60)
		_, err = getPodLogs(kc, "test-pod-couchdb-scaleup-2")
		if err != nil {
			assert.NoErrorf(t, err, "couldn't get pods logs for scaleup-2 - %s", err)
		}
		assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 1),
			"replica count should be %d after 1 minute", maxReplicaCount)
		time.Sleep(time.Second * 60)
	}
}

func testScaleDown(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale down ---")
	password := getPassword(kc, testNamespace)
	if WaitForStatefulsetReplicaReadyCount(t, kc, "test-release-couchdb", testNamespace, 3, 1, 1) {
		rev1, err := getPodLogs(kc, "test-pod-couchdb-scaleup-1")
		if err != nil {
			assert.NoErrorf(t, err, "couldn't get pods logs for scaleup-1 - %s", err)
		}
		deleteRecord(t, kc, password, "scaledown-1", "e488df68180d4c759d38bcf0881faca1", rev1)
		rev2, err := getPodLogs(kc, "test-pod-couchdb-scaleup-2")
		if err != nil {
			assert.NoErrorf(t, err, "couldn't get pods logs for scaleup-1 - %s", err)
		}
		deleteRecord(t, kc, password, "scaledown-2", "818bba66323b40bf83f42c04374cab23", rev2)
		assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 300, 1),
			"replica count should be %d after 5 minutes", minReplicaCount)
		time.Sleep(time.Second * 60)
	}
}

func getTemplateData(kc *kubernetes.Clientset) (templateData, []Template) {
	password := getPassword(kc, testNamespace)
	passwordEncoded := base64.StdEncoding.EncodeToString([]byte(password))
	connectionString := fmt.Sprintf("http://test-release-svc-couchdb.%s.svc.cluster.local:5984", testNamespace)
	hostName := fmt.Sprintf("test-release-svc-couchdb.%s.svc.cluster.local", testNamespace)
	base64ConnectionString := base64.StdEncoding.EncodeToString([]byte(connectionString))
	return templateData{
			TestNamespace:    testNamespace,
			DeploymentName:   deploymentName,
			HostName:         hostName,
			Port:             "5984",
			Username:         base64.StdEncoding.EncodeToString([]byte(couchdbUser)),
			Password:         passwordEncoded,
			TriggerAuthName:  triggerAuthName,
			SecretName:       secretName,
			ScaledObjectName: scaledObjectName,
			CouchDBNamespace: couchdbNamespace,
			Database:         couchdbDBName,
			Connection:       connectionString,
			Base64Connection: base64ConnectionString,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "triggerAuthTemplate", Config: triggerAuthTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}
