import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import test from 'ava'
import { createNamespace, waitForDeploymentReplicaCount } from './helpers'

const testNamespace = 'crontest'
const deployFile = tmp.fileSync()

// Get now + 1 min and now + 2 min for starting ang ending minutes
let now = new Date()
now.setMinutes(now.getMinutes()+ 1)
let start =  now.getMinutes()
now.setMinutes(now.getMinutes()+ 1)
let end = now.getMinutes()

test.before(t => {
  sh.config.silent = true
  createNamespace(testNamespace)

  fs.writeFileSync(deployFile.name, deployYaml.replace('{{START_MIN}}', start.toString())
                                .replace('{{END_MIN}}', end.toString()))
	t.is(
		0,
		sh.exec(`kubectl apply -f ${deployFile.name} --namespace ${testNamespace}`).code,
		'Deploying deployment should work.'
	)
})

test.serial('Deployment should have 1 replicas on start', t => {
  const replicaCount = sh.exec(
    `kubectl get deployment.apps/cron-tests-deployment --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
  ).stdout
  t.is(replicaCount, '1', 'replica count should start out as 1')
})

test.serial(`Deployment should scale to 4`, async t => {
    //The workload should scale to 4 instances in the next minute
    t.true(await waitForDeploymentReplicaCount(4, 'cron-tests-deployment', testNamespace, 30, 2000))
})

test.serial(`Deployment should scale to 1`, async t => {
    //The workload should scale to 1 instances in the next minute
    t.true(await waitForDeploymentReplicaCount(1, 'cron-tests-deployment', testNamespace, 30, 2000))
})


test.after.always.cb('clean up workload test related deployments', t => {
  const resources = [
    'scaledobject.keda.sh/cron-tests-scaledobject',
    'deployment.apps/cron-tests-deployment',
  ]

  for (const resource of resources) {
    sh.exec(`kubectl delete ${resource} --namespace ${testNamespace}`)
  }
  sh.exec(`kubectl delete namespace ${testNamespace}`)
  t.end()
})

const deployYaml = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: cron-tests-deployment
  labels:
    deploy: cron-tests
spec:
  replicas: 1
  selector:
    matchLabels:
      pod: cron-tests
  template:
    metadata:
      labels:
        pod: cron-tests
    spec:
      containers:
        - name: nginx
          image: 'nginx'
---
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: cron-tests-scaledobject
spec:
  scaleTargetRef:
    name: cron-tests-deployment
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
  - type: cron
    metadata:
      timezone: Etc/UTC
      start: {{START_MIN}} * * * *
      end: {{END_MIN}} * * * *
      desiredReplicas: "4"`
