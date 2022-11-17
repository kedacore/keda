//go:build e2e
// +build e2e

package helper

import (
	"fmt"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	"github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../../.env")

const (
	apachePulsarVersion = "2.10.2"
	messageCount        = 3
	minReplicaCount     = 0
	maxReplicaCount     = 5
	msgBacklog          = 10
)

type templateData struct {
	ApachePulsarVersion string
	TestName            string // Used for most resource names
	NumPartitions       int    // Use 0 to create a non-partitioned topic
	MessageCount        int
	MinReplicaCount     int
	MaxReplicaCount     int
	MsgBacklog          int
}

const authSecretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.TestName}}
  namespace: {{.TestName}}
data:
  key.pub: MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAnkggprp2GTl/2oQgLvnspbH0Lxthhmw3O3qpcx1FVUcJeD1JlUsuK6rO8uexfY/3JuZffzEm5busJB/5zuXQqO52ph8xDRiEeHOuFY0RKv8DAfpss+oG8Ou/LdHPYCbbyjbJXK/iVE/rUhicp7n6udv2/AaqJj/9535Qo49Q+3S/fbWqhNR6r84+Q+KTHtfwuoLsE4AbZ+g7FRpnyH3iYDxC4ISr1zIJiv4o41cwglaho/cOqCpBFwRHYyZTgeEIf9+7bjTPbpPThFztxO6DOAw73ikU7iT3T0H6hgpQqKa79kw1R8PAfeTYvkeQ4juQwlYmyGePTb9F4LZ+0w7a8wIDAQAB
  token.jwt: ZXlKaGJHY2lPaUpTVXpJMU5pSjkuZXlKemRXSWlPaUpoWkcxcGJpSjkubEg2TEVqcDU3Y2pFc2xhdWV2Z1ZKV1NTa19IaThFLVZGb29EZHVxUHRiQ1Q0U0NJQlluV0YtRlA5NzBMVUMxRzFWWnZFMmJFZGlkNGd3SzhKY3RnVHNMNGJTV2V5SW4yVVBNTnNnaDVGemhWQkQ4SXVaRnFLTXktLUZnUmtKWFZzWldrbUFwNW5yamU3MEZaRkJLME1uV0licWxSZ2Y2UUZKR2Vxd1FXbzlZV0RCOUh5cTRYR0oxUGx1SGR4T282eTJjVm1Ib3c2SFV3R0dfSDZfTmk0eTNBaU0zWEhvNlNvMkEtRGU5cGRBX3d6MHQzemFyXzhBNFJNeXdTYmtXYldNSVEwUnN5bEZhSk80SzYzT0lTRG5IQkp0TUNJTUNjNlo1WDFKYWt2eUdKek9FTVNQeDZRM1hXWG1MOFFDNjBrcG1xQkd0dXV4XzZlbWFSaHZTcDlB
`

const pulsarStatefulsetTemplate = `
apiVersion: apps/v1
kind: StatefulSet
metadata:
 name: {{.TestName}}
 namespace: {{.TestName}}
 labels:
  app: pulsar
spec:
  selector:
    matchLabels:
      app: pulsar
  replicas: 1
  serviceName: {{.TestName}}
  template:
    metadata:
      labels:
        app: pulsar
    spec:
      containers:
      - name: pulsar
        image: apachepulsar/pulsar:{{.ApachePulsarVersion}}
        imagePullPolicy: IfNotPresent
        volumeMounts:
        - name: auth-data
          mountPath: "/bin/pulsar"
          readOnly: true
        readinessProbe:
          tcpSocket:
            port: 8080
        ports:
        - name: pulsar
          containerPort: 6650
          protocol: TCP
        - name: admin
          containerPort: 8080
          protocol: TCP
        env:
        - name: PULSAR_PREFIX_tlsRequireTrustedClientCertOnConnect
          value: "true"
        - name: brokerDeleteInactiveTopicsEnabled
          value: "false"
        - name: authenticationEnabled
          value: "true"
        - name: authenticationProviders
          value: "org.apache.pulsar.broker.authentication.AuthenticationProviderToken"
        - name: PULSAR_PREFIX_tokenPublicKey
          value: "/bin/pulsar/key.pub"
        - name: brokerClientAuthenticationPlugin
          value: "org.apache.pulsar.client.impl.auth.AuthenticationToken"
        - name: brokerClientAuthenticationParameters
          value: "file:///bin/pulsar/token.jwt"
        command:
        - sh
        - -c
        args: ["bin/apply-config-from-env.py conf/client.conf && bin/apply-config-from-env.py conf/standalone.conf && exec bin/pulsar standalone -nfw -nss"]
      volumes:
      - name: auth-data
        secret:
          secretName: {{.TestName}}
