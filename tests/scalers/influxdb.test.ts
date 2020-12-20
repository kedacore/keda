import test from 'ava'
import * as eol from 'eol'
import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'

const influxdbJobName = 'influx-client-job'
const influxdbNamespaceName = 'influxdb'
const influxdbPodName = 'influxdb-0'

const authTokenPrefixConstant = 20
const nginxDeploymentName = 'nginx-deployment'
const orgNamePrefixConstant = 18

function runWriteJob(t) {
    const influxdbJobTmpFile = tmp.fileSync()
    fs.writeFileSync(influxdbJobTmpFile.name, influxdbWriteJobYaml)

    t.is(0, sh.exec(`kubectl apply --namespace ${influxdbNamespaceName} -f ${influxdbJobTmpFile.name}`).code)

    let influxdbJobStatus = '0'
    for (let i = 0; i < 15; i++) {
        influxdbJobStatus = sh.exec(`kubectl get job --namespace ${influxdbNamespaceName} ${influxdbJobName} -o jsonpath='{.status.succeeded}'`).stdout
        console.log(influxdbJobStatus)
        if (influxdbJobStatus !== '1') {
            sh.exec('sleep 2s')
        } else {
            break
        }
    }

    t.is('1', influxdbJobStatus, 'Job did not complete')
    // get stdout from logs in running job
    const podName = sh.exec(`kubectl get pods --namespace influxdb --template '{{range .items}}{{.metadata.name}}{{"\\n"}}{{end}}' | grep ${influxdbJobName} | head -1`).stdout
    console.log('This is the pod', podName)

    const infoOutput = sh.exec(`kubectl logs --namespace ${influxdbNamespaceName} ${podName}`).stdout


    const splitInfo = eol.split(infoOutput)

    const authToken = splitInfo[0].substring(authTokenPrefixConstant)
    const orgName = splitInfo[1].substring(orgNamePrefixConstant)

    return {
        authToken,
        orgName,
    }
}

test.before((t) => {
    const influxdbDeployTmpFile = tmp.fileSync()
    fs.writeFileSync(influxdbDeployTmpFile.name, influxdbDeployYaml)

    // Deploy influxdb instance
    t.is(0, sh.exec(`kubectl apply --namespace ${influxdbNamespaceName} -f ${influxdbDeployTmpFile.name}`).code)

    // Wait for influxdb instance to be ready
    let influxdbStatus = 'false'
    for (let i = 0; i < 25; i++) {
        influxdbStatus = sh.exec(`kubectl get pod ${influxdbPodName} --namespace ${influxdbNamespaceName} -o jsonpath='{.status.containerStatuses[0].started}'`).stdout
        if (influxdbStatus !== 'true') {
            sh.exec('sleep 2s')
        } else {
            break
        }
    }

    t.is('true', influxdbStatus, 'Influxdb is not in a ready state')
})

test.serial('Should start off deployment with 0 replicas and scale to 2 replicas when scaled object is applied', (t) => {
    const { authToken, orgName } = runWriteJob(t)
    const basicDeploymentTmpFile = tmp.fileSync()
    fs.writeFileSync(basicDeploymentTmpFile.name, basicDeploymentYaml)

    t.is(0, sh.exec(`kubectl apply --namespace ${influxdbNamespaceName} -f ${basicDeploymentTmpFile.name}`).code)

    const numReplicasBefore = sh.exec(`kubectl get deployment --namespace ${influxdbNamespaceName} ${nginxDeploymentName} -o jsonpath='{.spec.replicas}'`).stdout
    t.is(numReplicasBefore, '0', 'Number of replicas should be 0 to start with')

    const scaledObjectTmpFile = tmp.fileSync()
    fs.writeFileSync(scaledObjectTmpFile.name, scaledObjectYaml.replace('{{INFLUXDB_AUTH_TOKEN}}', authToken).replace('{{INFLUXDB_ORG_NAME}}', orgName))

    t.is(0, sh.exec(`kubectl apply --namespace ${influxdbNamespaceName} -f ${scaledObjectTmpFile.name}`).code)

    // polling/waiting for deployment to scale to desired amount of replicas
    let numReplicasAfter = '1'
    for (let i = 0; i < 15; i++){
        numReplicasAfter = sh.exec(`kubectl get deployment --namespace ${influxdbNamespaceName} ${nginxDeploymentName} -o jsonpath='{.spec.replicas}'`).stdout
        if (numReplicasAfter !== '2') {
            sh.exec('sleep 2s')
        } else {
            break
        }
    }

    t.is(numReplicasAfter, '2', 'Number of replicas should have scaled to 2')
})

test.after.always((t) => {
    t.is(0, sh.exec(`kubectl delete namespace ${influxdbNamespaceName}`).code, 'Should delete influxdb namespace')
})

const influxdbDeployYaml = `
---
apiVersion: v1
kind: Namespace
metadata:
    name: influxdb
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
    labels:
        app: influxdb
    name: influxdb
    namespace: influxdb
spec:
    replicas: 1
    selector:
        matchLabels:
            app: influxdb
    serviceName: influxdb
    template:
        metadata:
            labels:
                app: influxdb
        spec:
            containers:
              - image: quay.io/influxdb/influxdb:v2.0.1
                name: influxdb
                ports:
                  - containerPort: 8086
                    name: influxdb
                volumeMounts:
                  - mountPath: /root/.influxdbv2
                    name: data
    volumeClaimTemplates:
      - metadata:
            name: data
            namespace: influxdb
        spec:
            accessModes:
              - ReadWriteOnce
            resources:
                requests:
                    storage: 10G
---
apiVersion: v1
kind: Service
metadata:
    name: influxdb
    namespace: influxdb
spec:
    ports:
      - name: influxdb
        port: 8086
        targetPort: 8086
    selector:
        app: influxdb
    type: ClusterIP
`

const scaledObjectYaml = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: influxdb-scaler
  namespace: influxdb
spec:
  scaleTargetRef:
    name: nginx-deployment
  maxReplicaCount: 2
  triggers:
  - type: influxdb
    metadata:
      authToken: {{INFLUXDB_AUTH_TOKEN}}
      organizationName: {{INFLUXDB_ORG_NAME}}
      serverURL: http://influxdb.influxdb.svc:8086
      thresholdValue: "3"
      query: |
        from(bucket:"bucket")
        |> range(start: -1h)
        |> filter(fn: (r) => r._measurement == "stat")
`

const influxdbWriteJobYaml = `
apiVersion: batch/v1
kind: Job
metadata:
  name: influx-client-job
  namespace: influxdb
spec:
  template:
    spec:
      containers:
      - name: influx-client-job
        image: docker.io/yquansah/influxdb:2-client
        env:
        - name: INFLUXDB_SERVER_URL
          value: http://influxdb:8086
      restartPolicy: OnFailure
`

const basicDeploymentYaml = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx-deployment
spec:
  replicas: 0
  selector:
    matchLabels:
      app: nginx-deployment
  template:
    metadata:
      labels:
        app: nginx-deployment
    spec:
      containers:
      - name: nginx-deployment
        image: nginx:1.14.2
        ports:
        - containerPort: 80
`
