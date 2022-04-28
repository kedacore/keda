import test from 'ava'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import * as fs from 'fs'
import {createNamespace, waitForDeploymentReplicaCount, waitForRollout} from "./helpers";

const redisNamespace = 'redis-sentinel-streams'
const redisSentinelName = 'redis-sentinel-streams'
const redisSentinelMasterName = 'mymaster'
const redisStatefulSetName = 'redis-sentinel-streams-node'
const redisService = 'redis-sentinel-streams'
const testNamespace = 'redis-sentinel-streams-test'
const redisPassword = 'foobared'
let redisHost = ''
const numMessages = 100

test.before(t => {
    // Deploy Redis Sentinel.
    createNamespace(redisNamespace)
    sh.exec(`helm repo add bitnami https://charts.bitnami.com/bitnami`)

    let sentinelStatus = sh.exec(`helm install --timeout 900s ${redisSentinelName} --namespace ${redisNamespace} --set "sentinel.enabled=true" --set "master.persistence.enabled=false" --set "replica.persistence.enabled=false" --set "global.redis.password=${redisPassword}" bitnami/redis`).code
    t.is(0,
        sentinelStatus,
        'creating a Redis Sentinel setup should work.'
    )

    // Wait for Redis Sentinel to be ready.
    let exitCode = waitForRollout('statefulset', redisStatefulSetName, redisNamespace, 600)
    t.is(0, exitCode, 'expected rollout status for redis to finish successfully')

    // Get Redis Sentinel address.
    redisHost = sh.exec(`kubectl get svc ${redisService} -n ${redisNamespace} -o jsonpath='{.spec.clusterIP}'`)

    // Create test namespace.
    createNamespace(testNamespace)

    // Deploy streams consumer app, scaled object etc.
    const tmpFile = tmp.fileSync()
    const base64Password = Buffer.from(redisPassword).toString('base64')

    fs.writeFileSync(tmpFile.name, redisStreamsDeployYaml.replace('{{REDIS_PASSWORD}}', base64Password).replace('{{REDIS_SENTINEL_PASSWORD}}', base64Password).replace('{{REDIS_SENTINEL_MASTER}}', redisSentinelMasterName).replace('{{REDIS_HOSTS}}', redisHost))
    t.is(
        0,
        sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${testNamespace}`).code,
        'creating a deployment should work..'
    )
})

test.serial('Deployment should have 1 replica on start', t => {

  const replicaCount = sh.exec(
    `kubectl get deployment/redis-streams-consumer --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
  ).stdout
  t.is(replicaCount, '1', 'replica count should start out as 1')
})

test.serial(`Deployment should scale to 5 with ${numMessages} messages and back to 1`, async t => {
  // Publish messages to redis streams.
  const tmpFile = tmp.fileSync()
  fs.writeFileSync(tmpFile.name, producerDeployYaml.replace('{{NUM_MESSAGES}}', numMessages.toString())
    .replace('{{REDIS_SENTINEL_MASTER}}', redisSentinelMasterName)
    .replace('{{REDIS_HOSTS}}', redisHost))
  t.is(
    0,
    sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${testNamespace}`).code,
    'producer job should apply.'
  )

  // Wait for producer job to finish.
  for (let i = 0; i < 60; i++) {
    const succeeded = sh.exec(`kubectl get job  --namespace ${testNamespace} -o jsonpath='{.items[0].status.succeeded}'`).stdout
    if (succeeded == '1') {
      break
    }
    sh.exec('sleep 1s')
  }
  // With messages published, the consumer deployment should start receiving the messages.
  t.true(await waitForDeploymentReplicaCount(5, 'redis-streams-consumer', testNamespace, 30, 10000), 'Replica count should be 5 within 5 minutes')
  t.true(await waitForDeploymentReplicaCount(1, 'redis-streams-consumer', testNamespace, 60, 10000), 'Replica count should be 1 within 10 minutes')
})



test.after.always.cb('clean up deployment', t => {
  const resources = [
    'scaledobject.keda.sh/redis-streams-scaledobject',
    'triggerauthentications.keda.sh/keda-redis-stream-triggerauth',
    'secret/redis-password',
    'deployment/redis-streams-consumer',
    'job/redis-streams-producer',
  ]

  for (const resource of resources) {
    sh.exec(`kubectl delete ${resource} --namespace ${testNamespace}`)
  }
  sh.exec(`kubectl delete namespace ${testNamespace}`)

  sh.exec(`helm delete ${redisSentinelName} --namespace ${redisNamespace}`)
  sh.exec(`kubectl delete namespace ${redisNamespace}`)
  t.end()
})

const redisStreamsDeployYaml = `apiVersion: v1
kind: Secret
metadata:
  name: redis-password
type: Opaque
data:
  password: {{REDIS_PASSWORD}}
  sentinelPassword: {{REDIS_SENTINEL_PASSWORD}}
---
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: keda-redis-stream-triggerauth
spec:
  secretTargetRef:
    - parameter: password
      name: redis-password
      key: password
    - parameter: sentinelPassword
      name: redis-password
      key: sentinelPassword
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis-streams-consumer
spec:
  replicas: 1
  selector:
    matchLabels:
      app: redis-streams-consumer
  template:
    metadata:
      labels:
        app: redis-streams-consumer
    spec:
      containers:
        - name: redis-streams-consumer
          image: ghcr.io/kedacore/tests-redis-sentinel-streams
          command: ["./main"]
          args: ["consumer"]
          imagePullPolicy: Always
          env:
            - name: REDIS_HOSTS
              value: {{REDIS_HOSTS}}
            - name: REDIS_PORTS
              value: "26379"
            - name: REDIS_STREAM_NAME
              value: my-stream
            - name: REDIS_STREAM_CONSUMER_GROUP_NAME
              value: consumer-group-1
            - name: REDIS_SENTINEL_MASTER
              value: {{REDIS_SENTINEL_MASTER}}
            - name: REDIS_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: redis-password
                  key: password
            - name: REDIS_SENTINEL_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: redis-password
                  key: sentinelPassword
---
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: redis-streams-scaledobject
spec:
  scaleTargetRef:
    name: redis-streams-consumer
  pollingInterval: 5
  cooldownPeriod: 10
  minReplicaCount: 1
  maxReplicaCount: 5
  triggers:
    - type: redis-sentinel-streams
      metadata:
        hostsFromEnv: REDIS_HOSTS
        portsFromEnv: REDIS_PORTS
        sentinelMasterFromEnv: REDIS_SENTINEL_MASTER
        stream: my-stream
        consumerGroup: consumer-group-1
        pendingEntriesCount: "10"
      authenticationRef:
        name: keda-redis-stream-triggerauth
`

const producerDeployYaml = `apiVersion: batch/v1
kind: Job
metadata:
  name: redis-streams-producer
spec:
  template:
    spec:
      containers:
      - name: producer
        image: ghcr.io/kedacore/tests-redis-sentinel-streams
        command: ["./main"]
        args: ["producer"]
        imagePullPolicy: Always
        env:
            - name: REDIS_HOSTS
              value: {{REDIS_HOSTS}}
            - name: REDIS_PORTS
              value: "26379"
            - name: REDIS_STREAM_NAME
              value: my-stream
            - name: NUM_MESSAGES
              value: "{{NUM_MESSAGES}}"
            - name: REDIS_SENTINEL_MASTER
              value: "{{REDIS_SENTINEL_MASTER}}"
            - name: REDIS_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: redis-password
                  key: password
            - name: REDIS_SENTINEL_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: redis-password
                  key: sentinelPassword
      restartPolicy: Never
`
