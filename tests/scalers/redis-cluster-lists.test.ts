import test from 'ava'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import * as fs from 'fs'

const redisNamespace = 'redis-cluster'
const redisService = 'redis-cluster'
const testNamespace = 'redis-cluster-lists-test'
const redisStatefulSetName = 'redis-cluster'
const redisClusterName = 'redis-cluster'
const redisPassword = 'my-password'
let redisHost = ''
const redisPort = 6379
let redisAddress = ''
const listNameForHostPortRef = 'my-test-list-host-port-ref'
const listNameForAddressRef = 'my-test-list-address-ref'
const listNameForHostPortTriggerAuth = 'my-test-list-host-port-trigger'
const redisWorkerHostPortRefDeploymentName = 'redis-worker-test-hostport'
const redisWorkerAddressRefDeploymentName = 'redis-worker-test-address'
const redisWorkerHostPortRefTriggerAuthDeploymentName = 'redis-worker-test-hostport-triggerauth'
const itemsToWrite = 200
const deploymentContainerImage = 'goku321/redis-cluster-list:v1.7'
const writeJobNameForHostPortRef = 'redis-writer-host-port-ref'
const writeJobNameForAddressRef = 'redis-writer-address-ref'
const writeJobNameForHostPortInTriggerAuth = 'redis-writer-host-port-trigger-auth'

