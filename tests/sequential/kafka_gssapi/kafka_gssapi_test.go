//go:build e2e
// +build e2e

package kafka_gssapi_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "kafka-gssapi-test"

	realm           = "KEDA.TEST"
	clientPrincipal = "kedauser"
	clientPassword  = "kedapassword"
	brokerService   = "kafka"

	// Resolved by the scaler against /tmp/kerberos/ccache/.
	ccacheFileName = "keda.ccache"

	// Pinned to 1: the listener advertises the bootstrap Service, the only host the
	// broker keytab holds a principal for. Guarded in TestGSSAPIScaler.
	brokerReplicas = 1

	kinitContainerName   = "kinit"
	kerberosVolumeName   = "kerberos"
	krb5ClientVolumeName = "krb5-client"

	krb5Image = "debian:12-slim"

	kafkaClientImage = "confluentinc/cp-kafka:7.6.1"
)

var (
	testNamespace     = fmt.Sprintf("%s-ns", testName)
	kafkaName         = fmt.Sprintf("%s-kafka", testName)
	kafkaClientName   = fmt.Sprintf("%s-client", testName)
	kdcName           = fmt.Sprintf("%s-kdc", testName)
	kdcConfigMapName  = fmt.Sprintf("%s-kdc-config", testName)
	brokerSecretName  = fmt.Sprintf("%s-broker-krb5", testName)
	clientSecretName  = fmt.Sprintf("%s-client-krb5", testName)
	triggerAuthName   = fmt.Sprintf("%s-ta", testName)
	authSecretName    = fmt.Sprintf("%s-auth", testName)
	deploymentName    = fmt.Sprintf("%s-consumer", testName)
	scaledObjectName  = fmt.Sprintf("%s-so", testName)
	topicPartitions   = 3
	kdcHost           = fmt.Sprintf("%s.%s.svc.cluster.local", kdcName, testNamespace)
	bootstrapHost     = fmt.Sprintf("%s-kafka-bootstrap.%s.svc.cluster.local", kafkaName, testNamespace)
	bootstrapServer   = net.JoinHostPort(bootstrapHost, "9094")
	brokerPodHost     = fmt.Sprintf("%s-broker-0.%s-kafka-brokers.%s.svc.cluster.local", kafkaName, kafkaName, testNamespace)
	brokerJAASPrinc   = fmt.Sprintf("%s/%s@%s", brokerService, bootstrapHost, realm)
	clientFullPrinc   = fmt.Sprintf("%s@%s", clientPrincipal, realm)
	kerberosMountPath = "/tmp/kerberos"
)

type templateData struct {
	TestNamespace    string
	KafkaName        string
	KafkaClientName  string
	KafkaClientImage string
	KdcName          string
	KdcConfigMapName string
	KdcHost          string
	Krb5Image        string
	Realm            string
	BrokerService    string
	BrokerJAASPrinc  string
	BrokerPodHost    string
	BootstrapHost    string
	BootstrapServer  string
	BrokerSecretName string
	ClientSecretName string
	ClientPrincipal  string
	ClientPassword   string
	TopicName        string
	TopicPartitions  int
	BrokerReplicas   int
	DeploymentName   string
	ScaledObjectName string
	TriggerAuthName  string
	AuthSecretName   string
	ConsumerGroup    string

	// Keytab and KerberosConfig are base64 for the secret's `data`; the rest go in `stringData`.
	Username       string
	Password       string
	CcacheName     string
	Keytab         string
	KerberosConfig string
}

