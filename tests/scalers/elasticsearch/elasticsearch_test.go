//go:build e2e
// +build e2e

package elasticsearch_test

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"testing"
	"text/template"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "elasticsearch-test"
)

var (
	testNamespace         = fmt.Sprintf("%s-ns", testName)
	deploymentName        = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName      = fmt.Sprintf("%s-so", testName)
	secretName            = fmt.Sprintf("%s-secret", testName)
	password              = "passw0rd"
	indexName             = "keda"
	searchTemplateName    = "keda-search-template"
	maxReplicaCount       = 2
	minReplicaCount       = 0
	kubectlElasticExecCmd = fmt.Sprintf("kubectl exec -n %s elasticsearch-0 -- curl -sS -H 'Content-Type: application/json' -u 'elastic:%s'", testNamespace, password)
)

type templateData struct {
	TestNamespace         string
	DeploymentName        string
	ScaledObjectName      string
	SecretName            string
	ElasticPassword       string
	ElasticPasswordBase64 string
	IndexName             string
	SearchTemplateName    string
}

const (
	secretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  password: {{.ElasticPasswordBase64}}
`

	triggerAuthenticationTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: keda-trigger-auth-elasticsearch-secret
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
  - parameter: password
    name: {{.SecretName}}
    key: password
`

	deploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  replicas: 0
  selector:
    matchLabels:
      app: {{.DeploymentName}}
  template:
    metadata:
      labels:
        app: {{.DeploymentName}}
    spec:
      containers:
      - name: nginx
        image: ghcr.io/nginx/nginx-unprivileged:1.26
        ports:
        - containerPort: 80
`
	elasticsearchDeploymentTemplate = `
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: elasticsearch
  namespace: {{.TestNamespace}}
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
            value: "{{.ElasticPassword}}"
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
        readinessProbe:
          exec:
            command:
              - /usr/bin/curl
              - -sS
              - -u "elastic:{{.ElasticPassword}}"
              - http://localhost:9200
          failureThreshold: 3
          initialDelaySeconds: 10
          periodSeconds: 5
          successThreshold: 1
          timeoutSeconds: 5
  serviceName: {{.DeploymentName}}
`

	serviceTemplate = `apiVersion: v1
kind: Service
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
spec:
  type: ClusterIP
  ports:
  - name: http
    port: 9200
    targetPort: 9200
    protocol: TCP
  selector:
    name: elasticsearch
`

	scaledObjectTemplateSearchTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  minReplicaCount: 0
  maxReplicaCount: 2
  pollingInterval: 3
  cooldownPeriod:  5
  triggers:
    - type: elasticsearch
      metadata:
        addresses: "http://{{.DeploymentName}}.{{.TestNamespace}}.svc:9200"
        username: "elastic"
        index: {{.IndexName}}
        searchTemplateName: {{.SearchTemplateName}}
        valueLocation: "hits.total.value"
        targetValue: "1"
        activationTargetValue: "4"
        parameters: "dummy_value:1;dumb_value:oOooo"
      authenticationRef:
        name: keda-trigger-auth-elasticsearch-secret
`

	scaledObjectTemplateQuery = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  minReplicaCount: 0
  maxReplicaCount: 2
  pollingInterval: 3
  cooldownPeriod:  5
  triggers:
    - type: elasticsearch
      metadata:
        addresses: "http://{{.DeploymentName}}.{{.TestNamespace}}.svc:9200"
        username: "elastic"
        index: {{.IndexName}}
        query: |
          {
            "query": {
              "bool": {
                "must": [
                  {
                    "range": {
                      "@timestamp": {
                        "gte": "now-1m",
                        "lte": "now"
                      }
                    }
                  },
                  {
                    "match_all": {}
                  }
                ]
              }
            }
          }
        valueLocation: "hits.total.value"
        targetValue: "1"
        activationTargetValue: "4"
      authenticationRef:
        name: keda-trigger-auth-elasticsearch-secret
`

	elasticsearchCreateIndex = `
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

	elasticsearchSearchTemplate = `
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
                  "gte": "now-1m",
                  "lte": "now"
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

	elasticsearchDummyDoc = `
{
  "@timestamp": "{{.Timestamp}}",
  "dummy": 1,
  "dumb": "oOooo"
}`
)

func TestElasticsearchScaler(t *testing.T) {
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	// Create kubernetes resources
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	// setup elastic
	setupElasticsearch(t, kc)

	t.Run("test with searchTemplateName", func(t *testing.T) {
		t.Log("--- testing with searchTemplateName ---")

		// Create ScaledObject with searchTemplateName
		KubectlApplyWithTemplate(t, data, "scaledObjectTemplateSearchTemplate", scaledObjectTemplateSearchTemplate)

		testElasticsearchScaler(t, kc)

		// Delete ScaledObject
		KubectlDeleteWithTemplate(t, data, "scaledObjectTemplateSearchTemplate", scaledObjectTemplateSearchTemplate)
	})

	t.Run("test with query", func(t *testing.T) {
		t.Log("--- testing with query ---")

		// Create ScaledObject with query
		KubectlApplyWithTemplate(t, data, "scaledObjectTemplateQuery", scaledObjectTemplateQuery)

		testElasticsearchScaler(t, kc)

		// Delete ScaledObject
		KubectlDeleteWithTemplate(t, data, "scaledObjectTemplateQuery", scaledObjectTemplateQuery)
	})

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func setupElasticsearch(t *testing.T, kc *kubernetes.Clientset) {
	require.True(t, WaitForStatefulsetReplicaReadyCount(t, kc, "elasticsearch", testNamespace, 1, 60, 3),
		"elasticsearch should be up")
	// Create the index and the search template
	_, err := ExecuteCommand(fmt.Sprintf("%s -XPUT http://localhost:9200/%s -d '%s'", kubectlElasticExecCmd, indexName, elasticsearchCreateIndex))
	require.NoErrorf(t, err, "cannot execute command - %s", err)
	_, err = ExecuteCommand(fmt.Sprintf("%s -XPUT http://localhost:9200/_scripts/%s -d '%s'", kubectlElasticExecCmd, searchTemplateName, elasticsearchSearchTemplate))
	require.NoErrorf(t, err, "cannot execute command - %s", err)
}

func testElasticsearchScaler(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing activation ---")
	addElements(t, 3)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)

	t.Log("--- testing scale out ---")
	addElements(t, 10)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)

	t.Log("--- testing scale in ---")

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
}

func addElements(t *testing.T, count int) {
	for i := 0; i < count; i++ {
		result, err := getElasticsearchDoc()
		assert.NoErrorf(t, err, "cannot parse log - %s", err)
		_, err = ExecuteCommand(fmt.Sprintf("%s -XPOST http://localhost:9200/%s/_doc -d '%s'", kubectlElasticExecCmd, indexName, result))
		assert.NoErrorf(t, err, "cannot execute command - %s", err)
	}
}

func getElasticsearchDoc() (interface{}, error) {
	tmpl, err := template.New("elasticsearch doc").Parse(elasticsearchDummyDoc)
	var tpl bytes.Buffer
	if err := tmpl.Execute(&tpl, struct{ Timestamp string }{Timestamp: time.Now().Format(time.RFC3339)}); err != nil {
		return nil, err
	}
	result := tpl.String()
	return result, err
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:         testNamespace,
			DeploymentName:        deploymentName,
			ScaledObjectName:      scaledObjectName,
			SecretName:            secretName,
			ElasticPassword:       password,
			ElasticPasswordBase64: base64.StdEncoding.EncodeToString([]byte(password)),
			IndexName:             indexName,
			SearchTemplateName:    searchTemplateName,
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
			{Name: "serviceTemplate", Config: serviceTemplate},
			{Name: "elasticsearchDeploymentTemplate", Config: elasticsearchDeploymentTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
		}
}