test.before(t => {
    // Deploy Redis cluster.
    sh.exec(`kubectl create namespace ${redisNamespace}`)
    sh.exec(`helm repo add bitnami https://charts.bitnami.com/bitnami`)

    let clusterStatus = sh.exec(`helm install --timeout 600s ${redisClusterName} --namespace ${redisNamespace} --set "global.redis.password=${redisPassword}" bitnami/redis-cluster`).code
    t.is(0,
        clusterStatus,
        'creating a Redis cluster should work.'
    )

    // Wait for Redis cluster to be ready.
    let redisReplicaCount = '0'
    for (let i = 0; i < 30; i++) {
        redisReplicaCount = sh.exec(`kubectl get statefulset/${redisStatefulSetName} -n ${redisNamespace} -o jsonpath='{.spec.replicas}'`).stdout
        if (redisReplicaCount != '6') {
            sh.exec('sleep 2s')
        }
    }
    t.is('6', redisReplicaCount, 'Redis is not in a ready state')

    // Get Redis cluster address.
    redisHost = sh.exec(`kubectl get svc ${redisService} -n ${redisNamespace} -o jsonpath='{.spec.clusterIP}'`)
    redisAddress = `${redisHost}:${redisPort}`

    // Create test namespace.
    sh.exec(`kubectl create namespace ${testNamespace}`)

    const triggerAuthTmpFile = tmp.fileSync()
    const base64Password = Buffer.from(redisPassword).toString('base64')
    fs.writeFileSync(triggerAuthTmpFile.name, scaledObjectTriggerAuthYaml.replace('{{REDIS_PASSWORD}}', base64Password))

    t.is(
        0,
        sh.exec(`kubectl apply -f ${triggerAuthTmpFile.name} --namespace ${testNamespace}`).code,
        'creating trigger auth should work..'
    )

    const triggerAuthHostPortTmpFile = tmp.fileSync()

    fs.writeFileSync(triggerAuthHostPortTmpFile.name,
        scaledObjectTriggerAuthHostPortYaml.replace('{{REDIS_PASSWORD}}', base64Password)
            .replace('{{REDIS_HOSTS}}', Buffer.from(redisHost).toString('base64'))
            .replace('{{REDIS_PORTS}}', Buffer.from(redisPort.toString()).toString('base64'))
    )

    t.is(
        0,
        sh.exec(`kubectl apply -f ${triggerAuthHostPortTmpFile.name} --namespace ${testNamespace}`).code,
        'creating trigger auth with host port should work..'
    )

    // Create a deployment with host and port.
    const deploymentHostPortRefTmpFile = tmp.fileSync()

    fs.writeFileSync(deploymentHostPortRefTmpFile.name, redisListDeployHostPortYaml.replace(/{{REDIS_PASSWORD}}/g, redisPassword)
        .replace(/{{REDIS_HOSTS}}/g, redisHost)
        .replace(/{{REDIS_PORTS}}/g, redisPort.toString())
        .replace(/{{LIST_NAME}}/g, listNameForHostPortRef)
        .replace(/{{DEPLOYMENT_NAME}}/g, redisWorkerHostPortRefDeploymentName)
        .replace(/{{CONTAINER_IMAGE}}/g, deploymentContainerImage)
    )

    t.is(
        0,
        sh.exec(`kubectl apply -f ${deploymentHostPortRefTmpFile.name} --namespace ${testNamespace}`).code,
        'creating a deployment using redis host and port envs should work..'
    )

    const deploymentAddressRefTmpFile = tmp.fileSync()

    fs.writeFileSync(deploymentAddressRefTmpFile.name, redisListDeployAddressYaml.replace(/{{REDIS_PASSWORD}}/g, redisPassword)
        .replace(/{{REDIS_ADDRESSES}}/g, redisAddress)
        .replace(/{{LIST_NAME}}/g, listNameForAddressRef)
        .replace(/{{DEPLOYMENT_NAME}}/g, redisWorkerAddressRefDeploymentName)
        .replace(/{{CONTAINER_IMAGE}}/g, deploymentContainerImage)
    )

    t.is(
        0,
        sh.exec(`kubectl apply -f ${deploymentAddressRefTmpFile.name} --namespace ${testNamespace}`).code,
        'creating a deployment using redis address var should work..'
    )


    const deploymentHostPortRefTriggerAuthTmpFile = tmp.fileSync()

    fs.writeFileSync(deploymentHostPortRefTriggerAuthTmpFile.name, redisListDeployHostPortInTriggerAuhYaml.replace(/{{REDIS_PASSWORD}}/g, redisPassword)
        .replace(/{{REDIS_HOSTS}}/g, redisHost)
        .replace(/{{REDIS_PORTS}}/g, redisPort.toString())
        .replace(/{{LIST_NAME}}/g, listNameForHostPortTriggerAuth)
        .replace(/{{DEPLOYMENT_NAME}}/g, redisWorkerHostPortRefTriggerAuthDeploymentName)
        .replace(/{{CONTAINER_IMAGE}}/g, deploymentContainerImage)
    )

    t.is(
        0,
        sh.exec(`kubectl apply -f ${deploymentHostPortRefTriggerAuthTmpFile.name} --namespace ${testNamespace}`).code,
        'creating a deployment using redis host port in trigger auth should work..'
    )
})

test.serial('Deployment for redis host and port env vars should have 0 replica on start', t => {

    const replicaCount = sh.exec(
        `kubectl get deployment/${redisWorkerHostPortRefDeploymentName} --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
    ).stdout
    t.is(replicaCount, '0', 'replica count should start out as 0')
})


test.serial(`Deployment using redis host port env vars should max and scale to 5 with ${itemsToWrite} items written to list and back to 0`, t => {
    runWriteJob(t, writeJobNameForHostPortRef, listNameForHostPortRef)

    let replicaCount = '0'
    for (let i = 0; i < 30 && replicaCount !== '5'; i++) {
        replicaCount = sh.exec(
            `kubectl get deployment/${redisWorkerHostPortRefDeploymentName} --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
        ).stdout
        t.log('(scale up) replica count is:' + replicaCount)
        if (replicaCount !== '5') {
            sh.exec('sleep 3s')
        }
    }

    t.is('5', replicaCount, 'Replica count should be 5 within 60 seconds')

    for (let i = 0; i < 12 && replicaCount !== '0'; i++) {
        replicaCount = sh.exec(
            `kubectl get deployment/${redisWorkerHostPortRefDeploymentName} --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
        ).stdout
        t.log('(scale down) replica count is:' + replicaCount)
        if (replicaCount !== '0') {
            sh.exec('sleep 10s')
        }
    }

    t.is('0', replicaCount, 'Replica count should be 0 within 2 minutes')
})

test.serial('Deployment for redis address env var should have 0 replica on start', t => {

    const replicaCount = sh.exec(
        `kubectl get deployment/${redisWorkerAddressRefDeploymentName} --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
    ).stdout
    t.is(replicaCount, '0', 'replica count should start out as 0')
})



