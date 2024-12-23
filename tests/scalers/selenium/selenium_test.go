//go:build e2e
// +build e2e

package selenium_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "selenium-test"
)

var (
	testNamespace              = fmt.Sprintf("%s-ns", testName)
	secretName                 = fmt.Sprintf("%s-secret", testName)
	triggerAuthName            = fmt.Sprintf("%s-trigger-auth", testName)
	chromeDeploymentName       = fmt.Sprintf("%s-chrome", testName)
	firefoxDeploymentName      = fmt.Sprintf("%s-firefox", testName)
	edgeDeploymentName         = fmt.Sprintf("%s-edge", testName)
	hubDeploymentName          = fmt.Sprintf("%s-hub", testName)
	scaledObjectName           = fmt.Sprintf("%s-so", testName)
	hubHost                    = fmt.Sprintf("%s:%s@selenium-hub.%s", hubBasicAuthUsername, hubBasicAuthPassword, testNamespace)
	hubPort                    = 4444
	hubGraphURL                = fmt.Sprintf("http://selenium-hub.%s:%d/graphql", testNamespace, hubPort)
	hubBasicAuthUsername       = "admin"
	hubBasicAuthPassword       = "admin"
	hubBasicAuthUsernameB64enc = "YWRtaW4="
	hubBasicAuthPasswordB64enc = "YWRtaW4="
	hubBasicAuthHeader         = "YWRtaW46YWRtaW4="
	minReplicaCount            = 0
	maxReplicaCount            = 1
)

type templateData struct {
	TestNamespace              string
	SecretName                 string
	TriggerAuthName            string
	ChromeDeploymentName       string
	FirefoxDeploymentName      string
	EdgeDeploymentName         string
	HubDeploymentName          string
	HubHost                    string
	HubPort                    int
	HubGraphURL                string
	HubBasicAuthUsername       string
	HubBasicAuthPassword       string
	HubBasicAuthUsernameB64enc string
	HubBasicAuthPasswordB64enc string
	HubBasicAuthHeader         string
	WithVersion                bool
	JobName                    string
	ScaledObjectName           string
	MinReplicaCount            int
	MaxReplicaCount            int
}

const (
	eventBusConfigTemplate = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: selenium-event-bus-config
  namespace: {{.TestNamespace}}
  labels:
    app.kubernetes.io/managed-by: helm
    app.kubernetes.io/instance: selenium-hpa
    app.kubernetes.io/version: latest
    app.kubernetes.io/component: latest
    helm.sh/chart: latest
data:
  SE_EVENT_BUS_HOST: selenium-hub
  SE_EVENT_BUS_PUBLISH_PORT: "4442"
  SE_EVENT_BUS_SUBSCRIBE_PORT: "4443"
`

	chromeNodeServiceTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: selenium-chrome-node
  namespace: {{.TestNamespace}}
  labels:
    name: selenium-chrome-node
    app.kubernetes.io/managed-by: helm
    app.kubernetes.io/instance: selenium-hpa
    app.kubernetes.io/version: latest
    app.kubernetes.io/component: latest
    helm.sh/chart: latest
spec:
  type: ClusterIP
  selector:
    app: selenium-chrome-node
  ports:
  - name: tcp-chrome
    protocol: TCP
    port: 6900
    targetPort: 5900
`

	chromeNodeDeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.ChromeDeploymentName}}
  namespace: {{.TestNamespace}}
  labels: &chrome_node_labels
    app: selenium-chrome-node
    app.kubernetes.io/name: selenium-chrome-node
    app.kubernetes.io/managed-by: helm
    app.kubernetes.io/instance: selenium-hpa
    app.kubernetes.io/version: latest
    app.kubernetes.io/component: latest
    helm.sh/chart: latest
spec:
  replicas: 0
  selector:
    matchLabels:
      app: selenium-chrome-node
  template:
    metadata:
      labels: *chrome_node_labels
      annotations:
        checksum/event-bus-configmap: 0e5e9d25a669359a37dd0d684c485f4c05729da5a26a841ad9a2743d99460f73
    spec:
      containers:
      - name: selenium-chrome-node
        image: selenium/node-chrome:nightly
        imagePullPolicy: IfNotPresent
        env:
        - name: SE_NODE_BROWSER_VERSION
          value: ''
        - name: SE_NODE_PLATFORM_NAME
          value: ''
        envFrom:
        - configMapRef:
            name: selenium-event-bus-config
        ports:
        - containerPort: 5553
          protocol: TCP
        volumeMounts:
        - name: dshm
          mountPath: /dev/shm
      volumes:
      - name: dshm
        emptyDir:
          medium: Memory
          sizeLimit: 1Gi
