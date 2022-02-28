import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import test from 'ava'
import {createNamespace, waitForRollout} from "./helpers";

const testNamespace = 'elasticsearch-test'
const elasticsearchNamespace = 'elasticsearch'
const deploymentName = 'podinfo'
const indexName = 'keda'
const searchTemplateName = 'keda-search-template'
const elasticPassword = 'passw0rd!'
const kubectlExecCurl = `kubectl exec -n ${elasticsearchNamespace} elasticsearch-0 -- curl -sS -H "content-type: application/json" -u "elastic:${elasticPassword}"`

test.before(t => {
    // install elasticsearch
    createNamespace(elasticsearchNamespace)
    const elasticsearchTmpFile = tmp.fileSync()
    fs.writeFileSync(elasticsearchTmpFile.name, elasticsearchStatefulsetYaml.replace('{{ELASTIC_PASSWORD}}', elasticPassword))

    t.is(0, sh.exec(`kubectl apply --namespace ${elasticsearchNamespace} -f ${elasticsearchTmpFile.name}`).code, 'creating an elasticsearch statefulset should work.')
    t.is(0, waitForRollout('statefulset', "elasticsearch", elasticsearchNamespace))

    // Create the index and the search template
    sh.exec(`${kubectlExecCurl} -XPUT http://localhost:9200/${indexName} -d '${elastisearchCreateIndex}'`)
    sh.exec(`${kubectlExecCurl} -XPUT http://localhost:9200/_scripts/${searchTemplateName} -d '${elasticsearchSearchTemplate}'`)

    createNamespace(testNamespace)

    // deploy dummy app and scaled object
    const tmpFile = tmp.fileSync()
    fs.writeFileSync(tmpFile.name, deployYaml.replace(/{{DEPLOYMENT_NAME}}/g, deploymentName)
        .replace('{{ELASTICSEARCH_NAMESPACE}}', elasticsearchNamespace)
        .replace('{{SEARCH_TEMPLATE_NAME}}', searchTemplateName)
        .replace('{{INDEX_NAME}}', indexName)
        .replace('{{ELASTIC_PASSWORD_BASE64}}', Buffer.from(elasticPassword).toString('base64'))
    )

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

    for (let i = 0; i < 5; i++) {
        let doc = elasticsearchDummyDoc.replace("{{TIMESTAMP}}", new Date().toISOString())
        sh.exec(`${kubectlExecCurl} -XPOST http://localhost:9200/${indexName}/_doc -d '${doc}'`)
    }

    let replicaCount = '0'

    const maxReplicaCount = '5'

    for (let i = 0; i < 90 && replicaCount !== maxReplicaCount; i++) {
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

test.after.always.cb('clean up elasticsearch deployment', t => {
    sh.exec(`kubectl delete namespace ${testNamespace}`)

    // uninstall elasticsearch
    sh.exec(`kubectl delete --namespace ${elasticsearchNamespace} sts/elasticsearch`)
    sh.exec(`kubectl delete namespace ${elasticsearchNamespace}`)

    t.end()
})

const deployYaml = `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: {{DEPLOYMENT_NAME}}
  name: {{DEPLOYMENT_NAME}}
spec:
  replicas: 0
  selector:
    matchLabels:
      app: {{DEPLOYMENT_NAME}}
  template:
    metadata:
      labels:
        app: {{DEPLOYMENT_NAME}}
    spec:
      containers:
      - image: stefanprodan/podinfo
        name: {{DEPLOYMENT_NAME}}
---
apiVersion: v1
kind: Secret
metadata:
  name: elasticsearch-secrets
type: Opaque
data:
  password: {{ELASTIC_PASSWORD_BASE64}}
---
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: keda-trigger-auth-elasticsearch-secret
spec:
  secretTargetRef:
  - parameter: password
    name: elasticsearch-secrets
    key: password
---
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: elasticsearch-scaledobject
spec:
  minReplicaCount: 0
  maxReplicaCount: 5
  pollingInterval: 3
  cooldownPeriod:  5
  scaleTargetRef:
    name: {{DEPLOYMENT_NAME}}
  triggers:
    - type: elasticsearch
      metadata:
        addresses: "http://elasticsearch-svc.{{ELASTICSEARCH_NAMESPACE}}.svc.cluster.local:9200"
        username: "elastic"
        index: {{INDEX_NAME}}
        searchTemplateName: {{SEARCH_TEMPLATE_NAME}}
        valueLocation: "hits.total.value"
        targetValue: "1"
        parameters: "dummy_value:1;dumb_value:oOooo"
      authenticationRef:
        name: keda-trigger-auth-elasticsearch-secret
`

const elasticsearchStatefulsetYaml = `
kind: Service
apiVersion: v1
metadata:
  name: elasticsearch-svc
spec:
  type: ClusterIP
  ports:
  - name: http
    port: 9200
    targetPort: 9200
    protocol: TCP
  selector:
    name: elasticsearch
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: elasticsearch
spec:
  replicas: 1
  selector:
    matchLabels:
      name: elasticsearch
  template:
    metadata:
      labels:
        name: elasticsearch
    spec:
      containers:
      - name: elasticsearch
        image: docker.elastic.co/elasticsearch/elasticsearch:7.15.1
        imagePullPolicy: IfNotPresent
        env:
          - name: POD_IP
            valueFrom:
              fieldRef:
                apiVersion: v1
                fieldPath: status.podIP
          - name: POD_NAME
            valueFrom:
              fieldRef:
                apiVersion: v1
                fieldPath: metadata.name
          - name: NODE_NAME
            valueFrom:
              fieldRef:
                apiVersion: v1
                fieldPath: spec.nodeName
          - name: NAMESPACE
            valueFrom:
              fieldRef:
                apiVersion: v1
                fieldPath: metadata.namespace
          - name: ES_JAVA_OPTS
            value: -Xms256m -Xmx256m
          - name: cluster.name
            value: elasticsearch-keda
          - name: cluster.initial_master_nodes
            valueFrom:
              fieldRef:
                apiVersion: v1
                fieldPath: status.podIP
          - name: node.data
            value: "true"
          - name: node.ml
            value: "false"
          - name: node.ingest
            value: "false"
          - name: node.master
            value: "true"
          - name: node.remote_cluster_client
            value: "false"
          - name: node.transform
            value: "false"
          - name: ELASTIC_PASSWORD
            value: "{{ELASTIC_PASSWORD}}"
          - name: xpack.security.enabled
            value: "true"
          - name: node.store.allow_mmap
            value: "false"
        ports:
        - containerPort: 9200
          name: http
          protocol: TCP
        - containerPort: 9300
          name: transport
          protocol: TCP
        resources:
        readinessProbe:
          exec:
            command:
              - /usr/bin/curl
              - -sS
              - -u "elastic:{{ELASTIC_PASSWORD}}"
              - http://localhost:9200
          failureThreshold: 3
          initialDelaySeconds: 10
          periodSeconds: 5
          successThreshold: 1
          timeoutSeconds: 5


  serviceName: elasticsearch-svc
`

const elastisearchCreateIndex = `
{
  "mappings": {
    "properties": {
      "@timestamp": {
        "type": "date"
      },
      "dummy": {
        "type": "integer"
      },
      "dumb": {
        "type": "keyword"
      }
    }
  },
  "settings": {
    "number_of_replicas": 0,
    "number_of_shards": 1
  }
}`

const elasticsearchDummyDoc = `
{
  "@timestamp": "{{TIMESTAMP}}",
  "dummy": 1,
  "dumb": "oOooo"
}`

const elasticsearchSearchTemplate = `
{
  "script": {
    "lang": "mustache",
    "source": {
      "query": {
        "bool": {
          "filter": [
            {
              "range": {
                "@timestamp": {
                  "gte": "now-1m/m",
                  "lte": "now/m"
                }
              }
            },
            {
              "term": {
                "dummy": "{{dummy_value}}"
              }
            },
            {
              "term": {
                "dumb": "{{dumb_value}}"
              }
            }
          ]
        }
      }
    }
  }
}`
