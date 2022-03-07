/*
To use this test you will need:
* NewRelic License Key
* NewRelic API Key

You can get a free account on https://www.newrelic.com/

once you have your license and api key you need to setup the following
environment variables

NEWRELIC_API_KEY
NEWRELIC_LICENSE
NEWRELIC_ACCOUNT_ID

the API key starts with 'NRAK' and the license ends in 'NRAL'

 */
import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import test from 'ava'
import { createNamespace } from './helpers'

const newRelicApiKey = process.env['NEWRELIC_API_KEY']
const newRelicAccountId = process.env['NEWRELIC_ACCOUNT_ID']
const testNamespace = 'new-relic-test'
const newRelicNamespace = 'new-relic'
const newRelicRepoUrl = 'https://helm-charts.newrelic.com'
const newRelicHelmRepoName = 'new-relic'
const newRelicHelmPackageName = 'nri-bundle'
const newRelicLicenseKey = process.env['NEWRELIC_LICENSE']
const kuberneteClusterName = 'keda-new-relic'
let newRelicRegion = process.env['NEWRELIC_REGION']

test.before(t => {
  if (!newRelicApiKey) {
    t.fail('NEWRELIC_API_KEY environment variable is required for newrelic tests tests')
  }
  if (!newRelicLicenseKey) {
    t.fail('NEWRELIC_LICENSE environment variable is required for newrelic tests tests')
  }
  if (!newRelicAccountId) {
    t.fail('NEWRELIC_ACCOUNT_ID environment variable is required for newrelic tests tests')
  }
  if (!newRelicRegion) {
    newRelicRegion = 'EU'
  }
  createNamespace(newRelicNamespace)
  sh.exec(`helm repo add ${newRelicHelmRepoName} ${newRelicRepoUrl}`)
  sh.exec(`helm repo update`)
  let helmInstallStatus = sh.exec(`helm upgrade \
        --install --set global.cluster=${kuberneteClusterName} \
        --set prometheus.enabled=true \
        --set ksm.enabled=true \
        --set global.lowDataMode=true \
        --set global.licenseKey=${newRelicLicenseKey} \
        --timeout 600s \
        --set logging.enabled=false \
        --set ksm.enabled=true \
        --set logging.enabled=true \
        --namespace ${newRelicNamespace} \
        nri-keda ${newRelicHelmRepoName}/${newRelicHelmPackageName}`).code
  sh.echo(`${helmInstallStatus}`)
  t.is(0,
    helmInstallStatus,
    'creating a New Relic Bundle Install should work.'
  )

  sh.config.silent = true
  const tmpFile = tmp.fileSync()
  fs.writeFileSync(tmpFile.name, deployYaml
    .replace('{{NEWRELIC_API_KEY}}', Buffer.from(newRelicApiKey).toString('base64'))
    .replace('{{NEWRELIC_ACCOUNT_ID}}', newRelicAccountId)
    .replace('{{NEWRELIC_REGION}}', newRelicRegion)
  )
  createNamespace(testNamespace)
  sh.exec(`cp ${tmpFile.name} /tmp/paso.yaml`)
  t.is(
    0,
    sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${testNamespace}`).code,
    'creating a deployment should work.'
  )
  for (let i = 0; i < 10; i++) {
    const readyReplicaCount = sh.exec(`kubectl get deployment.apps/test-app \
      --namespace ${testNamespace} -o jsonpath="{.status.readyReplicas}"`).stdout
    if (readyReplicaCount != '1') {
      sh.exec('sleep 2s')
    }
  }
})

test.serial('Keda Deployment should have 0 replicas on start', t => {
  const replicaCount = sh.exec(
    `kubectl get deployment.apps/keda-test-app --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
  ).stdout
  t.is(replicaCount, '0', 'replica count should start out as 0')
})