`

	secretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
type: Opaque
data:
  username: '{{.HubBasicAuthUsernameB64enc}}'
  password: '{{.HubBasicAuthPasswordB64enc}}'
`

	scaledTriggerAuthTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthName}}
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

	chromeScaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: chrome-{{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  maxReplicaCount: 1
  pollingInterval: 5
  cooldownPeriod:  5
  scaleTargetRef:
    name: {{.ChromeDeploymentName}}
  triggers:
  - type: selenium-grid
    metadata:
      url: '{{.HubGraphURL}}'
      browserName: 'chrome'
      platformName: ''
      activationThreshold: '1'
    authenticationRef:
      name: '{{.TriggerAuthName}}'
`

	firefoxNodeServiceTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: selenium-firefox-node
  namespace: {{.TestNamespace}}
  labels:
    name: selenium-firefox-node
    app.kubernetes.io/managed-by: helm
    app.kubernetes.io/instance: selenium-hpa
    app.kubernetes.io/version: latest
    app.kubernetes.io/component: latest
    helm.sh/chart: latest
spec:
  type: ClusterIP
  selector:
    app: selenium-firefox-node
  ports:
  - name: tcp-firefox
    protocol: TCP
    port: 6900
    targetPort: 5900`

	firefoxNodeDeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.FirefoxDeploymentName}}
  namespace: {{.TestNamespace}}
  labels: &firefox_node_labels
    app: selenium-firefox-node
    app.kubernetes.io/name: selenium-firefox-node
    app.kubernetes.io/managed-by: helm
    app.kubernetes.io/instance: selenium-hpa
    app.kubernetes.io/version: latest
    app.kubernetes.io/component: latest
    helm.sh/chart: latest
spec:
  replicas: 0
  selector:
    matchLabels:
      app: selenium-firefox-node
  template:
    metadata:
      labels: *firefox_node_labels
      annotations:
        checksum/event-bus-configmap: 0e5e9d25a669359a37dd0d684c485f4c05729da5a26a841ad9a2743d99460f73
    spec:
      containers:
      - name: selenium-firefox-node
        image: selenium/node-firefox:nightly
        imagePullPolicy: IfNotPresent
        env:
        - name: SE_NODE_BROWSER_VERSION
          value: ''
        - name: SE_NODE_PLATFORM_NAME
          value: ''
        envFrom:
        - configMapRef:
            name: selenium-event-bus-config
        ports:
        - containerPort: 5553
          protocol: TCP
        volumeMounts:
        - name: dshm
          mountPath: /dev/shm
      volumes:
      - name: dshm
        emptyDir:
          medium: Memory
          sizeLimit: 1Gi
`

	firefoxScaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: firefox-{{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  maxReplicaCount: 1
  pollingInterval: 5
  cooldownPeriod:  5
  scaleTargetRef:
    name: {{.FirefoxDeploymentName}}
  triggers:
    - type: selenium-grid
      metadata:
        url: '{{.HubGraphURL}}'
        browserName: 'firefox'
        platformName: ''
        activationThreshold: '1'
      authenticationRef:
        name: '{{.TriggerAuthName}}'
`

	edgeNodeServiceTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: selenium-edge-node
  namespace: {{.TestNamespace}}
  labels:
    name: selenium-edge-node
    app.kubernetes.io/managed-by: helm
    app.kubernetes.io/instance: selenium-hpa
    app.kubernetes.io/version: latest
    app.kubernetes.io/component: latest
    helm.sh/chart: latest
spec:
  type: ClusterIP
  selector:
    app: selenium-edge-node
  ports:
  - name: tcp-edge
    protocol: TCP
    port: 6900
    targetPort: 5900
