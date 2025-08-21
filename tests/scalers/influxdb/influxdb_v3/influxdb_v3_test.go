//go:build e2e
// +build e2e

package influxdb_v3_test

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
var _ = godotenv.Load("../../../.env")

const (
	testName        = "influxdb-v3-test"
	influxdbJobName = "influxdb-v3-client-job"
	deploymentName  = "nginx-deployment"
	label           = "job-name=influxdb-v3-client-job"
)

var (
	testNamespace           = fmt.Sprintf("%s-ns", testName)
	influxdbStatefulsetName = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName        = fmt.Sprintf("%s-so", testName)
	authToken               = ""
	databaseName            = "testdb"
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
        app: influxdb-v3
    name: {{.InfluxdbStatefulsetName}}
    namespace: {{.TestNamespace}}
spec:
    replicas: 1
    selector:
        matchLabels:
            app: influxdb-v3
    serviceName: influxdb-v3
    template:
        metadata:
            labels:
                app: influxdb-v3
        spec:
            containers:
              - image: influxdb:3-core
                name: influxdb-v3
                args: ["influxdb3", "serve", "--node-id", "node1", "--object-store", "memory", "--disable-authz", "health"]
                ports:
                  - containerPort: 8181
                    name: influxdb-v3

                readinessProbe:
                  tcpSocket:
                    port: 8181
                  initialDelaySeconds: 15
                  periodSeconds: 10

---
apiVersion: v1
kind: Service
metadata:
    name: influxdb-v3
    namespace: {{.TestNamespace}}
spec:
    ports:
      - name: influxdb-v3
        port: 8181
        targetPort: 8181
    selector:
        app: influxdb-v3
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
      influxVersion: "3"
      authToken: {{.AuthToken}}
      organizationName: "testorg"
      serverURL: http://influxdb-v3.{{.TestNamespace}}.svc:8181
      database: {{.DatabaseName}}
      metricKey: "_value"
      thresholdValue: "5"
      activationThresholdValue: "10"
      query: |
        SELECT value FROM stat ORDER BY time DESC LIMIT 1
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
      influxVersion: "3"
      authToken: {{.AuthToken}}
      organizationName: "testorg"
      serverURL: http://influxdb-v3.{{.TestNamespace}}.svc:8181
      database: {{.DatabaseName}}
      metricKey: "value"
      thresholdValue: "5"
      query: |
        SELECT value FROM stat ORDER BY time DESC LIMIT 1
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
      - name: influx-v3-client-job
        image: influxdb:3-core
        env:
        - name: INFLUXDB3_HOST_URL
          value: http://influxdb-v3:8181
        - name: DATABASE_NAME
          value: {{.DatabaseName}}
        command: ["/bin/sh"]
        args:
        - -c
        - |
          set -e
          echo "Waiting for InfluxDB v3 to be ready..."
          until curl -m 5 -s -o /dev/null -w "%{http_code}" http://influxdb-v3:8181/health | grep -q "200"; do
            echo "InfluxDB v3 not ready, waiting..."
            sleep 5
          done
          echo "InfluxDB v3 is ready"

          # Create admin token (fresh memory instance, no existing tokens)
          echo "Creating admin token..."
          TOKEN=$(influxdb3 create token --admin --host http://influxdb-v3:8181 | grep "Token:" | cut -d' ' -f2)
          echo "AUTH_TOKEN=$TOKEN"

          # Create database
          echo "Creating database..."
          influxdb3 create database "$DATABASE_NAME" --host http://influxdb-v3:8181 --token "$TOKEN"
          echo "DATABASE_NAME=$DATABASE_NAME"

          # Write initial test data (below activation threshold 10)
          echo "Writing test data..."
          influxdb3 write --host http://influxdb-v3:8181 --token "$TOKEN" --database "$DATABASE_NAME" "stat,location=test value=3.0"
          echo "Initial test data written successfully"
      restartPolicy: OnFailure
`

	scalingDataJobTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  name: influx-v3-scaling-job
  namespace: {{.TestNamespace}}
spec:
  template:
    spec:
      containers:
      - name: influx-v3-scaling-job
        image: influxdb:3-core
        env:
        - name: INFLUXDB3_HOST_URL
          value: http://influxdb-v3:8181
        - name: DATABASE_NAME
          value: {{.DatabaseName}}
        - name: AUTH_TOKEN
          value: {{.AuthToken}}
        command: ["/bin/sh"]
        args:
        - -c
        - |
          set -e
          echo "Writing scaling data..."
          # Write data with values above scaling threshold (5)
          influxdb3 write --host http://influxdb-v3:8181 --token "$AUTH_TOKEN" --database "$DATABASE_NAME" "stat,location=test value=10.0"
          echo "Scaling data written successfully"
      restartPolicy: Never
`

	deploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    app: nginx-deployment-v3
spec:
  replicas: 0
  selector:
    matchLabels:
      app: nginx-deployment-v3
  template:
    metadata:
      labels:
        app: nginx-deployment-v3
    spec:
      containers:
      - name: nginx-deployment-v3
        image: ghcr.io/nginx/nginx-unprivileged:1.26
        ports:
        - containerPort: 80
`
)

func TestInfluxV3Scaler(t *testing.T) {
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
		"replica count should be 1 after a minute")

	// get token
	updateDataWithInfluxAuth(t, kc, &data)

	// test activation (should not scale with low values)
	testActivation(t, kc, data)

	// write higher values for scaling test
	writeScalingData(t, kc, data)

	// test scaling (should scale with high values)
	testScaleFloat(t, kc, data)

	// cleanup
}

func updateDataWithInfluxAuth(t *testing.T, kc *kubernetes.Clientset, data *templateData) {
	// run writeJob
	KubectlReplaceWithTemplate(t, data, "influxdbWriteJobTemplate", influxdbWriteJobTemplate)
	require.True(t, WaitForJobSuccess(t, kc, influxdbJobName, testNamespace, 30, 2), "Job should run successfully")

	// get pod logs
	log, err := FindPodLogs(kc, testNamespace, label, false)
	require.NoErrorf(t, err, "cannot get logs - %s", err)

	var lines []string
	sc := bufio.NewScanner(strings.NewReader(log[0]))
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}

	// Find the line containing AUTH_TOKEN=
	for _, line := range lines {
		if strings.HasPrefix(line, "AUTH_TOKEN=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				data.AuthToken = parts[1]
				break
			}
		}
	}

	data.DatabaseName = databaseName
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:           testNamespace,
			InfluxdbStatefulsetName: influxdbStatefulsetName,
			InfluxdbWriteJobName:    influxdbJobName,
			ScaledObjectName:        scaledObjectName,
			DeploymentName:          deploymentName,
			AuthToken:               authToken,
			DatabaseName:            databaseName,
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

func writeScalingData(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- writing scaling data ---")

	KubectlApplyWithTemplate(t, data, "scalingDataJobTemplate", scalingDataJobTemplate)
	require.True(t, WaitForJobSuccess(t, kc, "influx-v3-scaling-job", testNamespace, 30, 2), "Scaling data job should complete")
}

func testScaleFloat(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out float---")

	KubectlApplyWithTemplate(t, data, "scaledObjectTemplateFloat", scaledObjectTemplateFloat)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 2, 60, 1),
		"replica count should be 2 after a minute")
}
