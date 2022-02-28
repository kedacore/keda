import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import test from 'ava'
import { createNamespace, waitForDeploymentReplicaCount } from './helpers'

const testNamespace = 'memory-test'
const scaleUpValue = 1
const scaleDownValue = 45
const deploymentFile = tmp.fileSync()
const scaledObjectFile = tmp.fileSync()

test.before(t => {
  sh.config.silent = true
  createNamespace(testNamespace)

  fs.writeFileSync(deploymentFile.name, deploymentYaml)
  t.is(
    0,
    sh.exec(`kubectl apply -f ${deploymentFile.name} --namespace ${testNamespace}`).code,
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

test.serial(`Creating ScaledObject should work`, async t => {
  fs.writeFileSync(scaledObjectFile.name, scaledObjectYaml.
    replace('{{UTILIZATION_VALUE}}', scaleUpValue.toString()))
  t.is(
    0,
    sh.exec(`kubectl apply -f ${scaledObjectFile.name} --namespace ${testNamespace}`).code,
    'creating new ScaledObject should work.'
  )
})

test.serial(`Deployment should scale in next 3 minutes`, async t => {
  // check for increased replica count on constant triggering :
  t.true(await waitForDeploymentReplicaCount(2, 'php-apache', testNamespace, 18, 10000), 'Replica count should scale up in next 3 minutes')
})

test.serial(`Updsating ScaledObject should work`, async t => {
  fs.writeFileSync(scaledObjectFile.name, scaledObjectYaml.replace('{{UTILIZATION_VALUE}}', scaleDownValue.toString()))
  t.is(
    0,
    sh.exec(`kubectl apply -f ${scaledObjectFile.name} --namespace ${testNamespace}`).code,
    'Updating ScaledObject should work.'
  )
})

test.serial(`Deployment should scale back to 1 in next 3 minutes`, async t => {
  fs.writeFileSync(scaledObjectFile.name, scaledObjectYaml)
  t.is(
    0,
    sh.exec(`kubectl apply -f ${scaledObjectFile.name} --namespace ${testNamespace}`).code,
    'creating Scaled Object should work.'
  )
  // check for the scale down :
  t.true(await waitForDeploymentReplicaCount(1, 'php-apache', testNamespace, 18, 10000), 'Replica count should be 1 in next 3 minutes')
})

test.after.always.cb('clean up workload test related deployments', t => {
  const resources = [
    'deployment.apps/php-apache',
    'scaledobject.keda.sh/memory-scaledobject',
    'service/php-apache',
  ]
  for (const resource of resources) {
    sh.exec(`kubectl delete ${resource} --namespace ${testNamespace}`)
  }
  sh.exec(`kubectl delete namespace ${testNamespace}`)
  t.end()
})

const deploymentYaml = `apiVersion: apps/v1
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
            memory: 100Mi
          requests:
            memory: 50Mi
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
    run: php-apache`

const scaledObjectYaml = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: memory-scaledobject
  labels:
    run: php-apache
spec:
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          policies:
          - type: Pods
            value: 1
            periodSeconds: 10
          stabilizationWindowSeconds: 0
  maxReplicaCount: 2
  minReplicaCount: 1
  scaleTargetRef:
    name: php-apache
  triggers:
  - type: memory
    metadata:
      type: Utilization
      value: "{{UTILIZATION_VALUE}}"`
