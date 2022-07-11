// The tests in this file require an instance of Azure Application Inights as well as a service
// principal with "Monitoring Reader" permissions to the instance of Application Insights.
//
import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import test from 'ava'
import * as appinsights from 'applicationinsights'
import { createNamespace } from './helpers'

const namespacePrefix = 'azure-ai-test-'
const app_insights_app_id = process.env['AZURE_APP_INSIGHTS_APP_ID']
const app_insights_instrumentation_key = process.env['AZURE_APP_INSIGHTS_INSTRUMENTATION_KEY']
const sp_id = process.env['AZURE_SP_APP_ID']
const app_insights_connection_string = process.env['AZURE_APP_INSIGHTS_CONNECTION_STRING']
const sp_key = process.env['AZURE_SP_KEY']
const sp_tenant = process.env['AZURE_SP_TENANT']
const test_pod_id = process.env['TEST_POD_ID'] == "true"

const test_app_insights_metric = 'test-app-insights-metric'
const test_app_insights_metric_value = 10
const test_app_insights_role = 'test-app-insights-role'
const scaled_object_name = "test-app-insights-scaledobject"

class AppInsightsTestData {
  public name: string
  public namespace: string
  public deploymentYaml: string
  public scalerYaml: string
  public secretYaml: string
  public triggerAuthYaml: string

  constructor(name: string, deploymentYaml: string, scalerYaml: string, secretYaml: string = null, triggerAuthYaml: string = null) {
    this.name = name
    this.namespace = namespacePrefix + name
    this.deploymentYaml = deploymentYaml
    this.scalerYaml = scalerYaml
    this.secretYaml = secretYaml
    this.triggerAuthYaml = triggerAuthYaml
  }
}

function sleep(sec: number) {
  if (process.platform === "darwin") {
    sh.exec(`sleep ${sec}`)
  } else {
    sh.exec(`sleep ${sec}s`)
  }
}

function set_metric(metric_value, t, test_callback) {
  appinsights.setup(app_insights_connection_string).setUseDiskRetryCaching(true)
  appinsights.defaultClient.context.tags[appinsights.defaultClient.context.keys.cloudRole] = test_app_insights_role
  appinsights.defaultClient.trackMetric({name: test_app_insights_metric, value: metric_value});
  appinsights.defaultClient.flush({
    callback: function(response: string) {
      let resp_errors = JSON.parse(response)['errors']
      if (resp_errors != null && resp_errors != undefined) {
        t.is(0, resp_errors.length, `failed to set metric: ${JSON.stringify(resp_errors)}`)
      }
      test_callback()
    }
  })
}

function assert_replicas(t, namespace: string, name: string, replicas: number, wait_sec: number) {
  let expectedReplicas = String(replicas)
  let replicaCount = ''
  for (let i = 0; i < wait_sec && replicaCount !== expectedReplicas; i++) {
    replicaCount = sh.exec(
      `kubectl get statefulset.apps/${name} --namespace ${namespace} -o jsonpath="{.spec.replicas}"`
    ).stdout
    if (replicaCount !== expectedReplicas) {
      sleep(1)
    }
  }

  t.is(expectedReplicas, replicaCount, `Replica count for ${namespace}/${name} should be ${expectedReplicas} after some time`)
}

test.before(t => {
  if (!app_insights_app_id || !app_insights_instrumentation_key
    || !app_insights_connection_string || !sp_id || !sp_key || !sp_tenant) {
    t.fail('A required parameters app insights scaler was not resolved')
  }

  sh.config.silent = false

  if (test_pod_id) {
    test_data.push(pod_id_test_data)
  }

  for (let data of test_data) {
    createNamespace(data.namespace)
    t.is(
      0,
      sh.exec(`kubectl apply -f ${createYamlFile(data.deploymentYaml, data)}`).code,
      'creating a deployment should work.'
    )
  }
})

test.serial('Deployment should have 0 replica(s) on start', t => {
  for (let data of test_data) {
    const replicaCount = sh.exec(
      `kubectl get statefulset.apps/${data.name} --namespace ${data.namespace} -o jsonpath="{.spec.replicas}"`
    ).stdout
    t.is(replicaCount, '0', 'replica count should start out as 0')
  }
})

test.serial.cb('Auth trigger deployment should scale to 2 replicas', t => {
  set_metric(test_app_insights_metric_value + 1, t, function() {
    for (let data of test_data) {
      if (data.secretYaml != null) {
        t.is(
          0,
          sh.exec(`kubectl apply -f ${createYamlFile(data.secretYaml, data)}`).code,
          'creating a scaled object should work.'
        )
      }
      if (data.triggerAuthYaml != null) {
        t.is(
          0,
          sh.exec(`kubectl apply -f ${createYamlFile(data.triggerAuthYaml, data)}`).code,
          'creating a scaled object should work.'
        )
      }
      t.is(
        0,
        sh.exec(`kubectl apply -f ${createYamlFile(data.scalerYaml, data)}`).code,
        'creating a scaled object should work.'
      )
    }

    assert_replicas(t, auth_trigger_test_data.namespace, auth_trigger_test_data.name, 2, 240)
    t.end()
  })
})

