import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import test from 'ava'
import {waitForDeploymentReplicaCount} from "./helpers";

const testNamespace = 'kubernetes-workload-test'
const monitoredDeploymentFile = tmp.fileSync()
const sutDeploymentFile = tmp.fileSync()

test.before(t => {
  sh.config.silent = true
	sh.exec(`kubectl create namespace ${testNamespace}`)

  fs.writeFileSync(monitoredDeploymentFile.name, monitoredDeploymentYaml)
	t.is(
		0,
		sh.exec(`kubectl apply -f ${monitoredDeploymentFile.name} --namespace ${testNamespace}`).code,
		'Deploying monitored deployment should work.'
	)

  fs.writeFileSync(sutDeploymentFile.name, sutDeploymentYaml)
	t.is(
		0,
		sh.exec(`kubectl apply -f ${sutDeploymentFile.name} --namespace ${testNamespace}`).code,
		'Deploying monitored deployment should work.'
	)
})

test.serial('Deployment should have 1 replicas on start', t => {
  const replicaCount = sh.exec(
    `kubectl get deployment.apps/sut-deployment --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
  ).stdout
  t.is(replicaCount, '1', 'replica count should start out as 1')
})

test.serial(`Deployment should scale to fit the amount of pods which match the selector`, async t => {

  sh.exec(
    `kubectl scale deployment.apps/monitored-deployment --namespace ${testNamespace} --replicas=5`
  )
  t.true(await waitForDeploymentReplicaCount(5, 'sut-deployment', testNamespace, 6, 10000), 'Replica count should be 5 after 60 seconds')

  sh.exec(
    `kubectl scale deployment.apps/monitored-deployment --namespace ${testNamespace} --replicas=10`
  )
  t.true(await waitForDeploymentReplicaCount(10, 'sut-deployment', testNamespace, 6, 10000), 'Replica count should be 10 after 60 seconds')

  sh.exec(
    `kubectl scale deployment.apps/monitored-deployment --namespace ${testNamespace} --replicas=5`
  )
  t.true(await waitForDeploymentReplicaCount(5, 'sut-deployment', testNamespace, 6, 10000), 'Replica count should be 5 after 60 seconds')

  sh.exec(
    `kubectl scale deployment.apps/monitored-deployment --namespace ${testNamespace} --replicas=1`
  )
  t.true(await waitForDeploymentReplicaCount(1, 'sut-deployment', testNamespace, 6, 10000), 'Replica count should be 1 after 60 seconds')
})

test.after.always.cb('clean up workload test related deployments', t => {
  const resources = [
    'scaledobject.keda.sh/sut-scaledobject',
    'deployment.apps/sut-deployment',
    'deployment.apps/monitored-deployment',
  ]

  for (const resource of resources) {
    sh.exec(`kubectl delete ${resource} --namespace ${testNamespace}`)
  }
  sh.exec(`kubectl delete namespace ${testNamespace}`)
  t.end()
})

const monitoredDeploymentYaml = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: monitored-deployment
  labels:
    deploy: workload-test
spec:
  replicas: 1
  selector:
    matchLabels:
      pod: workload-test
  template:
    metadata:
      labels:
        pod: workload-test
    spec:
      containers:
        - name: nginx
          image: 'nginx'`

const sutDeploymentYaml = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: sut-deployment
  labels:
    deploy: workload-sut
spec:
  replicas: 1
  selector:
    matchLabels:
      pod: workload-sut
  template:
    metadata:
      labels:
        pod: workload-sut
    spec:
      containers:
        - name: nginx
          image: 'nginx'
---
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: sut-scaledobject
spec:
  scaleTargetRef:
    name: sut-deployment
  pollingInterval: 5
  cooldownPeriod: 5
  minReplicaCount: 1
  maxReplicaCount: 10
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 15
  triggers:
  - type: kubernetes-workload
    metadata:
      podSelector: 'pod=workload-test'
      namespace: ${testNamespace}
      value: '1'`
