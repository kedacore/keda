import * as async from 'async'
import * as azure from 'azure-storage'
import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import test from 'ava'

const testNamespace = 'azure-queue-auth-test'
const queueName = 'queue-name'
const connectionString = process.env['TEST_STORAGE_CONNECTION_STRING']

test.before(t => {
  if (!connectionString) {
    t.fail('TEST_STORAGE_CONNECTION_STRING environment variable is required for queue tests')
  }

  sh.config.silent = true
  const base64ConStr = Buffer.from(connectionString).toString('base64')
  const tmpFile = tmp.fileSync()
  fs.writeFileSync(tmpFile.name, deployYaml.replace(/{{CONNECTION_STRING_BASE64}}/g, base64ConStr))
  sh.exec(`kubectl create namespace ${testNamespace}`)
  t.is(
    0,
    sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${testNamespace}`).code,
    'creating a deployment should work.'
  )
})

test.serial('Deployment should have 0 replicas on start', t => {
  const replicaCount = sh.exec(
    `kubectl get deployment.apps/test-deployment --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
  ).stdout
  t.is(replicaCount, '0', 'replica count should start out as 0')
})

test.serial.cb(
  'Deployment should scale with messages on storage defined through trigger auth',
  t => {
    const queueSvc = azure.createQueueService(connectionString)
    queueSvc.messageEncoder = new azure.QueueMessageEncoder.TextBase64QueueMessageEncoder()
    queueSvc.createQueueIfNotExists(queueName, err => {
      t.falsy(err, 'unable to create queue')
      async.mapLimit(
        Array(1000).keys(),
        20,
        (n, cb) => queueSvc.createMessage(queueName, `test ${n}`, cb),
        () => {
          let replicaCount = '0'
          for (let i = 0; i < 10 && replicaCount !== '1'; i++) {
            replicaCount = sh.exec(
              `kubectl get deployment.apps/test-deployment --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
            ).stdout
            if (replicaCount !== '1') {
              sh.exec('sleep 1s')
            }
          }

          t.is('1', replicaCount, 'Replica count should be 1 after 10 seconds')
          queueSvc.deleteQueueIfExists(queueName, err => {
            t.falsy(err, `unable to delete queue ${queueName}`)
            for (let i = 0; i < 50 && replicaCount !== '0'; i++) {
              replicaCount = sh.exec(
                `kubectl get deployment.apps/test-deployment --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
              ).stdout
              if (replicaCount !== '0') {
                sh.exec('sleep 5s')
              }
            }

            t.is('0', replicaCount, 'Replica count should be 0 after 3 minutes')
            t.end()
          })
        }
      )
    })
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
        image: docker.io/kedacore/tests-azure-queue:824031e
        resources:
        ports:
        env:
        - name: FUNCTIONS_WORKER_RUNTIME
          value: node
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
  maxReplicaCount: 1
  cooldownPeriod: 10
  triggers:
  - type: azure-queue
    authenticationRef:
      name: azure-queue-auth
    metadata:
      queueName: ${queueName}
`
