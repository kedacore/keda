//go:build e2e
// +build e2e

package splunk_test

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName = "splunk-test"
)

var (
	testNamespace          = fmt.Sprintf("%s-ns", testName)
	configMapName          = fmt.Sprintf("%s-configmap", testName)
	deploymentName         = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName       = fmt.Sprintf("%s-so", testName)
	secretName             = fmt.Sprintf("%s-secret", testName)
	username               = "admin"
	password               = "password"
	savedSearchName        = "e2eSavedSearch"
	apiPort                = 8089
	maxReplicaCount        = 2
	minReplicaCount        = 0
	scaleInTargetValue     = "10"
	scaleInActivationValue = "15"
)

type templateData struct {
	TestNamespace        string
	ConfigMapName        string
	DeploymentName       string
	ScaledObjectName     string
	SecretName           string
	SplunkUsername       string
	SplunkUsernameBase64 string
	SplunkPassword       string
	SplunkPasswordBase64 string
	SavedSearchName      string
	APIPort              int
	MinReplicaCount      string
	MaxReplicaCount      string
	// Preconfigured saved search returns a static value of 10
	// so we need to change the scaled object values at different phases to test scale in + out
	TargetValue     string
	ActivationValue string
}

const (
	secretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  username: {{.SplunkUsernameBase64}}
  password: {{.SplunkPasswordBase64}}
`

	configMapTemplate = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{.ConfigMapName}}
  namespace: {{.TestNamespace}}
data:
  default.yml: |
    splunk:
      conf:
        - key: savedsearches
          value:
            directory: /opt/splunk/etc/users/admin/search/local
            content:
              {{.SavedSearchName}}:
                action.email.useNSSubject: 1
                action.webhook.enable_allowlist: 0
                alert.track: 0
                cron_schedule: '*/1 * * * *'
                dispatch.earliest_time: -15m
                dispatch.latest_time: now
                display.general.type: statistics
                display.page.search.tab: statistics
                display.visualizations.show: 0
                enableSched: 1
                request.ui_dispatch_app: search
                request.ui_dispatch_view: search
                search: index=_internal | tail | stats count
`

	triggerAuthenticationTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: keda-trigger-auth-splunk-secret
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
  - parameter: username
    name: {{.SecretName}}
    key: username
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
	splunkDeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: splunk
  namespace: {{.TestNamespace}}
spec:
  replicas: 1
  selector:
    matchLabels:
      name: splunk
  template:
    metadata:
      labels:
        name: splunk
    spec:
      containers:
        - name: splunk
          image: splunk/splunk:9.2
          imagePullPolicy: IfNotPresent
          env:
            - name: SPLUNK_START_ARGS
              value: --accept-license
            - name: SPLUNK_PASSWORD
              value: {{.SplunkPassword}}
          ports:
          - containerPort: {{.APIPort}}
            name: api
            protocol: TCP
          volumeMounts:
            - name: splunkconf-volume
              mountPath: /tmp/defaults
      volumes:
        - name: splunkconf-volume
          configMap:
            name: {{.ConfigMapName}}
`

	serviceTemplate = `apiVersion: v1
kind: Service
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
spec:
  type: ClusterIP
  ports:
  - name: api
    port: {{.APIPort}}
    targetPort: {{.APIPort}}
    protocol: TCP
  selector:
    name: splunk
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
  pollingInterval: 3
  cooldownPeriod:  1
  triggers:
    - type: splunk
      metadata:
        host: "https://{{.DeploymentName}}.{{.TestNamespace}}.svc:{{.APIPort}}"
        username:  {{.SplunkUsername}}
        unsafeSsl: "true"
        targetValue: "{{.TargetValue}}"
        activationValue: "{{.ActivationValue}}"
        savedSearchName: {{.SavedSearchName}}
        valueField: count
      authenticationRef:
        name: keda-trigger-auth-splunk-secret
`
)

func TestSplunkScaler(t *testing.T) {
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})

	// Create kubernetes resources
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	// Wait for splunk to start
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, "splunk", testNamespace, 1, 180, 3),
		"replica count should be %d after 3 minutes", 1)

	// Ensure nginx deployment is at min replica count
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 180, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)

	// test scaling
	testActivation(t, kc)
	testScaleOut(t, kc)
	testScaleIn(t, kc)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing activation ---")

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale out ---")

	// Saved Search returns 10, let's change the scaled object resource to force scaling out
	data := getScaledObjectTemplateData("1", "9")
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplateToScaleOut", scaledObjectTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")

	// Saved Search returns 10, let's change the scaled object resource to force scaling in
	data := getScaledObjectTemplateData(scaleInTargetValue, scaleInActivationValue)
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplateToScaleIn", scaledObjectTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:        testNamespace,
			ConfigMapName:        configMapName,
			DeploymentName:       deploymentName,
			ScaledObjectName:     scaledObjectName,
			SecretName:           secretName,
			SplunkUsername:       username,
			SplunkUsernameBase64: base64.StdEncoding.EncodeToString([]byte(username)),
			SplunkPassword:       password,
			SplunkPasswordBase64: base64.StdEncoding.EncodeToString([]byte(password)),
			SavedSearchName:      savedSearchName,
			APIPort:              apiPort,
			MinReplicaCount:      fmt.Sprintf("%v", minReplicaCount),
			MaxReplicaCount:      fmt.Sprintf("%v", maxReplicaCount),
			// Ensure no scaling out since saved search returns 10 by default
			TargetValue:     scaleInTargetValue,
			ActivationValue: scaleInActivationValue,
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "configMapTemplate", Config: configMapTemplate},
			{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
			{Name: "serviceTemplate", Config: serviceTemplate},
			{Name: "splunkDeploymentTemplate", Config: splunkDeploymentTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

func getScaledObjectTemplateData(targetValue, activationValue string) templateData {
	return templateData{
		TestNamespace:    testNamespace,
		DeploymentName:   deploymentName,
		ScaledObjectName: scaledObjectName,
		SplunkUsername:   username,
		SavedSearchName:  savedSearchName,
		APIPort:          apiPort,
		MinReplicaCount:  fmt.Sprintf("%v", minReplicaCount),
		MaxReplicaCount:  fmt.Sprintf("%v", maxReplicaCount),
		TargetValue:      targetValue,
		ActivationValue:  activationValue,
	}
}