test.serial(`Deployment using redis address env var should max and scale to 5 with ${itemsToWrite} items written to list and back to 0`, t => {

    runWriteJob(t, writeJobNameForAddressRef, listNameForAddressRef)

    let replicaCount = '0'
    for (let i = 0; i < 30 && replicaCount !== '5'; i++) {
        replicaCount = sh.exec(
            `kubectl get deployment/${redisWorkerAddressRefDeploymentName} --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
        ).stdout
        t.log('(scale up) replica count is:' + replicaCount)
        if (replicaCount !== '5') {
            sh.exec('sleep 3s')
        }
    }

    t.is('5', replicaCount, 'Replica count should be 5 within 60 seconds')

    for (let i = 0; i < 12 && replicaCount !== '0'; i++) {
        replicaCount = sh.exec(
            `kubectl get deployment/${redisWorkerAddressRefDeploymentName} --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
        ).stdout
        t.log('(scale down) replica count is:' + replicaCount)
        if (replicaCount !== '0') {
            sh.exec('sleep 10s')
        }
    }

    t.is('0', replicaCount, 'Replica count should be 0 within 2 minutes')
})


test.serial('Deployment for redis host and port in the trigger auth should have 0 replica on start', t => {

    const replicaCount = sh.exec(
        `kubectl get deployment/${redisWorkerHostPortRefTriggerAuthDeploymentName} --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
    ).stdout
    t.is(replicaCount, '0', 'replica count should start out as 0')
})


test.serial(`Deployment using redis host port in triggerAuth should max and scale to 5 with ${itemsToWrite} items written to list and back to 0`, t => {

    runWriteJob(t, writeJobNameForHostPortInTriggerAuth, listNameForHostPortTriggerAuth)

    let replicaCount = '0'
    for (let i = 0; i < 30 && replicaCount !== '5'; i++) {
        replicaCount = sh.exec(
            `kubectl get deployment/${redisWorkerHostPortRefTriggerAuthDeploymentName} --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
        ).stdout
        t.log('(scale up) replica count is:' + replicaCount)
        if (replicaCount !== '5') {
            sh.exec('sleep 3s')
        }
    }

    t.is('5', replicaCount, 'Replica count should be 5 within 60 seconds')

    for (let i = 0; i < 12 && replicaCount !== '0'; i++) {
        replicaCount = sh.exec(
            `kubectl get deployment/${redisWorkerHostPortRefTriggerAuthDeploymentName} --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
        ).stdout
        t.log('(scale down) replica count is:' + replicaCount)
        if (replicaCount !== '0') {
            sh.exec('sleep 10s')
        }
    }

    t.is('0', replicaCount, 'Replica count should be 0 within 2 minutes')
})


test.after.always.cb('clean up deployment', t => {
    const resources = [
        `job/${writeJobNameForHostPortRef}`,
        `job/${writeJobNameForAddressRef}`,
        `job/${writeJobNameForHostPortInTriggerAuth}`,
        `scaledobject.keda.sh/${redisWorkerHostPortRefDeploymentName}`,
        `scaledobject.keda.sh/${redisWorkerAddressRefDeploymentName}`,
        `scaledobject.keda.sh/${redisWorkerHostPortRefTriggerAuthDeploymentName}`,
        'triggerauthentication.keda.sh/keda-redis-cluster-list-triggerauth',
        'triggerauthentication.keda.sh/keda-redis-cluster-list-triggerauth-host-port',
        `deployment/${redisWorkerAddressRefDeploymentName}`,
        `deployment/${redisWorkerHostPortRefTriggerAuthDeploymentName}`,
        `deployment/${redisWorkerHostPortRefDeploymentName}`,
        'secret/redis-password',
    ]

    for (const resource of resources) {
        sh.exec(`kubectl delete ${resource} --namespace ${testNamespace}`)
    }
    sh.exec(`kubectl delete namespace ${testNamespace}`)

    sh.exec(`helm delete ${redisClusterName} --namespace ${redisNamespace}`)
    sh.exec(`kubectl delete namespace ${redisNamespace}`)
    t.end()
})

function runWriteJob(t, jobName, listName) {
    // write to list
    const tmpFile = tmp.fileSync()
    fs.writeFileSync(tmpFile.name, writeJobYaml.replace('{{REDIS_ADDRESSES}}', redisAddress).replace('{{REDIS_PASSWORD}}', redisPassword)
        .replace('{{LIST_NAME}}', listName)
        .replace('{{NUMBER_OF_ITEMS_TO_WRITE}}', itemsToWrite.toString())
        .replace('{{CONTAINER_IMAGE}}', deploymentContainerImage)
        .replace('{{JOB_NAME}}', jobName)
    )

    t.is(
        0,
        sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${testNamespace}`).code,
        'list writer job should apply.'
    )

    // wait for the write job to complete
    for (let i = 0; i < 20; i++) {
        const succeeded = sh.exec(`kubectl get job ${writeJobNameForHostPortRef} --namespace ${testNamespace} -o jsonpath='{.items[0].status.succeeded}'`).stdout
        if (succeeded == '1') {
            break
        }
        sh.exec('sleep 1s')
    }
}

