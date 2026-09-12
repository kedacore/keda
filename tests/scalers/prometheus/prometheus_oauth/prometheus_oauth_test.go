//go:build e2e
// +build e2e

package prometheus_oauth_test

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
	prometheus "github.com/kedacore/keda/v2/tests/scalers/prometheus"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../../.env")

const (
	testName = "prometheus-oauth-test"
)

var (
	testNamespace        = fmt.Sprintf("%s-ns", testName)
	deploymentName       = fmt.Sprintf("%s-deployment", testName)
	monitoredAppName     = fmt.Sprintf("%s-monitored-app", testName)
	scaledObjectName     = fmt.Sprintf("%s-so", testName)
	prometheusServerName = fmt.Sprintf("%s-server", testName)
	oauthProxyName       = fmt.Sprintf("%s-oauth-proxy", testName)
	enforcementJobName   = fmt.Sprintf("%s-oauth-proxy-enforcement", testName)
	triggerAuthName      = fmt.Sprintf("%s-ta", testName)
	triggerSecretName    = fmt.Sprintf("%s-ta-secret", testName)
	oauthClientID        = "keda-prometheus"
	oauthClientSecret    = "keda-prometheus-secret"
	oauthAccessToken     = "e2e-access-token"
	minReplicaCount      = 0
	maxReplicaCount      = 2
)

type templateData struct {
	TestNamespace           string
	DeploymentName          string
	MonitoredAppName        string
	ScaledObjectName        string
	PrometheusServerName    string
	OAuthProxyName          string
	EnforcementJobName      string
	TriggerAuthName         string
	TriggerSecretName       string
	OAuthClientID           string
	Base64ClientSecret      string
	Base64ClientCredentials string
	OAuthAccessToken        string
	MinReplicaCount         int
	MaxReplicaCount         int
}

