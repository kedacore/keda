import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import test from 'ava'
import { createNamespace } from './helpers'

const testNamespace = 'postgresql-hashicorp-vault'
const postgreSQLUsername = 'test-user'
const postgreSQLPassword = 'test-password'
const postgreSQLDatabase = 'test_db'
const deploymentName = 'worker'

test.before(t => {
  createNamespace(testNamespace)

  // install postgresql
  const postgreSQLTmpFile = tmp.fileSync()
  fs.writeFileSync(postgreSQLTmpFile.name, postgresqlDeploymentYaml.replace('{{POSTGRES_USER}}', postgreSQLUsername)
      .replace('{{POSTGRES_PASSWORD}}', postgreSQLPassword)
      .replace('{{POSTGRES_DB}}', postgreSQLDatabase)
      .replace('{{POSTGRES_DB}}', postgreSQLDatabase))

  t.is(0, sh.exec(`kubectl apply --namespace ${testNamespace} -f ${postgreSQLTmpFile.name}`).code, 'creating a POSTGRES deployment should work.')
  // wait for postgresql to load
  let postgresqlReadyReplicaCount = '0'
  for (let i = 0; i < 30; i++) {
    postgresqlReadyReplicaCount = sh.exec(`kubectl get deploy/postgresql -n ${testNamespace} -o jsonpath='{.status.readyReplicas}'`).stdout
      if (postgresqlReadyReplicaCount != '1') {
          sh.exec('sleep 2s')
      }
  }
  t.is('1', postgresqlReadyReplicaCount, 'Postgresql is not in a ready state')

  // create table that used by the job and the worker
  const postgresqlPod = sh.exec(`kubectl get po -n ${testNamespace} -o jsonpath='{.items[0].metadata.name}'`).stdout
  t.not(postgresqlPod, '')
  const createTableSQL = `CREATE TABLE task_instance (id serial PRIMARY KEY,state VARCHAR(10));`
  sh.exec( `kubectl exec -n ${testNamespace} ${postgresqlPod} -- psql -U ${postgreSQLUsername} -d ${postgreSQLDatabase} -c "${createTableSQL}"`)

  // deploy hashicorp vault
  sh.exec(`helm repo add hashicorp https://helm.releases.hashicorp.com`)
  sh.exec(`helm repo update`)
  let helmInstallStatus = sh.exec(`helm upgrade \
  	--install \
    --set "server.dev.enabled=true" \
	  --namespace ${testNamespace} \
    --wait \
    vault hashicorp/vault`).code
  t.is(0,
    helmInstallStatus,
    'deploying the Datadog Helm chart should work.'
  )

  // create a token and register the connection string
  const connectionString = `postgresql://${postgreSQLUsername}:${postgreSQLPassword}@postgresql.${testNamespace}.svc.cluster.local:5432/${postgreSQLDatabase}?sslmode=disable`
  let createSecret = sh.exec(`kubectl exec vault-0 --namespace ${testNamespace} -- vault kv put secret/keda connectionString=${connectionString}`).code
  t.is(0, createSecret,'create secret in vault should work')
  let response = JSON.parse(sh.exec(`kubectl exec vault-0 --namespace ${testNamespace} -- vault token create -format json`).stdout);

  sh.config.silent = true
  // deploy streams consumer app, scaled object etc.
  const tmpFile = tmp.fileSync()
  const base64ConnectionString = Buffer.from(connectionString).toString('base64')
  fs.writeFileSync(tmpFile.name, deployYaml.replace('{{HASHICORP_VAULT_TOKEN}}', response.auth.client_token)
            .replace('{{POSTGRES_CONNECTION_STRING}}', base64ConnectionString)
            .replace('{{DEPLOYMENT_NAME}}', deploymentName)
            .replace('{{NAMESPACE}}', testNamespace))
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

test.after.always.cb('clean up postgresql deployment', t => {
  const resources = [
      'scaledobject.keda.sh/postgresql-scaledobject',
      'triggerauthentication.keda.sh/keda-trigger-hashicorp-vault-secret',
      `deployment.apps/${deploymentName}`,
      'secret/postgresql-secrets',
      'job/postgresql-insert-job',
  ]

  for (const resource of resources) {
      sh.exec(`kubectl delete ${resource} --namespace ${testNamespace}`)
  }

  // uninstall vault
  sh.exec(`helm delete --namespace ${testNamespace} vault hashicorp/vault`)

  // uninstall postgresql
  sh.exec(`kubectl delete --namespace ${testNamespace} deploy/postgresql`)
  sh.exec(`kubectl delete namespace ${testNamespace}`)

  t.end()
})

const deployYaml = `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: postgresql-update-worker
  name: {{DEPLOYMENT_NAME}}
spec:
  replicas: 0
  selector:
    matchLabels:
      app: postgresql-update-worker
  template:
    metadata:
      labels:
        app: postgresql-update-worker
    spec:
      containers:
      - image: ghcr.io/kedacore/tests-postgresql
        imagePullPolicy: Always
        name: postgresql-processor-test
        command:
          - /app
          - update
        env:
          - name: TASK_INSTANCES_COUNT
            value: "10000"
          - name: CONNECTION_STRING
            valueFrom:
              secretKeyRef:
                name: postgresql-secrets
                key: postgresql_conn_str
---
apiVersion: v1
kind: Secret
metadata:
  name: postgresql-secrets
type: Opaque
data:
  postgresql_conn_str: {{POSTGRES_CONNECTION_STRING}}
---
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: keda-trigger-hashicorp-vault-secret
spec:
  hashiCorpVault:
    address: http://vault.{{NAMESPACE}}:8200
    authentication: token
    credential:
      token: {{HASHICORP_VAULT_TOKEN}}
    secrets:
    - parameter: connection
      key: connectionString
      path: secret/data/keda
---
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: postgresql-scaledobject
spec:
  scaleTargetRef:
    name: worker
  pollingInterval: 5
  cooldownPeriod:  10
  minReplicaCount: 0
  maxReplicaCount: 5
  triggers:
  - type: postgresql
    metadata:
      targetQueryValue: "4"
      query: "SELECT CEIL(COUNT(*) / 5) FROM task_instance WHERE state='running' OR state='queued'"
    authenticationRef:
      name: keda-trigger-hashicorp-vault-secret`

const insertRecordsJobYaml = `apiVersion: batch/v1
kind: Job
metadata:
  labels:
    app: postgresql-insert-job
  name: postgresql-insert-job
spec:
  template:
    metadata:
      labels:
        app: postgresql-insert-job
    spec:
      containers:
      - image: ghcr.io/kedacore/tests-postgresql
        imagePullPolicy: Always
        name: postgresql-processor-test
        command:
          - /app
          - insert
        env:
          - name: TASK_INSTANCES_COUNT
            value: "10000"
          - name: CONNECTION_STRING
            valueFrom:
              secretKeyRef:
                name: postgresql-secrets
                key: postgresql_conn_str
      restartPolicy: Never
  backoffLimit: 4`


const postgresqlDeploymentYaml = `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: postgresql
  name: postgresql
spec:
  replicas: 1
  selector:
    matchLabels:
      app: postgresql
  template:
    metadata:
      labels:
        app: postgresql
    spec:
      containers:
      - image: postgres:10.5
        name: postgresql
        env:
          - name: POSTGRES_USER
            value: {{POSTGRES_USER}}
          - name: POSTGRES_PASSWORD
            value: {{POSTGRES_PASSWORD}}
          - name: POSTGRES_DB
            value: {{POSTGRES_DB}}
        ports:
          - name: postgresql
            protocol: TCP
            containerPort: 5432
---
apiVersion: v1
kind: Service
metadata:
  labels:
    app: postgresql
  name: postgresql
spec:
  ports:
  - port: 5432
    protocol: TCP
    targetPort: 5432
  selector:
    app: postgresql
  type: ClusterIP`
