import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import test from 'ava'

const dataExplorerDb = process.env['AZURE_DATA_EXPLORER_DB']
const dataExplorerEndpoint = process.env['AZURE_DATA_EXPLORER_ENDPOINT']
const spId = process.env['AZURE_SP_ID']
const spSecret = process.env['AZURE_SP_KEY']
const spTenantId = process.env['AZURE_SP_TENANT']

const testName = 'test-azure-data-explorer'
const dataExplorerNamespace = `${testName}-ns`
const scaleInDesiredReplicaCount = '0'
const scaleInMetricValue = '0'
const scaleOutDesiredReplicaCount = '4'
const scaleOutMetricValue = '18'
const scaledObjectName = `${testName}-scaled-object`
const secretName = `${testName}-secret`
const serviceName = `${testName}-sts`
const triggerAuthThroughClientAndSecretName = `${testName}-trigger-auth-client-and-secret`
const triggerAuthThroughPodIdentityName = `${testName}-trigger-auth-pod-identity`

test.before(t => {
    if (!spId || !spSecret || !spTenantId) {
        t.fail('required parameters for data explorer e2e test were not resolved')
    }

    sh.config.silent = true

    // Clean namespace if it exists. Otherwise, create new one.
    if (sh.exec(`kubectl get namespace ${dataExplorerNamespace}`).code === 0) {
        t.is(
            0,
            sh.exec(`kubectl delete all --all -n ${dataExplorerNamespace}`).code,
            'Clean namespace should work.')
    }
    else {
        t.is(
            0,
            sh.exec(`kubectl create namespace ${dataExplorerNamespace}`).code,
            'Create namespace should work.')
    }

    // Create secret
    const secretFile = tmp.fileSync()
    fs.writeFileSync(
        secretFile.name,
        secretYaml
            .replace('{{CLIENT_ID}}', Buffer.from(spId).toString('base64'))
            .replace('{{CLIENT_SECRET}}', Buffer.from(spSecret).toString('base64'))
            .replace('{{TENANT_ID}}', Buffer.from(spTenantId).toString('base64')))
    t.is(
        0,
        sh.exec(`kubectl apply -f ${secretFile.name} -n ${dataExplorerNamespace}`).code,
        'Creating a secret should work.')

    // Create deployment
    t.is(
        0,
        sh.exec(`kubectl apply -f ${createYamlFile(stsYaml)} -n ${dataExplorerNamespace}`).code,
        'Creating a statefulset should work.')

    // Validate initial replica count
    const replicaCount = sh.exec(`kubectl get sts ${serviceName} -n ${dataExplorerNamespace} -o jsonpath="{.spec.replicas}"`).stdout
    t.is(
        replicaCount,
        scaleInDesiredReplicaCount,
        `Replica count should start with ${scaleInDesiredReplicaCount} replicas.`)
})

test.serial.cb(`Replica count should be scaled out to ${scaleOutDesiredReplicaCount} replicas [Pod Identity]`, t => {
    // Create trigger auth through Pod Identity
    t.is(
        0,
        sh.exec(`kubectl apply -f ${createYamlFile(triggerAuthPodIdentityYaml)}`).code,
        'Creating a trigger auth should work.')

    // Create scaled object
    t.is(
        0,
        sh.exec(`kubectl apply -f ${createYamlFile(scaledObjectYaml)}`).code,
        'Creating a scaled object should work.')

    // Test scale out [Pod Identity]
    testDeploymentScale(t, scaleOutDesiredReplicaCount)
    t.end()
})

test.serial.cb(`Replica count should be scaled in to ${scaleInDesiredReplicaCount} replicas [Pod Identity]`, t => {
    // Edit azure data explorer query in order to scale down to 0 replicas
    const scaledObjectFile = tmp.fileSync()
    fs.writeFileSync(scaledObjectFile.name, scaledObjectYaml.replace(scaleOutMetricValue, scaleInMetricValue))
    t.is(
        0,
        sh.exec(`kubectl apply -f ${scaledObjectFile.name} -n ${dataExplorerNamespace}`).code,
        'Edit scaled object query should work.')

    // Test scale in [Pod Identity]
    testDeploymentScale(t, scaleInDesiredReplicaCount)
    t.end()
})

