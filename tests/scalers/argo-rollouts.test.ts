import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import test from 'ava'
import {sleep, waitForRollout} from "./helpers";

const testNamespace = 'argo-rollouts-test'
const prometheusNamespace = 'argo-monitoring'
const prometheusDeploymentFile = 'scalers/prometheus-deployment.yaml'
const argoRolloutsNamespace = 'argo-rollouts'
const argoRolloutsYamlFile = tmp.fileSync()

test.before(async t => {
  // install prometheus
  sh.exec(`kubectl create namespace ${prometheusNamespace}`)
  t.is(0, sh.exec(`kubectl apply --namespace ${prometheusNamespace} -f ${prometheusDeploymentFile}`).code, 'creating a Prometheus deployment should work.')
  // wait for prometheus to load
  t.is(0, waitForRollout('deployment', "prometheus-server", prometheusNamespace))

  // install argo-rollouts
  sh.exec(`kubectl create namespace ${argoRolloutsNamespace}`)
  sh.exec(`curl -L https://raw.githubusercontent.com/argoproj/argo-rollouts/stable/manifests/install.yaml > ${argoRolloutsYamlFile.name}`)
	t.is(
		0,
		sh.exec(`kubectl apply -f ${argoRolloutsYamlFile.name} --namespace ${argoRolloutsNamespace}`).code,
		'Deploying argo-rollouts should work.'
	)

  sh.config.silent = true
  // create rollouts - there are two rollouts - both using the same image but one rollout
  // is directly tied to the KEDA HPA while the other is isolated that can be used for metrics
  // even when the KEDA deployment is at zero - the service points to both rollouts
  const tmpFile = tmp.fileSync()
  fs.writeFileSync(tmpFile.name, rollout.replace('{{PROMETHEUS_NAMESPACE}}', prometheusNamespace))
  sh.exec(`kubectl create namespace ${testNamespace}`)
  t.is(
    0,
    sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${testNamespace}`).code,
    'creating a rollouts should work.'
  )
  for (let i = 0; i < 10; i++) {
    const readyReplicaCount = sh.exec(`kubectl get rollouts.argoproj.io/test-app --namespace ${testNamespace} -o jsonpath="{.status.readyReplicas}"`).stdout
    if (readyReplicaCount != '1') {
      await sleep(2000)
    }
  }
})

test.serial('Rollouts should have 0 replicas on start', t => {
  const replicaCount = sh.exec(
    `kubectl get rollouts.argoproj.io/keda-test-app --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
  ).stdout
  t.is(replicaCount, '0', 'replica count should start out as 0')
})

test.serial(`Rollouts should scale to 5 (the max) with HTTP Requests exceeding in the rate then back to 0`, async t => {
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
    '2',
    sh.exec(
      `kubectl get rollouts.argoproj.io/test-app --namespace ${testNamespace} -o jsonpath="{.status.readyReplicas}"`
    ).stdout,
    'There should be 2 replica for the test-app rollout'
  )

  // keda based rollout should start scaling up with http requests issued
  let replicaCount = '0'
  for (let i = 0; i < 60 && replicaCount !== '5'; i++) {
    t.log(`Waited ${5 * i} seconds for prometheus-based rollout to scale up`)
    const jobLogs = sh.exec(`kubectl logs -l job-name=generate-requests -n ${testNamespace}`).stdout
    t.log(`Logs from the generate requests: ${jobLogs}`)

    replicaCount = sh.exec(
      `kubectl get rollouts.argoproj.io/keda-test-app --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
    ).stdout
    if (replicaCount !== '5') {
      await sleep(5000)
    }
  }

  t.is('5', replicaCount, 'Replica count should be maxed at 5')

  for (let i = 0; i < 50 && replicaCount !== '0'; i++) {
    replicaCount = sh.exec(
      `kubectl get rollouts.argoproj.io/keda-test-app --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
    ).stdout
    if (replicaCount !== '0') {
      await sleep(5000)
    }
  }

  t.is('0', replicaCount, 'Replica count should be 0 after 3 minutes')
})

test.after.always.cb('clean up argo-rollouts testing deployment', t => {
  const resources = [
    'scaledobject.keda.sh/prometheus-scaledobject',
    'rollouts.argoproj.io/test-app',
    'rollouts.argoproj.io/keda-test-app',
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

  // uninstall argo-rollouts
  sh.exec(`kubectl delete --namespace ${argoRolloutsNamespace} -f ${argoRolloutsYamlFile}`)
  sh.exec(`kubectl delete namespace ${argoRolloutsNamespace}`)

  t.end()
})

const rollout = `apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  labels:
    app: test-app
  name: test-app
spec:
  replicas: 2
  strategy:
    canary:
      steps:
      - setWeight: 50
      - pause: {duration: 10}
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
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  labels:
    app: keda-test-app
  name: keda-test-app
spec:
  replicas: 0
  strategy:
    canary:
      steps:
      - setWeight: 50
      - pause: {duration: 10}
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
    apiVersion: argoproj.io/v1alpha1
    kind: Rollout
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
