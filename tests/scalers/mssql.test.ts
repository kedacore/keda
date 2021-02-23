import test from 'ava'
import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'

const mssqlns = "mssql"
const testns = "mssql-app"
const mssqlName = "mssqlinst"
const password = "Pass@word1"
const hostname = `${mssqlName}.${mssqlns}.svc.cluster.local`
const database = "TestDB"
const sqlConnectionString = `Server=${hostname};Database=${database};User ID=sa;Password=${password};`
const appName = "consumer-app"

const getReplicaCountCommand = `kubectl get deployment.apps/${appName} -n ${testns} -o jsonpath="{.spec.replicas}"`

test.before(t => {
    sh.config.silent = true

    // deploy the mssql container
    sh.exec(`kubectl create namespace ${mssqlns}`)
    const mssqlDeploymentYamlFile = tmp.fileSync()
    fs.writeFileSync(mssqlDeploymentYamlFile.name, mssqlDeploymentYaml)
    t.is(0, sh.exec(`kubectl apply -n ${mssqlns} -f ${mssqlDeploymentYamlFile.name}`).code, 'creating the mssql deployment should work.')

    // wait for the mssql container to be ready
    let readyReplicaCount = '0'
    for (let i = 0; i < 30; i++) {
      readyReplicaCount = sh.exec(`kubectl get deploy/mssql-deployment -n ${mssqlns} -o jsonpath='{.status.readyReplicas}'`).stdout
        if (readyReplicaCount != '1') {
            sh.exec('sleep 2s')
        }
    }
    t.is('1', readyReplicaCount, 'mssql-deployment is not in a ready state!')

    // create the mssql database
    const mssqlPod = sh.exec(`kubectl get pods -n ${mssqlns} -o jsonpath='{.items[0].metadata.name}'`).stdout
    t.not(mssqlPod, '')
    sh.exec(`kubectl exec -n ${mssqlns} ${mssqlPod} -- /opt/mssql-tools/bin/sqlcmd -S . -U sa -P "${password}" -Q "CREATE DATABASE [${database}]"`)

    // create the table that KEDA will monitor for scale decisions
    const createTableSQL = "CREATE TABLE tasks ([id] int identity primary key, [status] varchar(10))"
    sh.exec(`kubectl exec -n ${mssqlns} ${mssqlPod} -- /opt/mssql-tools/bin/sqlcmd -S . -U sa -P "${password}" -d "${database}" -Q "${createTableSQL}"`)

    // deploy the test app
    sh.exec(`kubectl create namespace ${testns}`)
    const testAppYamlFile = tmp.fileSync()
    fs.writeFileSync(testAppYamlFile.name, testAppDeployYaml)
    t.is(0, sh.exec(`kubectl apply -n ${testns} -f ${testAppYamlFile.name}`).code, 'creating the test app deployment should work.')
})

test.serial('Deployment should have 0 replicas on start', t => {
    const replicaCount = sh.exec(getReplicaCountCommand).stdout
    t.is(replicaCount, '0', 'replica count should start out as 0')
})

test.serial(`Deployment should scale to 5 (the max) then back to 0`, t => {
    const jobYamlFile = tmp.fileSync()
    fs.writeFileSync(jobYamlFile.name, insertRecordsJobYaml)
    t.is(0, sh.exec(`kubectl apply -f ${jobYamlFile.name} -n ${testns}`).code, 'creating job should work.')

    const maxReplicaCount = '5'

    let replicaCount = '0'
    for (let i = 0; i < 30 && replicaCount !== maxReplicaCount; i++) {
        replicaCount = sh.exec(getReplicaCountCommand).stdout
        if (replicaCount !== maxReplicaCount) {
            sh.exec('sleep 2s')
        }
    }

    t.is(maxReplicaCount, replicaCount, `Replica count should be ${maxReplicaCount} after 60 seconds`)

    for (let i = 0; i < 36 && replicaCount !== '0'; i++) {
        replicaCount = sh.exec(getReplicaCountCommand).stdout
        if (replicaCount !== '0') {
            sh.exec('sleep 5s')
        }
    }

    t.is('0', replicaCount, 'Replica count should be 0 after 3 minutes')
})

