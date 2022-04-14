import * as azure from 'azure-storage'
import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import test from 'ava'
import {createNamespace, waitForDeploymentReplicaCount} from "./helpers";

const defaultNamespace = 'azure-queue-restore-original-replicas-test'
const queueName = 'queue-name-restore'
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
  createNamespace(defaultNamespace)
  t.is(
    0,
    sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${defaultNamespace}`).code,
    'creating a deployment should work.'
  )
})

test.serial('Deployment should have 2 replicas on start', async t => {
  t.true(await waitForDeploymentReplicaCount(2, 'test-deployment', defaultNamespace, 15, 1000), 'replica count should be 2 after 15 seconds')
})

test.serial('Creating ScaledObject should work', t => {
  const tmpFile = tmp.fileSync()
  fs.writeFileSync(tmpFile.name, scaledObjectYaml)

  t.is(
    0,
    sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${defaultNamespace}`).code,
    'creating a ScaledObject should work.'
  )
})


test.serial(
  'Deployment should scale to 0 and then shold be back to 2 after deletion of ScaledObject',
  async t => {
    t.true(await waitForDeploymentReplicaCount(0, 'test-deployment', defaultNamespace, 120, 1000), 'replica count should be 0 after 2 minutes')

    t.is(
      0,
      sh.exec(`kubectl delete scaledobject.keda.sh/test-scaledobject --namespace ${defaultNamespace}`).code,
      'deletion of ScaledObject should work.'
    )

    t.true(await waitForDeploymentReplicaCount(2, 'test-deployment', defaultNamespace, 120, 1000), 'replica count should be 2 after 2 minutes')
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
  replicas: 2
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
              key: AzureWebJobsStorage`


const scaledObjectYaml = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: test-scaledobject
spec:
  advanced:
    restoreToOriginalReplicaCount: true
  scaleTargetRef:
    name: test-deployment
  pollingInterval: 5
  maxReplicaCount: 4
  cooldownPeriod: 10
  triggers:
  - type: azure-queue
    metadata:
      queueName: ${queueName}
      connectionFromEnv: AzureWebJobsStorage`
