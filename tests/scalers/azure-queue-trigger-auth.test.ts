import * as async from 'async'
import * as azure from 'azure-storage'
import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import test from 'ava'
import {createNamespace, waitForDeploymentReplicaCount} from "./helpers";

const testNamespace = 'azure-queue-auth-test'
const queueName = 'queue-name-trigger'
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
  fs.writeFileSync(tmpFile.name, deployYaml.replace(/{{CONNECTION_STRING_BASE64}}/g, base64ConStr))
  createNamespace(testNamespace)
  t.is(
    0,
    sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${testNamespace}`).code,
    'creating a deployment should work.'
  )
  t.true(await waitForDeploymentReplicaCount(0, 'test-deployment', testNamespace, 60, 1000), 'replica count should be 0 after 1 minute')
})

test.serial(
  'Deployment should scale with messages on storage defined through trigger auth',
  async t => {
    const queueSvc = azure.createQueueService(connectionString)
    queueSvc.messageEncoder = new azure.QueueMessageEncoder.TextBase64QueueMessageEncoder()
    await async.mapLimit(
      Array(1000).keys(),
      20,
      (n, cb) => queueSvc.createMessage(queueName, `test ${n}`, cb)
    )

    // Scaling out when messages available
    t.true(await waitForDeploymentReplicaCount(1, 'test-deployment', testNamespace, 60, 1000), 'replica count should be 1 after 1 minute')

    queueSvc.clearMessages(queueName, _ => {})

    // Scaling in when no available messages
    t.true(await waitForDeploymentReplicaCount(0, 'test-deployment', testNamespace, 300, 1000), 'replica count should be 0 after 5 minute')
  }
)

test.after.always.cb('clean up azure-queue deployment', t => {
  const resources = [
    'scaledobject.keda.sh/test-scaledobject',
    'triggerauthentications.keda.sh/azure-queue-auth',
    'secret/test-auth-secrets',
    'deployment.apps/test-deployment',
  ]

  for (const resource of resources) {
    sh.exec(`kubectl delete ${resource} --namespace ${testNamespace}`)
  }
  sh.exec(`kubectl delete namespace ${testNamespace}`)

  // delete test queue
  const queueSvc = azure.createQueueService(connectionString)
  queueSvc.deleteQueueIfExists(queueName, err => {
    t.falsy(err, 'should delete test queue successfully')
    t.end()
  })
})

const deployYaml = `apiVersion: apps/v1
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
        image: ghcr.io/kedacore/tests-azure-queue
        resources:
        ports:
        env:
        - name: FUNCTIONS_WORKER_RUNTIME
          value: node
        - name: AzureWebJobsStorage
          valueFrom:
            secretKeyRef:
              name: test-auth-secrets
              key: connectionString
---
apiVersion: v1
kind: Secret
metadata:
  name: test-auth-secrets
  labels:
data:
  connectionString: {{CONNECTION_STRING_BASE64}}
---
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: azure-queue-auth
spec:
  secretTargetRef:
  - parameter: connection
    name: test-auth-secrets
    key: connectionString
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
  maxReplicaCount: 1
  triggers:
  - type: azure-queue
    authenticationRef:
      name: azure-queue-auth
    metadata:
      queueName: ${queueName}`