const redisListDeployHostPortYaml = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{DEPLOYMENT_NAME}}
  labels:
    app: {{DEPLOYMENT_NAME}}
spec:
  replicas: 0
  selector:
    matchLabels:
      app: {{DEPLOYMENT_NAME}}
  template:
    metadata:
      labels:
        app: {{DEPLOYMENT_NAME}}
    spec:
      containers:
      - name: redis-worker
        image: {{CONTAINER_IMAGE}}
        imagePullPolicy: IfNotPresent
        command: ["./main"]
        args: ["read"]
        env:
        - name: REDIS_HOSTS
          value: {{REDIS_HOSTS}}
        - name: REDIS_PORTS
          value: "{{REDIS_PORTS}}"
        - name: LIST_NAME
          value: {{LIST_NAME}}
        - name: REDIS_PASSWORD
          value: {{REDIS_PASSWORD}}
        - name: READ_PROCESS_TIME
          value: "500"
---
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{DEPLOYMENT_NAME}}
spec:
  scaleTargetRef:
    name: {{DEPLOYMENT_NAME}}
  pollingInterval: 5
  cooldownPeriod: 30
  minReplicaCount: 0
  maxReplicaCount: 5
  triggers:
  - type: redis-cluster
    metadata:
      hostsFromEnv: REDIS_HOSTS
      portsFromEnv: REDIS_PORTS
      listName: {{LIST_NAME}}
      listLength: "5"
    authenticationRef:
      name: keda-redis-cluster-list-triggerauth
