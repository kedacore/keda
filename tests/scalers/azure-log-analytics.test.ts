import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import test from 'ava'

const defaultNamespace = 'azure-log-analytics-test'
const la_workspace_id = process.env['TEST_LOG_ANALYTICS_WORKSPACE_ID']
const sp_id = process.env['AZURE_SP_ID']
const sp_key = process.env['AZURE_SP_KEY']
const sp_tenant = process.env['AZURE_SP_TENANT']

test.before(t => {
  if (!la_workspace_id || !sp_id || !sp_key || !sp_tenant) {
    t.fail('Connection parameter for LA scaler was not resolved')
  }

  sh.config.silent = true

  sh.exec(`kubectl create namespace ${defaultNamespace}`)
  t.is(
    0,
    sh.exec(`kubectl apply -f ${createYamlFile(deployYaml)}`).code,
    'creating a deployment should work.'
  )
})

test.serial('Deployment should have 0 replicas on start', t => {
  const replicaCount = sh.exec(
    `kubectl get statefulset.apps/test-deployment --namespace ${defaultNamespace} -o jsonpath="{.spec.replicas}"`
  ).stdout
  t.is(replicaCount, '0', 'replica count should start out as 0')
})

test.serial.cb('Deployment should scale to 2 replicas', t => {
  t.is(
    0,
    sh.exec(`kubectl apply -f ${createYamlFile(secretYaml)}`).code,
    'creating a scaled object should work.'
  )
  t.is(
    0,
    sh.exec(`kubectl apply -f ${createYamlFile(triggerAuthYaml)}`).code,
    'creating a scaled object should work.'
  )
  t.is(
    0,
    sh.exec(`kubectl apply -f ${createYamlFile(scaledObjectYaml)}`).code,
    'creating a scaled object should work.'
  )

  //Checking replicas
  let replicaCount = '0'
  for (let i = 0; i < 180 && replicaCount !== '2'; i++) {
    replicaCount = sh.exec(
      `kubectl get statefulset.apps/test-deployment --namespace ${defaultNamespace} -o jsonpath="{.spec.replicas}"`
    ).stdout
    if (replicaCount !== '2') {
      sh.exec('sleep 1s')
    }
  }

  t.is('2', replicaCount, 'Replica count should be 2 after some time')

  t.end()
})

test.serial.cb('Deployment should scale to 0 replicas', t => {
  //Let's change a query to scale down
  t.is(
    0,
    sh.exec(`kubectl patch scaledobject test-scaledobject -n ${defaultNamespace} --type='json' -p='[{"op": "replace", "path": "/spec/triggers/0/metadata/query", "value":"let x = 0; let y = 1; print MetricValue = x, Threshold = y;"}]'`).code,
    'changing a scaled object should work.'
  )

  //Checking replicas
  let replicaCount = '0'
  for (let i = 0; i < 180 && replicaCount !== '0'; i++) {
    replicaCount = sh.exec(
      `kubectl get statefulset.apps/test-deployment --namespace ${defaultNamespace} -o jsonpath="{.spec.replicas}"`
    ).stdout
    if (replicaCount !== '0') {
      sh.exec('sleep 1s')
    }
  }

  t.is('0', replicaCount, 'Replica count should be 0 after some time')

  t.end()
})

test.after.always.cb('clean up deployment', t => {
  const resources = [
    'scaledobject.keda.sh/test-scaledobject',
    'statefulset.apps/test-deployment',
  ]

  for (const resource of resources) {
    sh.exec(`kubectl delete ${resource} --namespace ${defaultNamespace}`)
  }
  sh.exec(`kubectl delete namespace ${defaultNamespace}`)

  t.end()
})

function createYamlFile(yaml: string) {
  const tmpFile = tmp.fileSync()
  fs.writeFileSync(tmpFile.name, yaml
    .replace('{{NAMESPACE}}', defaultNamespace)
    .replace('{{TENANT_ID}}', sp_tenant)
    .replace('{{CLIENT_ID}}', Buffer.from(sp_id).toString('base64'))
    .replace('{{CLIENT_SECRET}}', Buffer.from(sp_key).toString('base64'))
    .replace('{{WORKSPACE_ID}}', la_workspace_id))

  return tmpFile.name
}

const deployYaml = `apiVersion: v1
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: test-deployment
  namespace: {{NAMESPACE}}
spec:
  serviceName: "test-deployment"
  replicas: 0
  selector:
    matchLabels:
      app: test-deployment
  template:
    metadata:
      labels:
        app: test-deployment
    spec:
      containers:
      - name: nginx
        image: nginx:1.16.1
        ports:
        - containerPort: 80
`
const secretYaml = `apiVersion: v1
kind: Secret
metadata:
  name: test-scaledobject-secret
  namespace: {{NAMESPACE}}
type: Opaque
data:
  la-clientId: {{CLIENT_ID}}
  la-clientSecret: {{CLIENT_SECRET}}`

const triggerAuthYaml = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: test-scaledobject-trigger-auth
  namespace: {{NAMESPACE}}
spec:
  secretTargetRef:
    - parameter: clientId
      name: test-scaledobject-secret
      key: la-clientId
    - parameter: clientSecret
      name: test-scaledobject-secret
      key: la-clientSecret`

const scaledObjectYaml = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: test-scaledobject
  namespace: {{NAMESPACE}}
spec:
  scaleTargetRef:
    kind: StatefulSet
    name: test-deployment
  pollingInterval: 5
  cooldownPeriod: 5
  maxReplicaCount: 2
  triggers:
    - type: azure-log-analytics
      metadata:
        tenantId: "{{TENANT_ID}}"
        workspaceId: "{{WORKSPACE_ID}}"
        query: "let x = 10; let y = 1; print MetricValue = x, Threshold = y;"
        threshold: "1"
      authenticationRef:
        name: test-scaledobject-trigger-auth
`