const (
	kdcConfigMapTemplate = `apiVersion: v1
kind: ConfigMap
metadata:
  name: {{.KdcConfigMapName}}
  namespace: {{.TestNamespace}}
data:
  entrypoint.sh: |
    #!/usr/bin/env bash
    set -euo pipefail

    export DEBIAN_FRONTEND=noninteractive
    apt-get update -qq
    apt-get install -y -qq --no-install-recommends krb5-kdc krb5-admin-server krb5-user

    cat > /etc/krb5.conf <<'EOF'
    [libdefaults]
      default_realm = {{.Realm}}
      dns_lookup_realm = false
      dns_lookup_kdc = false
      rdns = false
      udp_preference_limit = 1
      forwardable = true

    [realms]
      {{.Realm}} = {
        kdc = {{.KdcHost}}
        admin_server = {{.KdcHost}}
      }

    [domain_realm]
      .svc.cluster.local = {{.Realm}}
      svc.cluster.local = {{.Realm}}
    EOF

    mkdir -p /etc/krb5kdc
    cat > /etc/krb5kdc/kdc.conf <<'EOF'
    [kdcdefaults]
      kdc_ports = 88
      kdc_tcp_ports = 88

    [realms]
      {{.Realm}} = {
        max_life = 24h 0m 0s
        max_renewable_life = 7d 0h 0m 0s
      }
    EOF

    echo '*/*@{{.Realm}} *' > /etc/krb5kdc/kadm5.acl

    kdb5_util create -s -P kedamasterkey -r {{.Realm}}

    # The principal must match the hostname the client dialled, hence two entries.
    kadmin.local -q "addprinc -randkey {{.BrokerService}}/{{.BootstrapHost}}@{{.Realm}}"
    kadmin.local -q "addprinc -randkey {{.BrokerService}}/{{.BrokerPodHost}}@{{.Realm}}"

    kadmin.local -q "addprinc -pw {{.ClientPassword}} {{.ClientPrincipal}}@{{.Realm}}"

    mkdir -p /keytabs
    kadmin.local -q "ktadd -k /keytabs/kafka.keytab {{.BrokerService}}/{{.BootstrapHost}}@{{.Realm}}"
    kadmin.local -q "ktadd -k /keytabs/kafka.keytab {{.BrokerService}}/{{.BrokerPodHost}}@{{.Realm}}"
    kadmin.local -q "ktadd -norandkey -k /keytabs/client.keytab {{.ClientPrincipal}}@{{.Realm}}"

    cp /etc/krb5.conf /keytabs/krb5.conf
    chmod 644 /keytabs/*

    touch /keytabs/.ready

    exec krb5kdc -n
`

	kdcTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.KdcName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.KdcName}}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: {{.KdcName}}
  template:
    metadata:
      labels:
        app: {{.KdcName}}
    spec:
      containers:
        - name: kdc
          image: {{.Krb5Image}}
          command: ["/bin/bash", "/opt/kdc/entrypoint.sh"]
          ports:
            - containerPort: 88
              protocol: TCP
            - containerPort: 88
              protocol: UDP
          volumeMounts:
            - name: config
              mountPath: /opt/kdc
            - name: keytabs
              mountPath: /keytabs
      volumes:
        - name: config
          configMap:
            name: {{.KdcConfigMapName}}
        - name: keytabs
          emptyDir: {}
---
apiVersion: v1
kind: Service
metadata:
  name: {{.KdcName}}
  namespace: {{.TestNamespace}}
spec:
  selector:
    app: {{.KdcName}}
  ports:
    - name: kerberos-tcp
      port: 88
      targetPort: 88
      protocol: TCP
    - name: kerberos-udp
      port: 88
      targetPort: 88
      protocol: UDP
`

	kafkaClusterTemplate = `apiVersion: kafka.strimzi.io/v1beta2
kind: Kafka
metadata:
  name: {{.KafkaName}}
  namespace: {{.TestNamespace}}
  annotations:
    strimzi.io/kraft: enabled
    strimzi.io/node-pools: enabled
