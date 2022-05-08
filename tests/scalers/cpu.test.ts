import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import test from 'ava'
import { createNamespace, waitForDeploymentReplicaCount } from './helpers'

const testNamespace = 'cpu-test'
const deployMentFile = tmp.fileSync()
const triggerFile = tmp.fileSync()


test.before(t => {
  sh.config.silent = true
  createNamespace(testNamespace)

  fs.writeFileSync(deployMentFile.name, deployMentYaml)
  t.is(
    0,
    sh.exec(`kubectl apply -f ${deployMentFile.name} --namespace ${testNamespace}`).code,
    'Deploying php deployment should work.'
  )
  t.is(0, sh.exec(`kubectl rollout status deploy/php-apache -n ${testNamespace}`).code, 'Deployment php rolled out succesfully')
})

test.serial('Deployment should have 1 replica on start', t => {
  const replicaCount = sh.exec(
    `kubectl get deployment.apps/php-apache --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
  ).stdout
  t.is(replicaCount, '1', 'replica count should start out as 1')
})

test.serial(`Creating Job should work`, async t => {
  fs.writeFileSync(triggerFile.name, triggerJob)
  t.is(
    0,
    sh.exec(`kubectl apply -f ${triggerFile.name} --namespace ${testNamespace}`).code,
    'creating job should work.'
  )
})

test.serial(`Deployment should scale in next 3 minutes`, async t => {
  // check for increased replica count on constant triggering :
  t.true(await waitForDeploymentReplicaCount(2, 'php-apache', testNamespace, 18, 10000), 'Replica count should scale up in next 3 minutes')
})

test.serial(`Deleting Job should work`, async t => {
  fs.writeFileSync(triggerFile.name, triggerJob)
  t.is(
    0,
    sh.exec(`kubectl delete -f ${triggerFile.name} --namespace ${testNamespace}`).code,
    'Deleting job should work.'
  )
})

test.serial(`Deployment should scale back to 1 in next 3 minutes`, async t => {
  // check for the scale down :
  t.true(await waitForDeploymentReplicaCount(1, 'php-apache', testNamespace, 18, 10000), 'Replica count should be 1 in next 3 minutes')
})

test.after.always.cb('clean up workload test related deployments', t => {
  const resources = [
    'deployment.apps/php-apache',
    'jobs.batch/trigger-job',
    'scaledobject.keda.sh/cpu-scaledobject',
    'service/php-apache',
  ]
  for (const resource of resources) {
    sh.exec(`kubectl delete ${resource} --namespace ${testNamespace}`)
  }
  sh.exec(`kubectl delete namespace ${testNamespace}`)
  t.end()
})

const deployMentYaml = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: php-apache
spec:
  selector:
    matchLabels:
      run: php-apache
  replicas: 1
  template:
    metadata:
      labels:
        run: php-apache
    spec:
      containers:
      - name: php-apache
        image: k8s.gcr.io/hpa-example
        ports:
        - containerPort: 80
        resources:
          limits:
            cpu: 500m
          requests:
            cpu: 200m
        imagePullPolicy: IfNotPresent
---
apiVersion: v1
kind: Service
metadata:
  name: php-apache
  labels:
    run: php-apache
spec:
  ports:
  - port: 80
  selector:
    run: php-apache
---
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: cpu-scaledobject
  labels:
    run: php-apache
spec:
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 0
  maxReplicaCount: 2
  minReplicaCount: 1
  scaleTargetRef:
    name: php-apache
  triggers:
  - type: cpu
    metadata:
      type: Utilization
      value: "50"`
const triggerJob = `apiVersion: batch/v1
kind: Job
metadata:
  name: trigger-job
spec:
  template:
    spec:
      containers:
      - image: busybox
        name: test
        command: ["/bin/sh"]
        args: ["-c", "for i in $(seq 1 400);do wget -q -O- http://php-apache.cpu-test.svc/;sleep 0.1;done"]
      restartPolicy: Never
  activeDeadlineSeconds: 400
  backoffLimit: 3`