test.after.always.cb('clean up deployment artifacts', t => {
    // delete all the app and job resources
    const resources = [
        'scaledobject.keda.sh/mssql-scaledobject',
        'triggerauthentication.keda.sh/keda-trigger-auth-mssql-secret',
        `deployment.apps/${appName}`,
        'secret/mssql-secrets',
        'job/mssql-producer-job',
    ]

    for (const resource of resources) {
        sh.exec(`kubectl delete ${resource} -n ${testns}`)
    }

    sh.exec(`kubectl delete namespace ${testns}`)

    // uninstall mssql
    sh.exec(`kubectl delete -n ${mssqlns} deploy/mssql`)
    sh.exec(`kubectl delete namespace ${mssqlns}`)

    t.end()
})

const testAppDeployYaml = `apiVersion: v1
kind: Secret
metadata:
  name: mssql-secrets
type: Opaque
stringData:
  mssql-sa-password: ${password}
  mssql-connection-string: ${sqlConnectionString}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: mssql-consumer-worker
  name: ${appName}
spec:
  replicas: 0
  selector:
    matchLabels:
      app: mssql-consumer-worker
  template:
    metadata:
      labels:
        app: mssql-consumer-worker
    spec:
      containers:
      - image: docker.io/cgillum/mssqlscalertest:latest
        imagePullPolicy: Always
        name: mssql-consumer-worker
        args: [consumer]
        env:
          - name: SQL_CONNECTION_STRING
            valueFrom:
              secretKeyRef:
                name: mssql-secrets
                key: mssql-connection-string
---
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: keda-trigger-auth-mssql-secret
spec:
  secretTargetRef:
  - parameter: password
    name: mssql-secrets
    key: mssql-sa-password
---
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: mssql-scaledobject
spec:
  scaleTargetRef:
    name: ${appName}
  pollingInterval: 5
  cooldownPeriod:  10
  minReplicaCount: 0
  maxReplicaCount: 5
  triggers:
  - type: mssql
    metadata:
      host: "${hostname}"
      port: "1433"
      database: "${database}"
      username: sa
      query: "SELECT COUNT(*) FROM tasks WHERE [status]='running' OR [status]='queued'"
      targetValue: "1" # one replica per row
    authenticationRef:
      name: keda-trigger-auth-mssql-secret`

const insertRecordsJobYaml = `apiVersion: batch/v1
kind: Job
metadata:
  labels:
    app: mssql-producer-job
  name: mssql-producer-job
spec:
  template:
    metadata:
      labels:
        app: mssql-producer-job
    spec:
      containers:
      - image: docker.io/cgillum/mssqlscalertest:latest
        imagePullPolicy: Always
        name: mssql-test-producer
        args: ["producer"]
        env:
          - name: SQL_CONNECTION_STRING
            valueFrom:
              secretKeyRef:
                name: mssql-secrets
                key: mssql-connection-string
      restartPolicy: Never
  backoffLimit: 4`

const mssqlDeploymentYaml = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: mssql-deployment
  labels:
    app: mssql
spec:
  replicas: 1
  selector:
     matchLabels:
       app: mssql
  template:
    metadata:
      labels:
        app: mssql
    spec:
      terminationGracePeriodSeconds: 30
      containers:
      - name: mssql
        image: mcr.microsoft.com/mssql/server:2019-latest
        ports:
        - containerPort: 1433
        env:
        - name: MSSQL_PID
          value: "Developer"
        - name: ACCEPT_EULA
          value: "Y"
        - name: SA_PASSWORD
          value: "${password}"
        readinessProbe:
          exec:
            command:
            - /bin/sh
            - -c
            - "/opt/mssql-tools/bin/sqlcmd -S . -U sa -P '${password}' -Q 'SELECT @@Version'"
---
apiVersion: v1
kind: Service
metadata:
  name: ${mssqlName}
spec:
  selector:
    app: mssql
  ports:
    - protocol: TCP
      port: 1433
      targetPort: 1433
  type: ClusterIP`