spec:
  kafka:
    version: "4.0.0"
    replicas: {{.BrokerReplicas}}
    listeners:
      - name: plain
        port: 9092
        type: internal
        tls: false
      - name: gssapi
        port: 9094
        type: internal
        tls: false
        configuration:
          useServiceDnsDomain: true
          brokers:
            - broker: 0
              advertisedHost: {{.BootstrapHost}}
        authentication:
          type: custom
          sasl: true
          listenerConfig:
            sasl.enabled.mechanisms: GSSAPI
            sasl.kerberos.service.name: {{.BrokerService}}
            gssapi.sasl.jaas.config: >-
              com.sun.security.auth.module.Krb5LoginModule required
              useKeyTab=true
              storeKey=true
              refreshKrb5Config=true
              keyTab="/mnt/krb5/kafka.keytab"
              principal="{{.BrokerJAASPrinc}}";
    template:
      pod:
        volumes:
          - name: broker-krb5
            secret:
              secretName: {{.BrokerSecretName}}
      kafkaContainer:
        volumeMounts:
          - name: broker-krb5
            mountPath: /mnt/krb5
    jvmOptions:
      javaSystemProperties:
        - name: java.security.krb5.conf
          value: /mnt/krb5/krb5.conf
    config:
      offsets.topic.replication.factor: 1
      transaction.state.log.replication.factor: 1
      transaction.state.log.min.isr: 1
    storage:
      type: ephemeral
  entityOperator:
    topicOperator: {}
    userOperator: {}
    template:
      topicOperatorContainer:
        env:
          - name: STRIMZI_USE_FINALIZERS
            value: "false"
---
apiVersion: kafka.strimzi.io/v1beta2
kind: KafkaNodePool
metadata:
  name: broker
  namespace: {{.TestNamespace}}
  labels:
    strimzi.io/cluster: {{.KafkaName}}
spec:
  replicas: {{.BrokerReplicas}}
  roles:
    - broker
    - controller
  storage:
    type: ephemeral
`

	kafkaTopicTemplate = `apiVersion: kafka.strimzi.io/v1beta2
kind: KafkaTopic
metadata:
  name: {{.TopicName}}
  namespace: {{.TestNamespace}}
  labels:
    strimzi.io/cluster: {{.KafkaName}}
spec:
  partitions: {{.TopicPartitions}}
  replicas: {{.BrokerReplicas}}
  config:
    retention.ms: 604800000
    segment.bytes: 1073741824
`

	kafkaClientTemplate = `apiVersion: v1
kind: Pod
metadata:
  name: {{.KafkaClientName}}
  namespace: {{.TestNamespace}}
spec:
  containers:
    - name: {{.KafkaClientName}}
      image: {{.KafkaClientImage}}
      command: ["sh", "-c", "exec tail -f /dev/null"]
      env:
        - name: KAFKA_OPTS
          value: -Djava.security.krb5.conf=/opt/krb5/krb5.conf
      volumeMounts:
        - name: krb5
          mountPath: /opt/krb5
        - name: client-config
          mountPath: /opt/kafka-client
  volumes:
    - name: krb5
      secret:
        secretName: {{.ClientSecretName}}
    - name: client-config
      configMap:
        name: {{.KafkaClientName}}-config
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{.KafkaClientName}}-config
  namespace: {{.TestNamespace}}
data:
  client.properties: |
    security.protocol=SASL_PLAINTEXT
    sasl.mechanism=GSSAPI
    sasl.kerberos.service.name={{.BrokerService}}
    sasl.jaas.config=com.sun.security.auth.module.Krb5LoginModule required useKeyTab=true storeKey=true keyTab="/opt/krb5/client.keytab" principal="{{.ClientPrincipal}}@{{.Realm}}";
`

	deploymentTemplate = `apiVersion: apps/v1
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
        - name: consumer
          image: {{.KafkaClientImage}}
          env:
            - name: KAFKA_OPTS
              value: -Djava.security.krb5.conf=/opt/krb5/krb5.conf
          command:
            - sh
            - -c
            # Offsets are never committed, so lag stays stable while the assertions run.
            - "kafka-console-consumer --bootstrap-server {{.BootstrapServer}} --topic {{.TopicName}} --group {{.ConsumerGroup}} --consumer.config /opt/kafka-client/client.properties --consumer-property enable.auto.commit=false"
          volumeMounts:
            - name: krb5
              mountPath: /opt/krb5
            - name: client-config
              mountPath: /opt/kafka-client
      volumes:
        - name: krb5
          secret:
            secretName: {{.ClientSecretName}}
        - name: client-config
          configMap:
            name: {{.KafkaClientName}}-config
