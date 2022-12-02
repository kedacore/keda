//go:build e2e
// +build e2e

package datadog_test

import (
	"encoding/base64"
	"fmt"
	"os"
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
	testName = "datadog-test"
)

var (
	testNamespace           = fmt.Sprintf("%s-ns", testName)
	deploymentName          = fmt.Sprintf("%s-deployment", testName)
	monitoredDeploymentName = fmt.Sprintf("%s-monitored-deployment", testName)
	servciceName            = fmt.Sprintf("%s-service", testName)
	triggerAuthName         = fmt.Sprintf("%s-ta", testName)
	scaledObjectName        = fmt.Sprintf("%s-so", testName)
	secretName              = fmt.Sprintf("%s-secret", testName)
	configName              = fmt.Sprintf("%s-config", testName)
	datadogAPIKey           = os.Getenv("DATADOG_API_KEY")
	datadogAppKey           = os.Getenv("DATADOG_APP_KEY")
	datadogSite             = os.Getenv("DATADOG_SITE")
	datadogHelmRepo         = "https://helm.datadoghq.com"
	kuberneteClusterName    = "keda-datadog-cluster"
	minReplicaCount         = 0
	maxReplicaCount         = 2
)

type templateData struct {
	TestNamespace           string
	DeploymentName          string
	MonitoredDeploymentName string
	ServciceName            string
	ScaledObjectName        string
	TriggerAuthName         string
	SecretName              string
	ConfigName              string
	DatadogAPIKey           string
	DatadogAppKey           string
	DatadogSite             string
	KuberneteClusterName    string
	MinReplicaCount         string
	MaxReplicaCount         string
}

const (
	secretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  apiKey: {{.DatadogAPIKey}}
  appKey: {{.DatadogAppKey}}
  datadogSite: {{.DatadogSite}}
`

	triggerAuthenticationTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthName}}
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
  - parameter: apiKey
    name: {{.SecretName}}
    key: apiKey
  - parameter: appKey
    name: {{.SecretName}}
    key: appKey
  - parameter: datadogSite
    name: {{.SecretName}}
    key: datadogSite
`
	configTemplate = `apiVersion: v1
kind: ConfigMap
metadata:
  name: {{.ConfigName}}
  namespace: {{.TestNamespace}}
data:
  status.conf: |
    server {
      listen 81;
      location /nginx_status {
        stub_status on;
      }
    }
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
  replicas: 1
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
        image: nginx:1.14.2
        ports:
        - containerPort: 80
`
	monitoredDeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: nginx
  name: {{.MonitoredDeploymentName}}
  namespace: {{.TestNamespace}}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      creationTimestamp: null
      labels:
        app: nginx
      annotations:
        ad.datadoghq.com/nginx.check_names: '["nginx"]'
        ad.datadoghq.com/nginx.init_configs: '[{}]'
        ad.datadoghq.com/nginx.instances: |
          [
            {
              "nginx_status_url":"http://%%host%%:81/nginx_status/"
            }
          ]
    spec:
      containers:
      - image: nginx
        name: nginx
        ports:
        - containerPort: 80
        - containerPort: 81
        volumeMounts:
        - mountPath: /etc/nginx/conf.d/status.conf
          subPath: status.conf
          readOnly: true
          name: "config"
      volumes:
      - name: "config"
        configMap:
          name: {{.ConfigName}}
`
	serviceTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: {{.ServciceName}}
  namespace: {{.TestNamespace}}
spec:
    ports:
      - name: default
        port: 80
        protocol: TCP
        targetPort: 80
      - name: status
        port: 81
        protocol: TCP
        targetPort: 81
    selector:
        app: nginx
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
  pollingInterval: 1
  cooldownPeriod:  1
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 10
  triggers:
  - type: datadog
    metadata:
      query: "avg:nginx.net.request_per_s{cluster_name:{{.KuberneteClusterName}}}"
      queryValue: "2"
      activationQueryValue: "3"
      age: "120"
    metricType: "Value"
    authenticationRef:
      name: {{.TriggerAuthName}}
`
	lightLoadTemplate = `apiVersion: v1
kind: Pod
metadata:
  name: fake-light-traffic
  namespace: {{.TestNamespace}}
spec:
  containers:
  - image: busybox
    name: test
    command: ["/bin/sh"]
    args: ["-c", "while true; do wget -O /dev/null -o /dev/null http://{{.ServciceName}}/; sleep 0.5; done"]`

	heavyLoadTemplate = `apiVersion: v1
kind: Pod
metadata:
  name: fake-heavy-traffic
  namespace: {{.TestNamespace}}
spec:
  containers:
  - image: busybox
    name: test
    command: ["/bin/sh"]
    args: ["-c", "while true; do wget -O /dev/null -o /dev/null http://{{.ServciceName}}/; sleep 0.1; done"]`
)

func TestDatadogScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	require.NotEmpty(t, datadogAppKey, "DATADOG_APP_KEY env variable is required for datadog tests")
	require.NotEmpty(t, datadogAPIKey, "DATADOG_API_KEY env variable is required for datadog tests")
	require.NotEmpty(t, datadogSite, "DATADOG_SITE env variable is required for datadog tests")
	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	CreateKubernetesResources(t, kc, testNamespace, data, templates)
	installDatadog(t)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 180, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)

	// test scaling
	testActivation(t, kc, data)
	testScaleOut(t, kc, data)
	testScaleIn(t, kc, data)

	// cleanup
	DeleteKubernetesResources(t, kc, testNamespace, data, templates)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")
	KubectlApplyWithTemplate(t, data, "lightLoadTemplate", lightLoadTemplate)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	KubectlApplyWithTemplate(t, data, "heavyLoadTemplate", heavyLoadTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale in ---")
	KubectlDeleteWithTemplate(t, data, "lightLoadTemplate", lightLoadTemplate)
	KubectlDeleteWithTemplate(t, data, "heavyLoadTemplate", heavyLoadTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
}

func installDatadog(t *testing.T) {
	_, err := ExecuteCommand(fmt.Sprintf("helm repo add datadog %s", datadogHelmRepo))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
	_, err = ExecuteCommand("helm repo update")
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
	_, err = ExecuteCommand(fmt.Sprintf(`helm upgrade --install --set datadog.apiKey=%s --set datadog.appKey=%s --set datadog.site=%s --set datadog.clusterName=%s --set datadog.kubelet.tlsVerify=false --namespace %s --wait %s datadog/datadog`,
		datadogAPIKey,
		datadogAppKey,
		datadogSite,
		kuberneteClusterName,
		testNamespace,
		testName))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:           testNamespace,
			DeploymentName:          deploymentName,
			MonitoredDeploymentName: monitoredDeploymentName,
			ServciceName:            servciceName,
			TriggerAuthName:         triggerAuthName,
			ScaledObjectName:        scaledObjectName,
			SecretName:              secretName,
			ConfigName:              configName,
			DatadogAPIKey:           base64.StdEncoding.EncodeToString([]byte(datadogAPIKey)),
			DatadogAppKey:           base64.StdEncoding.EncodeToString([]byte(datadogAppKey)),
			DatadogSite:             base64.StdEncoding.EncodeToString([]byte(datadogSite)),
			KuberneteClusterName:    kuberneteClusterName,
			MinReplicaCount:         fmt.Sprintf("%v", minReplicaCount),
			MaxReplicaCount:         fmt.Sprintf("%v", maxReplicaCount),
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "configTemplate", Config: configTemplate},
			{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
			{Name: "serviceTemplate", Config: serviceTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "monitoredDeploymentTemplate", Config: monitoredDeploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}