`


const redisListDeployAddressYaml = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{DEPLOYMENT_NAME}}
  labels:
    app: {{DEPLOYMENT_NAME}}
spec:
  replicas: 0
  selector:
    matchLabels:
      app: {{DEPLOYMENT_NAME}}
  template:
    metadata:
      labels:
        app: {{DEPLOYMENT_NAME}}
    spec:
      containers:
      - name: redis-worker
        image: {{CONTAINER_IMAGE}}
        imagePullPolicy: IfNotPresent
        command: ["./main"]
        args: ["read"]
        env:
        - name: REDIS_ADDRESSES
          value: {{REDIS_ADDRESSES}}
        - name: LIST_NAME
          value: {{LIST_NAME}}
        - name: REDIS_PASSWORD
          value: {{REDIS_PASSWORD}}
        - name: READ_PROCESS_TIME
          value: "500"
---
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{DEPLOYMENT_NAME}}
spec:
  scaleTargetRef:
    name: {{DEPLOYMENT_NAME}}
  pollingInterval: 5
  cooldownPeriod: 30
  minReplicaCount: 0
  maxReplicaCount: 5
  triggers:
  - type: redis-cluster
    metadata:
      addressesFromEnv: REDIS_ADDRESSES
      listName: {{LIST_NAME}}
      listLength: "5"
    authenticationRef:
      name: keda-redis-cluster-list-triggerauth
`

const redisListDeployHostPortInTriggerAuhYaml = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{DEPLOYMENT_NAME}}
  labels:
    app: {{DEPLOYMENT_NAME}}
spec:
  replicas: 0
  selector:
    matchLabels:
      app: {{DEPLOYMENT_NAME}}
  template:
    metadata:
      labels:
        app: {{DEPLOYMENT_NAME}}
    spec:
      containers:
      - name: redis-worker
        image: {{CONTAINER_IMAGE}}
        imagePullPolicy: IfNotPresent
        command: ["./main"]
        args: ["read"]
        env:
        - name: REDIS_HOSTS
          value: {{REDIS_HOSTS}}
        - name: REDIS_PORTS
          value: "{{REDIS_PORTS}}"
        - name: LIST_NAME
          value: {{LIST_NAME}}
        - name: REDIS_PASSWORD
          value: {{REDIS_PASSWORD}}
        - name: READ_PROCESS_TIME
          value: "500"
---
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{DEPLOYMENT_NAME}}
spec:
  scaleTargetRef:
    name: {{DEPLOYMENT_NAME}}
  pollingInterval: 5
  cooldownPeriod: 30
  minReplicaCount: 0
  maxReplicaCount: 5
  triggers:
  - type: redis-cluster
    metadata:
      listName: {{LIST_NAME}}
      listLength: "5"
    authenticationRef:
      name: keda-redis-cluster-list-triggerauth-host-port
`

const scaledObjectTriggerAuthHostPortYaml = `apiVersion: v1
kind: Secret
metadata:
  name: redis-config
type: Opaque
data:
  password: {{REDIS_PASSWORD}}
  redisHost: {{REDIS_HOSTS}}
  redisPort: {{REDIS_PORTS}}
---
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: keda-redis-cluster-list-triggerauth-host-port
spec:
  secretTargetRef:
    - parameter: password
      name: redis-config
      key: password
    - parameter: hosts
      name: redis-config
      key: redisHost
    - parameter: ports
      name: redis-config
      key: redisPort
`

const scaledObjectTriggerAuthYaml = `apiVersion: v1
kind: Secret
metadata:
  name: redis-password
type: Opaque
data:
  password: {{REDIS_PASSWORD}}
---
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: keda-redis-cluster-list-triggerauth
spec:
  secretTargetRef:
    - parameter: password
      name: redis-password
      key: password
`


const writeJobYaml = `apiVersion: batch/v1
kind: Job
metadata:
  name: {{JOB_NAME}}
spec:
  template:
    spec:
      containers:
      - name: redis
        image: {{CONTAINER_IMAGE}}
        imagePullPolicy: IfNotPresent
        command: ["./main"]
        env:
        - name: REDIS_ADDRESSES
          value: {{REDIS_ADDRESSES}}
        - name: REDIS_PASSWORD
          value: {{REDIS_PASSWORD}}
        - name: LIST_NAME
          value: {{LIST_NAME}}
        - name: NO_LIST_ITEMS_TO_WRITE
          value: "{{NUMBER_OF_ITEMS_TO_WRITE}}"
        args: ["write"]
      restartPolicy: Never
  backoffLimit: 4
`