const (
	// oauthProxyConfigMapTemplate serves two endpoints:
	// 1. A token endpoint handing out a static access token to the expected client credentials.
	// 2. A reverse proxy in front of Prometheus that only forwards requests carrying that access token.
	//
	// Together they verify that KEDA acquires a token with the configured credentials and presents it on the query request.
	oauthProxyConfigMapTemplate = `apiVersion: v1
kind: ConfigMap
metadata:
  name: {{.OAuthProxyName}}
  namespace: {{.TestNamespace}}
data:
  default.conf: |
    server {
      listen 8080;

      location = /healthz {
        access_log off;
        return 200;
      }

      location = /token {
        if ($request_method != POST) {
          return 405;
        }
        # Only the expected client credentials are issued a token, so a scaler that sends
        # the wrong ones cannot reach Prometheus. Requiring them in the Authorization header
        # also pins the client authentication style: golang.org/x/oauth2 prefers HTTP Basic
        # (RFC 6749 2.3.1) and only falls back to credentials in the request body once a
        # request has failed, so a fallback shows up as a failing test rather than passing
        # unnoticed.
        if ($http_authorization != "Basic {{.Base64ClientCredentials}}") {
          return 401;
        }
        default_type application/json;
        return 200 '{"access_token":"{{.OAuthAccessToken}}","token_type":"Bearer","expires_in":3600}';
      }

      location / {
        if ($http_authorization != "Bearer {{.OAuthAccessToken}}") {
          return 401;
        }
        proxy_pass http://{{.PrometheusServerName}}.{{.TestNamespace}}.svc;
      }
    }
`

	oauthProxyDeploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: {{.OAuthProxyName}}
  name: {{.OAuthProxyName}}
  namespace: {{.TestNamespace}}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: {{.OAuthProxyName}}
  template:
    metadata:
      labels:
        app: {{.OAuthProxyName}}
    spec:
      containers:
      - name: nginx
        image: ghcr.io/nginx/nginx-unprivileged:1.26
        ports:
        - containerPort: 8080
        volumeMounts:
        - name: config
          mountPath: /etc/nginx/conf.d
          readOnly: true
        readinessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 3
          periodSeconds: 3
        securityContext:
          allowPrivilegeEscalation: false
          runAsNonRoot: true
          capabilities:
            drop:
              - ALL
          seccompProfile:
            type: RuntimeDefault
      volumes:
      - name: config
        configMap:
          name: {{.OAuthProxyName}}
`

	oauthProxyServiceTemplate = `apiVersion: v1
kind: Service
metadata:
  labels:
    app: {{.OAuthProxyName}}
  name: {{.OAuthProxyName}}
  namespace: {{.TestNamespace}}
spec:
  ports:
  - name: http
    port: 8080
    protocol: TCP
    targetPort: 8080
  selector:
    app: {{.OAuthProxyName}}
`

	// oauthProxyEnforcementJobTemplate asserts that the proxy really is a gate before the scaler
	// is pointed at it. Without this, a proxy that forwarded everything, or a token endpoint that
	// issued tokens to anyone, would let the scaling assertions pass without any authentication
	// taking place. It also confirms the proxy is serving, so that the scaler's first poll is not
	// racing the proxy becoming reachable.
	oauthProxyEnforcementJobTemplate = `apiVersion: batch/v1
kind: Job
metadata:
  name: {{.EnforcementJobName}}
  namespace: {{.TestNamespace}}
spec:
  backoffLimit: 3
  template:
    spec:
      containers:
      - name: curl
        image: docker.io/curlimages/curl
        command:
        - sh
        - -e
        - -c
        - |
          proxy="http://{{.OAuthProxyName}}.{{.TestNamespace}}.svc:8080"
          query="$proxy/api/v1/query?query=up"

          status() {
            curl -s --max-time 10 -o /dev/null -w '%{http_code}' "$@"
          }

          # The token endpoint issues a token to the expected client credentials only.
          test "$(status -X POST -H 'Authorization: Basic {{.Base64ClientCredentials}}' "$proxy/token")" = 200
          test "$(status -X POST -H 'Authorization: Basic d3JvbmctY2xpZW50Ondyb25nLXNlY3JldA==' "$proxy/token")" = 401
          test "$(status -X POST "$proxy/token")" = 401

          # Prometheus is reachable with that access token only.
          test "$(status -H 'Authorization: Bearer {{.OAuthAccessToken}}' "$query")" = 200
          test "$(status -H 'Authorization: Bearer wrong-access-token' "$query")" = 401
          test "$(status "$query")" = 401
        securityContext:
          allowPrivilegeEscalation: false
          runAsNonRoot: true
          # The image declares its user by name, which the kubelet cannot verify as non-root,
          # so the uid behind that name is given explicitly.
          runAsUser: 100
          capabilities:
            drop:
              - ALL
          seccompProfile:
            type: RuntimeDefault
      restartPolicy: Never
`

	triggerSecretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.TriggerSecretName}}
  namespace: {{.TestNamespace}}
type: Opaque
data:
  clientSecret: {{.Base64ClientSecret}}
`

	triggerAuthTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthName}}
  namespace: {{.TestNamespace}}
spec:
  oauth2:
    type: clientCredentials
    clientId: {{.OAuthClientID}}
    clientSecret:
      valueFrom:
        secretKeyRef:
          name: {{.TriggerSecretName}}
          key: clientSecret
    tokenUrl: http://{{.OAuthProxyName}}.{{.TestNamespace}}.svc:8080/token
    scopes:
    - {{.OAuthClientID}}
