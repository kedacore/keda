import test from 'ava'
import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import { createNamespace } from './helpers'

const cassandraNamespace = 'cassandra-test'
const cassandraKeyspace = 'test_keyspace'
const cassandraTableName =  'test_table'
const cassandraUsername = 'cassandra'
const cassandraPassword = 'cassandra'
const nginxDeploymentName = 'nginx-deployment'

test.before(t => {
    // install cassandra
    console.log("Install cassandra")
    createNamespace(cassandraNamespace)
    const cassandraTmpFile = tmp.fileSync()
    fs.writeFileSync(cassandraTmpFile.name, cassandraDeployYaml)

    t.is(0, sh.exec(`kubectl apply --namespace ${cassandraNamespace} -f ${cassandraTmpFile.name}`).code, 'creating a Cassandra deployment should work.')

    // wait for cassandra to load
    console.log("wait for cassandra to load")
    let cassandraReadyReplicaCount = '0'
     for (let i = 0; i < 30; i++) {
        cassandraReadyReplicaCount = sh.exec(`kubectl get deploy/cassandra -n ${cassandraNamespace} -o jsonpath='{.status.readyReplicas}'`).stdout
        if (cassandraReadyReplicaCount != '1') {
            sh.exec('sleep 10s')
        }
    }
    t.is('1', cassandraReadyReplicaCount, 'Cassandra is not in a ready state')

    console.log("cassandraReadyCheck")
    let cassandraReady
    const cassandraPod = sh.exec(`kubectl get pods --selector=app=cassandra-app -n ${cassandraNamespace} -o jsonpath='{.items[0].metadata.name}'`).stdout
    for (let i = 0; i < 30; i++) {
      cassandraReady = sh.exec(`kubectl exec -n ${cassandraNamespace} ${cassandraPod} -- nodetool status | grep -w -o UN`)
      if (cassandraReady != "UN\n") {
        sh.exec('sleep 10s')
      }
      else {
        break
      }
    }

    // create cassandra-client
    console.log("create cassandra-client")
    const cassandraClientTmpFile = tmp.fileSync()
    fs.writeFileSync(cassandraClientTmpFile.name, cassandraClientDeployYaml)

    t.is(0, sh.exec(`kubectl apply --namespace ${cassandraNamespace} -f ${cassandraClientTmpFile.name}`).code, 'creating a Cassandra client deployment should work.')

    // wait for cassandra-client to load
    console.log("wait for cassandra-client to load")
    let cassandraClientReadyReplicaCount = '0'
    for (let i = 0; i < 30; i++) {
      cassandraClientReadyReplicaCount = sh.exec(`kubectl get deploy/cassandra-client -n ${cassandraNamespace} -o jsonpath='{.status.readyReplicas}'`).stdout
      if (cassandraClientReadyReplicaCount != '1') {
        sh.exec('sleep 10s')
      }
    }
    t.is('1', cassandraClientReadyReplicaCount, 'Cassandra client is not in a ready state')

    console.log("cassandraClientReadyCheck")
    let cassandraClientReady
    const cassandraClientPod = sh.exec(`kubectl get pods --selector=app=cassandra-client -n ${cassandraNamespace} -o jsonpath='{.items[0].metadata.name}'`).stdout
    for (let i = 0; i < 30; i++) {
      cassandraClientReady = sh.exec(`kubectl exec -n ${cassandraNamespace} ${cassandraClientPod} -- nodetool status | grep -w -o UN`)
      if (cassandraClientReady != "UN\n") {
        sh.exec('sleep 10s')
      }
      else {
        break
      }
    }

    // create table
    console.log("create table")
    const createKeyspace = `CREATE KEYSPACE IF NOT EXISTS ${cassandraKeyspace} WITH REPLICATION = {'class' : 'NetworkTopologyStrategy', 'datacenter1' : '1'};`
    const createTableCQL = `CREATE TABLE IF NOT EXISTS ${cassandraKeyspace}.${cassandraTableName} (name text, surname text, age int, PRIMARY KEY (name, surname));`
    t.not(cassandraClientPod, '')
    sh.exec(`kubectl exec ${cassandraClientPod} -n ${cassandraNamespace} -- bash cqlsh -u ${cassandraUsername} -p ${cassandraPassword} cassandra.${cassandraNamespace} --execute="${createKeyspace}"`)
    sh.exec(`kubectl exec ${cassandraClientPod} -n ${cassandraNamespace} -- bash cqlsh -u ${cassandraUsername} -p ${cassandraPassword} cassandra.${cassandraNamespace} --execute="${createTableCQL}"`)

    // deploy nginx, scaledobject etc.
    console.log("deploy nginx, scaledobject etc.")
    const nginxTmpFile = tmp.fileSync()
    fs.writeFileSync(nginxTmpFile.name, nginxDeployYaml)

    t.is(0, sh.exec(`kubectl apply --namespace ${cassandraNamespace} -f ${nginxTmpFile.name}`).code, 'creating nginx deployment should work.')
    // wait for nginx to load
    console.log("wait for nginx to load")
    let nginxReadyReplicaCount = '0'
    for (let i = 0; i < 30; i++) {
      nginxReadyReplicaCount = sh.exec(`kubectl get deploy/${nginxDeploymentName} -n ${cassandraNamespace} -o jsonpath='{.status.readyReplicas}'`).stdout
      if (nginxReadyReplicaCount != '') {
        sh.exec('sleep 10s')
      }
    }
    t.is('', nginxReadyReplicaCount, 'creating an Nginx deployment should work')

})

