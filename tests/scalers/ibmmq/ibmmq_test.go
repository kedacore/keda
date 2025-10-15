//go:build e2e
// +build e2e

package ibmmq_test

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
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "ibmmq-test"
)

var (
	ibmmqHelmRepo             = "https://raw.githubusercontent.com/IBM/charts/master/repo/stable"
	ibmmqHelmChartReleaseName = "ibm-mq-dev"
	queueManagerName          = "testqmgr"

	testNamespace    = fmt.Sprintf("%s-ns", testName)
	secretName       = fmt.Sprintf("%s-secret", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
	triggerAuthName  = fmt.Sprintf("%s-ta", testName)
	producerJobName  = fmt.Sprintf("%s-producer-job", testName)

	queueName                           = "DEV.QUEUE.1"
	channelName                         = "DEV.APP.SVRCONN"
	host                                = fmt.Sprintf("%s-ibm-mq.%s.svc", ibmmqHelmChartReleaseName, testNamespace)
	port                                = 1414
	adminUsername                       = "admin"
	adminPassword                       = "admin-passw0rd"
	appUsername                         = "app"
	appPassword                         = "app-passw0rd"
	minReplicaCount                     = 0
	maxReplicaCount                     = 2
	activationQueueDepth                = 5
	MqscAdminRestEndpoint               = fmt.Sprintf("https://%s:9443/ibmmq/rest/v2/admin/action/qmgr/%s/mqsc", host, queueManagerName)
	queueManagerStatusAdminRestEndpoint = fmt.Sprintf("https://%s:9443/ibmmq/rest/v2/admin/qmgr/%s", host, queueManagerName)
)

type templateData struct {
	TestNamespace                       string
	SecretName                          string
	DeploymentName                      string
	ScaledObjectName                    string
	TriggerAuthName                     string
	ProducerJobName                     string
	AdminUsername, Base64AdminUsername  string
	AdminPassword, Base64AdminPassword  string
	Base64AppUsername                   string
	Base64AppPassword                   string
	QueueManagerStatusAdminRestEndpoint string
	MinReplicaCount                     int
	MaxReplicaCount                     int
	ActivationQueueDepth                int
	QueueManagerName                    string
	QueueName                           string
	ChannelName                         string
	Host                                string
	Port                                int
	MqscAdminRestEndpoint               string
	NumberOfMessagesProduced            int
	ProducerSleepTime                   int
	ConsumerSleepTime                   int
}

const (
	checkQueueManagerRunningStatusJobTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  name: check-qmgr-running-status
  namespace: {{.TestNamespace}}
spec:
  template:
    spec:
      containers:
        - name: check-qmgr-running-status
          image: docker.io/curlimages/curl
          command:
            - sh
            - -c
            - |
              while true; do
                response=$(curl -k -u $ADMIN_USERNAME:$ADMIN_PASSWORD -s -w "%{http_code}" -o /tmp/response_body $QUEUE_MANAGER_STATUS_ADMIN_REST_ENDPOINT)
                if [ "$response" -eq 200 ]; then
                  body=$(cat /tmp/response_body)
                  echo "Received HTTP 200 from $QUEUE_MANAGER_STATUS_ADMIN_REST_ENDPOINT"
                  echo "Response body: $body"
                  break
                else
                  echo "Waiting for HTTP 200 from $QUEUE_MANAGER_STATUS_ADMIN_REST_ENDPOINT"
                  echo "Current response: $response"
                  sleep 10
                fi
              done
          env:
            - name: QUEUE_MANAGER_STATUS_ADMIN_REST_ENDPOINT
              value: {{.QueueManagerStatusAdminRestEndpoint}}
            - name: ADMIN_USERNAME
              value: {{.AdminUsername}}
            - name: ADMIN_PASSWORD
              value: {{.AdminPassword}}
      restartPolicy: Never
  backoffLimit: 1
`

	secretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName }}
  namespace: {{.TestNamespace}}
data:
  adminUsername: {{.Base64AdminUsername}}
  adminPassword: {{.Base64AdminPassword}}
  appUsername: {{.Base64AppUsername}}
  appPassword: {{.Base64AppPassword}}
`

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
        - name: {{.DeploymentName}}
          image: ghcr.io/kedacore/tests-ibmmq:latest
          imagePullPolicy: Always
          command:
            - "/app"
          args:
            - "consumer"
          env:
            - name: APP_USERNAME
              valueFrom:
                secretKeyRef:
                  name: {{.SecretName}}
                  key: appUsername
            - name: APP_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: {{.SecretName}}
                  key: appPassword
            - name: QUEUE_MANAGER
              value: {{.QueueManagerName}}
            - name: QUEUE_NAME
              value: {{.QueueName}}
            - name: HOST
              value: {{.Host}}
            - name: PORT
              value: "{{.Port}}"
            - name: CHANNEL
              value: {{.ChannelName}}
            - name: CONSUMER_SLEEP_TIME
              value: "{{.ConsumerSleepTime}}"
`

	scaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
  labels:
    deploymentName: {{.DeploymentName}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  pollingInterval: 5
  cooldownPeriod: 10
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  triggers:
    - type: ibmmq
      metadata:
        queueDepth: "3"
        activationQueueDepth: "{{.ActivationQueueDepth}}"
        host: {{.MqscAdminRestEndpoint}}
        queueName: {{.QueueName}}
        unsafeSsl: "true"
        usernameFromEnv: ""
        passwordFromEnv: ""
      authenticationRef:
        name: {{.TriggerAuthName}}
`

	triggerAuthTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthName}}
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
    - parameter: username
      name: {{.SecretName}}
      key: adminUsername
    - parameter: password
      name: {{.SecretName}}
      key: adminPassword
