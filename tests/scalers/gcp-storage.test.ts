import * as async from 'async'
import * as fs from 'fs'
import * as http from 'http'
import fetch from 'node-fetch'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import test from 'ava'

const testNamespace = 'gcp-storage-test'
const gcsNamespace = 'gcp-gcs-emulator'
const bucketName = 'test-bucket'
const deploymentName = 'dummy-consumer'
const maxReplicaCount = '3'

test.before(t => {
    // install fake gcs
    sh.exec(`kubectl create namespace ${gcsNamespace}`)
    const gcsTmpFile = tmp.fileSync()
    fs.writeFileSync(gcsTmpFile.name, gcsDeploymentYaml)

    t.is(0, sh.exec(`kubectl apply --namespace ${gcsNamespace} -f ${gcsTmpFile.name}`).code, 'creating a fake GCS deployment should work.')
    // wait for fake gcs to load
    let gcsReadyReplicaCount = '0'
    for (let i = 0; i < 30; i++) {
      gcsReadyReplicaCount = sh.exec(`kubectl get deploy/gcs -n ${gcsNamespace} -o jsonpath='{.status.readyReplicas}'`).stdout
        if (gcsReadyReplicaCount != '1') {
            sh.exec('sleep 2s')
        }
    }
    t.is('1', gcsReadyReplicaCount, 'Fake GCS is not in a ready state')

    sh.exec(`kubectl create namespace ${testNamespace}`)

    // deploy dummy consumer app, scaled object etc.
    const tmpFile = tmp.fileSync()
    fs.writeFileSync(tmpFile.name, deployYaml)

    t.is(
        0,
        sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${testNamespace}`).code,
        'creating a deployment should work..'
    )

})

test.serial('Deployment should have 0 replicas on start', t => {
    const replicaCount = sh.exec(
        `kubectl get deployment.apps/${deploymentName} --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
    ).stdout
    t.is(replicaCount, '0', 'replica count should start out as 0')
})

test.serial(`Deployment should scale to ${maxReplicaCount} (the max) then back to 0`, t => {
    // Create test bucket and files
    const numFiles = 30

    const endpoint = "http://gcs." + gcsNamespace + ":4443/storage/v1/b"
    let args = "'-d', '{ \"name\": \"" + bucketName + "\" }', '-H', 'Content-Type: application/json', '-X', 'POST', '" + endpoint + "'"
    // Upload 30 files to the test bucket by building a CURL argument of numFiles upload requests
    for (let i = 1; i <= numFiles; i++) {
      args += ",'--next', '-d', 'AAA', '-H', 'Content-Type: text/plain', '-X', 'POST', '" + endpoint + "/" + bucketName + "/o?name=test-object-" + i + "&uploadType=media'"
    }

    let tmpFile = tmp.fileSync()
    fs.writeFileSync(tmpFile.name, curlJobYaml.replace("{{ARGS}}", args).replace(/\{\{NAME\}\}/g, "create-data-job", ))

    t.is(
        0,
        sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${gcsNamespace}`).code,
        'creating the job for creating the bucket and inserting objects to it should work..'
    )

    let replicaCount = '0'

    // Wait for the number of replicas to be scaled up to maxReplicaCount
    for (let i = 0; i < 60 && replicaCount !== maxReplicaCount; i++) {
        replicaCount = sh.exec(
            `kubectl get deployment.apps/${deploymentName} --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
        ).stdout
        if (replicaCount !== maxReplicaCount) {
            sh.exec('sleep 2s')
        }
    }

    t.is(maxReplicaCount, replicaCount, `Replica count should be ${maxReplicaCount} after 120 seconds but is ${replicaCount}`)

    // Delete 30 files from the test bucket by building a CURL argument of numFiles delete requests
    args = ""
    for (let i = 1; i <= numFiles; i++) {
      args += "'-X', 'DELETE', '" + endpoint + "/" + bucketName + "/o/test-object-" + i + "'"
      if (i < numFiles) {
        args += ",'--next',"
      }
    }

    tmpFile = tmp.fileSync()
    fs.writeFileSync(tmpFile.name, curlJobYaml.replace("{{ARGS}}", args).replace(/\{\{NAME\}\}/g, "delete-data-job", ))

    t.is(
        0,
        sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${gcsNamespace}`).code,
        'creating the job for deleting objects from the test bucket should work..'
    )

    // Wait for the number of replicas to be scaled down to 0
    for (let i = 0; i < 10 && replicaCount !== '0'; i++) {
      replicaCount = sh.exec(
        `kubectl get deployment.apps/${deploymentName} --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
      ).stdout
      if (replicaCount !== '0') {
        sh.exec('sleep 5s')
      }
    }

    t.is('0', replicaCount, 'Replica count should be 0 after 3 minutes')
})

test.after.always.cb('clean up fake GCS deployment', t => {
  sh.exec(`kubectl delete deployment.apps/${deploymentName} --namespace ${testNamespace}`)
  sh.exec(`kubectl delete namespace ${testNamespace}`)

  // uninstall gcs
  sh.exec(`kubectl delete --namespace ${gcsNamespace} deploy/gcs`)
  sh.exec(`kubectl delete namespace ${gcsNamespace}`)

  t.end()
})


const gcsDeploymentYaml = `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: gcs
  name: gcs
spec:
  replicas: 1
  selector:
    matchLabels:
      app: gcs
  template:
    metadata:
      labels:
        app: gcs
    spec:
      containers:
      - image: fsouza/fake-gcs-server:1.34.1
        name: gcs
        command: [ "/bin/fake-gcs-server" ]
        args: [ "-scheme", "http" ]
---
apiVersion: v1
kind: Service
metadata:
  labels:
    app: gcs
  name: gcs
spec:
  ports:
  - port: 4443
    protocol: TCP
    targetPort: 4443
  selector:
    app: gcs
  type: ClusterIP`

const deployYaml = `
  apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: ${deploymentName}
    namespace: ${testNamespace}
    labels:
      app: ${deploymentName}
  spec:
    replicas: 0
    selector:
      matchLabels:
        app: ${deploymentName}
    template:
      metadata:
        labels:
          app: ${deploymentName}
      spec:
        containers:
        - name: noop-processor
          image: ubuntu:20.04
          command: [ "/bin/bash", "-c", "--" ]
          args: [ "sleep 10" ]
---
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: test-scaledobject
spec:
  scaleTargetRef:
    name: ${deploymentName}
  pollingInterval: 5
  maxReplicaCount: ${maxReplicaCount}
  cooldownPeriod: 10
  triggers:
  - type: gcp-storage
    metadata:
      bucketName: ${bucketName}
      targetObjectCount: '5'
      endpoint: http://gcs.${gcsNamespace}.svc.cluster.local:4443/storage/v1/`

const curlJobYaml = `
  apiVersion: batch/v1
  kind: Job
  metadata:
    labels:
      app: {{NAME}}
    name: {{NAME}}
  spec:
    template:
      metadata:
        labels:
          app: {{NAME}}
      spec:
        containers:
        - name: {{NAME}}-test
          image: curlimages/curl:7.81.0
          args: [ {{ARGS}} ]
        restartPolicy: Never
    backoffLimit: 4`