`

const pulsarServiceTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: {{.TestName}}
  namespace: {{.TestName}}
spec:
  type: ClusterIP
  ports:
  - name: http
    port: 8080
    targetPort: 8080
    protocol: TCP
  - name: pulsar
    port: 6650
    targetPort: 6650
    protocol: TCP
  selector:
    app: pulsar
`

const topicInitJobTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  name: {{.TestName}}-topic-init
  namespace: {{.TestName}}
spec:
  template:
    spec:
      containers:
      - name: pulsar-topic-init
        image: apachepulsar/pulsar:{{.ApachePulsarVersion}}
        imagePullPolicy: IfNotPresent
        volumeMounts:
        - name: auth-data
          mountPath: "/pulsar/auth"
          readOnly: true
        command:
        - sh
        - -c
        args: ["bin/pulsar-admin --admin-url http://{{.TestName}}.{{.TestName}}:8080 --auth-plugin org.apache.pulsar.client.impl.auth.AuthenticationToken --auth-params file:///pulsar/auth/token.jwt topics {{ if .NumPartitions }} create-partitioned-topic -p {{.NumPartitions}} {{ else }} create {{ end }} persistent://public/default/keda"]
      restartPolicy: Never
      volumes:
      - name: auth-data
        secret:
          secretName: {{.TestName}}
  backoffLimit: 4
`

const consumerTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.TestName}}-consumer
  namespace: {{.TestName}}
  labels:
    app: pulsar-consumer
spec:
  selector:
    matchLabels:
      app: pulsar-consumer
  template:
    metadata:
      labels:
        app: pulsar-consumer
    spec:
      containers:
        - name: pulsar-consumer
          image: apachepulsar/pulsar:{{.ApachePulsarVersion}}
          imagePullPolicy: IfNotPresent
          volumeMounts:
          - name: auth-data
            mountPath: "/pulsar/auth"
            readOnly: true
          command:
          - sh
          - -c
          args: ["bin/pulsar-perf consume --service-url pulsar://{{.TestName}}.{{.TestName}}:6650 --auth-plugin org.apache.pulsar.client.impl.auth.AuthenticationToken --auth-params file:///pulsar/auth/token.jwt --receiver-queue-size 1 --subscription-type Shared --rate 1 --subscriptions keda persistent://public/default/keda"]
      volumes:
      - name: auth-data
        secret:
          secretName: {{.TestName}}
`

const scaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.TestName}}
  namespace: {{.TestName}}
spec:
  scaleTargetRef:
    name: {{.TestName}}-consumer
  pollingInterval: 5 # Optional. Default: 30 seconds
  cooldownPeriod: 30 # Optional. Default: 300 seconds
  maxReplicaCount: {{.MaxReplicaCount}}
  minReplicaCount: {{.MinReplicaCount}}
  triggers:
    - type: pulsar
      metadata:
        msgBacklog: "{{.MsgBacklog}}"
        activationMsgBacklogThreshold: "5"
        adminURL: http://{{.TestName}}.{{.TestName}}:8080
        topic:  persistent://public/default/keda
        isPartitionedTopic: {{ if .NumPartitions }} "true" {{else}} "false" {{end}}
        authModes: "bearer"
        subscription: keda
      authenticationRef:
        name: {{.TestName}}
          `

const authenticationRefTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TestName}}
  namespace: {{.TestName}}
spec:
  secretTargetRef:
    - parameter: bearerToken
      name: {{.TestName}}
      key: token.jwt
`

const topicPublishJobTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  name: {{.TestName}}-producer
  namespace: {{.TestName}}
spec:
  template:
    spec:
      containers:
      - name: pulsar-producer
        image: apachepulsar/pulsar:{{.ApachePulsarVersion}}
        imagePullPolicy: IfNotPresent
        volumeMounts:
        - name: auth-data
          mountPath: "/pulsar/auth"
          readOnly: true
        command:
        - sh
        - -c
        args: ["bin/pulsar-perf produce --admin-url http://{{.TestName}}.{{.TestName}}:8080 --service-url pulsar://{{.TestName}}.{{.TestName}}:6650 --auth-plugin org.apache.pulsar.client.impl.auth.AuthenticationToken --auth-params file:///pulsar/auth/token.jwt --num-messages {{.MessageCount}} {{ if .NumPartitions }} --partitions {{.NumPartitions}} {{ end }} --batch-max-messages 1 persistent://public/default/keda"]
      restartPolicy: Never
      volumes:
      - name: auth-data
        secret:
          secretName: {{.TestName}}
  backoffLimit: 4
