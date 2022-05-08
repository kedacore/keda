import * as sh from 'shelljs'
import test from 'ava'
import { RabbitMQHelper } from './rabbitmq-helpers'
import { waitForDeploymentReplicaCount} from "./helpers";

const testNamespace = 'rabbitmq-queue-http-test'
const rabbitmqNamespace = 'rabbitmq-http-test'
const queueName = 'hello'
const username = "test-user"
const password = "test-password"
const vhost = "test-vh"
const connectionString = `amqp://${username}:${password}@rabbitmq.${rabbitmqNamespace}.svc.cluster.local/${vhost}`
const messageCount = 500

test.before(t => {
  RabbitMQHelper.installRabbit(t, username, password, vhost, rabbitmqNamespace)

  sh.config.silent = true
  // create deployment
  const httpConnectionString = `http://${username}:${password}@rabbitmq.${rabbitmqNamespace}.svc.cluster.local/${vhost}`

  RabbitMQHelper.createDeployment(t, testNamespace, deployYaml, connectionString, httpConnectionString, queueName)
})

test.serial('Deployment should have 0 replicas on start', async t => {
  t.true(await waitForDeploymentReplicaCount(0, 'test-deployment', testNamespace, 2, 1000), 'replica count should start out as 0')
})

test.serial(`Deployment should scale to 4 with ${messageCount} messages on the queue then back to 0`, async t => {
  RabbitMQHelper.publishMessages(t, testNamespace, connectionString, messageCount, queueName)

  // with messages published, the consumer deployment should start receiving the messages
  t.true(await waitForDeploymentReplicaCount(4, 'test-deployment', testNamespace, 30, 5000))
  t.true(await waitForDeploymentReplicaCount(0, 'test-deployment', testNamespace, 60, 5000))
})

test.after.always.cb('clean up rabbitmq-queue deployment', t => {
  const resources = [
    'scaledobject.keda.sh/test-scaledobject',
    'secret/test-secrets-api',
    'deployment.apps/test-deployment',
  ]

  for (const resource of resources) {
    sh.exec(`kubectl delete ${resource} --namespace ${testNamespace}`)
  }
  sh.exec(`kubectl delete namespace ${testNamespace}`)
  // remove rabbitmq
  RabbitMQHelper.uninstallRabbit(rabbitmqNamespace)
  t.end()
})

const deployYaml = `apiVersion: v1
kind: Secret
metadata:
  name: test-secrets-api
data:
  RabbitApiHost: {{CONNECTION_STRING_BASE64}}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
  labels:
    app: test-deployment
spec:
  replicas: 0
  selector:
    matchLabels:
      app: test-deployment
  template:
    metadata:
      labels:
        app: test-deployment
    spec:
      containers:
      - name: rabbitmq-consumer
        image: ghcr.io/kedacore/tests-rabbitmq
        imagePullPolicy: Always
        command:
          - receive
        args:
          - '{{CONNECTION_STRING}}'
        envFrom:
        - secretRef:
            name: test-secrets-api
---
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: test-scaledobject
spec:
  scaleTargetRef:
    name: test-deployment
  pollingInterval: 5
  cooldownPeriod: 10
  minReplicaCount: 0
  maxReplicaCount: 4
  triggers:
  - type: rabbitmq
    metadata:
      queueName: {{QUEUE_NAME}}
      hostFromEnv: RabbitApiHost
      protocol: http
      queueLength: '50'`
