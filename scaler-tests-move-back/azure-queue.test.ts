import * as async from 'async'
import * as azure from 'azure-storage'
import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import test from 'ava'

const defaultNamespace = 'azure-queue-test'
const connectionString = process.env['TEST_STORAGE_CONNECTION_STRING']

test.before(t => {
  if (!connectionString) {
    t.fail('TEST_STORAGE_CONNECTION_STRING environment variable is required for queue tests')
  }

  // poznamka - yaml subor je tu iba ako string, na to aby sa dal deploynut ako
  // apply -f ho musim zapisat do suboru, na to je tu generovanie nahodneho suboru
  // (fileSync), donho sa potom zapise yaml string
  sh.config.silent = true
  const base64ConStr = Buffer.from(connectionString).toString('base64')
  const tmpFile = tmp.fileSync()
  fs.writeFileSync(tmpFile.name, deployYaml.replace('{{CONNECTION_STRING_BASE64}}', base64ConStr))
  sh.exec(`kubectl create namespace ${defaultNamespace}`)
  t.is(
    0,
    sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${defaultNamespace}`).code,
    'creating a deployment should work.'
  )
})

test.serial('Deployment should have 0 replicas on start', t => {
  const replicaCount = sh.exec(
    `kubectl get deployment.apps/test-deployment --namespace ${defaultNamespace} -o jsonpath="{.spec.replicas}"`
  ).stdout
  t.is(replicaCount, '0', 'replica count should start out as 0')
})

test.serial.cb(
  'Deployment should scale to 4 with 10,000 messages on the queue then back to 0',
  t => {
    // add 10,000 messages
    const queueSvc = azure.createQueueService(connectionString)
    queueSvc.messageEncoder = new azure.QueueMessageEncoder.TextBase64QueueMessageEncoder()
    queueSvc.createQueueIfNotExists('queue-name', err => {
      t.falsy(err, 'unable to create queue')
      async.mapLimit(
        Array(10000).keys(),
        200,
        (n, cb) => queueSvc.createMessage('queue-name', `test ${n}`, cb),
        () => {
          let replicaCount = '0'
          for (let i = 0; i < 10 && replicaCount !== '4'; i++) {
            replicaCount = sh.exec(
              `kubectl get deployment.apps/test-deployment --namespace ${defaultNamespace} -o jsonpath="{.spec.replicas}"`
            ).stdout
            if (replicaCount !== '4') {
              sh.exec('sleep 1s')
            }
          }

          t.is('4', replicaCount, 'Replica count should be 4 after 10 seconds')

          for (let i = 0; i < 50 && replicaCount !== '0'; i++) {
            replicaCount = sh.exec(
              `kubectl get deployment.apps/test-deployment --namespace ${defaultNamespace} -o jsonpath="{.spec.replicas}"`
            ).stdout
            if (replicaCount !== '0') {
              sh.exec('sleep 5s')
            }
          }

          t.is('0', replicaCount, 'Replica count should be 0 after 3 minutes')
          t.end()
        }
      )
    })
  }
)

test.after.always.cb('clean up azure-queue deployment', t => {
  const resources = [
    'secret/test-secrets',
    'deployment.apps/test-deployment',
    'scaledobject.keda.k8s.io/test-scaledobject',
  ]

  for (const resource of resources) {
    sh.exec(`kubectl delete ${resource} --namespace ${defaultNamespace}`)
  }
  sh.exec(`kubectl delete namespace ${defaultNamespace}`)

  // delete test queue
  const queueSvc = azure.createQueueService(connectionString)
  queueSvc.deleteQueueIfExists('queue-name', err => {
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
        image: ahmelsayed/queue-consumer:latest
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
apiVersion: keda.k8s.io/v1alpha1
kind: ScaledObject
metadata:
  name: test-scaledobject
  labels:
    deploymentName: test-deployment
spec:
  scaleTargetRef:
    deploymentName: test-deployment
  pollingInterval: 5
  maxReplicaCount: 4
  cooldownPeriod: 10
  triggers:
  - type: azure-queue
    metadata:
      queueName: queue-name
      connection: AzureWebJobsStorage`
