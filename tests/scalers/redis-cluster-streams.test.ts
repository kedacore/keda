import test from 'ava'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import * as fs from 'fs'
import { waitForRollout } from "./helpers";

const redisNamespace = 'redis-cluster-streams'
const redisClusterName = 'redis-cluster-streams'
const redisStatefulSetName = 'redis-cluster-streams'
const redisService = 'redis-cluster-streams'
const testNamespace = 'redis-cluster-streams-test'
const redisPassword = 'foobared'
let redisHost = ''
const numMessages = 100

test.before(t => {
    // Deploy Redis cluster.
    sh.exec(`kubectl create namespace ${redisNamespace}`)
    sh.exec(`helm repo add bitnami https://charts.bitnami.com/bitnami`)

    let clusterStatus = sh.exec(`helm install --timeout 600s ${redisClusterName} --namespace ${redisNamespace} --set "global.redis.password=${redisPassword}" bitnami/redis-cluster`).code
    t.is(0,
        clusterStatus,
        'creating a Redis cluster should work.'
    )

    // Wait for Redis cluster to be ready.
    let exitCode = waitForRollout('statefulset', redisStatefulSetName, redisNamespace)
    t.is(0, exitCode, 'expected rollout status for redis to finish successfully')

    // Get Redis cluster address.
    redisHost = sh.exec(`kubectl get svc ${redisService} -n ${redisNamespace} -o jsonpath='{.spec.clusterIP}'`)

    // Create test namespace.
    sh.exec(`kubectl create namespace ${testNamespace}`)

    // Deploy streams consumer app, scaled object etc.
    const tmpFile = tmp.fileSync()
    const base64Password = Buffer.from(redisPassword).toString('base64')

    fs.writeFileSync(tmpFile.name, redisStreamsDeployYaml.replace('{{REDIS_PASSWORD}}', base64Password).replace('{{REDIS_HOSTS}}', redisHost))
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

test.serial(`Deployment should scale to 5 with ${numMessages} messages and back to 1`, t => {
  // Publish messages to redis streams.
  const tmpFile = tmp.fileSync()
  fs.writeFileSync(tmpFile.name, producerDeployYaml.replace('{{NUM_MESSAGES}}', numMessages.toString())
    .replace('{{REDIS_HOSTS}}', redisHost))
  t.is(
    0,
    sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${testNamespace}`).code,
    'producer job should apply.'
  )

  // Wait for producer job to finish.
  for (let i = 0; i < 20; i++) {
    const succeeded = sh.exec(`kubectl get job  --namespace ${testNamespace} -o jsonpath='{.items[0].status.succeeded}'`).stdout
    if (succeeded == '1') {
      break
    }
    sh.exec('sleep 1s')
  }
  // With messages published, the consumer deployment should start receiving the messages.
  let replicaCount = '0'
  for (let i = 0; i < 20 && replicaCount !== '5'; i++) {
    replicaCount = sh.exec(
      `kubectl get deployment/redis-streams-consumer --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
    ).stdout
    t.log('(scale up) replica count is:' + replicaCount)
    if (replicaCount !== '5') {
      sh.exec('sleep 3s')
    }
  }

  t.is('5', replicaCount, 'Replica count should be 5 within 60 seconds')

  for (let i = 0; i < 60 && replicaCount !== '1'; i++) {
    replicaCount = sh.exec(
      `kubectl get deployment/redis-streams-consumer --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
    ).stdout
    t.log('(scale down) replica count is:' + replicaCount)
    if (replicaCount !== '1') {
      sh.exec('sleep 10s')
    }
  }

  t.is('1', replicaCount, 'Replica count should be 1 within 10 minutes')
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

  sh.exec(`helm delete ${redisClusterName} --namespace ${redisNamespace}`)
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
          image: goku321/redis-cluster-streams:v2.5
          command: ["./main"]
          args: ["consumer"]
          imagePullPolicy: Always
          env:
            - name: REDIS_HOSTS
              value: {{REDIS_HOSTS}}
            - name: REDIS_PORTS
              value: "6379"
            - name: REDIS_STREAM_NAME
              value: my-stream
            - name: REDIS_STREAM_CONSUMER_GROUP_NAME
              value: consumer-group-1
            - name: REDIS_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: redis-password
                  key: password
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
    - type: redis-cluster-streams
      metadata:
        hostsFromEnv: REDIS_HOSTS
        portsFromEnv: REDIS_PORTS
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
        image: goku321/redis-cluster-streams:v2.5
        command: ["./main"]
        args: ["producer"]
        imagePullPolicy: Always
        env:
            - name: REDIS_HOSTS
              value: {{REDIS_HOSTS}}
            - name: REDIS_PORTS
              value: "6379"
            - name: REDIS_STREAM_NAME
              value: my-stream
            - name: NUM_MESSAGES
              value: "{{NUM_MESSAGES}}"
            - name: REDIS_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: redis-password
                  key: password
      restartPolicy: Never
`
