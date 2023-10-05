//go:build e2e
// +build e2e

package natsjetstream_standalone_test

import (
	"fmt"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	k8s "k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
	nats "github.com/kedacore/keda/v2/tests/scalers/nats_jetstream/helper"
)

// Load env variables from .env files
var _ = godotenv.Load("../../.env")

const (
	testName = "nats-jetstream-standalone"
)

var (
	testNamespace                = fmt.Sprintf("%s-test-ns", testName)
	natsNamespace                = fmt.Sprintf("%s-nats-ns", testName)
	natsAddress                  = fmt.Sprintf("nats://%s.%s.svc.cluster.local:4222", nats.NatsJetStreamName, natsNamespace)
	natsServerMonitoringEndpoint = fmt.Sprintf("%s.%s.svc.cluster.local:8222", nats.NatsJetStreamName, natsNamespace)
	messagePublishCount          = 300
	deploymentName               = "sub"
	minReplicaCount              = 0
	maxReplicaCount              = 2
)

const natsServerTemplate = `
# Source: https://github.com/nats-io/k8s/blob/main/nats-server/single-server-nats.yml
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: nats-config
  namespace: {{.NatsNamespace}}
data:
  nats.conf: |
    pid_file: "/var/run/nats/nats.pid"
    http: 8222
    jetstream {
      store_dir: /data/jetstream
      max_mem: 1G
      max_file: 10G
    }
---
apiVersion: v1
kind: Service
metadata:
  name: nats
  namespace: {{.NatsNamespace}}
  labels:
    app: nats
spec:
  selector:
    app: nats
  clusterIP: None
  ports:
  - name: client
    port: 4222
  - name: cluster
    port: 6222
  - name: monitor
    port: 8222
  - name: metrics
    port: 7777
  - name: leafnodes
    port: 7422
  - name: gateways
    port: 7522
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: nats
  namespace: {{.NatsNamespace}}
  labels:
    app: nats
spec:
  selector:
    matchLabels:
      app: nats
  replicas: 1
  serviceName: "nats"
  template:
    metadata:
      labels:
        app: nats
    spec:
      # Common volumes for the containers
      volumes:
      - name: config-volume
        configMap:
          name: nats-config
      - name: pid
        emptyDir: {}

      # Required to be able to HUP signal and apply config reload
      # to the server without restarting the pod.
      shareProcessNamespace: true

      #################
      #               #
      #  NATS Server  #
      #               #
      #################
      terminationGracePeriodSeconds: 60
      containers:
      - name: nats
        image: nats:{{.NatsVersion}}-alpine
        ports:
        - containerPort: 4222
          name: client
          hostPort: 4222
        - containerPort: 7422
          name: leafnodes
          hostPort: 7422
        - containerPort: 6222
          name: cluster
        - containerPort: 8222
          name: monitor
        - containerPort: 7777
          name: metrics
        command:
         - "nats-server"
         - "--config"
         - "/etc/nats-config/nats.conf"

        # Required to be able to define an environment variable
        # that refers to other environment variables.  This env var
        # is later used as part of the configuration file.
        env:
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: POD_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: CLUSTER_ADVERTISE
          value: $(POD_NAME).nats.$(POD_NAMESPACE).svc
        volumeMounts:
          - name: config-volume
            mountPath: /etc/nats-config
          - name: pid
            mountPath: /var/run/nats

        # Liveness/Readiness probes against the monitoring
        #
        livenessProbe:
          httpGet:
            path: /
            port: 8222
          initialDelaySeconds: 10
          timeoutSeconds: 5
        readinessProbe:
          httpGet:
            path: /
            port: 8222
          initialDelaySeconds: 10
          timeoutSeconds: 5

        # Gracefully stop NATS Server on pod deletion or image upgrade.
        #
        lifecycle:
          preStop:
            exec:
              # Using the alpine based NATS image, we add an extra sleep that is
              # the same amount as the terminationGracePeriodSeconds to allow
              # the NATS Server to gracefully terminate the client connections.
              #
              command: ["/bin/sh", "-c", "/nats-server -sl=ldm=/var/run/nats/nats.pid && /bin/sleep 60"]
  `

func TestNATSJetStreamScaler(t *testing.T) {
	// Create k8s resources.
	kc := GetKubernetesClient(t)

	// Deploy NATS server.
	installServerWithJetStream(t, kc, natsNamespace)
	assert.True(t, WaitForStatefulsetReplicaReadyCount(t, kc, nats.NatsJetStreamName, natsNamespace, 1, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)

	// Create k8s resources for testing.
	data, templates := nats.GetJetStreamDeploymentTemplateData(testNamespace, natsAddress, natsServerMonitoringEndpoint, messagePublishCount)
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	// Create stream and consumer.
	data.NatsStream = "standalone"
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", nats.ScaledObjectTemplate)
	installStreamAndConsumer(t, data.NatsStream, testNamespace, natsAddress)
	assert.True(t, WaitForJobSuccess(t, kc, "stream", testNamespace, 60, 3),
		"stream and consumer creation job should be success")

	testActivation(t, kc, data)
	testScaleOut(t, kc, data)
	testScaleIn(t, kc)

	// Cleanup nats namespace
	removeServerWithJetStream(t, natsNamespace)
	DeleteNamespace(t, natsNamespace)
	deleted := WaitForNamespaceDeletion(t, natsNamespace)
	assert.Truef(t, deleted, "%s namespace not deleted", natsNamespace)
	// Cleanup test namespace
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

// installStreamAndConsumer creates stream and consumer.
func installStreamAndConsumer(t *testing.T, stream, namespace, natsAddress string) {
	data := nats.JetStreamTemplateData{
		TestNamespace:  namespace,
		NatsAddress:    natsAddress,
		NatsConsumer:   nats.NatsJetStreamConsumerName,
		NatsStream:     stream,
		StreamReplicas: 1,
	}

	KubectlApplyWithTemplate(t, data, "streamAndConsumerTemplate", nats.StreamAndConsumerTemplate)
}

// installServerWithJetStream will deploy NATS server with JetStream.
func installServerWithJetStream(t *testing.T, kc *k8s.Clientset, namespace string) {
	CreateNamespace(t, kc, namespace)
	data := nats.JetStreamTemplateData{
		NatsNamespace: namespace,
		NatsVersion:   nats.NatsJetStreamServerVersion,
	}

	KubectlApplyWithTemplate(t, data, "natsServerTemplate", natsServerTemplate)
}

// removeServerWithJetStream will remove the NATS server and delete the namespace.
func removeServerWithJetStream(t *testing.T, namespace string) {
	data := nats.JetStreamTemplateData{
		NatsNamespace: namespace,
		NatsVersion:   nats.NatsJetStreamServerVersion,
	}

	KubectlDeleteWithTemplate(t, data, "natsServerTemplate", natsServerTemplate)
	DeleteNamespace(t, namespace)
}

func testActivation(t *testing.T, kc *k8s.Clientset, data nats.JetStreamDeploymentTemplateData) {
	t.Log("--- testing activation ---")
	data.NumberOfMessages = 10
	KubectlApplyWithTemplate(t, data, "activationPublishJobTemplate", nats.ActivationPublishJobTemplate)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *k8s.Clientset, data nats.JetStreamDeploymentTemplateData) {
	t.Log("--- testing scale out ---")
	KubectlApplyWithTemplate(t, data, "publishJobTemplate", nats.PublishJobTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *k8s.Clientset) {
	t.Log("--- testing scale in ---")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
}