test.serial('Should start off deployment with 0 replicas', t => {

    const replicaCount = sh.exec(`kubectl get deploy/${nginxDeploymentName} --namespace ${cassandraNamespace} -o jsonpath="{.spec.replicas}"`).stdout
    t.is(replicaCount, '0', 'Replica count should start out as 0')

})

test.serial(`Replicas should scale to 2 (the max) then back to 0`, t => {
    // insert data to cassandra
    console.log("insert data to cassandra")
    const insertData = `BEGIN BATCH
    INSERT INTO ${cassandraKeyspace}.${cassandraTableName} (name, surname, age) VALUES ('Mary', 'Paul', 30);
    INSERT INTO ${cassandraKeyspace}.${cassandraTableName} (name, surname, age) VALUES ('James', 'Miller', 25);
    INSERT INTO ${cassandraKeyspace}.${cassandraTableName} (name, surname, age) VALUES ('Lisa', 'Wilson', 29);
    INSERT INTO ${cassandraKeyspace}.${cassandraTableName} (name, surname, age) VALUES ('Bob', 'Taylor', 33);
    INSERT INTO ${cassandraKeyspace}.${cassandraTableName} (name, surname, age) VALUES ('Carol', 'Moore', 31);
    INSERT INTO ${cassandraKeyspace}.${cassandraTableName} (name, surname, age) VALUES ('Richard', 'Brown', 23);
    APPLY BATCH;`

    const cassandraClientPod = sh.exec(`kubectl get pods --selector=app=cassandra-client -n ${cassandraNamespace} -o jsonpath='{.items[0].metadata.name}'`).stdout
    t.not(cassandraClientPod, '')

    t.is(
        0,
        sh.exec(`kubectl exec ${cassandraClientPod} -n ${cassandraNamespace} -- bash cqlsh -u ${cassandraUsername} -p ${cassandraPassword} cassandra.${cassandraNamespace} --execute="${insertData}"`).code,
        'insert 6 cassandra record'
    )

    let replicaCount = '0'
    const maxReplicaCount = '2'

    for (let i = 0; i < 30 && replicaCount !== maxReplicaCount; i++) {
      replicaCount = sh.exec(
        `kubectl get deploy/${nginxDeploymentName} --namespace ${cassandraNamespace} -o jsonpath="{.spec.replicas}"`).stdout
        if (replicaCount !== maxReplicaCount) {
          sh.exec('sleep 10s')
        }
    }

    t.is(maxReplicaCount, replicaCount, `Replica count should be ${maxReplicaCount} after 300 seconds`)
    sh.exec('sleep 30s')

    // delete all data from cassandra
    console.log("delete all data from cassandra")
    const truncateData = `TRUNCATE ${cassandraKeyspace}.${cassandraTableName};`

    t.is(
        0,
        sh.exec(`kubectl exec ${cassandraClientPod} -n ${cassandraNamespace} -- bash cqlsh -u ${cassandraUsername} -p ${cassandraPassword} cassandra.${cassandraNamespace} --execute="${truncateData}"`).code,
        'delete all rows'
    )

    for (let i = 0; i < 30 && replicaCount !== '0'; i++) {
      replicaCount = sh.exec(
        `kubectl get deploy/${nginxDeploymentName} --namespace ${cassandraNamespace} -o jsonpath="{.spec.replicas}"`).stdout
      if (replicaCount !== '0') {
        sh.exec('sleep 10s')
      }
    }

     t.is('0', replicaCount, 'Replica count should be 0 after 5 minutes')

})