`

	edgeNodeDeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.EdgeDeploymentName}}
  namespace: {{.TestNamespace}}
  labels: &edge_node_labels
    app: selenium-edge-node
    app.kubernetes.io/name: selenium-edge-node
    app.kubernetes.io/managed-by: helm
    app.kubernetes.io/instance: selenium-hpa
    app.kubernetes.io/version: latest
    app.kubernetes.io/component: latest
    helm.sh/chart: latest
spec:
  replicas: 0
  selector:
    matchLabels:
      app: selenium-edge-node
  template:
    metadata:
      labels: *edge_node_labels
      annotations:
        checksum/event-bus-configmap: 0e5e9d25a669359a37dd0d684c485f4c05729da5a26a841ad9a2743d99460f73
    spec:
      containers:
      - name: selenium-edge-node
        image: selenium/node-edge:nightly
        imagePullPolicy: IfNotPresent
        env:
        - name: SE_NODE_BROWSER_VERSION
          value: ''
        - name: SE_NODE_PLATFORM_NAME
          value: ''
        envFrom:
        - configMapRef:
            name: selenium-event-bus-config
        ports:
        - containerPort: 5553
          protocol: TCP
        volumeMounts:
        - name: dshm
          mountPath: /dev/shm
      volumes:
      - name: dshm
        emptyDir:
          medium: Memory
          sizeLimit: 1Gi
`

	edgeScaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: edge-{{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  maxReplicaCount: 1
  pollingInterval: 5
  cooldownPeriod:  5
  scaleTargetRef:
    name: {{.EdgeDeploymentName}}
  triggers:
  - type: selenium-grid
    metadata:
      url: '{{.HubGraphURL}}'
      browserName: 'MicrosoftEdge'
      sessionBrowserName: 'msedge'
      platformName: ''
      activationThreshold: '1'
    authenticationRef:
      name: '{{.TriggerAuthName}}'
`

	hubServiceTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: selenium-hub
  namespace: {{.TestNamespace}}
  labels:
    app: selenium-hub
    app.kubernetes.io/managed-by: helm
    app.kubernetes.io/instance: selenium-hpa
    app.kubernetes.io/version: latest
    app.kubernetes.io/component: latest
    helm.sh/chart: latest
spec:
  selector:
    app: selenium-hub
  type: NodePort
  ports:
    - name: http-hub
      protocol: TCP
      port: 4444
      targetPort: 4444
    - name: tcp-hub-pub
      protocol: TCP
      port: 4442
      targetPort: 4442
    - name: tcp-hub-sub
      protocol: TCP
      port: 4443
      targetPort: 4443
`

	hubDeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.HubDeploymentName}}
  namespace: {{.TestNamespace}}
  labels: &hub_labels
    app: selenium-hub
    app.kubernetes.io/name: selenium-hub
    app.kubernetes.io/managed-by: helm
    app.kubernetes.io/instance: selenium-hpa
    app.kubernetes.io/version: latest
    app.kubernetes.io/component: latest
    helm.sh/chart: latest
spec:
  replicas: 1
  selector:
    matchLabels:
      app: selenium-hub
  template:
    metadata:
      labels: *hub_labels
    spec:
      containers:
      - name: selenium-hub
        image: selenium/hub:latest
        imagePullPolicy: IfNotPresent
        env:
        - name: SE_ROUTER_USERNAME
          value: '{{.HubBasicAuthUsername}}'
        - name: SE_ROUTER_PASSWORD
          value: '{{.HubBasicAuthPassword}}'
        ports:
        - containerPort: 4444
          protocol: TCP
        - containerPort: 4442
          protocol: TCP
        - containerPort: 4443
          protocol: TCP
        livenessProbe:
          httpGet:
            path: /wd/hub/status
            port: 4444
            httpHeaders:
              - name: Authorization
                value: Basic {{.HubBasicAuthHeader}}
          initialDelaySeconds: 10
          periodSeconds: 10
          timeoutSeconds: 10
          successThreshold: 1
          failureThreshold: 10
        readinessProbe:
          httpGet:
            path: /wd/hub/status
            port: 4444
            httpHeaders:
              - name: Authorization
                value: Basic {{.HubBasicAuthHeader}}
          initialDelaySeconds: 12
          periodSeconds: 10
          timeoutSeconds: 10
          successThreshold: 1
          failureThreshold: 10
`

	jobTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  labels:
    app: {{.JobName}}
  name: {{.JobName}}
  namespace: {{.TestNamespace}}
spec:
  ttlSecondsAfterFinished: 0
  template:
    metadata:
      labels:
        app: {{.JobName}}
    spec:
      containers:
      - name: selenium-random-tests
        image: ghcr.io/kedacore/tests-selenium-grid
        imagePullPolicy: Always
        env:
        - name: HOST_NAME
          value: "{{.HubHost}}"
        - name: PORT
          value: "{{.HubPort}}"
        - name: WITH_VERSION
          value: "{{.WithVersion}}"
      restartPolicy: Never
`
)