`

func TestScalerWithConfig(t *testing.T, testName string, numPartitions int) {
	// setup
	t.Log("--- setting up ---")

	// Create kubernetes resources
	kc := helper.GetKubernetesClient(t)
	data, templates := getTemplateData(testName, numPartitions)

	helper.CreateKubernetesResources(t, kc, testName, data, templates)

	assert.True(t, helper.WaitForStatefulsetReplicaReadyCount(t, kc, testName, testName, 1, 300, 1),
		"replica count should be 1 within 5 minutes")

	helper.KubectlApplyWithTemplate(t, data, "topicInitJobTemplate", topicInitJobTemplate)

	assert.True(t, helper.WaitForJobSuccess(t, kc, getTopicInitJobName(testName), testName, 300, 1),
		"job should succeed within 5 minutes")

	helper.KubectlApplyWithTemplate(t, data, "consumerTemplate", consumerTemplate)

	// run consumer for create subscription
	assert.True(t, helper.WaitForDeploymentReplicaReadyCount(t, kc, getConsumerDeploymentName(testName), testName, 1, 300, 1),
		"replica count should be 1 within 5 minutes")

	helper.KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	assert.True(t, helper.WaitForDeploymentReplicaReadyCount(t, kc, getConsumerDeploymentName(testName), testName, 0, 60, 1),
		"replica count should be 0 after a minute")

	testActivation(t, kc, data)
	// scale out
	testScaleOut(t, kc, data)
	// scale in
	testScaleIn(t, kc, testName)

	// cleanup
	helper.KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	helper.KubectlDeleteWithTemplate(t, data, "publishJobTemplate", topicPublishJobTemplate)
	helper.KubectlDeleteWithTemplate(t, data, "topicInitJobTemplate", topicInitJobTemplate)

	helper.DeleteKubernetesResources(t, kc, testName, data, templates)
}

func getTemplateData(testName string, numPartitions int) (templateData, []helper.Template) {
	return templateData{
			ApachePulsarVersion: apachePulsarVersion,
			TestName:            testName,
			NumPartitions:       numPartitions,
			MessageCount:        messageCount,
			MinReplicaCount:     minReplicaCount,
			MaxReplicaCount:     maxReplicaCount,
			MsgBacklog:          msgBacklog,
		}, []helper.Template{
			{Name: "statefulsetTemplate", Config: pulsarStatefulsetTemplate},
			{Name: "serviceTemplate", Config: pulsarServiceTemplate},
			{Name: "authenticationRefTemplate", Config: authenticationRefTemplate},
			{Name: "secretTemplate", Config: authSecretTemplate},
		}
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")
	// publish message and less than MsgBacklog
	helper.KubectlApplyWithTemplate(t, data, "publishJobTemplate", topicPublishJobTemplate)
	helper.AssertReplicaCountNotChangeDuringTimePeriod(t, kc, getConsumerDeploymentName(data.TestName), data.TestName, data.MinReplicaCount, 60)
	helper.KubectlDeleteWithTemplate(t, data, "publishJobTemplate", topicPublishJobTemplate)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	data.MessageCount = 100
	helper.KubectlApplyWithTemplate(t, data, "publishJobTemplate", topicPublishJobTemplate)
	assert.True(t, helper.WaitForDeploymentReplicaReadyCount(t, kc, getConsumerDeploymentName(data.TestName), data.TestName, 5, 300, 1),
		"replica count should be 5 within 5 minute")
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, testName string) {
	t.Log("--- testing scale in ---")
	// Check if deployment scale in to 0 after 5 minutes
	assert.True(t, helper.WaitForDeploymentReplicaReadyCount(t, kc, getConsumerDeploymentName(testName), testName, 0, 300, 1),
		"Replica count should be 0 within 5 minutes")
}

func getConsumerDeploymentName(testName string) string {
	return fmt.Sprintf("%s-consumer", testName)
}

func getTopicInitJobName(testName string) string {
	return fmt.Sprintf("%s-topic-init", testName)
}
