import * as async from 'async'
import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import test from 'ava'
import {waitForRollout} from "./helpers";

const testNamespace = 'prometheus-test'
const prometheusNamespace = 'monitoring'
const prometheusDeploymentFile = 'scalers/prometheus-deployment.yaml'

test.before(t => {
  // install prometheus
  sh.exec(`kubectl create namespace ${prometheusNamespace}`)
  t.is(0, sh.exec(`kubectl apply --namespace ${prometheusNamespace} -f ${prometheusDeploymentFile}`).code, 'creating a Prometheus deployment should work.')
  // wait for prometheus to load
  t.is(0, waitForRollout('deployment', "prometheus-server", prometheusNamespace))

  sh.config.silent = true
  // create deployments - there are two deployments - one triggered with
  // AverageValue metric type, and the other one with Value metric type
  const tmpFile = tmp.fileSync()
  fs.writeFileSync(tmpFile.name, deployYaml)
  sh.exec(`kubectl create namespace ${testNamespace}`)
  t.is(
    0,
    sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${testNamespace}`).code,
    'creating a deployment should work.'
  )
})

test.serial(`Value deployment should scale to 8, average value deployment should only scale to 2`, t => {
  // we use a constant vector so that the current metric value is always 2.

  // for Value metric type, we expect the replica count to double itself every time:
  // desiredReplicas = ceil[currentReplicas * ( currentMetricValue / desiredMetricValue )]
  // = currentReplicas * 2 / 1

  // for AverageValue metric type, we expect the deployment to scale to 2 and stay there,
  // since the currentReplicas isn't part of the equation:
  // desiredReplicas = ceil[currentReplicas * ( (currentMetricValue / currentReplicas) / desiredMetricValue )]
  // = ceil[currentMetricValue / desiredMetricValue] = 2 / 1 = 2
  let averageValueReplicaCount = '0', valueReplicaCount = '0'
  for (let i = 0; i < 60 && (averageValueReplicaCount !== '2' || valueReplicaCount !== '8'); i++) {
    t.log(`Waited ${5 * i} seconds for prometheus-based deployments to scale up`)

    averageValueReplicaCount = sh.exec(
      `kubectl get deployment.apps/average-value-test-app --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
    ).stdout
    valueReplicaCount = sh.exec(
      `kubectl get deployment.apps/value-test-app --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
    ).stdout

    // replica count should always double itself for value metric type
    t.assert(['1', '2', '4', '8'].includes(valueReplicaCount))
    if (averageValueReplicaCount !== '2' || valueReplicaCount !== '8') {
      sh.exec('sleep 10s')
    }
  }

  t.is('2', averageValueReplicaCount, 'Average value replica count should be 2')
  t.is('8', valueReplicaCount, 'Value replica count should be maxed at 8')
})

test.after.always.cb('clean up prometheus deployment', t => {
  const resources = [
    'scaledobject.keda.sh/test-scaledobject',
    'deployment.apps/test-app',
  ]

  for (const resource of resources) {
    sh.exec(`kubectl delete ${resource} --namespace ${testNamespace}`)
  }
  sh.exec(`kubectl delete namespace ${testNamespace}`)

  // uninstall prometheus
  sh.exec(`kubectl delete --namespace ${prometheusNamespace} -f ${prometheusDeploymentFile}`)
  sh.exec(`kubectl delete namespace ${prometheusNamespace}`)

  t.end()
})

const deployYaml = `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: average-value-test-app
  name: average-value-test-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: average-value-test-app
  template:
    metadata:
      labels:
        app: average-value-test-app
    spec:
      containers:
      - name: prom-average-value-test-app
        image: nginx:1.16.1
        imagePullPolicy: IfNotPresent
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: value-test-app
  name: value-test-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: value-test-app
  template:
    metadata:
      labels:
        app: value-test-app
    spec:
      containers:
      - name: prom-value-test-app
        image: nginx:1.16.1
        imagePullPolicy: IfNotPresent
---
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: average-value-test-scaledobject
spec:
  scaleTargetRef:
    name: average-value-test-app
  minReplicaCount: 1
  maxReplicaCount: 8
  pollingInterval: 5
  cooldownPeriod:  10
  triggers:
  - type: prometheus
    metricType: AverageValue
    metadata:
      serverAddress: http://prometheus-server.${prometheusNamespace}.svc
      metricName: two
      threshold: '1'
      query: vector(2)
---
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: value-test-scaledobject
spec:
  scaleTargetRef:
    name: value-test-app
  minReplicaCount: 1
  maxReplicaCount: 8
  pollingInterval: 5
  cooldownPeriod:  10
  triggers:
  - type: prometheus
    metricType: Value
    metadata:
      serverAddress: http://prometheus-server.${prometheusNamespace}.svc
      metricName: two
      threshold: '1'
      query: vector(2)`
