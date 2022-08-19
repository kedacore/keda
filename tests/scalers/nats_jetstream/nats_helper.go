//go:build e2e
// +build e2e

package natsjetstream

import (
	"testing"

	k8s "k8s.io/client-go/kubernetes"

	"github.com/kedacore/keda/v2/tests/helper"
)

type templateData struct {
	NatsNamespace string
	TestNamespace string
	NatsAddress   string
}

const (
	natsServerTemplate = `
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
        image: nats:2.8.4-alpine
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

	streamAndConsumerTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  name: stream
  namespace: {{.TestNamespace}}
spec:
  template:
    spec:
      containers:
      - name: stream
        image: "goku321/nats-stream:v0.3"
        imagePullPolicy: Always
        command: ["./main"]
        env:
        - name: NATS_ADDRESS
          value: {{.NatsAddress}}
      restartPolicy: Never
  backoffLimit: 4
  `
)

// InstallServerWithJetStream will deploy NATS server with JetStream.
func InstallServerWithJetStream(t *testing.T, kc *k8s.Clientset, namespace string) {
	helper.CreateNamespace(t, kc, namespace)
	data := templateData{
		NatsNamespace: namespace,
	}

	helper.KubectlApplyWithTemplate(t, data, "natsServerTemplate", natsServerTemplate)
}

// RemoveServer will remove the NATS server and delete the namespace.
func RemoveServer(t *testing.T, kc *k8s.Clientset, namespace string) {
	data := templateData{
		NatsNamespace: namespace,
	}

	helper.KubectlDeleteWithTemplate(t, data, "natsServerTemplate", natsServerTemplate)
	helper.DeleteNamespace(t, kc, namespace)
}

// InstallStreamAndConsumer creates stream and consumer.
func InstallStreamAndConsumer(t *testing.T, kc *k8s.Clientset, namespace, natsAddress string) {
	data := templateData{
		TestNamespace: namespace,
		NatsAddress:   natsAddress,
	}

	helper.KubectlApplyWithTemplate(t, data, "streamAndConsumerTemplate", streamAndConsumerTemplate)
}
