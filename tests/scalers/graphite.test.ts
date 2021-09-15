import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import test from 'ava'
import {waitForRollout} from "./helpers";

const graphiteNamespace = 'graphite'
const graphiteDeploymentFile = 'scalers/graphite-deployment.yaml'

test.before(t => {
  // install graphite
  sh.exec(`kubectl create namespace ${graphiteNamespace}`)
  t.is(0,
    sh.exec(`kubectl apply --namespace ${graphiteNamespace} -f ${graphiteDeploymentFile}`).code,
    'creating a Graphite deployment should work.'
  )
  // wait for graphite to load
  t.is(0, waitForRollout('statefulset', "graphite", graphiteNamespace))

  sh.config.silent = true
  const tmpFile = tmp.fileSync()
  fs.writeFileSync(tmpFile.name, deployYaml.replace('{{GRAPHITE_NAMESPACE}}', graphiteNamespace))
  t.is(
    0,
    sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${graphiteNamespace}`).code,
    'creating graphite scaling test deployment should work.'
  )
  for (let i = 0; i < 10; i++) {
    const readyReplicaCount = sh.exec(`kubectl get deployment php-apache-graphite --namespace ${graphiteNamespace} -o jsonpath="{.status.readyReplicas}`).stdout
    if (readyReplicaCount != '1') {
      sh.exec('sleep 2s')
    }
  }
})

test.serial('Deployment should have 0 replica on start', t => {
  const replicaCount = sh.exec(
    `kubectl get deployment php-apache-graphite --namespace ${graphiteNamespace} -o jsonpath="{.spec.replicas}"`
  ).stdout
  t.is(replicaCount, '0', 'replica count should start out as 0')
})

test.serial(`Deployment should scale to 5 (the max) with HTTP Requests exceeding in the rate then back to 0`, t => {
  const tmpFile = tmp.fileSync()
  t.log(tmpFile.name)

  fs.writeFileSync(tmpFile.name, generateRequestsYaml.replace('{{GRAPHITE_NAMESPACE}}', graphiteNamespace))
  t.is(
    0,
    sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${graphiteNamespace}`).code,
    'creating job should work.'
  )

  // keda based deployment should start scaling up with http requests issued
  let replicaCount = '0'
  for (let i = 0; i < 60 && replicaCount !== '5'; i++) {
    t.log(`Waited ${5 * i} seconds for graphite-based deployments to scale up`)
    const jobLogs = sh.exec(`kubectl logs -l job-name=generate-graphite-metrics -n ${graphiteNamespace}`).stdout
    t.log(`Logs from the generate requests: ${jobLogs}`)

    replicaCount = sh.exec(
      `kubectl get deployment php-apache-graphite --namespace ${graphiteNamespace} -o jsonpath="{.spec.replicas}"`
    ).stdout
    if (replicaCount !== '5') {
      sh.exec('sleep 5s')
    }
  }

  t.is('5', replicaCount, 'Replica count should be maxed at 5')

  for (let i = 0; i < 50 && replicaCount !== '0'; i++) {
    replicaCount = sh.exec(
      `kubectl get deployment php-apache-graphite --namespace ${graphiteNamespace} -o jsonpath="{.spec.replicas}"`
    ).stdout
    if (replicaCount !== '0') {
      sh.exec('sleep 5s')
    }
  }

  t.is('0', replicaCount, 'Replica count should be 0 after 3 minutes')
})

test.after.always.cb('clean up graphite deployment', t => {
  const resources = [
    'scaledobject.keda.sh/graphite-scaledobject',
    'deployment.apps/php-apache-graphite',
    'job/generate-graphite-metrics',
  ]

  for (const resource of resources) {
    sh.exec(`kubectl delete ${resource} --namespace ${graphiteNamespace}`)
  }

  // uninstall graphite
  sh.exec(`kubectl delete --namespace ${graphiteNamespace} -f ${graphiteDeploymentFile}`)
  sh.exec(`kubectl delete namespace ${graphiteNamespace}`)

  t.end()
})

const deployYaml = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: php-apache-graphite
spec:
  selector:
    matchLabels:
      run: php-apache-graphite
  replicas: 0
  template:
    metadata:
      labels:
        run: php-apache-graphite
    spec:
      containers:
      - name: php-apache-graphite
        image: k8s.gcr.io/hpa-example
        ports:
        - containerPort: 80
---
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: graphite-scaledobject
spec:
  scaleTargetRef:
    name: php-apache-graphite
  minReplicaCount: 0
  maxReplicaCount: 5
  pollingInterval: 5
  cooldownPeriod:  10
  triggers:
  - type: graphite
    metadata:
      serverAddress: http://graphite.{{GRAPHITE_NAMESPACE}}.svc:8080
      metricName: https_metric
      threshold: '100'
      query: "https_metric"
      queryTime: '-10Seconds'`

const generateRequestsYaml = `apiVersion: batch/v1
kind: Job
metadata:
  name: generate-graphite-metrics
spec:
  template:
    spec:
      containers:
      - image: busybox
        name: generate-graphite-metrics
        command: ["/bin/sh"]
        args: ["-c", "for i in $(seq 1 60);do echo $i; echo \\"https_metric 1000 $(date +%s)\\" | nc graphite.{{GRAPHITE_NAMESPACE}}.svc 2003; echo 'data sent :)'; sleep 1; done"]
      restartPolicy: Never
  activeDeadlineSeconds: 120
  backoffLimit: 2`
