import * as async from 'async'
import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import test from 'ava'

const testNamespace = 'prometheus-test'
const prometheusNamespace = 'monitoring'
const prometheusDeploymentFile = 'scalers/prometheus-deployment.yaml'

test.before(t => {
  // install prometheus
  sh.exec(`kubectl create namespace ${prometheusNamespace}`)
  t.is(0, sh.exec(`kubectl apply --namespace ${prometheusNamespace} -f ${prometheusDeploymentFile}`).code, 'creating a Prometheus deployment should work.')
  // wait for prometheus to load
  let prometheusReadyReplicaCount = '0'
  for (let i = 0; i < 30; i++) {
    prometheusReadyReplicaCount = sh.exec(`kubectl get deploy/prometheus-server -n ${prometheusNamespace} -o jsonpath='{.status.readyReplicas}'`).stdout
    if (prometheusReadyReplicaCount != '1') {
      sh.exec('sleep 2s')
    }
  }
  t.is('1', prometheusReadyReplicaCount, 'Prometheus is not in a ready state')

  sh.config.silent = true
  // create deployments - there are two deployments - both using the same image but one deployment
  // is directly tied to the KEDA HPA while the other is isolated that can be used for metrics
  // even when the KEDA deployment is at zero - the service points to both deployments
  const tmpFile = tmp.fileSync()
  fs.writeFileSync(tmpFile.name, deployYaml.replace('{{PROMETHEUS_NAMESPACE}}', prometheusNamespace))
  sh.exec(`kubectl create namespace ${testNamespace}`)
  t.is(
    0,
    sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${testNamespace}`).code,
    'creating a deployment should work.'
  )
  for (let i = 0; i < 10; i++) {
    const readyReplicaCount = sh.exec(`kubectl get deployment.apps/test-app --namespace ${testNamespace} -o jsonpath="{.status.readyReplicas}`).stdout
    if (readyReplicaCount != '1') {
      sh.exec('sleep 2s')
    }
  }
})

test.serial('Deployment should have 0 replicas on start', t => {
  const replicaCount = sh.exec(
    `kubectl get deployment.apps/keda-test-app --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
  ).stdout
  t.is(replicaCount, '0', 'replica count should start out as 0')
})

test.serial(`Deployment should scale to 5 (the max) with HTTP Requests exceeding in the rate then back to 0`, t => {
  // generate a large number of HTTP requests (using Apache Bench) that will take some time
  // so prometheus has some time to scrape it
  const tmpFile = tmp.fileSync()
  fs.writeFileSync(tmpFile.name, generateRequestsYaml.replace('{{NAMESPACE}}', testNamespace))
  t.is(
    0,
    sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${testNamespace}`).code,
    'creating job should work.'
  )

  t.is(
    '1',
    sh.exec(
      `kubectl get deployment.apps/test-app --namespace ${testNamespace} -o jsonpath="{.status.readyReplicas}"`
    ).stdout,
    'There should be 1 replica for the test-app deployment'
  )

  // keda based deployment should start scaling up with http requests issued
  let replicaCount = '0'
  for (let i = 0; i < 60 && replicaCount !== '5'; i++) {
    t.log(`Waited ${5 * i} seconds for prometheus-based deployments to scale up`)
    const jobLogs = sh.exec(`kubectl logs -l job-name=generate-requests -n ${testNamespace}`).stdout
    t.log(`Logs from the generate requests: ${jobLogs}`)

    replicaCount = sh.exec(
      `kubectl get deployment.apps/keda-test-app --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
    ).stdout
    if (replicaCount !== '5') {
      sh.exec('sleep 5s')
    }
  }

  t.is('5', replicaCount, 'Replica count should be maxed at 5')

  for (let i = 0; i < 50 && replicaCount !== '0'; i++) {
    replicaCount = sh.exec(
      `kubectl get deployment.apps/keda-test-app --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
    ).stdout
    if (replicaCount !== '0') {
      sh.exec('sleep 5s')
    }
  }

  t.is('0', replicaCount, 'Replica count should be 0 after 3 minutes')
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
  sh.exec(`kubectl delete --namespace ${prometheusNamespace} -f ${prometheusDeploymentFile}`)
  sh.exec(`kubectl delete namespace ${prometheusNamespace}`)

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
  maxReplicaCount: 5
  pollingInterval: 5
  cooldownPeriod:  10
  triggers:
  - type: prometheus
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
        args: ["-c", "for i in $(seq 1 60);do echo $i;ab -c 5 -n 1000 -v 2 http://test-app.{{NAMESPACE}}.svc/;sleep 1;done"]
      restartPolicy: Never
  activeDeadlineSeconds: 120
  backoffLimit: 2`
