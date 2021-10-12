import * as async from 'async'
import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import test from 'ava'

const mongoDBNamespace = 'mongodb'
const testNamespace = 'mongodb-test'
const mongoDBUsername = 'test_user'
const mongoDBPassword = 'test_password'
const mongoDBDatabase = 'test'
const mongodbCollection = "test_collection"
const mongoJobName = "mongodb-job"

test.before(t => {
    // install mongoDB
    sh.exec(`kubectl create namespace ${mongoDBNamespace}`)
    const mongoDBTmpFile = tmp.fileSync()
    fs.writeFileSync(mongoDBTmpFile.name, mongoDBdeployYaml)

    t.is(0, sh.exec(`kubectl apply --namespace ${mongoDBNamespace} -f ${mongoDBTmpFile.name}`).code, 'creating a MongoDB deployment should work.')
    // wait for mongoDB to load
    let mongoDBReadyReplicaCount = '0'
    for (let i = 0; i < 30; i++) {
        mongoDBReadyReplicaCount = sh.exec(`kubectl get deploy/mongodb -n ${mongoDBNamespace} -o jsonpath='{.status.readyReplicas}'`).stdout
        if (mongoDBReadyReplicaCount != '1') {
            sh.exec('sleep 2s')
        }
    }
    t.is('1', mongoDBReadyReplicaCount, 'MongoDB is not in a ready state')

    const createUserJS = `db.createUser({ user:"${mongoDBUsername}",pwd:"${mongoDBPassword}",roles:[{ role:"readWrite", db: "${mongoDBDatabase}"}]})`
    const LoginJS = `db.auth("${mongoDBUsername}","${mongoDBPassword}")`

    const mongoDBPod = sh.exec(`kubectl get po -n ${mongoDBNamespace} -o jsonpath='{.items[0].metadata.name}'`).stdout
    t.not(mongoDBPod, '')
    sh.exec(`kubectl exec -n ${mongoDBNamespace} ${mongoDBPod} -- mongo --eval \'${createUserJS}\'`)
    sh.exec(`kubectl exec -n ${mongoDBNamespace} ${mongoDBPod} -- mongo --eval \'${LoginJS}\'`)

    sh.config.silent = true
    // create test namespace
    sh.exec(`kubectl create namespace ${testNamespace}`)

    // deploy streams consumer app, scaled job etc.
    const tmpFile = tmp.fileSync()
    const ConnectionString = Buffer.from(`mongodb://${mongoDBUsername}:${mongoDBPassword}@mongodb-svc.${mongoDBNamespace}.svc.cluster.local:27017/${mongoDBDatabase}`).toString()
    const base64ConnectionString = Buffer.from(`mongodb://${mongoDBUsername}:${mongoDBPassword}@mongodb-svc.${mongoDBNamespace}.svc.cluster.local:27017/${mongoDBDatabase}`).toString('base64')

    fs.writeFileSync(tmpFile.name, deployYaml.
    replace(/{{MONGODB_CONNECTION_STRING_BASE64}}/g, base64ConnectionString).
    replace(/{{MONGODB_JOB_NAME}}/g, mongoJobName).
    replace(/{{MONGODB_DATABASE}}/g, mongoDBDatabase).
    replace(/{{MONGODB_COLLECTION}}/g, mongodbCollection).
    replace(/{{MONGODB_CONNECTION_STRING}}/g,ConnectionString))

    t.is(
        0,
        sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${testNamespace}`).code,
        'creating a deployment should work..'
    )

})

test.serial('Job should have 0 job on start', t => {
    const jobCount = sh.exec(
        `kubectl get job --namespace ${testNamespace}`
    ).stdout
    t.is(jobCount, '', 'job count should start out as 0')
})

test.serial(`Job should scale to 5 then back to 0`, t => {
    // insert data to mongodb
    const InsertJS = `db.${mongodbCollection}.insert([
    {"region":"eu-1","state":"running","plan":"planA","goods":"apple"},
    {"region":"eu-1","state":"running","plan":"planA","goods":"orange"},
    {"region":"eu-1","state":"running","plan":"planA","goods":"strawberry"},
    {"region":"eu-1","state":"running","plan":"planA","goods":"cherry"},
    {"region":"eu-1","state":"running","plan":"planA","goods":"pineapple"}
    ])`
    const mongoDBPod = sh.exec(`kubectl get po -n ${mongoDBNamespace} -o jsonpath='{.items[0].metadata.name}'`).stdout
    t.not(mongoDBPod, '')

    t.is(
        0,
        sh.exec(`kubectl exec -n ${mongoDBNamespace} ${mongoDBPod} -- mongo --eval \'${InsertJS}\'`).code,
        'insert 5 mongo record'
    )

    let jobCount = '0'
    // maxJobCount = real Job + first line of output
    const maxJobCount = '6'

    for (let i = 0; i < 30 && jobCount !== maxJobCount; i++) {
        jobCount = sh.exec(
            `kubectl get job --namespace ${testNamespace} | wc -l`
        ).stdout.replace(/[\r\n]/g,"")

        if (jobCount !== maxJobCount) {
            sh.exec('sleep 2s')
        }
    }

    t.is(maxJobCount, jobCount, `Job count should be ${maxJobCount} after 60 seconds`)

    for (let i = 0; i < 36 && jobCount !== '0'; i++) {
        jobCount = sh.exec(
            `kubectl get job --namespace ${testNamespace} | wc -l`
        ).stdout.replace(/[\r\n]/g,"")
        if (jobCount !== '0') {
            sh.exec('sleep 5s')
        }
    }

    t.is('0', jobCount, 'Job count should be 0 after 3 minutes')
})