test.after.always((t) => {
     t.is(0, sh.exec(`kubectl delete namespace ${cassandraNamespace}`).code, 'Should delete Cassandra namespace')

})

const cassandraDeployYaml = `
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: cassandra-app
  name: cassandra
spec:
  replicas: 1
  selector:
    matchLabels:
      app: cassandra-app
  template:
    metadata:
      labels:
        app: cassandra-app
    spec:
      containers:
      - image: bitnami/cassandra:4.0.1
        imagePullPolicy: IfNotPresent
        name: cassandra
        ports:
        - containerPort: 9042
---
apiVersion: v1
kind: Service
metadata:
    name: cassandra
spec:
    ports:
      - name: cql
        port: 9042
        protocol: TCP
        targetPort: 9042
    selector:
        app: cassandra-app
`

const cassandraClientDeployYaml = `
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: cassandra-client
  name: cassandra-client
spec:
  replicas: 1
  selector:
    matchLabels:
      app: cassandra-client
  template:
    metadata:
      labels:
        app: cassandra-client
    spec:
      containers:
      - image: bitnami/cassandra:4.0.1
        imagePullPolicy: IfNotPresent
        name: cassandra-client
`

const nginxDeployYaml = `
---
apiVersion: v1
kind: Secret
metadata:
  name: cassandra-secrets
type: Opaque
data:
  cassandra_password: Y2Fzc2FuZHJh
---
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: keda-trigger-auth-cassandra-secret
spec:
  secretTargetRef:
  - parameter: password
    name: cassandra-secrets
    key: cassandra_password
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: nginx
  name: ${nginxDeploymentName}
spec:
  replicas: 0
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - image: nginx
        name: nginx
        ports:
        - containerPort: 80
---
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: cassandra-scaledobject
spec:
  minReplicaCount: 0
  maxReplicaCount: 2
  pollingInterval: 1  # Optional. Default: 30 seconds
  cooldownPeriod: 1 # Optional. Default: 300 seconds
  scaleTargetRef:
    name: ${nginxDeploymentName}
  triggers:
  - type: cassandra
    metadata:
      username: "cassandra"
      clusterIPAddress: "cassandra.${cassandraNamespace}"
      consistency: "Quorum"
      protocolVersion: "4"
      port: "9042"
      keyspace: "${cassandraKeyspace}"
      query: "SELECT COUNT(*) FROM ${cassandraKeyspace}.${cassandraTableName};"
      targetQueryValue: "1"
      metricName: "${cassandraKeyspace}"
    authenticationRef:
      name: keda-trigger-auth-cassandra-secret
`
