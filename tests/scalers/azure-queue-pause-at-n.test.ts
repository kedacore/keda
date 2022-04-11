import * as async from 'async'
import * as fs from 'fs'
import * as azure from 'azure-storage'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import test from 'ava'
import {createNamespace, waitForDeploymentReplicaCount} from "./helpers";

const deploymentFile = tmp.fileSync()
const scaledObjectFile = tmp.fileSync()
const scaledObjectFileWithAnnotation = tmp.fileSync()
const pauseCount = 5

const testNamespace = 'pause-at-n-test'
const queueName = 'paused-replicas-at-n-queue-name'
const connectionString = process.env['AZURE_STORAGE_CONNECTION_STRING']

test.before(async t => {
  if (!connectionString) {
    t.fail('AZURE_STORAGE_CONNECTION_STRING environment variable is required for queue tests')
  }

  const createQueueAsync = () => new Promise((resolve, _) => {
    const queueSvc = azure.createQueueService(connectionString)
    queueSvc.messageEncoder = new azure.QueueMessageEncoder.TextBase64QueueMessageEncoder()
    queueSvc.createQueueIfNotExists(queueName, _ => {
      resolve(undefined);
    })
  })
  await createQueueAsync()

  sh.config.silent = true
  const base64ConStr = Buffer.from(connectionString).toString('base64')
  const tmpFile = tmp.fileSync()
  fs.writeFileSync(tmpFile.name, deployYaml.replace('{{CONNECTION_STRING_BASE64}}', base64ConStr))
  createNamespace(testNamespace)
  t.is(
    0,
    sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${testNamespace}`).code,
    'creating a deployment should work.'
  )
  t.true(await waitForDeploymentReplicaCount(1, 'test-deployment', testNamespace, 60, 1000), 'replica count should be 1 after 1 minute')
})

test.serial(`Creating ScaledObject should work`, async t => {
  fs.writeFileSync(scaledObjectFile.name, scaledObjectYaml)
  t.is(
    0,
    sh.exec(`kubectl apply -f ${scaledObjectFile.name} --namespace ${testNamespace}`).code,
    'Updating ScaledObject should work.'
  )
})

test.serial(
  'Deployment should scale up to 2 - to minReplicaCount',
  async t => {
    t.true(await waitForDeploymentReplicaCount(2, 'test-deployment', testNamespace, 60, 3000), 'replica count should be 2 after 3 minutes')
  }
)

test.serial.cb(
  'Deployment should scale up with messages on storage',
  t => {
    const queueSvc = azure.createQueueService(connectionString)
    queueSvc.messageEncoder = new azure.QueueMessageEncoder.TextBase64QueueMessageEncoder()
    async.mapLimit(
      Array(2000).keys(),
      20,
      (n, cb) => queueSvc.createMessage(queueName, `test ${n}`, cb),
      async () => {
         // Scaling out when messages available
        t.true(await waitForDeploymentReplicaCount(4, 'test-deployment', testNamespace, 60, 4000), 'replica count should be 4 after 4 minutes')
        queueSvc.clearMessages(queueName, _ => {})
        t.end()
      }
    )
  }
)

test.serial(`Updsating ScaledObject (with annotation) should work`, async t => {
  fs.writeFileSync(scaledObjectFileWithAnnotation.name, scaledObjectYamlWithAnnotation.
    replace('{{PAUSED_REPLICA_COUNT}}', pauseCount.toString()))
  t.is(
    0,
    sh.exec(`kubectl apply -f ${scaledObjectFileWithAnnotation.name} --namespace ${testNamespace}`).code,
    'Updating ScaledObject should work.'
  )
})

test.serial(
  'Deployment should scale to 5 - to pausedReplicaCount',
  async t => {
    t.true(await waitForDeploymentReplicaCount(5, 'test-deployment', testNamespace, 60, 1000), 'replica count should be 5 after 1 minute')
  }
)

test.serial(`Updating ScaledObject (delete annotation) should work`, async t => {
  fs.writeFileSync(scaledObjectFile.name, scaledObjectYaml)
  t.is(
    0,
    sh.exec(`kubectl apply -f ${scaledObjectFile.name} --namespace ${testNamespace}`).code,
    'Updating ScaledObject should work.'
  )
})

test.serial(
  'Deployment should scale scale down',
  async t => {
    t.true(await waitForDeploymentReplicaCount(4, 'test-deployment', testNamespace, 60, 1000), 'replica count should scale dwon after 1 minute')
  }
)

test.after.always.cb('clean up workload test related deployments', t => {
  const resources = [
    'deployment.apps/test-deployment',
    'scaledobject.keda.sh/pause-scaledobject',
    'secret/test-secrets',
  ]
  for (const resource of resources) {
    sh.exec(`kubectl delete ${resource} --namespace ${testNamespace}`)
  }
  sh.exec(`kubectl delete namespace ${testNamespace}`)
  t.end()
})

const deployYaml = `apiVersion: v1
kind: Secret
metadata:
  name: test-secrets
  labels:
data:
  AzureWebJobsStorage: {{CONNECTION_STRING_BASE64}}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
  labels:
    app: test-deployment
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test-deployment
  template:
    metadata:
      name:
      namespace:
      labels:
        app: test-deployment
    spec:
      containers:
      - name: test-deployment
        image: docker.io/kedacore/tests-azure-queue:824031e
        resources:
        ports:
        env:
        - name: FUNCTIONS_WORKER_RUNTIME
          value: node
        - name: AzureWebJobsStorage
          valueFrom:
            secretKeyRef:
              name: test-secrets
              key: AzureWebJobsStorage`

const scaledObjectYaml = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: pause-scaledobject
spec:
  scaleTargetRef:
    name: test-deployment
  pollingInterval: 5
  minReplicaCount: 2
  maxReplicaCount: 4
  cooldownPeriod: 10
  triggers:
  - type: azure-queue
    metadata:
      queueName: ${queueName}
      connectionFromEnv: AzureWebJobsStorage`

const scaledObjectYamlWithAnnotation = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: pause-scaledobject
  annotations:
    autoscaling.keda.sh/paused-replicas: "{{PAUSED_REPLICA_COUNT}}"
spec:
  scaleTargetRef:
    name: test-deployment
  pollingInterval: 5
  minReplicaCount: 2
  maxReplicaCount: 4
  cooldownPeriod: 10
  triggers:
  - type: azure-queue
    metadata:
      queueName: ${queueName}
      connectionFromEnv: AzureWebJobsStorage`
