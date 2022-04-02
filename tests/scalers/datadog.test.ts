/*
To use this test you will need:
* Datadog API Key
* Datadog App Key
* Datadog site

You can get a free account on https://www.datadoghq.com/free-datadog-trial/

once you have your Datadog account set up, you need to setup the following
environment variables

DATADOG_API_KEY
DATADOG_APP_KEY
DATADOG_SITE (optional, default datadoghq.com)

*/
import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import test from 'ava'
import { createNamespace } from './helpers'

const datadogApiKey = process.env['DATADOG_API_KEY']
const datadogAppKey = process.env['DATADOG_APP_KEY']
let datadogSite = process.env['DATADOG_SITE']
const testNamespace = 'datadog-test'
const datadogNamespace = 'datadog'
const datadogHelmRepo = 'https://helm.datadoghq.com'
const datadogHelmRelease = 'datadogkeda'
const kuberneteClusterName = 'keda-datadog-cluster'

test.before(t => {
  if (!datadogApiKey) {
    t.fail('DATADOG_API_KEY environment variable is required for Datadog tests')
  }
  if (!datadogAppKey) {
    t.fail('DATADOG_APP_KEY environment variable is required for Datadog tests')
  }
  if (!datadogSite) {
    datadogSite = 'datadoghq.com'
  }

  sh.config.silent = true

  sh.exec(`kubectl delete namespace ${datadogNamespace} --force`)
  sh.exec(`kubectl delete namespace ${testNamespace} --force`)

  sh.exec(`helm repo add datadog ${datadogHelmRepo}`)
  sh.exec(`helm repo update`)

  sh.config.silent = false
  createNamespace(datadogNamespace)
  let helmInstallStatus = sh.exec(`helm upgrade \
  		--install \
  		--set datadog.apiKey=${datadogApiKey} \
  		--set datadog.appKey=${datadogAppKey} \
		  --set datadog.site=${datadogSite} \
		  --set datadog.clusterName=${kuberneteClusterName} \
		  --set datadog.kubelet.tlsVerify=false \
		  --namespace ${datadogNamespace} \
        ${datadogHelmRelease} datadog/datadog`).code
  t.is(0,
    helmInstallStatus,
    'deploying the Datadog Helm chart should work.'
  )

  sh.config.silent = true

  // Let's wait until the Datadog agent is ready
  let datadogDesired = sh.exec(`kubectl get daemonset ${datadogHelmRelease} \
    --namespace ${datadogNamespace} -o jsonpath="{.status.desiredNumberScheduled}"`).stdout

  let datadogReady = sh.exec(`kubectl get daemonset ${datadogHelmRelease} \
    --namespace ${datadogNamespace} -o jsonpath="{.status.numberReady}"`).stdout

  while (datadogReady != datadogDesired) {
    sh.exec('sleep 2')
    datadogDesired = sh.exec(`kubectl get daemonset ${datadogHelmRelease} \
      --namespace ${datadogNamespace} -o jsonpath="{.status.desiredNumberScheduled}"`).stdout

    datadogReady = sh.exec(`kubectl get daemonset ${datadogHelmRelease} \
      --namespace ${datadogNamespace} -o jsonpath="{.status.numberReady}"`).stdout
  }

  createNamespace(testNamespace)

  var create_secret =  sh.exec(`kubectl create secret generic datadog-secrets --from-literal=apiKey=${datadogApiKey} \
  --from-literal=appKey=${datadogAppKey} --from-literal=datadogSite=${datadogSite} --namespace ${testNamespace}`)

  t.is(
    0,
   create_secret.code,
    'creating a generic secret should work: '.concat(create_secret.stderr)
  )

  const tmpFile = tmp.fileSync()
  fs.writeFileSync(tmpFile.name, deployYaml)

  var create_deploy = sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${testNamespace}`)
  t.is(
    0,
    create_deploy.code,
    'creating a deployment should work: '.concat(create_deploy.stderr)
  )

  // Sleeping 1m to make sure we are already sending nginx metrics to Datadog
  sh.exec('sleep 1m')
})

test.serial('Deployment should have 1 replicas on start', t => {
  const replicaCount = sh.exec(
    `kubectl get deployment nginx --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
  ).stdout
  t.is(replicaCount, '1', 'replica count should start out as 1')
})


