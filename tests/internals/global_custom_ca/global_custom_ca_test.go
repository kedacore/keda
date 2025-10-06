//go:build e2e
// +build e2e

package global_custom_ca_test

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "global-custom-ca-test"
)

var (
	testNamespace               = fmt.Sprintf("%s-ns", testName)
	deploymentName              = fmt.Sprintf("%s-deployment", testName)
	metricsServerDeploymentName = fmt.Sprintf("%s-metrics-server", testName)
	servciceName                = fmt.Sprintf("%s-service", testName)
	triggerAuthName             = fmt.Sprintf("%s-ta", testName)
	scaledObjectName            = fmt.Sprintf("%s-so", testName)
	secretName                  = fmt.Sprintf("%s-secret", testName)
	metricsServerEndpoint       = fmt.Sprintf("http://%s.%s.svc.cluster.local:8080/api/value", servciceName, testNamespace)
	metricsServerHTTPSEndpoint  = fmt.Sprintf("https://%s.%s.svc.cluster.local:4333/api/value", servciceName, testNamespace)
	minReplicaCount             = 0
	maxReplicaCount             = 2
)

type templateData struct {
	TestNamespace               string
	DeploymentName              string
	MetricsServerDeploymentName string
	MetricsServerEndpoint       string
	MetricsServerHTTPSEndpoint  string
	ServciceName                string
	ScaledObjectName            string
	TriggerAuthName             string
	TLSCertificate              string
	TLSKey                      string
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

	tlsSecretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}-tls
  namespace: {{.TestNamespace}}
data:
  tls.crt: {{.TLSCertificate}}
  tls.key: {{.TLSKey}}
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
      volumes:
      - name: certificates
        secret:
          defaultMode: 420
          secretName: {{.SecretName}}-tls
      containers:
      - name: metrics
        image: ghcr.io/kedacore/tests-metrics-api
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 4333
          name: https
        envFrom:
        - secretRef:
            name: {{.SecretName}}
        env:
        - name: USE_TLS
          value: "true"
        volumeMounts:
        - mountPath: /certs
          name: certificates
          readOnly: true
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
    name: http
  - port: 4333
    targetPort: 4333
    name: https
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
        image: nginx:1.14.2
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
      url: "{{.MetricsServerHTTPSEndpoint}}"
      valueLocation: 'value'
      authMode: "basic"
      method: "query"
    authenticationRef:
      name: {{.TriggerAuthName}}
`
	updateMetricTemplate = `apiVersion: batch/v1
kind: Job
metadata:
  name: update-metric-value
  namespace: {{.TestNamespace}}
spec:
  ttlSecondsAfterFinished: 0
  template:
    spec:
      containers:
      - name: curl-client
        image: docker.io/curlimages/curl
        imagePullPolicy: Always
        command: ["curl", "-X", "POST", "{{.MetricsServerEndpoint}}/{{.MetricValue}}"]
      restartPolicy: Never`
)

func TestCustomCa(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData(t)
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 180, 3),
		"replica count should be %d after 9 minutes", minReplicaCount)

	// test scaling
	testScaleOut(t, kc, data)
	testScaleIn(t, kc, data)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	data.MetricValue = 50
	KubectlReplaceWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale in ---")
	data.MetricValue = 0
	KubectlReplaceWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
}

func getTemplateData(t *testing.T) (templateData, []Template) {
	tlsCrt, TLSKey := GenerateServerCert(t, fmt.Sprintf("%s.%s.svc.cluster.local", servciceName, testNamespace))
	return templateData{
			TestNamespace:               testNamespace,
			DeploymentName:              deploymentName,
			MetricsServerDeploymentName: metricsServerDeploymentName,
			ServciceName:                servciceName,
			TriggerAuthName:             triggerAuthName,
			ScaledObjectName:            scaledObjectName,
			SecretName:                  secretName,
			MetricsServerEndpoint:       metricsServerEndpoint,
			MetricsServerHTTPSEndpoint:  metricsServerHTTPSEndpoint,
			MinReplicaCount:             fmt.Sprintf("%v", minReplicaCount),
			MaxReplicaCount:             fmt.Sprintf("%v", maxReplicaCount),
			TLSCertificate:              base64.StdEncoding.EncodeToString([]byte(tlsCrt)),
			TLSKey:                      base64.StdEncoding.EncodeToString([]byte(TLSKey)),
			MetricValue:                 0,
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "metricsServerdeploymentTemplate", Config: metricsServerdeploymentTemplate},
			{Name: "serviceTemplate", Config: serviceTemplate},
			{Name: "tlsSecretTemplate", Config: tlsSecretTemplate},
			{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}
