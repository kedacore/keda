import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import test from 'ava'
import { PrometheusServer } from './prometheus-server-helpers'
import { createNamespace, waitForDeploymentReplicaCount } from './helpers'

const testNamespace = 'prometheus-test-metric-type'
const prometheusNamespace = 'prometheus-test-monitoring-metric-type'
const loadGeneratorJob = tmp.fileSync()

test.before(async t => {
  // install prometheus
  PrometheusServer.install(t, prometheusNamespace)

  sh.config.silent = true
  // create deployments - there are two deployments - both using the same image but one deployment
  // is directly tied to the KEDA HPA while the other is isolated that can be used for metrics
  // even when the KEDA deployment is at zero - the service points to both deployments
  const tmpFile = tmp.fileSync()
  fs.writeFileSync(tmpFile.name, deployYaml.replace('{{PROMETHEUS_NAMESPACE}}', prometheusNamespace))
  createNamespace(testNamespace)
  t.is(
    0,
    sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${testNamespace}`).code,
    'creating a deployment should work.'
  )
  t.true(await waitForDeploymentReplicaCount(1, 'test-app', testNamespace, 60, 1000), 'test-app replica count should be 1 after 1 minute')

  fs.writeFileSync(loadGeneratorJob.name, generateRequestsYaml.replace('{{NAMESPACE}}', testNamespace))
})

test.serial('Metric type should be "Value"',async t => {
  const scaledObjectMetricType = sh.exec(
    `kubectl get scaledobject.keda.sh/prometheus-scaledobject --namespace ${testNamespace} -o jsonpath="{.spec.triggers[0].metricType}"`
  ).stdout
  const hpaMetricType = sh.exec(
    `kubectl get hpa.v2beta2.autoscaling/keda-hpa-prometheus-scaledobject --namespace ${testNamespace} -o jsonpath="{.spec.metrics[0].external.target.type}"`
  ).stdout

  t.is('Value', scaledObjectMetricType, 'prometheus-scaledobject trigger metric type should be "Value"')
  t.is('Value', hpaMetricType, 'keda-hpa-prometheus-scaledobject metric target type should be "Value"')
})

test.serial('Deployment should have 0 replicas on start', async t => {
  t.true(await waitForDeploymentReplicaCount(0, 'keda-test-app', testNamespace, 60, 1000), 'keda-test-app replica count should be 0 after 1 minute')
})

test.serial(`Deployment should scale to 2 (the max) with HTTP Requests exceeding in the rate`, async t => {
  // generate a large number of HTTP requests (using Apache Bench) that will take some time
  // so prometheus has some time to scrape it
  t.is(
    0,
    sh.exec(`kubectl apply -f ${loadGeneratorJob.name} --namespace ${testNamespace}`).code,
    'creating job should work.'
  )

  t.true(await waitForDeploymentReplicaCount(2, 'keda-test-app', testNamespace, 600, 1000), 'keda-test-app replica count should be 2 after 10 minutes')
})

test.serial(`Deployment should scale to 0`, async t => {
  // Stop the load
  t.is(
    0,
    sh.exec(`kubectl delete -f ${loadGeneratorJob.name} --namespace ${testNamespace}`).code,
    'deleting job should work.'
  )

  t.true(await waitForDeploymentReplicaCount(0, 'keda-test-app', testNamespace, 300, 1000), 'keda-test-app replica count should be 0 after 5 minutes')

})


test.after.always.cb('clean up prometheus deployment', t => {
  const resources = [
    'scaledobject.keda.sh/prometheus-scaledobject',
    'deployment.apps/test-app',
    'deployment.apps/keda-test-app',
    'service/test-app',
    'job/generate-requests',
  ]

  for (const resource of resources) {
    sh.exec(`kubectl delete ${resource} --namespace ${testNamespace}`)
  }
  sh.exec(`kubectl delete namespace ${testNamespace}`)

  // uninstall prometheus
  PrometheusServer.uninstall(prometheusNamespace)

  t.end()
})

const deployYaml = `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: test-app
  name: test-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test-app
  template:
    metadata:
      labels:
        app: test-app
        type: keda-testing
    spec:
      containers:
      - name: prom-test-app
        image: tbickford/simple-web-app-prometheus:a13ade9
        imagePullPolicy: IfNotPresent
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: keda-test-app
  name: keda-test-app
spec:
  replicas: 0
  selector:
    matchLabels:
      app: keda-test-app
  template:
    metadata:
      labels:
        app: keda-test-app
        type: keda-testing
    spec:
      containers:
      - name: prom-test-app
        image: tbickford/simple-web-app-prometheus:a13ade9
        imagePullPolicy: IfNotPresent
---
apiVersion: v1
kind: Service
metadata:
  labels:
    app: test-app
  annotations:
    prometheus.io/scrape: "true"
  name: test-app
spec:
  ports:
  - name: http
    port: 80
    protocol: TCP
    targetPort: 8080
  selector:
    type: keda-testing
---
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: prometheus-scaledobject
spec:
  scaleTargetRef:
    name: keda-test-app
  minReplicaCount: 0
  maxReplicaCount: 2
  pollingInterval: 5
  cooldownPeriod:  10
  triggers:
  - type: prometheus
    metricType: Value
    metadata:
      serverAddress: http://prometheus-server.{{PROMETHEUS_NAMESPACE}}.svc
      metricName: http_requests_total
      threshold: '100'
      query: sum(rate(http_requests_total{app="test-app"}[2m]))`

const generateRequestsYaml = `apiVersion: batch/v1
kind: Job
metadata:
  name: generate-requests
spec:
  template:
    spec:
      containers:
      - image: jordi/ab
        name: test
        command: ["/bin/sh"]
        args: ["-c", "for i in $(seq 1 600);do echo $i;ab -c 5 -n 1000 -v 2 http://test-app.{{NAMESPACE}}.svc/;sleep 1;done"]
      restartPolicy: Never
  activeDeadlineSeconds: 600
  backoffLimit: 5`