test.serial.cb(`Replica count should be scaled out to ${scaleOutDesiredReplicaCount} replicas [clientId & clientSecret]`, t => {
    // Create trigger auth through clientId, clientSecret and tenantId
    t.is(
        0,
        sh.exec(`kubectl apply -f ${createYamlFile(triggerAuthClientIdAndSecretYaml)} -n ${dataExplorerNamespace}`).code,
        'Change trigger of scaled object auth from pod identity to aad app should work.')

    // Change trigger auth of scaled object from pod identity to clientId and clientSecret
    const scaledObjectFile = tmp.fileSync()
    fs.writeFileSync(scaledObjectFile.name, scaledObjectYaml.replace(triggerAuthThroughPodIdentityName, triggerAuthThroughClientAndSecretName))
    t.is(
        0,
        sh.exec(`kubectl apply -f ${scaledObjectFile.name} -n ${dataExplorerNamespace}`).code,
        'Change trigger of scaled object auth from pod identity to aad app should work.')

    // Test scale out [clientId & clientSecret]
    testDeploymentScale(t, scaleOutDesiredReplicaCount)
    t.end()
})

test.after.always.cb('Clean up E2E K8s objects', t => {
    const resources = [
        `scaledobject.keda.sh/${scaledObjectName}`,
        `triggerauthentications.keda.sh/${triggerAuthThroughClientAndSecretName}`,
        `triggerauthentications.keda.sh/${triggerAuthThroughPodIdentityName}`,
        `statefulsets.apps/${serviceName}`,
        `v1/${secretName}`,
        `v1/${dataExplorerNamespace}`
    ]

    for (const resource of resources) {
        sh.exec(`kubectl delete ${resource} -n ${dataExplorerNamespace}`)
    }

    t.end()
})

function testDeploymentScale(t, desiredReplicaCount: string) {
    let currentReplicaCount = '-1'
    for (let i = 0; i < 120 && currentReplicaCount !== desiredReplicaCount; i++) {
        currentReplicaCount = sh.exec(`kubectl get sts ${serviceName} -n ${dataExplorerNamespace} -o jsonpath="{.spec.replicas}"`).stdout
        if (currentReplicaCount !== desiredReplicaCount) {
            sh.exec(`sleep 2s`)
        }
    }
    t.is(desiredReplicaCount, currentReplicaCount, `Replica count should be ${desiredReplicaCount} after some time`)
}

function createYamlFile(yaml: string) {
    const tmpFile = tmp.fileSync()
    fs.writeFileSync(tmpFile.name, yaml)
    return tmpFile.name
}

const stsYaml =
`apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: ${serviceName}
  namespace: ${dataExplorerNamespace}
spec:
  serviceName: ${serviceName}
  replicas: ${scaleInDesiredReplicaCount}
  selector:
    matchLabels:
      app: ${serviceName}
  template:
    metadata:
      labels:
        app: ${serviceName}
    spec:
      containers:
      - name: nginx
        image: nginx:1.16.1`

const scaledObjectYaml =
`apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: ${scaledObjectName}
  namespace: ${dataExplorerNamespace}
  labels:
    deploymentName: ${serviceName}
spec:
  scaleTargetRef:
    kind: StatefulSet
    name: ${serviceName}
  cooldownPeriod: 10
  minReplicaCount: 0
  maxReplicaCount: 10
  pollingInterval: 30
  triggers:
  - type: azure-data-explorer
    metadata:
      databaseName: ${dataExplorerDb}
      endpoint: ${dataExplorerEndpoint}
      query: print result = ${scaleOutMetricValue}
      threshold: "5"
    authenticationRef:
      name: ${triggerAuthThroughPodIdentityName}`

// K8s manifests for auth through Pod Identity
const triggerAuthPodIdentityYaml =
`apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: ${triggerAuthThroughPodIdentityName}
  namespace: ${dataExplorerNamespace}
spec:
  podIdentity:
    provider: azure`

// K8s manifests for auth through clientId, clientSecret and tenantId
const secretYaml =
`apiVersion: v1
kind: Secret
metadata:
  name: ${secretName}
  namespace: ${dataExplorerNamespace}
type: Opaque
data:
  clientId: {{CLIENT_ID}}
  clientSecret: {{CLIENT_SECRET}}
  tenantId: {{TENANT_ID}}`

const triggerAuthClientIdAndSecretYaml =
`apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: ${triggerAuthThroughClientAndSecretName}
  namespace: ${dataExplorerNamespace}
spec:
  secretTargetRef:
    - parameter: clientId
      name: ${secretName}
      key: clientId
    - parameter: clientSecret
      name: ${secretName}
      key: clientSecret
    - parameter: tenantId
      name: ${secretName}
      key: tenantId`
