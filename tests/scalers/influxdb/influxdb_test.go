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
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName        = "influx-db-test"
	influxdbJobName = "influx-client-job"
	deploymentName  = "nginx-deployment"
	label           = "job-name=influx-client-job"
)

var (
	testNamespace           = fmt.Sprintf("%s-ns", testName)
	influxdbStatefulsetName = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName        = fmt.Sprintf("%s-so", testName)
	authToken               = ""
	orgName                 = ""
)

type templateData struct {
	TestNamespace           string
	InfluxdbStatefulsetName string
	InfluxdbWriteJobName    string
	ScaledObjectName        string
	DeploymentName          string
	AuthToken               string
	OrgName                 string
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
      organizationName: {{.OrgName}}
      serverURL: http://influxdb.{{.TestNamespace}}.svc:8086
      thresholdValue: "80"
      activationThresholdValue: "110"
      query: |
        from(bucket:"bucket")
        |> range(start: -1h)
        |> filter(fn: (r) => r._measurement == "stat")
        |> map(fn: (r) => ({r with _value: float(v: r._value)}))
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
      organizationName: {{.OrgName}}
      serverURL: http://influxdb.{{.TestNamespace}}.svc:8086
      thresholdValue: "3"
      query: |
        from(bucket:"bucket")
        |> range(start: -1h)
        |> filter(fn: (r) => r._measurement == "stat")
        |> map(fn: (r) => ({r with _value: float(v: r._value)}))
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
        image: ghcr.io/nginx/nginx-unprivileged:1.26
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
	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})

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
}

func updateDataWithInfluxAuth(t *testing.T, kc *kubernetes.Clientset, data *templateData) {
	// run writeJob
	KubectlReplaceWithTemplate(t, data, "influxdbWriteJobTemplate", influxdbWriteJobTemplate)
	require.True(t, WaitForJobSuccess(t, kc, influxdbJobName, testNamespace, 30, 2), "Job should run successfully")

	// get pod logs
	log, err := FindPodLogs(kc, testNamespace, label, false)
	require.NoErrorf(t, err, "cannotget logs - %s", err)

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
