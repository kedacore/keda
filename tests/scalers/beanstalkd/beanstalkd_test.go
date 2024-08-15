//go:build e2e
// +build e2e

package beanstalkd_test

import (
	"fmt"
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
	testName             = "beanstalkd-test"
	deploymentName       = "beanstalkd-consumer-deployment"
	beanstalkdPutJobName = "beanstalkd-put-job"
)

var (
	testNamespace            = fmt.Sprintf("%s-ns", testName)
	beanstalkdDeploymentName = fmt.Sprintf("%s-beanstalkd-deployment", testName)
	scaledObjectName         = fmt.Sprintf("%s-so", testName)
	beanstalkdTubeName       = "default"
)

type templateData struct {
	TestNamespace            string
	BeanstalkdDeploymentName string
	BeanstalkdPutJobName     string
	ScaledObjectName         string
	DeploymentName           string
	BeanstalkdTubeName       string
}

const (
	beanstalkdDeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
    labels:
      app: beanstalkd
    name: {{.BeanstalkdDeploymentName}}
    namespace: {{.TestNamespace}}
spec:
  selector:
    matchLabels:
      app: beanstalkd
  template:
    metadata:
      labels:
        app: beanstalkd
    spec:
      containers:
        - image: docker.io/schickling/beanstalkd
          name: beanstalkd
          ports:
            - containerPort: 11300
              name: beanstalkd
          readinessProbe:
            tcpSocket:
              port: 11300
            initialDelaySeconds: 5
            periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: beanstalkd
  namespace: {{.TestNamespace}}
spec:
  ports:
    - name: beanstalkd
      port: 11300
      targetPort: 11300
  selector:
    app: beanstalkd
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
  maxReplicaCount: 3
  triggers:
  - type: beanstalkd
    metadata:
      server: beanstalkd.{{.TestNamespace}}:11300
      value: "5"
      activationValue: "15"
      tube: {{.BeanstalkdTubeName}}
`

	beanstalkdPutJobsTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  name: {{.BeanstalkdPutJobName}}
  namespace: {{.TestNamespace}}
spec:
  template:
    spec:
      containers:
      - name: beanstalkd-put-job
        image: docker.io/sitecrafting/beanstalkd-cli
        command: ["/bin/sh"]
        args: ["-c", "for run in $(seq 1 10); do beanstalkd-cli --host=beanstalkd put \"Test Job\"; done;"]
      restartPolicy: OnFailure
`

	scaledObjectDelayedTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  maxReplicaCount: 5
  minReplicaCount: 1
  triggers:
  - type: beanstalkd
    metadata:
      server: beanstalkd.{{.TestNamespace}}:11300
      value: "5"
      tube: {{.BeanstalkdTubeName}}
      includeDelayed: "true"
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

func TestBeanstalkdScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, beanstalkdDeploymentName, testNamespace, 1, 60, 1),
		"replica count should be 0 after a minute")

	// Add beanstalkd jobs
	addBeanstalkdJobs(t, kc, &data)

	// test activation
	testActivation(t, kc, data)

	// test scaling
	testScale(t, kc, data)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:            testNamespace,
			ScaledObjectName:         scaledObjectName,
			DeploymentName:           deploymentName,
			BeanstalkdDeploymentName: beanstalkdDeploymentName,
			BeanstalkdTubeName:       beanstalkdTubeName,
			BeanstalkdPutJobName:     beanstalkdPutJobName,
		}, []Template{
			{Name: "beanstalkdDeploymentTemplate", Config: beanstalkdDeploymentTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
		}
}

func addBeanstalkdJobs(t *testing.T, kc *kubernetes.Clientset, data *templateData) {
	// run putJob
	KubectlReplaceWithTemplate(t, data, "beanstalkdPutJobsTemplate", beanstalkdPutJobsTemplate)
	require.True(t, WaitForJobSuccess(t, kc, beanstalkdPutJobName, testNamespace, 30, 2), "Job should run successfully")
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation---")

	KubectlApplyWithTemplate(t, data, "scaledObjectActivationTemplate", scaledObjectActivationTemplate)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 30)
}

func testScale(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scaling---")
	KubectlApplyWithTemplate(t, data, "scaledObjectDelayedTemplate", scaledObjectDelayedTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 2, 60, 1),
		"replica count should be 2 after a minute")
}
