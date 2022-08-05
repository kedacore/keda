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

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName            = "influx-db-test"
	influxdbJobName     = "influx-client-job"
	basicDeploymentName = "nginx-deployment"
	label               = "job-name=influx-client-job"
)

var (
	testNamespace           = fmt.Sprintf("%s-ns", testName)
	influxdbStatefulsetName = fmt.Sprintf("%s-deployment", testName)
	scaledObjectFloatName   = fmt.Sprintf("%s-so-float", testName)
	scaledObjectIntName     = fmt.Sprintf("%s-so-int", testName)
	authToken               = ""
	orgName                 = ""
)

type templateData struct {
	TestNamespace           string
	InfluxdbStatefulsetName string
	InfluxdbWriteJobName    string
	ScaledObjectIntName     string
	ScaledObjectFloatName   string
	BasicDeploymentName     string
	AuthToken               string
	OrgName                 string
}

type templateValues map[string]string

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
              - image: quay.io/influxdb/influxdb:v2.0.1
                name: influxdb
                ports:
                  - containerPort: 8086
                    name: influxdb
                readinessProbe:
                  tcpSocket:
                    port: 8086
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
        port: 8086
        targetPort: 8086
    selector:
        app: influxdb
    type: ClusterIP
`

	scaledObjectTemplateFloat = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectFloatName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.BasicDeploymentName}}
  maxReplicaCount: 2
  triggers:
  - type: influxdb
    metadata:
      authToken: {{.AuthToken}}
      organizationName: {{.OrgName}}
      serverURL: http://influxdb.{{.TestNamespace}}.svc:8086
      thresholdValue: "3"
      query: |
        from(bucket:"bucket")
        |> range(start: -1h)
        |> filter(fn: (r) => r._measurement == "stat")
        |> map(fn: (r) => ({r with _value: float(v: r._value)}))
`

	scaledObjectTemplateInt = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectIntName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.BasicDeploymentName}}
  maxReplicaCount: 2
  triggers:
  - type: influxdb
    metadata:
      authToken: {{.AuthToken}}
      organizationName: {{.OrgName}}
      serverURL: http://influxdb.{{.TestNamespace}}.svc:8086
      thresholdValue: "3"
      query: |
        from(bucket:"bucket")
        |> range(start: -1h)
        |> filter(fn: (r) => r._measurement == "stat")
        |> map(fn: (r) => ({r with _value: int(v: r._value)}))
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
        image: docker.io/yquansah/influxdb:2-client
        env:
        - name: INFLUXDB_SERVER_URL
          value: http://influxdb:8086
      restartPolicy: OnFailure
`

	basicDeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.BasicDeploymentName}}
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
        image: nginx:1.14.2
        ports:
        - containerPort: 80
`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templateValues{"influxdbStatefulsetTemplate": influxdbStatefulsetTemplate})

	assert.True(t, WaitForStatefulsetReplicaReadyCount(t, kc, influxdbStatefulsetName, testNamespace, 1, 60, 1),
		"replica count should be 0 after a minute")

	// test scaling
	testScaleFloat(t, kc)
	testScaleInt(t, kc)

	// cleanup
	DeleteKubernetesResources(t, kc, testNamespace, data, templates)
}

func runWriteJob(t *testing.T, kc *kubernetes.Clientset) templateData {
	// run writeJob
	data, _ := getTemplateData()
	KubectlApplyWithTemplate(t, data, "influxdbWriteJobTemplate", influxdbWriteJobTemplate)
	assert.True(t, WaitForJobSuccess(t, kc, influxdbJobName, testNamespace, 30, 2), "Job should run successfully")

	// get pod logs
	log := FindPodLogs(t, kc, testNamespace, label)

	var lines []string
	sc := bufio.NewScanner(strings.NewReader(log[0]))
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	authToken = (strings.SplitN(lines[0], "=", 2))[1]
	orgName = (strings.SplitN(lines[1], "=", 2))[1]
	data = templateData{
		TestNamespace:           testNamespace,
		InfluxdbStatefulsetName: influxdbStatefulsetName,
		InfluxdbWriteJobName:    influxdbJobName,
		ScaledObjectIntName:     scaledObjectIntName,
		ScaledObjectFloatName:   scaledObjectFloatName,
		BasicDeploymentName:     basicDeploymentName,
		AuthToken:               authToken,
		OrgName:                 orgName,
	}
	return data
}

func getTemplateData() (templateData, templateValues) {
	return templateData{
			TestNamespace:           testNamespace,
			InfluxdbStatefulsetName: influxdbStatefulsetName,
			InfluxdbWriteJobName:    influxdbJobName,
			ScaledObjectIntName:     scaledObjectIntName,
			ScaledObjectFloatName:   scaledObjectFloatName,
			BasicDeploymentName:     basicDeploymentName,
		}, templateValues{
			"influxdbStatefulsetTemplate": influxdbStatefulsetTemplate,
			"scaledObjectTemplateFloat":   scaledObjectTemplateFloat,
			"scaledObjectTemplateInt":     scaledObjectTemplateInt,
			"basicDeploymentTemplate":     basicDeploymentTemplate,
		}
}

func testScaleFloat(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale up float---")
	data := runWriteJob(t, kc)
	KubectlApplyWithTemplate(t, data, "basicDeploymentTemplate", basicDeploymentTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, basicDeploymentName, testNamespace, 0, 60, 1),
		"replica count should be 1 after a minute")

	KubectlApplyWithTemplate(t, data, "scaledObjectTemplateFloat", scaledObjectTemplateFloat)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, basicDeploymentName, testNamespace, 2, 60, 1),
		"replica count should be 2 after a minute")
}

func testScaleInt(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale up with int ---")
	data := runWriteJob(t, kc)
	KubectlApplyWithTemplate(t, data, "basicDeploymentTemplate", basicDeploymentTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, basicDeploymentName, testNamespace, 0, 60, 1),
		"replica count should be 1 after a minute")

	KubectlApplyWithTemplate(t, data, "scaledObjectTemplateInt", scaledObjectTemplateInt)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, basicDeploymentName, testNamespace, 2, 60, 1),
		"replica count should be 2 after a minute")
}
