import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import test from 'ava'

const defaultNamespace = 'azure-queue-restore-original-replicas-test'
const connectionString = process.env['TEST_STORAGE_CONNECTION_STRING']

test.before(t => {
  if (!connectionString) {
    t.fail('TEST_STORAGE_CONNECTION_STRING environment variable is required for queue tests')
  }

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

test.serial('Deployment should have 2 replicas on start', t => {
  const replicaCount = sh.exec(
    `kubectl get deployment.apps/test-deployment --namespace ${defaultNamespace} -o jsonpath="{.spec.replicas}"`
  ).stdout
  t.is(replicaCount, '2', 'replica count should start out as 2')
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
  t => {
    let replicaCount = '100'
    for (let i = 0; i < 50 && replicaCount !== '0'; i++) {
      replicaCount = sh.exec(
        `kubectl get deployment.apps/test-deployment --namespace ${defaultNamespace} -o jsonpath="{.spec.replicas}"`
      ).stdout
      if (replicaCount !== '0') {
        sh.exec('sleep 5s')
      }
    }
    t.is('0', replicaCount, 'Replica count should be 0')


    t.is(
      0,
      sh.exec(`kubectl delete scaledobject.keda.sh/test-scaledobject --namespace ${defaultNamespace}`).code,
      'deletion of ScaledObject should work.'
    )

    for (let i = 0; i < 50 && replicaCount !== '2'; i++) {
      replicaCount = sh.exec(
        `kubectl get deployment.apps/test-deployment --namespace ${defaultNamespace} -o jsonpath="{.spec.replicas}"`
      ).stdout
      if (replicaCount !== '2') {
        sh.exec('sleep 5s')
      }
    }
    t.is('2', replicaCount, 'Replica count should be back at orignal 2')
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
      queueName: queue-name
      connectionFromEnv: AzureWebJobsStorage`
