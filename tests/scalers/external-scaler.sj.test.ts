import * as sh from "shelljs"
import test from "ava"
import { createNamespace, createYamlFile, waitForDeploymentReplicaCount, waitForJobCount } from "./helpers"

const testName = "test-external-scaler-sj"
const testNamespace = `${testName}-ns`
const scalerName = `${testName}-scaler`
const serviceName = `${testName}-service`
const deploymentName = `${testName}-deployment`
const scaledJobName = `${testName}-scaled-job`

const maxReplicaCount = 5
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

    // Create scaled job
    t.is(
        sh.exec(`kubectl apply -f ${createYamlFile(scaledJobYaml.replace("{{VALUE}}", "0"))} -n ${testNamespace}`).code,
        0,
        "Creating a scaled job should work"
    )

    t.true(await waitForJobCount(0, testNamespace, 60, 1000),`Replica count should be 0 after 1 minute`)
})

test.serial("Deployment should scale up to maxReplicaCount", async t => {
    // Modify scaled job's metricValue to induce scaling
    t.is(
        sh.exec(`kubectl apply -f ${createYamlFile(scaledJobYaml.replace("{{VALUE}}", `${threshold * 2}`))} -n ${testNamespace}`).code,
        0,
        "Modifying scaled job should work"
    )

    t.true(await waitForJobCount(maxReplicaCount, testNamespace, 60, 1000),`Replica count should be 0 after 1 minute`)
})

test.serial("Deployment should scale back down to 0", async t => {
    // Modify scaled job's metricValue to induce scaling
    t.is(
        sh.exec(`kubectl apply -f ${createYamlFile(scaledJobYaml.replace("{{VALUE}}", "0"))} -n ${testNamespace}`).code,
        0,
        "Modifying scaled job should work"
    )

    t.true(await waitForJobCount(0, testNamespace, 120, 1000),`Replica count should be 0 after 2 minute`)
})

test.after.always("Clean up E2E K8s objects", async t => {
    const resources = [
        `scaledjob.keda.sh/${scaledJobName}`,
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

// scaled job
const scaledJobYaml =
`
apiVersion: keda.sh/v1alpha1
kind: ScaledJob
metadata:
  name: ${scaledJobName}
  namespace: ${testNamespace}
spec:
  jobTargetRef:
    template:
      spec:
        containers:
          - name: external-executor
            image: busybox
            command:
            - sleep 
            - "30"
            imagePullPolicy: IfNotPresent
        restartPolicy: Never
    backoffLimit: 1
  pollingInterval: 5
  maxReplicaCount: ${maxReplicaCount}
  successfulJobsHistoryLimit: 0
  failedJobsHistoryLimit: 10
  triggers:
  - type: external
    metadata:
      scalerAddress: ${serviceName}.${testNamespace}:6000
      metricThreshold: "${threshold}"
      metricValue: "{{VALUE}}"
`