`

	producerJobTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  name: {{.ProducerJobName}}
  namespace: {{.TestNamespace}}
spec:
  template:
    spec:
      containers:
        - name: {{.ProducerJobName}}
          image: ghcr.io/kedacore/tests-ibmmq:latest
          imagePullPolicy: Always
          command:
            - "/app"
          args:
            - "producer"
          env:
            - name: APP_USERNAME
              valueFrom:
                secretKeyRef:
                  name: {{.SecretName}}
                  key: appUsername
            - name: APP_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: {{.SecretName}}
                  key: appPassword
            - name: QUEUE_MANAGER
              value: {{.QueueManagerName}}
            - name: QUEUE_NAME
              value: {{.QueueName}}
            - name: HOST
              value: {{.Host}}
            - name: PORT
              value: "{{.Port}}"
            - name: CHANNEL
              value: {{.ChannelName}}
            - name: PRODUCER_SLEEP_TIME
              value: "{{.ProducerSleepTime}}"
            - name: NUM_MESSAGES
              value: "{{.NumberOfMessagesProduced}}"
      restartPolicy: Never
  backoffLimit: 1
`
)

func TestScaler(t *testing.T) {
	t.Log("--- setting up ---")
	kc := GetKubernetesClient(t)
	data, templates := getTemplate()

	t.Cleanup(func() {
		uninstallIbmmq(t, data)
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})

	CreateNamespace(t, kc, testNamespace)
	installIbmmq(t, kc, data)

	KubectlApplyMultipleWithTemplate(t, data, templates)

	testActivation(t, kc, data)
	testScaleOut(t, kc, data)
	testScaleIn(t, kc)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")

	data.NumberOfMessagesProduced = activationQueueDepth - 1
	data.ProducerSleepTime = 2
	KubectlApplyWithTemplate(t, data, "producerJobTemplate", producerJobTemplate)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
	assert.True(t, WaitForJobSuccess(t, kc, producerJobName, testNamespace, 1, 0),
		"producer job didn't ran successfully!")
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")

	data.NumberOfMessagesProduced = 50
	data.ProducerSleepTime = 0
	KubectlReplaceWithTemplate(t, data, "producerJobTemplate", producerJobTemplate)

	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 180, 1),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")
	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 180, 1),
		"replica count should be %d after 3 minutes", minReplicaCount)
}