`

	authSecretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.AuthSecretName}}
  namespace: {{.TestNamespace}}
type: Opaque
data:
  kerberosConfig: {{.KerberosConfig}}
{{- if .Keytab}}
  keytab: {{.Keytab}}
{{- end}}
stringData:
  username: {{.Username}}
  realm: {{.Realm}}
{{- if .Password}}
  password: {{.Password}}
{{- end}}
{{- if .CcacheName}}
  ccacheName: {{.CcacheName}}
{{- end}}
`

	triggerAuthTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthName}}
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
    - parameter: username
      name: {{.AuthSecretName}}
      key: username
    - parameter: realm
      name: {{.AuthSecretName}}
      key: realm
    - parameter: kerberosConfig
      name: {{.AuthSecretName}}
      key: kerberosConfig
{{- if .Password}}
    - parameter: password
      name: {{.AuthSecretName}}
      key: password
{{- end}}
{{- if .Keytab}}
    - parameter: keytab
      name: {{.AuthSecretName}}
      key: keytab
{{- end}}
{{- if .CcacheName}}
    - parameter: ccacheName
      name: {{.AuthSecretName}}
      key: ccacheName
{{- end}}
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
  minReplicaCount: 0
  maxReplicaCount: {{.TopicPartitions}}
  cooldownPeriod: 10
  triggers:
    - type: kafka
      metadata:
        bootstrapServers: {{.BootstrapServer}}
        topic: {{.TopicName}}
        consumerGroup: {{.ConsumerGroup}}
        lagThreshold: '1'
        activationLagThreshold: '1'
        offsetResetPolicy: 'earliest'
        sasl: 'gssapi'
        tls: 'disable'
      authenticationRef:
        name: {{.TriggerAuthName}}