test.serial('Auth env deployment should scale to 2 replicas', t => {
  assert_replicas(t, auth_env_test_data.namespace, auth_env_test_data.name, 2, 180)
})

test.serial('Pod identity deployment should scale to 2 replicas', t => {
  if (test_pod_id) {
    assert_replicas(t, pod_id_test_data.namespace, pod_id_test_data.name, 2, 180)
  } else {
    t.pass()
  }
})

test.serial.cb('Auth trigger deployment should scale to 0 replicas', t => {
  // Switch to min for aggregation and set a negative metric value to scale down more quickly.
  for (let data of test_data) {
    t.is(
      0,
      sh.exec(`kubectl patch scaledobject ${data.name} -n ${data.namespace} --type='json' -p='[{"op": "replace", "path": "/spec/triggers/0/metadata/metricAggregationType", "value":"min"}]'`).code,
      'changing a scaled object should work.'
    )
  }

  set_metric(-(test_app_insights_metric_value + 1), t, function() {
    assert_replicas(t, auth_trigger_test_data.namespace, auth_trigger_test_data.name, 0, 240)
    t.end()
  })
})

test.serial('Auth env deployment should scale to 0 replicas', t => {
  assert_replicas(t, auth_env_test_data.namespace, auth_env_test_data.name, 0, 240)
})

test.serial('Pod identity deployment should scale to 0 replicas', t => {
  if (test_pod_id) {
    assert_replicas(t, auth_env_test_data.namespace, auth_env_test_data.name, 0, 240)
  } else {
    t.pass()
  }
})

test.after.always.cb('clean up deployment', t => {
  for (let data of test_data) {
    sh.exec(`kubectl delete -f ${createYamlFile(data.scalerYaml, data)}`)
    if (data.triggerAuthYaml != null) {
      sh.exec(`kubectl delete -f ${createYamlFile(data.triggerAuthYaml, data)}`)
    }
    if (data.secretYaml != null) {
      sh.exec(`kubectl delete -f ${createYamlFile(data.secretYaml, data)}`)
    }
    sh.exec(`kubectl delete -f ${createYamlFile(data.deploymentYaml, data)}`)
    sh.exec(`kubectl delete namespace ${data.namespace}`)
  }

  t.end()
})

function createYamlFile(yaml: string, test_data: AppInsightsTestData) {
  const tmpFile = tmp.fileSync()
  fs.writeFileSync(tmpFile.name, yaml
    .replace(/{{NAME}}/g, test_data.name)
    .replace('{{NAMESPACE}}', test_data.namespace)
    .replace('{{TENANT_ID}}', Buffer.from(sp_tenant).toString('base64'))
    .replace('{{CLIENT_ID}}', Buffer.from(sp_id).toString('base64'))
    .replace('{{CLIENT_SECRET}}', Buffer.from(sp_key).toString('base64'))
    .replace('{{APP_INSIGHTS_APP_ID}}', Buffer.from(app_insights_app_id).toString('base64'))
    .replace('{{APP_INSIGHTS_METRIC}}', test_app_insights_metric)
    .replace('{{APP_INSIGHTS_ROLE}}', test_app_insights_role)
    .replace('{{SCALED_OBJECT_NAME}}', scaled_object_name)
    .replace('{{TEST_APP_INSIGHTS_METRIC_VALUE}}', test_app_insights_metric_value.toString()))

  return tmpFile.name
}

let auth_trigger_test_data = new AppInsightsTestData(
'auth-trigger',
`apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: {{NAME}}
  namespace: {{NAMESPACE}}
spec:
  serviceName: "{{NAME}}"
  replicas: 0
  selector:
    matchLabels:
      app: {{NAME}}
  template:
    metadata:
      labels:
        app: {{NAME}}
    spec:
      containers:
      - name: app-insights-scaler-test
        image: nginx:1.16.1`,
`apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name:  {{NAME}}
  namespace: {{NAMESPACE}}
spec:
  scaleTargetRef:
    kind: StatefulSet
    name: {{NAME}}
  pollingInterval: 5
  cooldownPeriod: 5
  minReplicaCount: 0
  maxReplicaCount: 2
  triggers:
    - type: azure-app-insights
      metadata:
        metricId: "customMetrics/{{APP_INSIGHTS_METRIC}}"
        metricAggregationTimespan: "0:5"
        metricAggregationType: max
        metricFilter: cloud/roleName eq '{{APP_INSIGHTS_ROLE}}'
        targetValue: "{{TEST_APP_INSIGHTS_METRIC_VALUE}}"
      authenticationRef:
        name: {{NAME}}`,
`apiVersion: v1
kind: Secret
metadata:
  name: {{NAME}}
  namespace: {{NAMESPACE}}
type: Opaque
data:
  applicationInsightsId: {{APP_INSIGHTS_APP_ID}}
  clientId: {{CLIENT_ID}}
  clientSecret: {{CLIENT_SECRET}}
  tenantId: {{TENANT_ID}}`,
`apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{NAME}}
  namespace: {{NAMESPACE}}
spec:
  secretTargetRef:
    - parameter: applicationInsightsId
      name: {{NAME}}
      key: applicationInsightsId
    - parameter: activeDirectoryClientId
      name: {{NAME}}
      key: clientId
    - parameter: activeDirectoryClientPassword
      name: {{NAME}}
      key: clientSecret
    - parameter: tenantId
      name: {{NAME}}
      key: tenantId`)

