//go:build e2e
// +build e2e

package helper

import (
	h "github.com/kedacore/keda/v2/tests/helper"
)

type JetStreamTemplateData struct {
	NatsNamespace  string
	TestNamespace  string
	NatsAddress    string
	NatsConsumer   string
	NatsStream     string
	StreamReplicas int
	NatsVersion    string
}

const (
	NatsJetStreamName              = "nats"
	NatsJetStreamConsumerName      = "PULL_CONSUMER"
	Natsv2_10JetStreamChartVersion = "2.10.3"
	NatsJetStreamChartVersion      = "0.18.2"
	NatsJetStreamServerVersion     = "2.9.3"
)

type JetStreamDeploymentTemplateData struct {
	TestNamespace                string
	NatsAddress                  string
	NatsConsumer                 string
	NatsStream                   string
	NatsServerMonitoringEndpoint string
	NumberOfMessages             int
}

func GetJetStreamDeploymentTemplateData(
	testNamespace string,
	natsAddress string,
	natsServerMonitoringEndpoint string,
	messagePublishCount int,
) (JetStreamDeploymentTemplateData, []h.Template) {
	return JetStreamDeploymentTemplateData{
			TestNamespace:                testNamespace,
			NatsAddress:                  natsAddress,
			NatsServerMonitoringEndpoint: natsServerMonitoringEndpoint,
			NumberOfMessages:             messagePublishCount,
			NatsConsumer:                 NatsJetStreamConsumerName,
		}, []h.Template{
			{Name: "deploymentTemplate", Config: DeploymentTemplate},
		}
}

const (
	DeleteStreamTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  name: delete-stream
  namespace: {{.TestNamespace}}
spec:
  ttlSecondsAfterFinished: 15
  template:
    spec:
      containers:
      - name: stream
        image: "natsio/nats-box:0.13.2"
        imagePullPolicy: Always
        command: [
          'sh', '-c', 'nats context save local --server {{.NatsAddress}} --select &&
          nats stream rm {{.NatsStream}} -f ;'
        ]
      restartPolicy: OnFailure
  backoffLimit: 4
  `

	StreamAndConsumerTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  name: stream
  namespace: {{.TestNamespace}}
spec:
  ttlSecondsAfterFinished: 15
  template:
    spec:
      containers:
      - name: stream
        image: "natsio/nats-box:0.13.2"
        imagePullPolicy: Always
        command: [
          'sh', '-c', 'nats context save local --server {{.NatsAddress}} --select &&
          nats stream add {{.NatsStream}} --replicas={{.StreamReplicas}} --storage=memory --subjects="ORDERS.*"
                                          --retention=limits --discard=old --max-msgs="-1" --max-msgs-per-subject="-1"
                                          --max-bytes="-1" --max-age="-1" --max-msg-size="-1" --dupe-window=2m
                                          --allow-rollup --no-deny-delete --no-deny-purge &&
          nats consumer add {{.NatsStream}} {{.NatsConsumer}} --pull --deliver=all --ack=explicit --replay=instant
                                                              --filter="" --max-deliver="-1" --max-pending=1000
                                                              --no-headers-only --wait=5s --backoff=none'
        ]
      restartPolicy: OnFailure
  backoffLimit: 4
  `

	DeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sub
  namespace: {{.TestNamespace}}
spec:
  replicas: 0
  selector:
    matchLabels:
      app: sub
  template:
    metadata:
      labels:
        app: sub
    spec:
      containers:
      - name: sub
        image: "ghcr.io/kedacore/tests-nats-jetstream"
        imagePullPolicy: Always
        command:
        - /app
        - consumer
        env:
        - name: NATS_ADDRESS
          value: {{.NatsAddress}}
`

	PublishJobTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  name: pub
  namespace: {{.TestNamespace}}
spec:
  ttlSecondsAfterFinished: 15
  template:
    spec:
      containers:
      - name: pub
        image: "ghcr.io/kedacore/tests-nats-jetstream"
        imagePullPolicy: Always
        command:
        - /app
        - publisher
        env:
        - name: NATS_ADDRESS
          value: {{.NatsAddress}}
        - name: NUM_MESSAGES
          value: "{{.NumberOfMessages}}"
      restartPolicy: OnFailure
  backoffLimit: 4
`

	ActivationPublishJobTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  name: pub
  namespace: {{.TestNamespace}}
spec:
  ttlSecondsAfterFinished: 0
  template:
    spec:
      containers:
      - name: pub
        image: "ghcr.io/kedacore/tests-nats-jetstream"
        imagePullPolicy: Always
        command:
        - /app
        - publisher
        env:
        - name: NATS_ADDRESS
          value: {{.NatsAddress}}
        - name: NUM_MESSAGES
          value: "{{.NumberOfMessages}}"
      restartPolicy: OnFailure
  backoffLimit: 4
`

	ScaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: nats-jetstream-scaledobject
  namespace: {{.TestNamespace}}
spec:
  pollingInterval: 3
  cooldownPeriod: 10
  minReplicaCount: 0
  maxReplicaCount: 2
  scaleTargetRef:
    name: sub
  triggers:
  - type: nats-jetstream
    metadata:
      natsServerMonitoringEndpoint: {{.NatsServerMonitoringEndpoint}}
      account: "$G"
      stream: {{.NatsStream}}
      consumer: {{.NatsConsumer}}
      lagThreshold: "10"
      activationLagThreshold: "15"
  `
)