test.serial(`NGINX deployment should scale to 3 (the max) when getting too many HTTP requests then back to 1`, t => {
  // generate fake traffic to the NGINX pod to for scaling up
  const tmpFile = tmp.fileSync()
  fs.writeFileSync(tmpFile.name, generateRequestsYaml)

  var create_traffic = sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${testNamespace}`)
  t.is(
    0,
    create_traffic.code,
    'creating fake-traffic should work: '.concat(create_traffic.stderr)
  )

  // keda based deployment should start scaling up with http requests issued
  let replicaCount = '1'
  for (let i = 0; i < 60 && replicaCount !== '3'; i++) {
    t.log(`Waited ${15 * i} seconds for nginx deployment to scale up`)

    replicaCount = sh.exec(
      `kubectl get deployment nginx --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
    ).stdout
    if (replicaCount !== '3') {
      sh.exec('sleep 15')
    }
  }

  t.is('3', replicaCount, 'Replica count should be maxed at 3')

  // Delete fake-traffic to force scaling down
  var delete_traffic = sh.exec(`kubectl delete -f ${tmpFile.name} --namespace ${testNamespace}`)
  t.is(
    0,
    delete_traffic.code,
    'deleting fake-traffic should work: '.concat(delete_traffic.stderr)
  )

  for (let i = 0; i < 50 && replicaCount !== '1'; i++) {
    t.log(`Waited ${15 * i} seconds for nginx deployment to scale down`)
    replicaCount = sh.exec(
      `kubectl get deployment nginx --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
    ).stdout
    if (replicaCount !== '1') {
      sh.exec('sleep 15')
    }
  }

  t.is('1', replicaCount, 'Replica count should be back to 1 after removing the fake traffic')
  sh.exec('sleep 10')
})

test.after.always.cb('clean up datadog resources', t => {
  sh.exec(`kubectl delete scaledobject -n ${testNamespace} --all`)
  sh.exec(`helm repo rm datadog`)
  sh.exec(`kubectl delete namespace ${datadogNamespace} --force`)
  sh.exec(`kubectl delete namespace ${testNamespace} --force`)
  t.end()
})

const generateRequestsYaml = `apiVersion: v1
kind: Pod
metadata:
  name: fake-traffic
spec:
  containers:
  - image: busybox
    name: test
    command: ["/bin/sh"]
    args: ["-c", "while true; do wget -O /dev/null -o /dev/null http://nginx/; sleep 0.1; done"]`

const deployYaml = `apiVersion: v1
kind: ConfigMap
metadata:
  name: nginx-conf
data:
  status.conf: |
    server {
      listen 81;

      location /nginx_status {
        stub_status on;
      }
    }
---
apiVersion: apps/v1
kind: Deployment
metadata:
  creationTimestamp: null
  labels:
    app: nginx
  name: nginx
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      creationTimestamp: null
      labels:
        app: nginx
      annotations:
        ad.datadoghq.com/nginx.check_names: '["nginx"]'
        ad.datadoghq.com/nginx.init_configs: '[{}]'
        ad.datadoghq.com/nginx.instances: |
          [
            {
              "nginx_status_url":"http://%%host%%:81/nginx_status/"
            }
          ]
    spec:
      containers:
      - image: nginx
        name: nginx
        ports:
        - containerPort: 80
        - containerPort: 81
        volumeMounts:
        - mountPath: /etc/nginx/conf.d/status.conf
          subPath: status.conf
          readOnly: true
          name: "config"
      volumes:
      - name: "config"
        configMap:
          name: "nginx-conf"
---
apiVersion: v1
kind: Service
metadata:
  labels:
    app: nginx
  name: nginx
spec:
  ports:
  - port: 80
    protocol: TCP
    targetPort: 80
    name: default
  - port: 81
    protocol: TCP
    targetPort: 81
    name: status
  selector:
    app: nginx
---
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: keda-trigger-auth-datadog-secret
spec:
  secretTargetRef:
  - parameter: apiKey
    name: datadog-secrets
    key: apiKey
  - parameter: appKey
    name: datadog-secrets
    key: appKey
  - parameter: datadogSite
    name: datadog-secrets
    key: datadogSite
---
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: datadog-scaledobject
spec:
  scaleTargetRef:
    name: nginx
  minReplicaCount: 1
  maxReplicaCount: 3
  cooldownPeriod: 10
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 10
  triggers:
  - type: datadog
    metadata:
      query: "avg:nginx.net.request_per_s{cluster_name:keda-datadog-cluster}"
      queryValue: "2"
      type: "global"
      age: "120"
    authenticationRef:
      name: keda-trigger-auth-datadog-secret
`
