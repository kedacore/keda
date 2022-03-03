import * as crypto from 'crypto'
import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import test from 'ava'
import { createNamespace, sleep, waitForDeploymentReplicaCount } from './helpers';

const gcpKey = process.env['GCP_SP_KEY']
const projectId = JSON.parse(gcpKey).project_id
const gcpAccount = JSON.parse(gcpKey).client_email
const testNamespace = 'gcp-pubsub-test'
const topicId = `projects/${projectId}/topics/keda-test-topic-` + crypto.randomBytes(6).toString('hex')
const subscriptionName = `keda-test-topic-sub-` + crypto.randomBytes(6).toString('hex')
const subscriptionId = `projects/${projectId}/subscriptions/${subscriptionName}`
const deploymentName = 'dummy-consumer'
const maxReplicaCount = '4'
const gsPrefix = `kubectl exec --namespace ${testNamespace} deployment.apps/gcp-sdk -- `

test.before(t => {
    createNamespace(testNamespace)

    // deploy dummy consumer app, scaled object etc.
    const tmpFile = tmp.fileSync()
    fs.writeFileSync(tmpFile.name, deployYaml.replace("{{GCP_CREDS}}", Buffer.from(gcpKey).toString("base64")))

    t.is(
        0,
        sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${testNamespace}`).code,
        'creating a deployment should work..'
    )
})

test.serial('Deployment should have 0 replicas on start', async t => {
    t.true(await waitForDeploymentReplicaCount(0, deploymentName, testNamespace, 30, 2000), 'replica count should start out as 0')
})

test.serial('creating the gcp-sdk pod should work..', async t => {
    let tmpFile = tmp.fileSync()
    fs.writeFileSync(tmpFile.name, gcpSdkYaml)
    t.is(
        0,
        sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${testNamespace}`).code,
        'creating the gcp-sdk pod should work..'
    )

    // wait for the gcp-sdk pod to be ready
    t.true(await waitForDeploymentReplicaCount(1, 'gcp-sdk', testNamespace, 60, 2000), 'GCP-SDK pod is not in a ready state')
})

test.serial('initializing the gcp-sdk pod should work..', t => {
    sh.exec(`kubectl wait --for=condition=ready --namespace ${testNamespace} pod -l app=gcp-sdk --timeout=30s`)
    sh.exec('sleep 5s')

    // Authenticate to GCP
    t.is(0,
      sh.exec(gsPrefix + `gcloud auth activate-service-account ${gcpAccount} --key-file /etc/secret-volume/creds.json --project=${projectId}`).code,
      'Setting GCP authentication on gcp-sdk should work..')

    // Create topic and subscription
    t.is(
        0,
        sh.exec(gsPrefix + `gcloud pubsub topics create ${topicId}`).code,
        'Creating a topic should work..'
    )

    t.is(
        0,
        sh.exec(gsPrefix + `gcloud pubsub subscriptions create ${subscriptionId} --topic=${topicId}`).code,
        'Creating a subscription should work..'
    )
})

test.serial(`Publishing to pubsub`, t => {
    // Publish 30 messages
    var cmd = gsPrefix + ' /bin/bash -c -- "cd .'
    for (let i = 0; i < 30; i++) {
        cmd += ` && gcloud pubsub topics publish ${topicId} --message=AAAAAAAAAA`
    }
    cmd += '"'
    t.is(0,sh.exec(cmd).code,'Publishing messages to pub/sub should work..')
})

test.serial(`Deployment should scale to ${maxReplicaCount} (the max) then back to 0`, async t => {
    // Wait for the number of replicas to be scaled up to maxReplicaCount
    t.true(
      await waitForDeploymentReplicaCount(parseInt(maxReplicaCount, 10), deploymentName, testNamespace, 150, 2000),
      `Replica count should be ${maxReplicaCount} after 120 seconds`)

    // Purge all messages
    sh.exec(gsPrefix + `gcloud pubsub subscriptions seek ${subscriptionId} --time=p0s`)

    // Wait for the number of replicas to be scaled down to 0
    t.true(
      await waitForDeploymentReplicaCount(0, deploymentName, testNamespace, 30, 10000),
      `Replica count should be 0 after 3 minutes`)
})

test.after.always.cb('clean up', t => {
    // Delete the subscription and topic
    sh.exec(gsPrefix + `gcloud pubsub subscriptions delete ${subscriptionId}`)
    sh.exec(gsPrefix + `gcloud pubsub topics delete ${topicId}`)

    sh.exec(`kubectl delete deployment.apps/${deploymentName} --namespace ${testNamespace}`)
    sh.exec(`kubectl delete namespace ${testNamespace}`)

    t.end()
})


const deployYaml = `apiVersion: apps/v1
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
        - name: ${deploymentName}-processor
          image: google/cloud-sdk:slim
          # Consume a message
          command: [ "/bin/bash", "-c", "--" ]
          args: [ "gcloud auth activate-service-account --key-file /etc/secret-volume/creds.json && \
          while true; do gcloud pubsub subscriptions pull ${subscriptionId} --auto-ack; sleep 20; done" ]
          env:
            - name: GOOGLE_APPLICATION_CREDENTIALS_JSON
              valueFrom:
                secretKeyRef:
                  name: pubsub-secrets
                  key: creds.json
          volumeMounts:
            - name: secret-volume
              mountPath: /etc/secret-volume
      volumes:
        - name: secret-volume
          secret:
            secretName: pubsub-secrets
---
apiVersion: v1
kind: Secret
metadata:
  name: pubsub-secrets
type: Opaque
data:
  creds.json: {{GCP_CREDS}}
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
    - type: gcp-pubsub
      metadata:
        subscriptionName: ${subscriptionName}
        mode: SubscriptionSize
        value: "5"
        credentialsFromEnv: GOOGLE_APPLICATION_CREDENTIALS_JSON
`

const gcpSdkYaml = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: gcp-sdk
  namespace: ${testNamespace}
  labels:
    app: gcp-sdk
spec:
  replicas: 1
  selector:
    matchLabels:
      app: gcp-sdk
  template:
    metadata:
      labels:
        app: gcp-sdk
    spec:
      containers:
        - name: gcp-sdk-container
          image: google/cloud-sdk:slim
          # Just spin & wait forever
          command: [ "/bin/bash", "-c", "--" ]
          args: [ "while true; do sleep 30; done;" ]
          volumeMounts:
            - name: secret-volume
              mountPath: /etc/secret-volume
      volumes:
        - name: secret-volume
          secret:
            secretName: pubsub-secrets
`
