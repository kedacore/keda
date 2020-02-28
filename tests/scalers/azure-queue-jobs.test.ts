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

  sh.config.silent = true
  const base64ConStr = Buffer.from(connectionString).toString('base64')
  const tmpFile = tmp.fileSync()
  fs.writeFileSync(tmpFile.name, jobYaml.replace('{{CONNECTION_STRING_BASE64}}', base64ConStr))
  sh.exec(`kubectl create namespace ${defaultNamespace}`)
  t.is(
    0,
    sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${defaultNamespace}`).code,
    'creating a job scaledObject should work.'
  )
})

test.serial('Job scaledObject should have 0 job on start', t => {
  const replicaCount = sh.exec(
    `kubectl get jobs --namespace ${defaultNamespace} -o jsonpath="{range .items[*]}{.spec.completions}{"\n"}{end}" | wc -l`
  ).stdout.trim()
  t.is(replicaCount, '0', 'replica count should start out as 0')
})

test.serial.cb(
  'Deployment should scale to 5 with 10 messages on the queue then complete the 10',
  t => {
    // add 5 messages
    const queueSvc = azure.createQueueService(connectionString)
    queueSvc.messageEncoder = new azure.QueueMessageEncoder.TextBase64QueueMessageEncoder()
    queueSvc.createQueueIfNotExists('queue-name', err => {
      t.falsy(err, 'unable to create queue')
      async.mapLimit(
        Array(10).keys(),
        2,
        (n, cb) => queueSvc.createMessage('queue-name', `test ${n}`, cb),
        () => {
          let replicaCount = '0'
          for (let i = 0; i < 10 && replicaCount !== '5'; i++) {
            replicaCount = sh.exec(
              `kubectl get jobs --namespace ${defaultNamespace} -o jsonpath="{range .items[*]}{.spec.completions}{"\n"}{end}" | wc -l`
            ).stdout.trim()
            if (replicaCount !== '5') {
              sh.exec('sleep 1s')
            }
          }

          t.is('5', replicaCount, 'Job count should be 5 after 10 seconds')

          for (let i = 0; i < 50 && replicaCount !== '10'; i++) {
            replicaCount = sh.exec(
              `kubectl get jobs --namespace ${defaultNamespace} -o jsonpath="{range .items[*]}{.spec.completions}{"\n"}{end}" | grep 1 | wc -l`
            ).stdout
            if (replicaCount !== '10') {
              sh.exec('sleep 5s')
            }
          }

          t.is('10', replicaCount, 'Job Completion count should be 10 after 3 minutes')
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

const jobYaml = `apiVersion: v1
kind: Secret
metadata:
  name: test-secrets
  labels:
data:
  AzureWebJobsStorage: {{CONNECTION_STRING_BASE64}}
---
apiVersion: keda.k8s.io/v1alpha1
kind: ScaledObject
metadata:
  name: test-scaledobject
spec:
  scaleType: job
  pollingInterval: 5
  triggers:
  - type: azure-queue
    metadata:
      queueName: queue-name
      connection: AzureWebJobsStorage
      queueLength: '5'
  jobTargetRef:
    parallelism: 1
    completions: 1
    activeDeadlineSeconds: 3600
    backoffLimit: 2
    template:
      spec:
        containers:
        - name: test-job
        image: thomaslamure/job-queue-consumer:latest
        env:
        - name: CompletionTime
          value: 60
        - name: QueueReads
          value: 1
        - name: AzureQueueName
          value: queue-name
        - name: AzureWebJobsStorage
          valueFrom:
            secretKeyRef:
              name: test-secrets
              key: AzureWebJobsStorage
`
