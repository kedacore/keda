import * as sh from 'shelljs'
import test from 'ava'
import { createNamespace, waitForDeploymentReplicaCount } from './helpers'
import * as tmp from 'tmp'
import * as fs from 'fs'

const testNamespace = 'metrics-api-trigger-test'
const maxReplicaCount = 2
const endpoint = `http://metrics-entrypoint.${testNamespace}.svc.cluster.local:8080/api`

test.before((t) => {
  sh.config.silent = true
  createDeployment(t)
})

test.serial('Deployment should have 0 replicas on start', async (t) => {
  t.true(
    await waitForDeploymentReplicaCount(0, 'test-deployment', testNamespace, 10, 5000),
    'Replica count should start out as 0'
  )
})

test.serial(
  `Deployment should scale to ${maxReplicaCount} when target value > 5 then back to 0`,
  async (t) => {
    updateMetricValue(10)
    t.true(
      await waitForDeploymentReplicaCount(
        maxReplicaCount,
        'test-deployment',
        testNamespace,
        20,
        5000
      ),
      `Replica count should be ${maxReplicaCount}`
    )

    updateMetricValue(0)

    t.true(
      await waitForDeploymentReplicaCount(0, 'test-deployment', testNamespace, 32, 10000),
      'Replica count should be 0 after 4 minutes'
    )
  }
)

test.after.always.cb('clean up metrics-api deployment', (t) => {
  const resources = [
    'scaledobject.keda.sh/test-scaledobject',
    'triggerauthentication.keda.sh/keda-metric-api-creds',
  ]

  for (const resource of resources) {
    sh.exec(`kubectl delete ${resource} --namespace ${testNamespace}`)
  }
  sh.exec(`kubectl delete namespace ${testNamespace}`)
  t.end()
})

function createDeployment(t) {
  const tmpFile = tmp.fileSync()
  fs.writeFileSync(
    tmpFile.name,
    deployYaml
      .replace('{{METRIC_SERVER_ENDPOINT}}', endpoint + '/basic/value')
      .replace('{{MAX_REPLICA_COUNT}}', maxReplicaCount.toString())
  )

  createNamespace(testNamespace)
  sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${testNamespace}`)
}

function updateMetricValue(value: number) {
  sh.exec(`kubectl delete jobs/update-metric-value --namespace ${testNamespace}`)

  const tmpFile = tmp.fileSync()
  fs.writeFileSync(
    tmpFile.name,
    updateMetricValueYaml
      .replace('{{METRIC_SERVER_ENDPOINT}}', endpoint)
      .replace('{{VALUE}}', value.toString())
  )
  sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${testNamespace}`)
}

const updateMetricValueYaml = `apiVersion: batch/v1
kind: Job
metadata:
  name: update-metric-value
spec:
  template:
    spec:
      containers:
      - name: curl-client
        image: curlimages/curl
        imagePullPolicy: Always
        command: ["curl", "-X", "POST", "{{METRIC_SERVER_ENDPOINT}}/value/{{VALUE}}"]
      restartPolicy: Never`

const deployYaml = `apiVersion: v1
kind: Secret
metadata:
  name: metrics-secrets
stringData:
  username: "user"
  password: "SECRET"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
spec:
  selector:
    matchLabels:
      app: testapp
  replicas: 0
  template:
    metadata:
      labels:
        app: testapp
    spec:
      containers:
      - name: nginx
        image: nginx:1.14.2
        ports:
        - containerPort: 80
---
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: keda-metric-api-creds
spec:
  secretTargetRef:
    - parameter: username
      name: metrics-secrets
      key: username
    - parameter: password
      name: metrics-secrets
      key: password
---
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: metric-api-scaledobject
  labels:
    deploymentName: http-scaled
spec:
  maxReplicaCount: {{MAX_REPLICA_COUNT}}
  scaleTargetRef:
    name: test-deployment
  triggers:
    - type: metrics-api
      metadata:
        targetValue: "5"
        url: "{{METRIC_SERVER_ENDPOINT}}"
        valueLocation: 'value'
        authMode: "basic"
        method: "query"
      authenticationRef:
        name: keda-metric-api-creds
---
apiVersion: v1
kind: Service
metadata:
  name: metrics-entrypoint
spec:
  selector:
    app: metrics
  ports:
  - port: 8080
    targetPort: 8080
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: metrics-deployment
  labels:
    app: metrics
spec:
  replicas: 1
  selector:
    matchLabels:
      app: metrics
  template:
    metadata:
      labels:
        app: metrics
    spec:
      containers:
      - name: metrics
        image: ghcr.io/kedacore/tests-metrics-api:latest
        ports:
        - containerPort: 8080
        imagePullPolicy: Always
        env:
        - name: AUTH_USERNAME
          valueFrom:
            secretKeyRef:
              name: metrics-secrets
              key: username
        - name: AUTH_PASSWORD
          valueFrom:
            secretKeyRef:
              name: metrics-secrets
              key: password`
