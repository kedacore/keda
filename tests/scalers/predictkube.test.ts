import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import test from 'ava'
import { PrometheusServer } from './prometheus-server-helpers'

const predictkubeApiKey = process.env['PREDICTKUBE_API_KEY']
const testNamespace = 'predictkube-test'
const prometheusNamespace = 'predictkube-test-monitoring'

test.before(t => {
    // install prometheus
    PrometheusServer.install(t, prometheusNamespace)

    sh.config.silent = true
    // create deployments - there are two deployments - both using the same image but one deployment
    // is directly tied to the KEDA HPA while the other is isolated that can be used for metrics
    // even when the KEDA deployment is at zero - the service points to both deployments
    const tmpFile = tmp.fileSync()
    fs.writeFileSync(tmpFile.name, deployYaml
        .replace('{{PREDICTKUBE_API_KEY}}', Buffer.from(predictkubeApiKey).toString('base64'))
        .replace('{{PROMETHEUS_NAMESPACE}}', prometheusNamespace)
    )
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
        t.log(`Waited ${10 * i} seconds for predictkube-based deployments to scale up`)
        const jobLogs = sh.exec(`kubectl logs -l job-name=generate-requests -n ${testNamespace}`).stdout
        t.log(`Logs from the generate requests: ${jobLogs}`)

        replicaCount = sh.exec(
            `kubectl get deployment.apps/keda-test-app --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
        ).stdout
        if (replicaCount !== '5') {
            sh.exec('sleep 10s')
        }
    }

    t.is('5', replicaCount, 'Replica count should be maxed at 5')

    for (let i = 0; i < 60 && replicaCount !== '0'; i++) {
        t.log(`Waited ${5 * i} seconds for predictkube-based deployments to scale down`)
        replicaCount = sh.exec(
            `kubectl get deployment.apps/keda-test-app --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
        ).stdout
        if (replicaCount !== '0') {
            sh.exec('sleep 10s')
        }
    }

    t.is('0', replicaCount, 'Replica count should be 0 after 3 minutes')
})

test.after.always.cb('clean up predictkube deployment', t => {
    const resources = [
        'scaledobject.keda.sh/predictkube-scaledobject',
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
kind: TriggerAuthentication
metadata:
  name: predictkube-trigger
spec:
  secretTargetRef:
  - parameter: apiKey
    name: predictkube-secret
    key: apiKey
---
apiVersion: v1
kind: Secret
metadata:
  name: predictkube-secret
type: Opaque
data:
  apiKey: {{PREDICTKUBE_API_KEY}}
---
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: predictkube-scaledobject
spec:
  scaleTargetRef:
    name: keda-test-app
  minReplicaCount: 0
  maxReplicaCount: 5
  pollingInterval: 5
  cooldownPeriod:  10
  triggers:
  - type: predictkube
    metadata:
      predictHorizon: "2h"
      historyTimeWindow: "7d"
      prometheusAddress: http://prometheus-server.{{PROMETHEUS_NAMESPACE}}.svc
      threshold: '100'
      query: sum(rate(http_requests_total{app="test-app"}[2m]))
      queryStep: "2m"
    authenticationRef:
      name: predictkube-trigger`

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
