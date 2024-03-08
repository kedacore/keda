//go:build e2e
// +build e2e

package influx_db_test

import (
	"bufio"
	"fmt"
	"strings"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName        = "influx-v3-db-test"
	influxdbJobName = "influx-v3-client-job"
	deploymentName  = "nginx-deployment"
	label           = "job-name=influx-v3-client-job"
)

var (
	testNamespace           = fmt.Sprintf("%s-ns", testName)
	influxdbStatefulsetName = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName        = fmt.Sprintf("%s-so", testName)
	authToken               = ""
	databaseName            = ""
)

type templateData struct {
	TestNamespace           string
	InfluxdbStatefulsetName string
	InfluxdbWriteJobName    string
	ScaledObjectName        string
	DeploymentName          string
	AuthToken               string
	DatabaseName            string
}

const (
	influxdbStatefulsetTemplate = `
apiVersion: apps/v1
kind: StatefulSet
metadata:
    labels:
        app: influxdb
    name: {{.InfluxdbStatefulsetName}}
    namespace: {{.TestNamespace}}
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
              - image: ghcr.io/metrico/influxdb-edge-musl:latest
                name: influxdb
                ports:
                  - containerPort: 8080
                    name: influxdb
				env:
				- name: INFLUXDB_IOX_ROUTER_HTTP_BIND_ADDR
				  value: http://iox:8080
				- name: INFLUXDB_IOX_OBJECT_STORE
				  value: file
				- name: INFLUXDB_IOX_DB_DIR
				  value: /data/db
				- name: INFLUXDB_IOX_BUCKET
				  value: iox
				- name: INFLUXDB_IOX_CATALOG_DSN
				  value: sqlite:///data/catalog.sqlite
				- name: INFLUXDB_IOX_WAL_DIRECTORY
				  value: /data/wal
                readinessProbe:
                  tcpSocket:
                    port: 8080
                  initialDelaySeconds: 5
                  periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
    name: influxdb
    namespace: {{.TestNamespace}}
spec:
    ports:
      - name: influxdb
        port: 8080
        targetPort: 8080
    selector:
        app: influxdb
    type: ClusterIP
`

	scaledObjectActivationTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  maxReplicaCount: 2
  triggers:
  - type: influxdb
    metadata:
      authToken: {{.AuthToken}}
      databaseName: {{.DatabaseName}}
      serverURL: http://influxdb.{{.TestNamespace}}.svc:8080
      influxVersion: "3"
      thresholdValue: "3"
      queryType: "InfluxQL"
      metricKey: "stat"
      query: 'SELECT something AS "stat" FROM "something" GROUP BY time(5m) ORDER BY time DESC LIMIT 1;'
`
	scaledObjectTemplateFloat = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  maxReplicaCount: 2
  triggers:
  - type: influxdb
    metadata:
      authToken: {{.AuthToken}}
      databaseName: {{.DatabaseName}}
      serverURL: http://influxdb.{{.TestNamespace}}.svc:8080
      influxVersion: "3"
      thresholdValue: "3"
      queryType: "InfluxQL"
      metricKey: "stat"
      query: 'SELECT something AS "stat" FROM "something" GROUP BY time(5m) ORDER BY time DESC LIMIT 1;'
`

	influxdbWriteJobTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  name: {{.InfluxdbWriteJobName}}
  namespace: {{.TestNamespace}}
spec:
  template:
    spec:
      containers:
      - name: influx-client-job
        image: ghcr.io/metrico/iox:latest
        env:
        - name: INFLUXDB_SERVER_URL
          value: http://influxdb:8086
      restartPolicy: OnFailure
`

	deploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
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
        image: nginxinc/nginx-unprivileged
        ports:
        - containerPort: 80
`
)

func TestInfluxScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForStatefulsetReplicaReadyCount(t, kc, influxdbStatefulsetName, testNamespace, 1, 60, 1),
		"replica count should be 0 after a minute")

	// get token
	updateDataWithInfluxAuth(t, kc, &data)

	// test activation
	testActivation(t, kc, data)
	// test scaling
	testScaleFloat(t, kc, data)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func updateDataWithInfluxAuth(t *testing.T, kc *kubernetes.Clientset, data *templateData) {
	// run writeJob
	KubectlReplaceWithTemplate(t, data, "influxdbWriteJobTemplate", influxdbWriteJobTemplate)
	assert.True(t, WaitForJobSuccess(t, kc, influxdbJobName, testNamespace, 30, 2), "Job should run successfully")

	// get pod logs
	log, err := FindPodLogs(kc, testNamespace, label, false)
	assert.NoErrorf(t, err, "cannotget logs - %s", err)

	var lines []string
	sc := bufio.NewScanner(strings.NewReader(log[0]))
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	data.AuthToken = (strings.SplitN(lines[0], "=", 2))[1]
	data.OrgName = (strings.SplitN(lines[1], "=", 2))[1]
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:           testNamespace,
			InfluxdbStatefulsetName: influxdbStatefulsetName,
			InfluxdbWriteJobName:    influxdbJobName,
			ScaledObjectName:        scaledObjectName,
			DeploymentName:          deploymentName,
			AuthToken:               authToken,
			OrgName:                 orgName,
		}, []Template{
			{Name: "influxdbStatefulsetTemplate", Config: influxdbStatefulsetTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
		}
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation---")

	KubectlApplyWithTemplate(t, data, "scaledObjectActivationTemplate", scaledObjectActivationTemplate)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 30)
}

func testScaleFloat(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out float---")

	KubectlApplyWithTemplate(t, data, "scaledObjectTemplateFloat", scaledObjectTemplateFloat)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 2, 60, 1),
		"replica count should be 2 after a minute")
}