test.serial('Deployment should have 1 replicas on start', t => {
  const replicaCount = sh.exec(
    `kubectl get deployment.apps/test-app --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
  ).stdout
  t.is(replicaCount, '1', 'replica count should start out as 0')
})

test.serial(`Deployment should scale to 3 (the max) with HTTP Requests exceeding in the rate then back to 0`, t => {
  // generate a large number of HTTP requests (using Apache Bench) that will take some time
  // so prometheus has some time to scrape it
  const loadGeneratorFile = tmp.fileSync()
  fs.writeFileSync(loadGeneratorFile.name, generateRequestsYaml.replace('{{NAMESPACE}}', testNamespace))
  t.is(
    0,
    sh.exec(`kubectl apply -f ${loadGeneratorFile.name} --namespace ${testNamespace}`).code,
    'creating job should work.'
  )

  t.is(
    '1',
    sh.exec(
      `kubectl get deployment.apps/test-app --namespace ${testNamespace} -o jsonpath="{.status.readyReplicas}"`
    ).stdout,
    'There should be 1 replica for the test-app deployment'
  )

  // keda based deployment should start scaling up with http requests issued
  let replicaCount = '0'
  for (let i = 0; i < 60 && replicaCount !== '3'; i++) {
    t.log(`Waited ${5 * i} seconds for new-relic-based deployments to scale up`)
    const jobLogs = sh.exec(`kubectl logs -l job-name=generate-requests -n ${testNamespace}`).stdout
    t.log(`Logs from the generate requests: ${jobLogs}`)

    replicaCount = sh.exec(
      `kubectl get deployment.apps/keda-test-app --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
    ).stdout
    if (replicaCount !== '3') {
      sh.exec('sleep 10s')
    }
  }

  t.is('3', replicaCount, 'Replica count should be maxed at 3')

  t.is(
    0,
    sh.exec(`kubectl delete -f ${loadGeneratorFile.name} --namespace ${testNamespace}`).code,
    'deleting job should work.'
  )

  for (let i = 0; i < 60 && replicaCount !== '0'; i++) {
    replicaCount = sh.exec(
      `kubectl get deployment.apps/keda-test-app --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
    ).stdout
    if (replicaCount !== '0') {
      sh.exec('sleep 10s')
    }
  }

  t.is('0', replicaCount, 'Replica count should be 0 after 6 minutes')
  sh.exec('sleep 10s')
})


test.after.always.cb('clean up newrelic resources', t => {
  sh.exec(`helm delete --namespace ${newRelicNamespace} nri-keda`)
  sh.exec(`helm repo rm ${newRelicHelmRepoName}`)
  sh.exec(`kubectl delete namespace ${newRelicNamespace} --force`)
  sh.exec(`kubectl delete namespace ${testNamespace} --force`)
  t.end()
})

const generateRequestsYaml = `apiVersion: batch/v1
kind: Job
metadata:
  name: generate-requests
spec:
  template:
    spec:
      containers:
      - image: jordi/ab
        name: test
        command: ["/bin/sh"]
        args: ["-c", "for i in $(seq 1 60);do echo $i;ab -c 5 -n 10000 -v 2 http://test-app/;sleep 1;done"]
      restartPolicy: Never
  activeDeadlineSeconds: 600
  backoffLimit: 2`

const deployYaml = `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: test-app
  name: test-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test-app
  template:
    metadata:
      labels:
        app: test-app
        type: keda-testing
    spec:
      containers:
      - name: prom-test-app
        image: tbickford/simple-web-app-prometheus:a13ade9
        imagePullPolicy: IfNotPresent
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: keda-test-app
  name: keda-test-app
spec:
  replicas: 0
  selector:
    matchLabels:
      app: keda-test-app
  template:
    metadata:
      labels:
        app: keda-test-app
        type: keda-testing
    spec:
      containers:
      - name: prom-test-app
        image: tbickford/simple-web-app-prometheus:a13ade9
        imagePullPolicy: IfNotPresent
---
apiVersion: v1
kind: Service
metadata:
  labels:
    app: test-app
  annotations:
    prometheus.io/scrape: "true"
  name: test-app
spec:
  ports:
  - name: http
    port: 80
    protocol: TCP
    targetPort: 8080
  selector:
    type: keda-testing
---
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: newrelic-trigger
spec:
  secretTargetRef:
  - parameter: queryKey
    name: newrelic-secret
    key: newRelicApiKey
---
apiVersion: v1
kind: Secret
metadata:
  name: newrelic-secret
type: Opaque
data:
  newRelicApiKey: {{NEWRELIC_API_KEY}}
---
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: new-relic-scaledobject
spec:
  scaleTargetRef:
    name: keda-test-app
  minReplicaCount: 0
  maxReplicaCount: 3
  pollingInterval: 5
  cooldownPeriod:  10
  triggers:
  - type: new-relic
    metadata:
      account: '{{NEWRELIC_ACCOUNT_ID}}'
      region: '{{NEWRELIC_REGION}}'
      threshold: '10'
      nrql: SELECT average(\`http_requests_total\`) FROM Metric where serviceName='test-app' and namespaceName='new-relic-test' since 60 seconds ago
    authenticationRef:
        name: newrelic-trigger
`
