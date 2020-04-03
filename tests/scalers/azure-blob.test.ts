import * as async from 'async'
import * as azure from 'azure-storage'
import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import test from 'ava'

const defaultNamespace = 'azure-blob-test'
const connectionString = process.env['TEST_STORAGE_CONNECTION_STRING']
// const blobSubPath = process.env['BLOB_SUB_PATH'];

test.before(t => {
    if (!connectionString) {
        t.fail('TEST_STORAGE_CONNECTION_STRING environment variable is required for blob tests')
    }
    // if (!blobSubPath) {
    //   t.fail('BLOB_SUB_PATH environment variable is required for blob tests');
  // }


    sh.config.silent = true
    const base64ConStr = Buffer.from(connectionString).toString('base64')
    const tmpFile = tmp.fileSync()
    fs.writeFileSync(tmpFile.name, deployYaml.replace('{{CONNECTION_STRING_BASE64}}', base64ConStr))
    sh.exec(`kubectl create namespace ${defaultNamespace}`)
    t.is(0, sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${defaultNamespace}`).code, 'creating a deployment should work.')
})

test.serial('Deployment should have 0 replicas on start', t => {
    const replicaCount = sh.exec(`kubectl get deployment.apps/test-deployment --namespace ${defaultNamespace} -o jsonpath="{.spec.replicas}"`).stdout
    t.is(replicaCount, '0', 'replica count should start out as 0')
})

test.serial.cb('Deployment should scale to 2 with 2000 blobs on the blob container then back to 0', t => {
    // add 2000 files
    const blobSvc = azure.createBlobService(connectionString)
    blobSvc.createContainerIfNotExists('container-name', err => {
        t.falsy(err, 'unable to create blob')
        async.mapLimit(Array(2000).keys(), 500, (n, cb) => blobSvc.createBlockBlobFromText('container-name',`blobsubpath/blob-name-${n}`,'test text', cb), () => {
            let replicaCount = '0'
            for (let i = 0; i < 40 && replicaCount !== '2'; i++) {
                replicaCount = sh.exec(`kubectl get deployment.apps/test-deployment --namespace ${defaultNamespace} -o jsonpath="{.spec.replicas}"`).stdout
                if (replicaCount !== '2') {
                    sh.exec('sleep 1s')
                }
            }

            t.is('2', replicaCount, 'Replica count should be 2 after 40 seconds')

            for (let i = 0; i < 50 && replicaCount !== '0'; i++) {
                replicaCount = sh.exec(`kubectl get deployment.apps/test-deployment --namespace ${defaultNamespace} -o jsonpath="{.spec.replicas}"`).stdout
                if (replicaCount !== '0') {
                    sh.exec('sleep 5s')
                }
            }

            t.is('0', replicaCount, 'Replica count should be 0 after 3 minutes')
            t.end()
        })
    })
})

test.after.always('clean up azure-blob deployment', t => {
    const resources = [
        'secret/test-secrets',
        'deployment.apps/test-deployment',
        'scaledobject.keda.sh/test-scaledobject'
    ]

    for (const resource of resources) {
        sh.exec(`kubectl delete ${resource} --namespace ${defaultNamespace}`)
    }
    sh.exec(`kubectl delete namespace ${defaultNamespace}`)
})

const deployYaml = `apiVersion: v1
kind: Secret
metadata:
  name: test-secrets
  labels:
data:
  AzureWebJobsStorage: {{CONNECTION_STRING_BASE64}}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
  labels:
    app: test-deployment
spec:
  replicas: 0
  selector:
    matchLabels:
      app: test-deployment
  template:
    metadata:
      name:
      namespace:
      labels:
        app: test-deployment
    spec:
      containers:
      - name: test-deployment
        image: slurplk/blob-consumer:latest
        resources:
        ports:
        env:
        - name: FUNCTIONS_WORKER_RUNTIME
          value: dotnet
        - name: AzureFunctionsWebHost__hostid
          value: blobtestsampleapp
        - name: BLOB_SUB_PATH
          value: blobsubpath/
        - name: AzureWebJobsStorage
          valueFrom:
            secretKeyRef:
              name: test-secrets
              key: AzureWebJobsStorage
        - name: TEST_STORAGE_CONNECTION_STRING
          valueFrom:
            secretKeyRef:
              name: test-secrets
              key: AzureWebJobsStorage
      nodeSelector:
        beta.kubernetes.io/os: linux
---
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: test-scaledobject
spec:
  scaleTargetRef:
    name: test-deployment
  pollingInterval: 20
  maxReplicaCount: 2
  cooldownPeriod: 10
  triggers:
  - type: azure-blob
    metadata:
      blobContainerName: container-name
      blobPrefix: blobsubpath
      connection: AzureWebJobsStorage`