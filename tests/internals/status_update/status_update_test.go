//go:build e2e
// +build e2e

package status_update_test

import (
	"fmt"
	"testing"

	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName = "status-update-test"
)

var (
	testNamespace               = fmt.Sprintf("%s-ns", testName)
	deploymentName              = fmt.Sprintf("%s-deployment", testName)
	metricsServerDeploymentName = fmt.Sprintf("%s-metrics-server", testName)
	servciceName                = fmt.Sprintf("%s-service", testName)
	triggerAuthName             = fmt.Sprintf("%s-ta", testName)
	scaledObjectName            = fmt.Sprintf("%s-so", testName)
	scaledJobName               = fmt.Sprintf("%s-sj", testName)
	secretName                  = fmt.Sprintf("%s-secret", testName)
	metricsServerEndpoint       = fmt.Sprintf("http://%s.%s.svc.cluster.local:8080/api/value", servciceName, testNamespace)
	minReplicaCount             = 0
	maxReplicaCount             = 2
)

type templateData struct {
	TestNamespace               string
	DeploymentName              string
	MetricsServerDeploymentName string
	MetricsServerEndpoint       string
	ServciceName                string
	ScaledObjectName            string
	ScaledJobName               string
	TriggerAuthName             string
	SecretName                  string
	MetricValue                 int
	MinReplicaCount             string
	MaxReplicaCount             string
}

const (
	secretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  AUTH_PASSWORD: U0VDUkVUCg==
  AUTH_USERNAME: VVNFUgo=
`

	triggerAuthenticationTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthName}}
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
    - parameter: username
      name: {{.SecretName}}
      key: AUTH_USERNAME
    - parameter: password
      name: {{.SecretName}}
      key: AUTH_PASSWORD
`

	metricsServerdeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.MetricsServerDeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.MetricsServerDeploymentName}}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: {{.MetricsServerDeploymentName}}
  template:
    metadata:
      labels:
        app: {{.MetricsServerDeploymentName}}
    spec:
      containers:
      - name: metrics
        image: ghcr.io/kedacore/tests-metrics-api
        ports:
        - containerPort: 8080
        envFrom:
        - secretRef:
            name: {{.SecretName}}
        imagePullPolicy: Always
`

	serviceTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: {{.ServciceName}}
  namespace: {{.TestNamespace}}
spec:
  selector:
    app: {{.MetricsServerDeploymentName}}
  ports:
  - port: 8080
    targetPort: 8080
`

	deploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: {{.DeploymentName}}
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
spec:
  selector:
    matchLabels:
      app: {{.DeploymentName}}
  replicas: 0
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

	scaledObjectTemplate = `
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
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  cooldownPeriod:  1
  triggers:
  - type: metrics-api
    metadata:
      targetValue: "5"
      activationTargetValue: "20"
      url: "{{.MetricsServerEndpoint}}"
      valueLocation: 'value'
      authMode: "basic"
      method: "query"
    authenticationRef:
      name: {{.TriggerAuthName}}
  - type: corn
    metadata:
      timezone: Asia/Kolkata
      start: 0 6 * * *
      end: 0 8 * * *
      desiredReplicas: "9"
  - type: corn
    metadata:
      timezone: Asia/Kolkata
      start: 0 22 * * *
      end: 0 23 * * *
      desiredReplicas: "9"`

	scaledJobTemplate = `
      apiVersion: keda.sh/v1alpha1
      kind: ScaledJob
      metadata:
        name: {{.ScaledJobName}}
        namespace: {{.TestNamespace}}
      spec:
        jobTargetRef:
          template:
            spec:
              containers:
                - name: external-executor
                  image: busybox
                  command:
                  - sleep
                  - "30"
                  imagePullPolicy: IfNotPresent
              restartPolicy: Never
          backoffLimit: 1
        pollingInterval: 5
        minReplicaCount: {{.MinReplicaCount}}
        maxReplicaCount: {{.MaxReplicaCount}}
        successfulJobsHistoryLimit: 0
        failedJobsHistoryLimit: 0
        triggers:
        - type: metrics-api
          metadata:
            targetValue: "5"
            activationTargetValue: "20"
            url: "{{.MetricsServerEndpoint}}"
            valueLocation: 'value'
            authMode: "basic"
            method: "query"
          authenticationRef:
            name: {{.TriggerAuthName}}
        - type: corn
          metadata:
            timezone: Asia/Kolkata
            start: 0 6 * * *
            end: 0 8 * * *
            desiredReplicas: "9"
        - type: corn
          metadata:
            timezone: Asia/Kolkata
            start: 0 22 * * *
            end: 0 23 * * *
            desiredReplicas: "9"`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	// test
	testTriggersAndAuthenticationsTypes(t)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func testTriggersAndAuthenticationsTypes(t *testing.T) {
	otherparameter := `-o jsonpath="{.status.triggersTypes}"`
	CheckKubectlGetResult(t, "ScaledObject", scaledObjectName, testNamespace, otherparameter, "metrics-api,corn")
	otherparameter = `-o jsonpath="{.status.authenticationsTypes}"`
	CheckKubectlGetResult(t, "ScaledObject", scaledObjectName, testNamespace, otherparameter, triggerAuthName)
	otherparameter = `-o jsonpath="{.status.triggersTypes}"`
	CheckKubectlGetResult(t, "ScaledJob", scaledJobName, testNamespace, otherparameter, "metrics-api,corn")
	otherparameter = `-o jsonpath="{.status.authenticationsTypes}"`
	CheckKubectlGetResult(t, "ScaledJob", scaledJobName, testNamespace, otherparameter, triggerAuthName)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:               testNamespace,
			DeploymentName:              deploymentName,
			MetricsServerDeploymentName: metricsServerDeploymentName,
			ServciceName:                servciceName,
			TriggerAuthName:             triggerAuthName,
			ScaledObjectName:            scaledObjectName,
			ScaledJobName:               scaledJobName,
			SecretName:                  secretName,
			MetricsServerEndpoint:       metricsServerEndpoint,
			MinReplicaCount:             fmt.Sprintf("%v", minReplicaCount),
			MaxReplicaCount:             fmt.Sprintf("%v", maxReplicaCount),
			MetricValue:                 0,
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "metricsServerdeploymentTemplate", Config: metricsServerdeploymentTemplate},
			{Name: "serviceTemplate", Config: serviceTemplate},
			{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
			{Name: "scaledJobTemplate", Config: scaledJobTemplate},
		}
}
