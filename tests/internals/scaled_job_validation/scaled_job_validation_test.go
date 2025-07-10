//go:build e2e
// +build e2e

package cache_metrics_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName = "scaled-job-validation-test"
)

var (
	testNamespace           = fmt.Sprintf("%s-ns", testName)
	emptyTriggersSjName     = fmt.Sprintf("%s-sj-empty-triggers", testName)
	excludedLabelsSjName    = fmt.Sprintf("%s-sj-excluded-labels", testName)
	monitoredDeploymentName = fmt.Sprintf("%s-monitored-deployment", testName)
)

type templateData struct {
	TestNamespace           string
	EmptyTriggersSjName     string
	ExcludedLabelsSjName    string
	MonitoredDeploymentName string
}

const (
	emptyTriggersSjTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledJob
metadata:
  name: {{.EmptyTriggersSjName}}
  namespace: {{.TestNamespace}}
spec:
  jobTargetRef:
    template:
      spec:
        containers:
        - name: demo-rabbitmq-client
          image: demo-rabbitmq-client:1
          imagePullPolicy: Always
          command: ["receive",  "amqp://user:PASSWORD@rabbitmq.default.svc.cluster.local:5672"]
          envFrom:
            - secretRef:
                name: rabbitmq-consumer-secrets
        restartPolicy: Never
  triggers: []
`

	monitoredDeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.MonitoredDeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.MonitoredDeploymentName}}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: {{.MonitoredDeploymentName}}
  template:
    metadata:
      labels:
        app: {{.MonitoredDeploymentName}}
    spec:
      containers:
        - name: nginx
          image: 'ghcr.io/nginx/nginx-unprivileged:1.26'
`

	scaledJobTemplateWithExcludedLabels = `
apiVersion: keda.sh/v1alpha1
kind: ScaledJob
metadata:
  name: {{.ExcludedLabelsSjName}}
  namespace: {{.TestNamespace}}
  annotations:
    scaledjob.keda.sh/job-excluded-labels: "foo.bar/environment,foo.bar/version"
  labels:
    team: backend
    foo.bar/environment: bf5011472247b67cce3ee7b24c9a08c5
    foo.bar/version: "1"
spec:
  jobTargetRef:
    template:
      spec:
        containers:
        - name: external-executor
          image: busybox
          command: ["sleep", "60"]
          imagePullPolicy: IfNotPresent
        restartPolicy: Never
  triggers:
    - type: kubernetes-workload
      metadata:
        podSelector: 'app={{.MonitoredDeploymentName}}'
        value: '1'
`
)

func TestScaledJobValidations(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	testTriggersWithEmptyArray(t, data)

	testScaledJobWithExcludedLabels(t, data)

	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func testTriggersWithEmptyArray(t *testing.T, data templateData) {
	t.Log("--- triggers with empty array ---")

	err := KubectlApplyWithErrors(t, data, "emptyTriggersSjTemplate", emptyTriggersSjTemplate)
	assert.Errorf(t, err, "can deploy the scaledJob - %s", err)
	assert.Contains(t, err.Error(), "no triggers defined in the ScaledObject/ScaledJob")
}

func testScaledJobWithExcludedLabels(t *testing.T, data templateData) {
	t.Log("--- scaled job with excluded labels ---")

	err := KubectlApplyWithErrors(t, data, "monitoredDeploymentTemplate", monitoredDeploymentTemplate)
	assert.NoError(t, err, "monitoredDeployment should be deployed")

	err = KubectlApplyWithErrors(t, data, "scaledJobTemplateWithExcludedLabels", scaledJobTemplateWithExcludedLabels)
	assert.NoError(t, err, "scaledJob should be deployed")

	job, err := WaitForJobCreation(t, GetKubernetesClient(t), data.ExcludedLabelsSjName, data.TestNamespace, 10, 5)
	assert.NoError(t, err, "job should be created")
	assert.NotNil(t, job, "job should be created")

	// Ensure that foo.bar/environment and foo.bar/version labels are not propagated
	assert.Equal(t, "", job.Labels["foo.bar/environment"], "job should not have the 'foo.bar/environment' label")
	assert.Equal(t, "", job.Labels["foo.bar/version"], "job should not have the 'foo.bar/version' label")

	KubectlDeleteWithTemplate(t, data, "scaledJobTemplateWithExcludedLabels", scaledJobTemplateWithExcludedLabels)
	KubectlDeleteWithTemplate(t, data, "monitoredDeploymentTemplate", monitoredDeploymentTemplate)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
		TestNamespace:           testNamespace,
		EmptyTriggersSjName:     emptyTriggersSjName,
		ExcludedLabelsSjName:    excludedLabelsSjName,
		MonitoredDeploymentName: monitoredDeploymentName,
	}, []Template{}
}
