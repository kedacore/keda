import test from 'ava'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import * as fs from 'fs'

const redisNamespace = 'redis-ns'
const testNamespace = 'redis-streams-ns'
const redisDeploymentName = 'redis'
const redisPassword = 'foobared'
const redisHost = `redis-service.${redisNamespace}.svc.cluster.local:6379`
const numMessages = 100

test.before(t => {
  // setup Redis
  sh.exec(`kubectl create namespace ${redisNamespace}`)

  const tmpFile1 = tmp.fileSync()
  fs.writeFileSync(tmpFile1.name, redisDeployYaml.replace('{{REDIS_PASSWORD}}', redisPassword))

  t.is(0, sh.exec(`kubectl apply --namespace ${redisNamespace} -f ${tmpFile1.name}`).code, 'creating a Redis deployment should work.')

  // wait for redis to be ready
  let redisReplicaCount = '0'
  for (let i = 0; i < 30; i++) {
    redisReplicaCount = sh.exec(`kubectl get deploy/${redisDeploymentName} -n ${redisNamespace} -o jsonpath='{.spec.replicas}'`).stdout
    if (redisReplicaCount != '1') {
      sh.exec('sleep 2s')
    }
  }
  t.is('1', redisReplicaCount, 'Redis is not in a ready state')

  sh.exec(`kubectl create namespace ${testNamespace}`)

  // deploy streams consumer app, scaled object etc.
  const tmpFile = tmp.fileSync()
  const base64Password = Buffer.from(redisPassword).toString('base64')

  fs.writeFileSync(tmpFile.name, redisStreamsDeployYaml.replace('{{REDIS_PASSWORD}}', base64Password).replace('{{REDIS_HOST}}', redisHost))
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
  // publish messages
  const tmpFile = tmp.fileSync()
  fs.writeFileSync(tmpFile.name, producerDeployYaml.replace('{{NUM_MESSAGES}}', numMessages.toString())
    .replace('{{REDIS_HOST}}', redisHost))
  t.is(
    0,
    sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${testNamespace}`).code,
    'producer job should apply.'
  )

  // wait for the producer job to complete
  for (let i = 0; i < 20; i++) {
    const succeeded = sh.exec(`kubectl get job  --namespace ${testNamespace} -o jsonpath='{.items[0].status.succeeded}'`).stdout
    if (succeeded == '1') {
      break
    }
    sh.exec('sleep 1s')
  }
  // with messages published, the consumer deployment should start receiving the messages
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
    'secret/redis-password',
    'deployment/redis-streams-consumer',
    'scaledobject.keda.k8s.io/redis-streams-scaledobject',
    'triggerauthentications.keda.k8s.io/keda-redis-stream-triggerauth'
  ]

  for (const resource of resources) {
    sh.exec(`kubectl delete ${resource} --namespace ${testNamespace}`)
  }
  sh.exec(`kubectl delete namespace ${testNamespace}`)

  sh.exec(`kubectl delete namespace ${redisNamespace}`)
  t.end()
})

const redisDeployYaml = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis
spec:
  selector:
    matchLabels:
      app: redis
  replicas: 1
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
      - name: master
        image: redis
        command: ["redis-server", "--requirepass", "{{REDIS_PASSWORD}}"]
        ports:
        - containerPort: 6379
---
apiVersion: v1
kind: Service
metadata:
  name: redis-service
  labels:
    app: redis
spec:
  ports:
  - port: 6379
    targetPort: 6379
  selector:
    app: redis
`

const redisStreamsDeployYaml = `apiVersion: v1
kind: Secret
metadata:
  name: redis-password
type: Opaque
data:
  password: {{REDIS_PASSWORD}}
---
apiVersion: keda.k8s.io/v1alpha1
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
          image: abhirockzz/redis-streams-consumer
          imagePullPolicy: Always
          env:
            - name: REDIS_HOST
              value: {{REDIS_HOST}}
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
apiVersion: keda.k8s.io/v1alpha1
kind: ScaledObject
metadata:
  name: redis-streams-scaledobject
  labels:
    deploymentName: redis-streams-consumer
spec:
  scaleTargetRef:
    deploymentName: redis-streams-consumer
  pollingInterval: 5
  cooldownPeriod: 10
  minReplicaCount: 1
  maxReplicaCount: 5
  triggers:
    - type: redis-streams
      metadata:
        address: REDIS_HOST
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
        image: abhirockzz/redis-streams-producer
        imagePullPolicy: Always
        env:
            - name: REDIS_HOST
              value: {{REDIS_HOST}}
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