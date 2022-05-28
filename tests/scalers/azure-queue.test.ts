import * as async from 'async'
import * as azure from 'azure-storage'
import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import test from 'ava'
import {createNamespace, waitForDeploymentReplicaCount} from "./helpers";

const defaultNamespace = 'azure-queue-test'
const connectionString = process.env['AZURE_STORAGE_CONNECTION_STRING']
const queueName = 'queue-single-name'

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
  createNamespace(defaultNamespace)
  t.is(
    0,
    sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${defaultNamespace}`).code,
    'creating a deployment should work.'
  )
  t.true(await waitForDeploymentReplicaCount(0, 'test-deployment', defaultNamespace, 60, 1000), 'replica count should be 0 after 1 minute')
})

test.serial(
  'Deployment should scale to 4 with 10,000 messages on the queue then back to 0',
  async t => {
    const queueSvc = azure.createQueueService(connectionString)
    queueSvc.messageEncoder = new azure.QueueMessageEncoder.TextBase64QueueMessageEncoder()
    await async.mapLimit(
      Array(1000).keys(),
      20,
      (n, cb) => queueSvc.createMessage(queueName, `test ${n}`, cb)
    )

    // Scaling out when messages available
    t.true(await waitForDeploymentReplicaCount(1, 'test-deployment', defaultNamespace, 60, 1000), 'replica count should be 1 after 1 minutes')

    queueSvc.clearMessages(queueName, _ => {})

    // Scaling in when no available messages
    t.true(await waitForDeploymentReplicaCount(0, 'test-deployment', defaultNamespace, 300, 1000), 'replica count should be 0 after 5 minute')
  }
)

test.after.always.cb('clean up azure-queue deployment', t => {
  const resources = [
    'scaledobject.keda.sh/test-scaledobject',
    'secret/test-secrets',
    'deployment.apps/test-deployment',
  ]

  for (const resource of resources) {
    sh.exec(`kubectl delete ${resource} --namespace ${defaultNamespace}`)
  }
  sh.exec(`kubectl delete namespace ${defaultNamespace}`)

  // delete test queue
  const queueSvc = azure.createQueueService(connectionString)
  queueSvc.deleteQueueIfExists(queueName, err => {
    t.falsy(err, 'should delete test queue successfully')
    t.end()
  })
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
  replicas: 0
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
        image: ghcr.io/kedacore/tests-azure-queue
        resources:
        ports:
        env:
        - name: FUNCTIONS_WORKER_RUNTIME
          value: node
        - name: AzureWebJobsStorage
          valueFrom:
            secretKeyRef:
              name: test-secrets
              key: AzureWebJobsStorage
---
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: test-scaledobject
spec:
  scaleTargetRef:
    name: test-deployment
  pollingInterval: 5
  minReplicaCount: 0
  maxReplicaCount: 1
  cooldownPeriod: 10
  triggers:
  - type: azure-queue
    metadata:
      queueName: ${queueName}
      connectionFromEnv: AzureWebJobsStorage`
