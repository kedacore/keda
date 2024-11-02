//go:build e2e
// +build e2e

package artemis_test

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "artemis-test"
)

var (
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
	secretName       = fmt.Sprintf("%s-secret", testName)
	artemisUser      = "admin"
	artemisPassword  = "admin"
	minReplicaCount  = 0
	maxReplicaCount  = 2
)

type templateData struct {
	TestNamespace         string
	DeploymentName        string
	ScaledObjectName      string
	SecretName            string
	ArtemisPasswordBase64 string
	ArtemisUserBase64     string
	MessageCount          int
}

const (
	secretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  artemis-password: {{.ArtemisPasswordBase64}}
  artemis-username: {{.ArtemisUserBase64}}
`

	triggerAuthenticationTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: keda-trigger-auth-artemis-secret
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
    - parameter: username
      name: {{.SecretName}}
      key: artemis-username
    - parameter: password
      name: {{.SecretName}}
      key: artemis-password
`

	deploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  selector:
    matchLabels:
      app: kedartemis-consumer
  replicas: 0
  template:
    metadata:
      labels:
        app: kedartemis-consumer
    spec:
      containers:
      - name: kedartemis-consumer
        image: ghcr.io/kedacore/tests-artemis
        args: ["consumer"]
        env:
          - name: ARTEMIS_PASSWORD
            valueFrom:
              secretKeyRef:
                name: {{.SecretName}}
                key: artemis-password
          - name: ARTEMIS_USERNAME
            valueFrom:
              secretKeyRef:
                name: {{.SecretName}}
                key: artemis-username
          - name: ARTEMIS_SERVER_HOST
            value: "artemis-activemq.{{.TestNamespace}}"
          - name: ARTEMIS_SERVER_PORT
            value: "61616"
          - name: ARTEMIS_MESSAGE_SLEEP_MS
            value: "70"
`

	artemisDeploymentTemplate = `apiVersion: apps/v1
apiVersion: apps/v1
kind: Deployment
metadata:
  name: artemis-activemq
  namespace: {{.TestNamespace}}
  labels:
    app: activemq-artemis
spec:
  replicas: 1
  revisionHistoryLimit: 1
  selector:
    matchLabels:
      app: activemq-artemis
  template:
    metadata:
      name: artemis-activemq-artemis
      labels:
        app: activemq-artemis
    spec:
      initContainers:
        - name: configure-cluster
          image: docker.io/vromero/activemq-artemis:2.6.2
          command: ["/bin/sh", "/data/etc-override/configure-cluster.sh"]
          volumeMounts:
            - name: config-override
              mountPath: /var/lib/artemis/etc-override
            - name: configmap-override
              mountPath: /data/etc-override/
      containers:
        - name: artemis-activemq-artemis
          image: docker.io/vromero/activemq-artemis:2.6.2
          imagePullPolicy:
          env:
            - name: ARTEMIS_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: {{.SecretName}}
                  key: artemis-password
            - name: ARTEMIS_USERNAME
              valueFrom:
                secretKeyRef:
                  name: {{.SecretName}}
                  key: artemis-username
            - name: ARTEMIS_PERF_JOURNAL
              value: "AUTO"
            - name: ENABLE_JMX_EXPORTER
              value: "true"
          ports:
            - name: http
              containerPort: 8161
            - name: core
              containerPort: 61616
            - name: amqp
              containerPort: 5672
            - name: jmxexporter
              containerPort: 9404
          livenessProbe:
            tcpSocket:
              port: http
            initialDelaySeconds: 10
            periodSeconds: 10
          readinessProbe:
            tcpSocket:
              port: core
            initialDelaySeconds: 10
            periodSeconds: 10
          volumeMounts:
            - name: data
              mountPath: /var/lib/artemis/data
            - name: config-override
              mountPath: /var/lib/artemis/etc-override
      volumes:
        - name: data
          emptyDir: {}
        - name: config-override
          emptyDir: {}
        - name: configmap-override
          configMap:
            name:  artemis-activemq-cm
`
	artemisServiceTemplate = `apiVersion: v1
kind: Service
metadata:
  name: artemis-activemq
  namespace: {{.TestNamespace}}
spec:
  ports:
    - name: http
      port: 8161
      targetPort: http
    - name: core
      port: 61616
      targetPort: core
    - name: amqp
      port: 5672
      targetPort: amqp
    - name: jmx
      port: 9494
      targetPort: jmxexporter
  selector:
    app: activemq-artemis
`
	artemisConfigTemplate = `apiVersion: v1
kind: ConfigMap
metadata:
  name: artemis-activemq-cm
  namespace: {{.TestNamespace}}
data:
  broker-00.xml: |
    <?xml version="1.0" encoding="UTF-8" standalone="no"?>

    <configuration xmlns="urn:activemq" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:activemq /schema/artemis-configuration.xsd">

      <core xmlns="urn:activemq:core" xsi:schemaLocation="urn:activemq:core ">
         <name>artemis-activemq</name>
         <addresses>
           <address name="test">
           <anycast>
             <queue name="test"/>
           </anycast>
         </address>
        </addresses>
      </core>
    </configuration>

  configure-cluster.sh: |

    set -e
    echo Copying common configuration
    cp /data/etc-override/*.xml /var/lib/artemis/etc-override/broker-10.xml
`

	scaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  minReplicaCount: 0
  maxReplicaCount: 2
  pollingInterval: 1
  cooldownPeriod:  1
  triggers:
  - type: artemis-queue
    metadata:
      managementEndpoint: "artemis-activemq.{{.TestNamespace}}:8161"
      queueName: "test"
      queueLength: "50"
      activationQueueLength: "5"
      brokerName: "artemis-activemq"
      brokerAddress: "test"
    authenticationRef:
      name: keda-trigger-auth-artemis-secret
`

	producerJob = `
apiVersion: batch/v1
kind: Job
metadata:
  name: artemis-producer
  namespace: {{.TestNamespace}}
spec:
  ttlSecondsAfterFinished: 10
  template:
    spec:
      containers:
        - name: artemis-producer
          image: ghcr.io/kedacore/tests-artemis
          args: ["producer"]
          env:
            - name: ARTEMIS_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: {{.SecretName}}
                  key: artemis-password
            - name: ARTEMIS_USERNAME
              valueFrom:
                secretKeyRef:
                  name: {{.SecretName}}
                  key: artemis-username
            - name: ARTEMIS_SERVER_HOST
              value: "artemis-activemq.{{.TestNamespace}}"
            - name: ARTEMIS_SERVER_PORT
              value: "61616"
            - name: ARTEMIS_MESSAGE_COUNT
              value: "{{.MessageCount}}"
      restartPolicy: Never
  backoffLimit: 4
`
)

func TestArtemisScaler(t *testing.T) {
	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, "artemis-activemq", testNamespace, 1, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)

	testActivation(t, kc, data)
	testScaleOut(t, kc, data)
	testScaleIn(t, kc)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")
	data.MessageCount = 1
	KubectlReplaceWithTemplate(t, data, "triggerJobTemplate", producerJob)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	data.MessageCount = 1000
	KubectlReplaceWithTemplate(t, data, "triggerJobTemplate", producerJob)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:         testNamespace,
			DeploymentName:        deploymentName,
			ScaledObjectName:      scaledObjectName,
			SecretName:            secretName,
			ArtemisPasswordBase64: base64.StdEncoding.EncodeToString([]byte(artemisPassword)),
			ArtemisUserBase64:     base64.StdEncoding.EncodeToString([]byte(artemisUser)),
			MessageCount:          0,
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
			{Name: "artemisServiceTemplate", Config: artemisServiceTemplate},
			{Name: "artemisConfigTemplate", Config: artemisConfigTemplate},
			{Name: "artemisDeploymentTemplate", Config: artemisDeploymentTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}
