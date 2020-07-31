import test from 'ava'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import * as fs from 'fs'

const redisNamespace = 'redis'
const testNamespace = 'redis-lists-test'
const redisDeploymentName = 'redis'
const redisPassword = 'my-password'
const redisHost = `redis.${redisNamespace}.svc.cluster.local`
const redisPort = 6379
const redisAddress = `${redisHost}:${redisPort}`
const listNameForHostPortRef = 'my-test-list-host-port-ref'
const listNameForAddressRef = 'my-test-list-address-ref'
const listNameForHostPortTriggerAuth = 'my-test-list-host-port-trigger'
const redisWorkerHostPortRefDeploymentName = 'redis-worker-test-hostport'
const redisWorkerAddressRefDeploymentName = 'redis-worker-test-address'
const redisWorkerHostPortRefTriggerAuthDeploymentName = 'redis-worker-test-hostport-triggerauth'
const itemsToWrite = 200
const deploymentContainerImage = 'kedacore/tests-redis-lists:824031e'
const writeJobNameForHostPortRef = 'redis-writer-host-port-ref'
const writeJobNameForAddressRef = 'redis-writer-address-ref'
const writeJobNameForHostPortInTriggerAuth = 'redis-writer-host-port-trigger-auth'

test.before(t => {
    // setup Redis
    sh.exec(`kubectl create namespace ${redisNamespace}`)

    const redisDeployTmpFile = tmp.fileSync()
    fs.writeFileSync(redisDeployTmpFile.name, redisDeployYaml.replace('{{REDIS_PASSWORD}}', redisPassword))

    t.is(0, sh.exec(`kubectl apply --namespace ${redisNamespace} -f ${redisDeployTmpFile.name}`).code, 'creating a Redis deployment should work.')

    // wait for redis to be ready
    let redisReplicaCount = '0'
    for (let i = 0; i < 30; i++) {
        redisReplicaCount = sh.exec(`kubectl get deploy/${redisDeploymentName} -n ${redisNamespace} -o jsonpath='{.spec.replicas}'`).stdout
        if (redisReplicaCount != '1') {
            sh.exec('sleep 2s')
        }
    }
    t.is('1', redisReplicaCount, 'Redis is not in a ready state')

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
            .replace('{{REDIS_HOST}}', Buffer.from(redisHost).toString('base64'))
            .replace('{{REDIS_PORT}}', Buffer.from(redisPort.toString()).toString('base64'))
    )

    t.is(
        0,
        sh.exec(`kubectl apply -f ${triggerAuthHostPortTmpFile.name} --namespace ${testNamespace}`).code,
        'creating trigger auth with host port should work..'
    )

    const deploymentHostPortRefTmpFile = tmp.fileSync()

    fs.writeFileSync(deploymentHostPortRefTmpFile.name, redisRedisListDeployHostPortYaml.replace(/{{REDIS_PASSWORD}}/g, redisPassword)
        .replace(/{{REDIS_HOST}}/g, redisHost)
        .replace(/{{REDIS_PORT}}/g, redisPort.toString())
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
        .replace(/{{REDIS_ADDRESS}}/g, redisAddress)
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
        .replace(/{{REDIS_HOST}}/g, redisHost)
        .replace(/{{REDIS_PORT}}/g, redisPort.toString())
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
    for (let i = 0; i < 20 && replicaCount !== '5'; i++) {
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
    for (let i = 0; i < 20 && replicaCount !== '5'; i++) {
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
    for (let i = 0; i < 20 && replicaCount !== '5'; i++) {
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
        `scaledobject.keda.k8s.io/${redisWorkerHostPortRefDeploymentName}`,
        `scaledobject.keda.k8s.io/${redisWorkerAddressRefDeploymentName}`,
        `scaledobject.keda.k8s.io/${redisWorkerHostPortRefTriggerAuthDeploymentName}`,
        'triggerauthentications.keda.k8s.io/keda-redis-list-triggerauth',
        'triggerauthentications.keda.k8s.io/keda-redis-list-triggerauth-host-port',
        `deployment/${redisWorkerAddressRefDeploymentName}`,
        `deployment/${redisWorkerHostPortRefTriggerAuthDeploymentName}`,
        `deployment/${redisWorkerHostPortRefDeploymentName}`,
        'secret/redis-password',
    ]

    for (const resource of resources) {
        sh.exec(`kubectl delete ${resource} --namespace ${testNamespace}`)
    }
    sh.exec(`kubectl delete namespace ${testNamespace}`)

    sh.exec(`kubectl delete namespace ${redisNamespace}`)
    t.end()
})

function runWriteJob(t, jobName, listName) {
    // write to list
    const tmpFile = tmp.fileSync()
    fs.writeFileSync(tmpFile.name, writeJobYaml.replace('{{REDIS_ADDRESS}}', redisAddress).replace('{{REDIS_PASSWORD}}', redisPassword)
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

const redisDeployYaml = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis
  namespace: redis
spec:
  selector:
    matchLabels:
      app: redis
  replicas: 1
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
      - name: master
        image: redis:6.0.6
        command: ["redis-server", "--requirepass", {{REDIS_PASSWORD}}]
        ports:
        - containerPort: 6379
---
apiVersion: v1
kind: Service
metadata:
  name: redis
  namespace: redis
  labels:
    app: redis
spec:
  ports:
  - port: 6379
    targetPort: 6379
  selector:
    app: redis
`

const redisRedisListDeployHostPortYaml = `apiVersion: apps/v1
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
        args: ["read"]
        env:
        - name: REDIS_HOST
          value: {{REDIS_HOST}}
        - name: REDIS_PORT
          value: "{{REDIS_PORT}}"
        - name: LIST_NAME
          value: {{LIST_NAME}}
        - name: REDIS_PASSWORD
          value: {{REDIS_PASSWORD}}
        - name: READ_PROCESS_TIME
          value: "200"
---
apiVersion: keda.k8s.io/v1alpha1
kind: ScaledObject
metadata:
  name: {{DEPLOYMENT_NAME}}
  labels:
    deploymentName: {{DEPLOYMENT_NAME}}
spec:
  scaleTargetRef:
    deploymentName: {{DEPLOYMENT_NAME}}
  pollingInterval: 5 
  cooldownPeriod: 30
  minReplicaCount: 0
  maxReplicaCount: 5
  triggers:
  - type: redis
    metadata:
      host: REDIS_HOST
      port: REDIS_PORT
      listName: {{LIST_NAME}} 
      listLength: "5"
    authenticationRef:
      name: keda-redis-list-triggerauth
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
        args: ["read"]
        env:
        - name: REDIS_ADDRESS
          value: {{REDIS_ADDRESS}}
        - name: LIST_NAME
          value: {{LIST_NAME}}
        - name: REDIS_PASSWORD
          value: {{REDIS_PASSWORD}}
        - name: READ_PROCESS_TIME
          value: "500"
---
apiVersion: keda.k8s.io/v1alpha1
kind: ScaledObject
metadata:
  name: {{DEPLOYMENT_NAME}}
  labels:
    deploymentName: {{DEPLOYMENT_NAME}}
spec:
  scaleTargetRef:
    deploymentName: {{DEPLOYMENT_NAME}}
  pollingInterval: 5 
  cooldownPeriod: 30
  minReplicaCount: 0
  maxReplicaCount: 5
  triggers:
  - type: redis
    metadata:
      address: REDIS_ADDRESS
      listName: {{LIST_NAME}} 
      listLength: "5"
    authenticationRef:
      name: keda-redis-list-triggerauth
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
        args: ["read"]
        env:
        - name: REDIS_HOST
          value: {{REDIS_HOST}}
        - name: REDIS_PORT
          value: "{{REDIS_PORT}}"
        - name: LIST_NAME
          value: {{LIST_NAME}}
        - name: REDIS_PASSWORD
          value: {{REDIS_PASSWORD}}
        - name: READ_PROCESS_TIME
          value: "200"
---
apiVersion: keda.k8s.io/v1alpha1
kind: ScaledObject
metadata:
  name: {{DEPLOYMENT_NAME}}
  labels:
    deploymentName: {{DEPLOYMENT_NAME}}
spec:
  scaleTargetRef:
    deploymentName: {{DEPLOYMENT_NAME}}
  pollingInterval: 5 
  cooldownPeriod: 30
  minReplicaCount: 0
  maxReplicaCount: 5
  triggers:
  - type: redis
    metadata:
      listName: {{LIST_NAME}} 
      listLength: "5"
    authenticationRef:
      name: keda-redis-list-triggerauth-host-port
`

const scaledObjectTriggerAuthHostPortYaml = `apiVersion: v1
kind: Secret
metadata:
  name: redis-config
type: Opaque
data:
  password: {{REDIS_PASSWORD}}
  redisHost: {{REDIS_HOST}}
  redisPort: {{REDIS_PORT}}
---
apiVersion: keda.k8s.io/v1alpha1
kind: TriggerAuthentication
metadata:
  name: keda-redis-list-triggerauth-host-port
spec:
  secretTargetRef:
    - parameter: password
      name: redis-config
      key: password
    - parameter: host
      name: redis-config
      key: redisHost
    - parameter: port
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
apiVersion: keda.k8s.io/v1alpha1
kind: TriggerAuthentication
metadata:
  name: keda-redis-list-triggerauth
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
        env: 
        - name: REDIS_ADDRESS
          value: {{REDIS_ADDRESS}}
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