test.after.always.cb('clean up mongodb deployment', t => {
    const resources = [
        `scaledJob.keda.sh/${mongoJobName}`,
        'triggerauthentication.keda.sh/mongodb-trigger',
        `deployment.apps/mongodb`,
        'secret/mongodb-secret',
    ]

    for (const resource of resources) {
        sh.exec(`kubectl delete ${resource} --namespace ${testNamespace}`)
    }
    sh.exec(`kubectl delete namespace ${testNamespace}`)

    // uninstall mongodb
    sh.exec(`kubectl delete --namespace ${mongoDBNamespace} deploy/mongodb`)
    sh.exec(`kubectl delete namespace ${mongoDBNamespace}`)

    t.end()
})

const mongoDBdeployYaml = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mongodb
spec:
  replicas: 1
  selector:
    matchLabels:
      name: mongodb
  template:
    metadata:
      labels:
        name: mongodb
    spec:
      containers:
      - name: mongodb
        image: mongo:4.2.1
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 27017
          name: mongodb
          protocol: TCP
---
kind: Service
apiVersion: v1
metadata:
  name: mongodb-svc
spec:
  type: ClusterIP
  ports:
  - name: mongodb
    port: 27017
    targetPort: 27017
    protocol: TCP
  selector:
    name: mongodb
`

const deployYaml = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: mongodb-trigger
spec:
  secretTargetRef:
    - parameter: connectionString
      name: mongodb-secret
      key: connect
---
apiVersion: v1
kind: Secret
metadata:
  name: mongodb-secret
type: Opaque
data:
  connect: {{MONGODB_CONNECTION_STRING_BASE64}}
---
apiVersion: keda.sh/v1alpha1
kind: ScaledJob
metadata:
  name: {{MONGODB_JOB_NAME}}
spec:
  jobTargetRef:
    template:
      spec:
        containers:
          - name: mongodb-update
            image: 1314520999/mongodb-update:latest
            args:
            - --connectStr={{MONGODB_CONNECTION_STRING}}
            - --dataBase={{MONGODB_DATABASE}}
            - --collection={{MONGODB_COLLECTION}}
            imagePullPolicy: IfNotPresent
        restartPolicy: Never
    backoffLimit: 1
  pollingInterval: 20
  successfulJobsHistoryLimit: 0
  failedJobsHistoryLimit: 10
  triggers:
    - type: mongodb
      metadata:
        dbName: {{MONGODB_DATABASE}}
        collection: {{MONGODB_COLLECTION}}
        query: '{"region":"eu-1","state":"running","plan":"planA"}'
        queryValue: "1"
      authenticationRef:
        name: mongodb-trigger
---
`