`

	deploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: test-app
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
spec:
  replicas: 0
  selector:
    matchLabels:
      app: test-app
  template:
    metadata:
      labels:
        app: test-app
        type: keda-testing
    spec:
      containers:
      - name: prom-test-app
        image: ghcr.io/kedacore/tests-prometheus:latest
        imagePullPolicy: IfNotPresent
        securityContext:
          allowPrivilegeEscalation: false
          runAsNonRoot: true
          capabilities:
            drop:
              - ALL
          seccompProfile:
            type: RuntimeDefault
`

	monitoredAppDeploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: {{.MonitoredAppName}}
  name: {{.MonitoredAppName}}
  namespace: {{.TestNamespace}}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: {{.MonitoredAppName}}
  template:
    metadata:
      labels:
        app: {{.MonitoredAppName}}
        type: {{.MonitoredAppName}}
    spec:
      containers:
      - name: prom-test-app
        image: ghcr.io/kedacore/tests-prometheus:latest
        imagePullPolicy: IfNotPresent
        securityContext:
          allowPrivilegeEscalation: false
          runAsNonRoot: true
          capabilities:
            drop:
              - ALL
          seccompProfile:
            type: RuntimeDefault
`

	monitoredAppServiceTemplate = `apiVersion: v1
kind: Service
metadata:
  labels:
    app: {{.MonitoredAppName}}
  name: {{.MonitoredAppName}}
  namespace: {{.TestNamespace}}
  annotations:
    prometheus.io/scrape: "true"
spec:
  ports:
  - name: http
    port: 80
    protocol: TCP
    targetPort: 8080
  selector:
    type: {{.MonitoredAppName}}
`

	// The scaler reaches Prometheus through the token gated proxy, so scaling can only
	// happen once the OAuth client credentials flow has succeeded.
	scaledObjectTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  pollingInterval: 3
  cooldownPeriod:  1
  triggers:
  - type: prometheus
    metadata:
      serverAddress: http://{{.OAuthProxyName}}.{{.TestNamespace}}.svc:8080
      threshold: '20'
      activationThreshold: '20'
      query: sum(rate(http_requests_total{app="{{.MonitoredAppName}}"}[2m]))
      authModes: "oauth"
    authenticationRef:
      name: {{.TriggerAuthName}}
`

	generateLowLevelLoadJobTemplate = `apiVersion: batch/v1
kind: Job
metadata:
  name: generate-low-level-requests-job
  namespace: {{.TestNamespace}}
spec:
  template:
    spec:
      containers:
      - image: ghcr.io/kedacore/tests-hey
        name: test
        command: ["/bin/sh"]
        args: ["-c", "for i in $(seq 1 60);do echo $i;/hey -c 5 -n 30 http://{{.MonitoredAppName}}.{{.TestNamespace}}.svc;sleep 1;done"]
        securityContext:
          allowPrivilegeEscalation: false
          runAsNonRoot: true
          capabilities:
            drop:
              - ALL
          seccompProfile:
            type: RuntimeDefault
      restartPolicy: Never
  activeDeadlineSeconds: 100
  backoffLimit: 2
`

	generateLoadJobTemplate = `apiVersion: batch/v1
kind: Job
metadata:
  name: generate-requests-job
  namespace: {{.TestNamespace}}
spec:
  template:
    spec:
      containers:
      - image: ghcr.io/kedacore/tests-hey
        name: test
        command: ["/bin/sh"]
        args: ["-c", "for i in $(seq 1 60);do echo $i;/hey -c 5 -n 80 http://{{.MonitoredAppName}}.{{.TestNamespace}}.svc;sleep 1;done"]
        securityContext:
          allowPrivilegeEscalation: false
          runAsNonRoot: true
          capabilities:
            drop:
              - ALL
          seccompProfile:
            type: RuntimeDefault
      restartPolicy: Never
  activeDeadlineSeconds: 100
  backoffLimit: 2
`
)