`
)

func TestGSSAPIScaler(t *testing.T) {
	require.Equalf(t, 1, brokerReplicas,
		"the GSSAPI listener pins advertisedHost to the bootstrap Service, which only "+
			"resolves to a principal in the broker keytab while there is a single broker; "+
			"supporting %d would need a principal and an advertisedHost per broker",
		brokerReplicas)

	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	t.Log("--- setting up ---")
	CreateKubernetesResources(t, kc, testNamespace, data, templates)
	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})

	kdcPod := waitForKDC(t, kc)

	brokerKeytab := readFileFromKDC(t, kdcPod, "/keytabs/kafka.keytab")
	clientKeytab := readFileFromKDC(t, kdcPod, "/keytabs/client.keytab")
	krb5Conf := readFileFromKDC(t, kdcPod, "/keytabs/krb5.conf")

	createSecret(t, kc, testNamespace, brokerSecretName, map[string][]byte{
		"kafka.keytab": brokerKeytab,
		"krb5.conf":    krb5Conf,
	})
	createSecret(t, kc, testNamespace, clientSecretName, map[string][]byte{
		"client.keytab": clientKeytab,
		"krb5.conf":     krb5Conf,
	})
	// The kinit init container needs it in the KEDA namespace too.
	createSecret(t, kc, KEDANamespace, clientSecretName, map[string][]byte{
		"client.keytab": clientKeytab,
		"krb5.conf":     krb5Conf,
	})
	t.Cleanup(func() {
		_ = kc.CoreV1().Secrets(KEDANamespace).Delete(context.Background(), clientSecretName, metav1.DeleteOptions{})
	})

	data.KerberosConfig = base64.StdEncoding.EncodeToString(krb5Conf)
	data.Username = clientPrincipal

	addCluster(t, data)
	KubectlApplyWithTemplate(t, data, "kafkaClientTemplate", kafkaClientTemplate)
	require.True(t, WaitForAllPodRunningInNamespace(t, kc, testNamespace, 60, 2),
		"kafka client pod should be running")

	patchOperatorForKerberos(t, kc)

	testPasswordAuth(t, kc, data)
	testKeytabAuth(t, kc, data, clientKeytab)
	testCcacheAuth(t, kc, data)

	testRejectsMultipleCredentials(t, data, clientKeytab)
	testRejectsMissingCredential(t, data)
}

func testPasswordAuth(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing GSSAPI with password ---")
	data.TopicName = "gssapi-password-topic"
	data.ConsumerGroup = "gssapi-password-group"
	data.Password = clientPassword
	data.Keytab = ""
	data.CcacheName = ""
	assertScalesOut(t, kc, data)
}

func testKeytabAuth(t *testing.T, kc *kubernetes.Clientset, data templateData, clientKeytab []byte) {
	t.Log("--- testing GSSAPI with keytab ---")
	data.TopicName = "gssapi-keytab-topic"
	data.ConsumerGroup = "gssapi-keytab-group"
	data.Password = ""
	data.Keytab = base64.StdEncoding.EncodeToString(clientKeytab)
	data.CcacheName = ""
	assertScalesOut(t, kc, data)
}

func testCcacheAuth(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing GSSAPI with ccache ---")
	data.TopicName = "gssapi-ccache-topic"
	data.ConsumerGroup = "gssapi-ccache-group"
	data.Password = ""
	data.Keytab = ""
	data.CcacheName = ccacheFileName
	assertScalesOut(t, kc, data)
}

func testRejectsMultipleCredentials(t *testing.T, data templateData, clientKeytab []byte) {
	t.Log("--- testing GSSAPI rejects multiple credentials ---")
	data.TopicName = "gssapi-multi-cred-topic"
	data.ConsumerGroup = "gssapi-multi-cred-group"
	data.Password = clientPassword
	data.Keytab = base64.StdEncoding.EncodeToString(clientKeytab)
	data.CcacheName = ccacheFileName
	assertCredentialsRejected(t, data)
}

func testRejectsMissingCredential(t *testing.T, data templateData) {
	t.Log("--- testing GSSAPI rejects a missing credential ---")
	data.TopicName = "gssapi-no-cred-topic"
	data.ConsumerGroup = "gssapi-no-cred-group"
	data.Password = ""
	data.Keytab = ""
	data.CcacheName = ""
	assertCredentialsRejected(t, data)
}

// No topic needed: parsing rejects the metadata before any broker call.
func assertCredentialsRejected(t *testing.T, data templateData) {
	KubectlApplyWithTemplate(t, data, "authSecretTemplate", authSecretTemplate)
	defer KubectlDeleteWithTemplate(t, data, "authSecretTemplate", authSecretTemplate)
	KubectlApplyWithTemplate(t, data, "triggerAuthTemplate", triggerAuthTemplate)
	defer KubectlDeleteWithTemplate(t, data, "triggerAuthTemplate", triggerAuthTemplate)
	KubectlApplyWithTemplate(t, data, "deploymentTemplate", deploymentTemplate)
	defer KubectlDeleteWithTemplate(t, data, "deploymentTemplate", deploymentTemplate)
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	defer KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	reason, message := waitForScaledObjectFailure(t, 12, 5)
	assert.Equal(t, "ScaledObjectCheckFailed", reason,
		"scaled object should report that the scaler could not be built")
	assert.Contains(t, message, "exactly one of 'password', 'keytab' or 'ccacheName' must be provided",
		"failure should name the credential rule that was broken")
}

func waitForScaledObjectFailure(t *testing.T, iterations, intervalSeconds int) (string, string) {
	for i := 0; i < iterations; i++ {
		status, reason, message := scaledObjectReadyCondition(t)
		if status == "False" && reason != "" {
			return reason, message
		}
		t.Logf("waiting for scaled object to report a failure - status: %q, reason: %q", status, reason)
		time.Sleep(time.Duration(intervalSeconds) * time.Second)
	}
	return "", ""
}

func scaledObjectReadyCondition(t *testing.T) (string, string, string) {
	out, err := ExecuteCommand(fmt.Sprintf(
		"kubectl get scaledobject/%s -n %s -o json", scaledObjectName, testNamespace))
	require.NoErrorf(t, err, "cannot read scaled object - %s", err)

	var scaledObject struct {
		Status struct {
			Conditions []struct {
				Type    string `json:"type"`
				Status  string `json:"status"`
				Reason  string `json:"reason"`
				Message string `json:"message"`
			} `json:"conditions"`
		} `json:"status"`
	}
	require.NoErrorf(t, json.Unmarshal(out, &scaledObject), "cannot decode scaled object - %s", err)

	for _, condition := range scaledObject.Status.Conditions {
		if condition.Type == "Ready" {
			return condition.Status, condition.Reason, condition.Message
		}
	}
	return "", "", ""
}

func assertScalesOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	addTopic(t, data)
	defer KubectlDeleteWithTemplate(t, data, "kafkaTopicTemplate", kafkaTopicTemplate)

	KubectlApplyWithTemplate(t, data, "authSecretTemplate", authSecretTemplate)
	defer KubectlDeleteWithTemplate(t, data, "authSecretTemplate", authSecretTemplate)
	KubectlApplyWithTemplate(t, data, "triggerAuthTemplate", triggerAuthTemplate)
	defer KubectlDeleteWithTemplate(t, data, "triggerAuthTemplate", triggerAuthTemplate)
	KubectlApplyWithTemplate(t, data, "deploymentTemplate", deploymentTemplate)
	defer KubectlDeleteWithTemplate(t, data, "deploymentTemplate", deploymentTemplate)
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	defer KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	// One message sits on the activation threshold, so nothing should start.
	publishMessage(t, data)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 30)

	// Push lag past maxReplicaCount so the expected count is the cap.
	for i := 0; i < topicPartitions+1; i++ {
		publishMessage(t, data)
	}
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, topicPartitions, 60, 2),
		"replica count should reach %d within 2 minutes", topicPartitions)
}

func publishMessage(t *testing.T, data templateData) {
	cmd := fmt.Sprintf(
		`echo "msg" | kafka-console-producer --broker-list %s --topic %s --producer.config /opt/kafka-client/client.properties`,
		data.BootstrapServer, data.TopicName)
	_, _, err := ExecCommandOnSpecificPod(t, kafkaClientName, testNamespace, cmd)
	require.NoErrorf(t, err, "cannot publish message - %s", err)
}

func waitForKDC(t *testing.T, kc *kubernetes.Clientset) string {
	t.Log("--- waiting for KDC ---")
	require.True(t, WaitForDeploymentReplicaReadyCount(t, kc, kdcName, testNamespace, 1, 60, 5),
		"KDC deployment should become ready")

	out, err := ExecuteCommand(fmt.Sprintf(
		"kubectl get pods -n %s -l app=%s -o jsonpath={.items[0].metadata.name}", testNamespace, kdcName))
	require.NoErrorf(t, err, "cannot get KDC pod name - %s", err)
	podName := strings.TrimSpace(string(out))
	require.NotEmpty(t, podName, "KDC pod name should not be empty")

	// The realm is provisioned after apt-get, so pod readiness is not enough.
	ok, _, _, err := WaitForSuccessfulExecCommandOnSpecificPod(t, podName, testNamespace,
		"test -f /keytabs/.ready", 60, 5)
	require.NoErrorf(t, err, "error waiting for KDC provisioning - %s", err)
	require.True(t, ok, "KDC should finish provisioning the realm")

	t.Log("--- KDC ready ---")
	return podName
}

// base64 plus the non-TTY exec keeps the keytab intact: newline translation corrupts it.
func readFileFromKDC(t *testing.T, podName, path string) []byte {
	out, errOut, err := ExecCommandOnSpecificPodWithoutTTY(t, podName, testNamespace,
		fmt.Sprintf("base64 -w0 %s", path))
	require.NoErrorf(t, err, "cannot read %s from KDC - %s (%s)", path, err, errOut)

	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(out))
	require.NoErrorf(t, err, "cannot decode %s - %s", path, err)
	require.NotEmpty(t, decoded, "%s should not be empty", path)
	return decoded
}

// Replaces any leftover: the KEDA-namespace copy outlives the test namespace.
func createSecret(t *testing.T, kc *kubernetes.Clientset, namespace, name string, contents map[string][]byte) {
	ctx := context.Background()
	_ = kc.CoreV1().Secrets(namespace).Delete(ctx, name, metav1.DeleteOptions{})

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Data:       contents,
	}
	_, err := kc.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	require.NoErrorf(t, err, "cannot create secret %s/%s - %s", namespace, name, err)
}

func stripKerberosPatch(operator *appsv1.Deployment) {
	spec := &operator.Spec.Template.Spec
	spec.InitContainers = slices.DeleteFunc(spec.InitContainers, func(c corev1.Container) bool {
		return c.Name == kinitContainerName
	})
	spec.Volumes = slices.DeleteFunc(spec.Volumes, func(v corev1.Volume) bool {
		return v.Name == kerberosVolumeName || v.Name == krb5ClientVolumeName
	})
	for i := range spec.Containers {
		spec.Containers[i].VolumeMounts = slices.DeleteFunc(spec.Containers[i].VolumeMounts,
			func(m corev1.VolumeMount) bool { return m.Name == kerberosVolumeName })
	}
}

// Mounts /tmp/kerberos and kinits into it so the ccache mode has a cache to read.
func patchOperatorForKerberos(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- patching KEDA operator for Kerberos ---")
	ctx := context.Background()

	operator, err := kc.AppsV1().Deployments(KEDANamespace).Get(ctx, KEDAOperator, metav1.GetOptions{})
	require.NoErrorf(t, err, "cannot get keda operator deployment - %s", err)

	// A timed-out run skips t.Cleanup; without this the appends below duplicate names.
	stripKerberosPatch(operator)
	original := operator.Spec.Template.DeepCopy()

	kinitCmd := fmt.Sprintf(
		"set -euo pipefail; export DEBIAN_FRONTEND=noninteractive; apt-get update -qq; "+
			"apt-get install -y -qq --no-install-recommends krb5-user; "+
			"mkdir -p %s/ccache; "+
			"kinit -k -t /opt/krb5/client.keytab -c %s/ccache/%s %s; "+
			"chmod 644 %s/ccache/%s",
		kerberosMountPath, kerberosMountPath, ccacheFileName, clientFullPrinc, kerberosMountPath, ccacheFileName)

	runAsRoot := int64(0)
	notNonRoot := false
	operator.Spec.Template.Spec.InitContainers = append(operator.Spec.Template.Spec.InitContainers, corev1.Container{
		Name:    kinitContainerName,
		Image:   krb5Image,
		Command: []string{"/bin/bash", "-c", kinitCmd},
		Env: []corev1.EnvVar{
			{Name: "KRB5_CONFIG", Value: "/opt/krb5/krb5.conf"},
		},
		VolumeMounts: []corev1.VolumeMount{
			{Name: kerberosVolumeName, MountPath: kerberosMountPath},
			{Name: krb5ClientVolumeName, MountPath: "/opt/krb5"},
		},
		// The pod template sets runAsNonRoot; installing packages needs root.
		SecurityContext: &corev1.SecurityContext{
			RunAsUser:    &runAsRoot,
			RunAsNonRoot: &notNonRoot,
		},
	})

	operator.Spec.Template.Spec.Volumes = append(operator.Spec.Template.Spec.Volumes,
		corev1.Volume{
			Name:         kerberosVolumeName,
			VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
		},
		corev1.Volume{
			Name: krb5ClientVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{SecretName: clientSecretName},
			},
		},
	)

	for i, container := range operator.Spec.Template.Spec.Containers {
		if container.Name == KEDAOperator {
			operator.Spec.Template.Spec.Containers[i].VolumeMounts = append(
				operator.Spec.Template.Spec.Containers[i].VolumeMounts,
				corev1.VolumeMount{Name: kerberosVolumeName, MountPath: kerberosMountPath},
			)
		}
	}

	// Registered before the update: a readiness failure below still runs t.Cleanup, and
	// leaving the operator patched would break it once the secret it mounts is deleted.
	t.Cleanup(func() {
		t.Log("--- restoring KEDA operator ---")
		current, err := kc.AppsV1().Deployments(KEDANamespace).Get(ctx, KEDAOperator, metav1.GetOptions{})
		if err != nil {
			t.Logf("cannot get keda operator deployment for restore: %s", err)
			return
		}
		current.Spec.Template = *original
		if _, err := kc.AppsV1().Deployments(KEDANamespace).Update(ctx, current, metav1.UpdateOptions{}); err != nil {
			t.Logf("cannot restore keda operator deployment: %s", err)
			return
		}
		WaitForDeploymentReplicaReadyCount(t, kc, KEDAOperator, KEDANamespace, 1, 60, 5)
	})

	_, err = kc.AppsV1().Deployments(KEDANamespace).Update(ctx, operator, metav1.UpdateOptions{})
	require.NoErrorf(t, err, "cannot patch keda operator - %s", err)
	require.True(t, WaitForDeploymentReplicaReadyCount(t, kc, KEDAOperator, KEDANamespace, 1, 60, 5),
		"keda operator should become ready after patching")
	t.Log("--- KEDA operator patched ---")
}

func addCluster(t *testing.T, data templateData) {
	t.Log("--- adding kafka cluster ---")
	KubectlApplyWithTemplate(t, data, "kafkaClusterTemplate", kafkaClusterTemplate)
	_, err := ExecuteCommand(fmt.Sprintf(
		"kubectl wait kafka/%s --for=condition=Ready --timeout=600s --namespace %s", kafkaName, testNamespace))
	require.NoErrorf(t, err, "cannot execute command - %s", err)
	t.Log("--- kafka cluster added ---")
}

func addTopic(t *testing.T, data templateData) {
	t.Log("--- adding kafka topic ---")
	KubectlApplyWithTemplate(t, data, "kafkaTopicTemplate", kafkaTopicTemplate)
	_, err := ExecuteCommand(fmt.Sprintf(
		"kubectl wait kafkatopic/%s --for=condition=Ready --timeout=480s --namespace %s", data.TopicName, testNamespace))
	require.NoErrorf(t, err, "cannot execute command - %s", err)
	t.Log("--- kafka topic added ---")
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:    testNamespace,
			KafkaName:        kafkaName,
			KafkaClientName:  kafkaClientName,
			KafkaClientImage: kafkaClientImage,
			KdcName:          kdcName,
			KdcConfigMapName: kdcConfigMapName,
			KdcHost:          kdcHost,
			Krb5Image:        krb5Image,
			Realm:            realm,
			BrokerService:    brokerService,
			BrokerJAASPrinc:  brokerJAASPrinc,
			BrokerPodHost:    brokerPodHost,
			BootstrapHost:    bootstrapHost,
			BootstrapServer:  bootstrapServer,
			BrokerSecretName: brokerSecretName,
			ClientSecretName: clientSecretName,
			ClientPrincipal:  clientPrincipal,
			ClientPassword:   clientPassword,
			TopicPartitions:  topicPartitions,
			BrokerReplicas:   brokerReplicas,
			DeploymentName:   deploymentName,
			ScaledObjectName: scaledObjectName,
			TriggerAuthName:  triggerAuthName,
			AuthSecretName:   authSecretName,
		}, []Template{
			{Name: "kdcConfigMapTemplate", Config: kdcConfigMapTemplate},
			{Name: "kdcTemplate", Config: kdcTemplate},
		}
}