let auth_env_test_data = new AppInsightsTestData(
'auth-env',
`apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: {{NAME}}
  namespace: {{NAMESPACE}}
spec:
  serviceName: "{{NAME}}"
  replicas: 0
  selector:
    matchLabels:
      app: {{NAME}}
  template:
    metadata:
      labels:
        app: {{NAME}}
    spec:
      containers:
      - name: app-insights-scaler-test
        image: nginx:1.16.1
        env:
        - name: ACTIVE_DIRECTORY_PASSWORD
          valueFrom:
            secretKeyRef:
              name: {{NAME}}
              key: clientSecret
        - name: ACTIVE_DIRECTORY_USERNAME
          valueFrom:
            secretKeyRef:
              name: {{NAME}}
              key: clientId
        - name: APP_INSIGHTS_APP_ID
          valueFrom:
            secretKeyRef:
              name: {{NAME}}
              key: applicationInsightsId
        - name: TENANT_ID
          valueFrom:
            secretKeyRef:
              name: {{NAME}}
              key: tenantId`,
`apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name:  {{NAME}}
  namespace: {{NAMESPACE}}
spec:
  scaleTargetRef:
    kind: StatefulSet
    name: {{NAME}}
  pollingInterval: 5
  cooldownPeriod: 5
  minReplicaCount: 0
  maxReplicaCount: 2
  triggers:
    - type: azure-app-insights
      metadata:
        metricId: "customMetrics/{{APP_INSIGHTS_METRIC}}"
        metricAggregationTimespan: "0:5"
        metricAggregationType: max
        metricFilter: cloud/roleName eq '{{APP_INSIGHTS_ROLE}}'
        targetValue: "{{TEST_APP_INSIGHTS_METRIC_VALUE}}"
        activeDirectoryClientIdFromEnv: ACTIVE_DIRECTORY_USERNAME
        activeDirectoryClientPasswordFromEnv: ACTIVE_DIRECTORY_PASSWORD
        applicationInsightsIdFromEnv: APP_INSIGHTS_APP_ID
        tenantIdFromEnv: TENANT_ID`,
`apiVersion: v1
kind: Secret
metadata:
  name: {{NAME}}
  namespace: {{NAMESPACE}}
type: Opaque
data:
  applicationInsightsId: {{APP_INSIGHTS_APP_ID}}
  clientId: {{CLIENT_ID}}
  clientSecret: {{CLIENT_SECRET}}
  tenantId: {{TENANT_ID}}`)

let pod_id_test_data = new AppInsightsTestData(
'pod-id',
`apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: {{NAME}}
  namespace: {{NAMESPACE}}
spec:
  serviceName: "{{NAME}}"
  replicas: 0
  selector:
    matchLabels:
      app: {{NAME}}
  template:
    metadata:
      labels:
        app: {{NAME}}
    spec:
      containers:
      - name: app-insights-scaler-test
        image: nginx:1.16.1`,
`apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name:  {{NAME}}
  namespace: {{NAMESPACE}}
spec:
  scaleTargetRef:
    kind: StatefulSet
    name: {{NAME}}
  pollingInterval: 5
  cooldownPeriod: 5
  minReplicaCount: 0
  maxReplicaCount: 2
  triggers:
    - type: azure-app-insights
      metadata:
        metricId: "customMetrics/{{APP_INSIGHTS_METRIC}}"
        metricAggregationTimespan: "0:5"
        metricAggregationType: max
        metricFilter: cloud/roleName eq '{{APP_INSIGHTS_ROLE}}'
        targetValue: "{{TEST_APP_INSIGHTS_METRIC_VALUE}}"
      authenticationRef:
        name: {{NAME}}`,
`apiVersion: v1
kind: Secret
metadata:
  name: {{NAME}}
  namespace: {{NAMESPACE}}
type: Opaque
data:
  applicationInsightsId: {{APP_INSIGHTS_APP_ID}}
  tenantId: {{TENANT_ID}}`,
`apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{NAME}}
  namespace: {{NAMESPACE}}
spec:
  podIdentity:
    provider: azure
  secretTargetRef:
    - parameter: applicationInsightsId
      name: {{NAME}}
      key: applicationInsightsId
    - parameter: tenantId
      name: {{NAME}}
      key: tenantId`)

let test_data = [auth_trigger_test_data, auth_env_test_data]