// TestPrometheusOAuthScaler verifies that the Prometheus scaler authenticates with an OAuth2 client credentials TriggerAuthentication.
// Prometheus sits behind a proxy that rejects every request without a valid access token,
// so the scaler can only observe the metric, and therefore only scale, when the token has been acquired and presented.
func TestPrometheusOAuthScaler(t *testing.T) {
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	t.Cleanup(func() {
		prometheus.Uninstall(t, prometheusServerName, testNamespace, nil)
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})

	// Prometheus is installed first, so that its service can be resolved by the proxy.
	prometheus.Install(t, kc, prometheusServerName, testNamespace, nil)

	KubectlApplyMultipleWithTemplate(t, data, proxyTemplates())
	require.True(t, WaitForDeploymentReplicaReadyCount(t, kc, oauthProxyName, testNamespace, 1, 60, 3),
		"oauth proxy replica count should be 1 after 3 minutes")
	require.True(t, WaitForDeploymentReplicaReadyCount(t, kc, monitoredAppName, testNamespace, 1, 60, 3),
		"monitored app replica count should be 1 after 3 minutes")

	t.Log("--- verifying oauth proxy enforcement ---")
	KubectlApplyWithTemplate(t, data, "oauthProxyEnforcementJobTemplate", oauthProxyEnforcementJobTemplate)
	require.True(t, WaitForJobSuccess(t, kc, enforcementJobName, testNamespace, 30, 5),
		"oauth proxy should issue a token to the expected client credentials only, and gate Prometheus behind it")

	// The scaler is created once the proxy is known to be serving,
	// so that its first poll does not race the proxy becoming reachable.
	KubectlApplyMultipleWithTemplate(t, data, scalerTemplates())
	require.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)

	testActivation(t, kc, data)
	testScaleOut(t, kc, data)
	testScaleIn(t, kc)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")
	KubectlReplaceWithTemplate(t, data, "generateLowLevelLoadJobTemplate", generateLowLevelLoadJobTemplate)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	KubectlReplaceWithTemplate(t, data, "generateLoadJobTemplate", generateLoadJobTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 5),
		"replica count should be %d after 5 minutes", minReplicaCount)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
		TestNamespace:           testNamespace,
		DeploymentName:          deploymentName,
		MonitoredAppName:        monitoredAppName,
		ScaledObjectName:        scaledObjectName,
		PrometheusServerName:    prometheusServerName,
		OAuthProxyName:          oauthProxyName,
		EnforcementJobName:      enforcementJobName,
		TriggerAuthName:         triggerAuthName,
		TriggerSecretName:       triggerSecretName,
		OAuthClientID:           oauthClientID,
		Base64ClientSecret:      base64.StdEncoding.EncodeToString([]byte(oauthClientSecret)),
		Base64ClientCredentials: basicClientCredentials(),
		OAuthAccessToken:        oauthAccessToken,
		MinReplicaCount:         minReplicaCount,
		MaxReplicaCount:         maxReplicaCount,
	}, append(proxyTemplates(), scalerTemplates()...)
}

// proxyTemplates are the token endpoint and the gated Prometheus, everything the scaler
// authenticates against.
func proxyTemplates() []Template {
	return []Template{
		{Name: "oauthProxyConfigMapTemplate", Config: oauthProxyConfigMapTemplate},
		{Name: "oauthProxyDeploymentTemplate", Config: oauthProxyDeploymentTemplate},
		{Name: "oauthProxyServiceTemplate", Config: oauthProxyServiceTemplate},
		{Name: "monitoredAppDeploymentTemplate", Config: monitoredAppDeploymentTemplate},
		{Name: "monitoredAppServiceTemplate", Config: monitoredAppServiceTemplate},
	}
}

// scalerTemplates are the KEDA resources and their scale target,
// applied only once the proxy they authenticate against is known to be enforcing.
func scalerTemplates() []Template {
	return []Template{
		{Name: "triggerSecretTemplate", Config: triggerSecretTemplate},
		{Name: "triggerAuthTemplate", Config: triggerAuthTemplate},
		{Name: "deploymentTemplate", Config: deploymentTemplate},
		{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
	}
}

// basicClientCredentials is the credential the scaler is expected to present to the token
// endpoint. golang.org/x/oauth2 URL encodes both halves before base64 encoding them, so the
// expected value is derived the same way rather than hardcoded.
func basicClientCredentials() string {
	credentials := url.QueryEscape(oauthClientID) + ":" + url.QueryEscape(oauthClientSecret)
	return base64.StdEncoding.EncodeToString([]byte(credentials))
}