func TestSeleniumScaler(t *testing.T) {
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})

	// Create kubernetes resources
	CreateKubernetesResources(t, kc, testNamespace, data, templates)
	require.True(t, WaitForDeploymentReplicaReadyCount(t, kc, hubDeploymentName, testNamespace, 1, 60, 1),
		"replica count should be 1 after 1 minute")
	require.True(t, WaitForDeploymentReplicaReadyCount(t, kc, chromeDeploymentName, testNamespace, minReplicaCount, 60, 1),
		"replica count should be 0 after 1 minute")
	require.True(t, WaitForDeploymentReplicaReadyCount(t, kc, firefoxDeploymentName, testNamespace, minReplicaCount, 60, 1),
		"replica count should be 0 after 1 minute")
	require.True(t, WaitForDeploymentReplicaReadyCount(t, kc, edgeDeploymentName, testNamespace, minReplicaCount, 60, 1),
		"replica count should be 0 after 1 minute")

	testActivation(t, kc, data)
	testScaleOut(t, kc, data)
	testScaleIn(t, kc)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")
	data.JobName = "activation"
	data.WithVersion = false
	KubectlApplyWithTemplate(t, data, "jobTemplate", jobTemplate)

	// Instead of waiting a minute with every one, we sleep the time and check them later
	time.Sleep(time.Second * 60)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, chromeDeploymentName, testNamespace, minReplicaCount, 5)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, firefoxDeploymentName, testNamespace, minReplicaCount, 5)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, edgeDeploymentName, testNamespace, minReplicaCount, 5)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")

	data.JobName = "scaleup"
	data.WithVersion = false
	KubectlApplyWithTemplate(t, data, "jobTemplate", jobTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, chromeDeploymentName, testNamespace, maxReplicaCount, 60, 1),
		"replica count should be %s after 1 minute", maxReplicaCount)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, firefoxDeploymentName, testNamespace, maxReplicaCount, 60, 1),
		"replica count should be %s after 1 minute", maxReplicaCount)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, edgeDeploymentName, testNamespace, maxReplicaCount, 60, 1),
		"replica count should be %s after 1 minute", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, chromeDeploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %s after 3 minutes", minReplicaCount)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, firefoxDeploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %s after 3 minutes", minReplicaCount)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, edgeDeploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %s after 3 minutes", minReplicaCount)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:              testNamespace,
			SecretName:                 secretName,
			TriggerAuthName:            triggerAuthName,
			ChromeDeploymentName:       chromeDeploymentName,
			FirefoxDeploymentName:      firefoxDeploymentName,
			EdgeDeploymentName:         edgeDeploymentName,
			HubDeploymentName:          hubDeploymentName,
			HubHost:                    hubHost,
			HubPort:                    hubPort,
			HubGraphURL:                hubGraphURL,
			HubBasicAuthUsername:       hubBasicAuthUsername,
			HubBasicAuthPassword:       hubBasicAuthPassword,
			HubBasicAuthUsernameB64enc: hubBasicAuthUsernameB64enc,
			HubBasicAuthPasswordB64enc: hubBasicAuthPasswordB64enc,
			HubBasicAuthHeader:         hubBasicAuthHeader,
			ScaledObjectName:           scaledObjectName,
			MinReplicaCount:            minReplicaCount,
			MaxReplicaCount:            maxReplicaCount,
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "scaledTriggerAuthTemplate", Config: scaledTriggerAuthTemplate},
			{Name: "eventBusConfigTemplate", Config: eventBusConfigTemplate},
			{Name: "hubDeploymentTemplate", Config: hubDeploymentTemplate},
			{Name: "hubServiceTemplate", Config: hubServiceTemplate},
			{Name: "chromeNodeServiceTemplate", Config: chromeNodeServiceTemplate},
			{Name: "chromeNodeDeploymentTemplate", Config: chromeNodeDeploymentTemplate},
			{Name: "chromeScaledObjectTemplate", Config: chromeScaledObjectTemplate},
			{Name: "firefoxNodeServiceTemplate", Config: firefoxNodeServiceTemplate},
			{Name: "firefoxNodeDeploymentTemplate", Config: firefoxNodeDeploymentTemplate},
			{Name: "firefoxScaledObjectTemplate", Config: firefoxScaledObjectTemplate},
			{Name: "edgeNodeServiceTemplate", Config: edgeNodeServiceTemplate},
			{Name: "edgeNodeDeploymentTemplate", Config: edgeNodeDeploymentTemplate},
			{Name: "edgeScaledObjectTemplate", Config: edgeScaledObjectTemplate},
		}
}
