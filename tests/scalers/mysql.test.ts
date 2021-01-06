import * as async from 'async'
import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import test from 'ava'

const testNamespace = 'mysql-test'
const mySQLNamespace = 'mysql'
const mySQLUsername = 'test-user'
const mySQLPassword = 'test-password'
const mySQLDatabase = 'test_db'
const mySQLRootPassword = 'some-password'
const deploymentName = 'worker'

test.before(t => {
    // install mysql
    sh.exec(`kubectl create namespace ${mySQLNamespace}`)
    const mySQLTmpFile = tmp.fileSync()
    fs.writeFileSync(mySQLTmpFile.name, mysqlDeploymentYaml.replace('{{MYSQL_USER}}', mySQLUsername)
        .replace('{{MYSQL_PASSWORD}}', mySQLPassword)
        .replace('{{MYSQL_DATABASE}}', mySQLDatabase)
        .replace('{{MYSQL_ROOT_PASSWORD}}', mySQLRootPassword))

    t.is(0, sh.exec(`kubectl apply --namespace ${mySQLNamespace} -f ${mySQLTmpFile.name}`).code, 'creating a MySQL deployment should work.')
    // wait for mysql to load
    let mysqlReadyReplicaCount = '0'
    for (let i = 0; i < 30; i++) {
        mysqlReadyReplicaCount = sh.exec(`kubectl get deploy/mysql -n ${mySQLNamespace} -o jsonpath='{.status.readyReplicas}'`).stdout
        if (mysqlReadyReplicaCount != '1') {
            sh.exec('sleep 2s')
        }
    }
    t.is('1', mysqlReadyReplicaCount, 'MySQL is not in a ready state')

    // create table that used by the job and the worker
    const createTableSQL = `CREATE TABLE ${mySQLDatabase}.task_instance (id INT AUTO_INCREMENT PRIMARY KEY,state VARCHAR(10));`
    const mysqlPod = sh.exec(`kubectl get po -n ${mySQLNamespace} -o jsonpath='{.items[0].metadata.name}'`).stdout
    t.not(mysqlPod, '')
    sh.exec( `kubectl exec -n ${mySQLNamespace} ${mysqlPod} -- mysql -u${mySQLUsername} -p${mySQLPassword} -e \"${createTableSQL}\"`)

    sh.config.silent = true

    sh.exec(`kubectl create namespace ${testNamespace}`)

    // deploy streams consumer app, scaled object etc.
    const tmpFile = tmp.fileSync()
    const base64ConnectionString = Buffer.from(`${mySQLUsername}:${mySQLPassword}@tcp(mysql.${mySQLNamespace}.svc.cluster.local:3306)/${mySQLDatabase}`).toString('base64')

    fs.writeFileSync(tmpFile.name, deployYaml.replace('{{MYSQL_CONNECTION_STRING}}', base64ConnectionString).replace('{{DEPLOYMENT_NAME}}', deploymentName))

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

test.serial(`Deployment should scale to 5 (the max) then back to 0`, t => {
    const tmpFile = tmp.fileSync()
    fs.writeFileSync(tmpFile.name, insertRecordsJobYaml)
    t.is(
        0,
        sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${testNamespace}`).code,
        'creating job should work.'
    )

    let replicaCount = '0'

    const maxReplicaCount = '5'

    for (let i = 0; i < 30 && replicaCount !== maxReplicaCount; i++) {
        replicaCount = sh.exec(
            `kubectl get deployment.apps/${deploymentName} --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
        ).stdout
        if (replicaCount !== maxReplicaCount) {
            sh.exec('sleep 2s')
        }
    }

    t.is(maxReplicaCount, replicaCount, `Replica count should be ${maxReplicaCount} after 60 seconds`)

    for (let i = 0; i < 36 && replicaCount !== '0'; i++) {
      replicaCount = sh.exec(
        `kubectl get deployment.apps/${deploymentName} --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
      ).stdout
      if (replicaCount !== '0') {
        sh.exec('sleep 5s')
      }
    }

    t.is('0', replicaCount, 'Replica count should be 0 after 3 minutes')
})

test.after.always.cb('clean up mysql deployment', t => {
    const resources = [
        'scaledobject.keda.sh/mysql-scaledobject',
        'triggerauthentication.keda.sh/keda-trigger-auth-mysql-secret',
        `deployment.apps/${deploymentName}`,
        'secret/mysql-secrets',
        'job/mysql-insert-job',
    ]

    for (const resource of resources) {
        sh.exec(`kubectl delete ${resource} --namespace ${testNamespace}`)
    }
    sh.exec(`kubectl delete namespace ${testNamespace}`)

    // uninstall mysql
    sh.exec(`kubectl delete --namespace ${mySQLNamespace} deploy/mysql`)
    sh.exec(`kubectl delete namespace ${mySQLNamespace}`)

    t.end()
})

const deployYaml = `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: mysql-update-worker
  name: {{DEPLOYMENT_NAME}}
spec:
  replicas: 0
  selector:
    matchLabels:
      app: mysql-update-worker
  template:
    metadata:
      labels:
        app: mysql-update-worker
    spec:
      containers:
      - image: docker.io/kedacore/tests-mysql:824031e
        imagePullPolicy: Always
        name: mysql-processor-test
        command:
          - /app
          - update
        env:
          - name: TASK_INSTANCES_COUNT
            value: "10000"
          - name: CONNECTION_STRING
            valueFrom:
              secretKeyRef:
                name: mysql-secrets
                key: mysql_conn_str
---
apiVersion: v1
kind: Secret
metadata:
  name: mysql-secrets
type: Opaque
data:
  mysql_conn_str: {{MYSQL_CONNECTION_STRING}}
---
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: keda-trigger-auth-mysql-secret
spec:
  secretTargetRef:
  - parameter: connectionString
    name: mysql-secrets
    key: mysql_conn_str
---
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: mysql-scaledobject
spec:
  scaleTargetRef:
    name: worker
  pollingInterval: 5
  cooldownPeriod:  10
  minReplicaCount: 0
  maxReplicaCount: 5
  triggers:
  - type: mysql
    metadata:
      queryValue: "4"
      query: "SELECT CEIL(COUNT(*) / 5) FROM task_instance WHERE state='running' OR state='queued'"
    authenticationRef:
      name: keda-trigger-auth-mysql-secret`

const insertRecordsJobYaml = `apiVersion: batch/v1
kind: Job
metadata:
  labels:
    app: mysql-insert-job
  name: mysql-insert-job
spec:
  template:
    metadata:
      labels:
        app: mysql-insert-job
    spec:
      containers:
      - image: docker.io/kedacore/tests-mysql:824031e
        imagePullPolicy: Always
        name: mysql-processor-test
        command:
          - /app
          - insert
        env:
          - name: TASK_INSTANCES_COUNT
            value: "2000"
          - name: CONNECTION_STRING
            valueFrom:
              secretKeyRef:
                name: mysql-secrets
                key: mysql_conn_str
      restartPolicy: Never
  backoffLimit: 4`


const mysqlDeploymentYaml = `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: mysql
  name: mysql
spec:
  replicas: 1
  selector:
    matchLabels:
      app: mysql
  template:
    metadata:
      labels:
        app: mysql
    spec:
      containers:
      - image: mysql:8.0.20
        name: mysql
        env:
          - name: MYSQL_ROOT_PASSWORD
            value: {{MYSQL_ROOT_PASSWORD}}
          - name: MYSQL_USER
            value: {{MYSQL_USER}}
          - name: MYSQL_PASSWORD
            value: {{MYSQL_PASSWORD}}
          - name: MYSQL_DATABASE
            value: {{MYSQL_DATABASE}}
        ports:
          - name: mysql
            protocol: TCP
            containerPort: 3600
        readinessProbe:
          exec:
            command:
            - sh
            - -c
            - "mysqladmin ping -u root -p{{MYSQL_ROOT_PASSWORD}}"
---
apiVersion: v1
kind: Service
metadata:
  labels:
    app: mysql
  name: mysql
spec:
  ports:
  - port: 3306
    protocol: TCP
    targetPort: 3306
  selector:
    app: mysql
  type: ClusterIP`