func installIbmmq(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	_, err := ExecuteCommand(fmt.Sprintf("helm repo add ibm-stable-charts %s", ibmmqHelmRepo))
	require.NoErrorf(t, err, "cannot execute command - %s", err)
	_, err = ExecuteCommand("helm repo update")
	require.NoErrorf(t, err, "cannot execute command - %s", err)

	tempDir, err := os.MkdirTemp("", testName)
	require.NoErrorf(t, err, "cannot create temp directory - %s", err)
	defer os.RemoveAll(tempDir)

	_, err = ExecuteCommand(fmt.Sprintf("helm pull ibm-mqadvanced-server-dev --repo %s --untar --untardir %s", ibmmqHelmRepo, tempDir))
	require.NoErrorf(t, err, "cannot execute command - %s", err)
	chartTempDir := fmt.Sprintf("%s/ibm-mqadvanced-server-dev", tempDir)

	// Update deprecated keys for the statefulset `nodeAffinity` object: `beta.kubernetes.io/os` and `beta.kubernetes.io/arch`
	// by removing the `beta` prefix if it exists.
	_, err = ExecuteCommand(fmt.Sprintf("find %s/templates -type f -name '*.yaml' -exec sed -i -e s/beta.kubernetes.io\\/os/kubernetes.io\\/os/g {} ;", chartTempDir))
	require.NoErrorf(t, err, "cannot execute command - %s", err)
	_, err = ExecuteCommand(fmt.Sprintf("find %s/templates -type f -name '*.yaml' -exec sed -i -e s/beta.kubernetes.io\\/arch/kubernetes.io\\/arch/g {} ;", chartTempDir))
	require.NoErrorf(t, err, "cannot execute command - %s", err)

	// Creates the secret that contains the 'app' and 'admin' user passwords
	// which will be referenced by the chart.
	KubectlApplyWithTemplate(t, data, "secretTemplate", secretTemplate)

	t.Logf("installing IBM MQ helm chart '%s'", ibmmqHelmChartReleaseName)
	todebug, err := ExecuteCommand(fmt.Sprintf(
		"helm install %s %s "+
			"--set license=accept "+
			"--set persistence.enabled=false "+
			"--set persistence.useDynamicProvisioning=false "+
			"--set image.tag=9.2.4.0-r1 "+
			"--set queueManager.name=%s "+
			"--set queueManager.multiInstance=false "+
			"--set queueManager.dev.secret.name=%s "+
			"--set queueManager.dev.secret.adminPasswordKey=adminPassword "+
			"--set queueManager.dev.secret.appPasswordKey=appPassword "+
			"--namespace %s --wait --debug",
		ibmmqHelmChartReleaseName, chartTempDir, queueManagerName, secretName, testNamespace))
	// temp for debugging purpose
	t.Log(string(todebug))
	require.NoErrorf(t, err, "cannot execute command - %s", err)

	KubectlApplyWithTemplate(t, data, "checkQueueManagerRunningStatusJobTemplate", checkQueueManagerRunningStatusJobTemplate)
	t.Logf("waiting for the queue manager '%s' to be in a running state", queueManagerName)
	assert.Truef(t, WaitForJobSuccess(t, kc, "check-qmgr-running-status", testNamespace, 60, 10),
		"queue manager '%s' should be in a running state after maximum 10 minutes", queueManagerName)
}

func uninstallIbmmq(t *testing.T, data templateData) {
	t.Logf("uninstalling IBM MQ helm chart '%s'", ibmmqHelmChartReleaseName)
	_, err := ExecuteCommand(fmt.Sprintf("helm uninstall %s --namespace %s", ibmmqHelmChartReleaseName, testNamespace))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)

	KubectlDeleteMultipleWithTemplate(t, data,
		[]Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "checkQueueManagerRunningStatusJobTemplate", Config: checkQueueManagerRunningStatusJobTemplate},
		})
}

func getTemplate() (templateData, []Template) {
	return templateData{
			TestNamespace:                       testNamespace,
			SecretName:                          secretName,
			DeploymentName:                      deploymentName,
			ScaledObjectName:                    scaledObjectName,
			TriggerAuthName:                     triggerAuthName,
			ProducerJobName:                     producerJobName,
			AdminUsername:                       adminUsername,
			AdminPassword:                       adminPassword,
			Base64AdminUsername:                 base64.StdEncoding.EncodeToString([]byte(adminUsername)),
			Base64AdminPassword:                 base64.StdEncoding.EncodeToString([]byte(adminPassword)),
			Base64AppUsername:                   base64.StdEncoding.EncodeToString([]byte(appUsername)),
			Base64AppPassword:                   base64.StdEncoding.EncodeToString([]byte(appPassword)),
			QueueManagerStatusAdminRestEndpoint: queueManagerStatusAdminRestEndpoint,
			MinReplicaCount:                     minReplicaCount,
			MaxReplicaCount:                     maxReplicaCount,
			ActivationQueueDepth:                activationQueueDepth,
			QueueManagerName:                    queueManagerName,
			QueueName:                           queueName,
			ChannelName:                         channelName,
			Host:                                host,
			Port:                                port,
			ConsumerSleepTime:                   1,
			MqscAdminRestEndpoint:               MqscAdminRestEndpoint,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
			{Name: "triggerAuthTemplate", Config: triggerAuthTemplate},
		}
}
