import * as sh from "shelljs"
import test from "ava"
import { createNamespace, createYamlFile, waitForDeploymentReplicaCount } from "./helpers"

const testName = "test-external-scaler"
const testNamespace = `${testName}-ns`
const scalerName = `${testName}-scaler`
const serviceName = `${testName}-service`
const deploymentName = `${testName}-deployment`
const scaledObjectName = `${testName}-scaled-object`

const idleReplicaCount = 0
const minReplicaCount = 1
const maxReplicaCount = 2
const threshold = 10

test.before(async t => {
    sh.config.silent = true

    // Create Kubernetes Namespace
    createNamespace(testNamespace)

    // Create external scaler deployment
    t.is(
        sh.exec(`kubectl apply -f ${createYamlFile(scalerYaml)} -n ${testNamespace}`).code,
        0,
        "Createing a external scaler deployment should work"
    )

    // Create service
    t.is(
        sh.exec(`kubectl apply -f ${createYamlFile(serviceYaml)} -n ${testNamespace}`).code,
        0,
        "Createing a service should work"
    )

    // Create deployment
    t.is(
        sh.exec(`kubectl apply -f ${createYamlFile(deploymentYaml)} -n ${testNamespace}`).code,
        0,
        "Createing a deployment should work"
    )

    // Create scaled object
    t.is(
        sh.exec(`kubectl apply -f ${createYamlFile(scaledObjectYaml.replace("{{VALUE}}", "0"))} -n ${testNamespace}`).code,
        0,
        "Creating a scaled object should work"
    )

    t.true(await waitForDeploymentReplicaCount(idleReplicaCount, deploymentName, testNamespace, 60, 1000),
        `Replica count should be ${idleReplicaCount} after 1 minute`)
})

test.serial("Deployment should scale up to minReplicaCount", async t => {
    // Modify scaled object's metricValue to induce scaling
    t.is(
        sh.exec(`kubectl apply -f ${createYamlFile(scaledObjectYaml.replace("{{VALUE}}", `${threshold}`))} -n ${testNamespace}`).code,
        0,
        "Modifying scaled object should work"
    )

    t.true(await waitForDeploymentReplicaCount(minReplicaCount, deploymentName, testNamespace, 60, 1000),
    `Replica count should be ${minReplicaCount} after 1 minute`)
})

test.serial("Deployment should scale up to maxReplicaCount", async t => {
    // Modify scaled object's metricValue to induce scaling
    t.is(
        sh.exec(`kubectl apply -f ${createYamlFile(scaledObjectYaml.replace("{{VALUE}}", `${threshold * 2}`))} -n ${testNamespace}`).code,
        0,
        "Modifying scaled object should work"
    )

    t.true(await waitForDeploymentReplicaCount(maxReplicaCount, deploymentName, testNamespace, 60, 1000),
    `Replica count should be ${maxReplicaCount} after 1 minute`)
})

test.serial("Deployment should scale back down to idleReplicaCount", async t => {
    // Modify scaled object's metricValue to induce scaling
    t.is(
        sh.exec(`kubectl apply -f ${createYamlFile(scaledObjectYaml.replace("{{VALUE}}", "0"))} -n ${testNamespace}`).code,
        0,
        "Modifying scaled object should work"
    )

    t.true(await waitForDeploymentReplicaCount(idleReplicaCount, deploymentName, testNamespace, 60, 1000),
    `Replica count should be ${idleReplicaCount} after 1 minute`)
})

test.after.always("Clean up E2E K8s objects", async t => {
    const resources = [
        `scaledobject.keda.sh/${scaledObjectName}`,
        `deployments.apps/${deploymentName}`,
        `service/${serviceName}`,
        `deployments.apps/${scalerName}`,
    ]

    for (const resource of resources) {
        sh.exec(`kubectl delete ${resource} -n ${testNamespace}`)
    }

    sh.exec(`kubectl delete ns ${testNamespace}`)
})

// YAML Definitions for Kubernetes resources
// External Scaler Deployment
const scalerYaml =
`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ${scalerName}
  namespace: ${testNamespace}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ${scalerName}
  template:
    metadata:
      labels:
        app: ${scalerName}
    spec:
      containers:
      - name: scaler
        image: ghcr.io/kedacore/tests-external-scaler-e2e:latest
        imagePullPolicy: Always
        ports:
        - containerPort: 6000
`

const serviceYaml =
`
apiVersion: v1
kind: Service
metadata:
  name: ${serviceName}
  namespace: ${testNamespace}
spec:
  ports:
  - port: 6000
    targetPort: 6000
  selector:
    app: ${scalerName}
`

// Deployment
const deploymentYaml =
`apiVersion: apps/v1
kind: Deployment
metadata:
  name: ${deploymentName}
  namespace: ${testNamespace}
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
      - name: nginx
        image: nginx:1.16.1
`

// Scaled Object
const scaledObjectYaml =
`
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: ${scaledObjectName}
  namespace: ${testNamespace}
spec:
  scaleTargetRef:
    name: ${deploymentName}
  pollingInterval: 5
  cooldownPeriod: 10
  idleReplicaCount: ${idleReplicaCount}
  minReplicaCount: ${minReplicaCount}
  maxReplicaCount: ${maxReplicaCount}
  triggers:
  - type: external
    metadata:
      scalerAddress: ${serviceName}.${testNamespace}:6000
      metricThreshold: "${threshold}"
      metricValue: "{{VALUE}}"
`
