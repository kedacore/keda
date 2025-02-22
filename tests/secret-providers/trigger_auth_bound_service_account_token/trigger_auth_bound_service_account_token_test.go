//go:build e2e
// +build e2e

package trigger_auth_bound_service_account_token_test

import (
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
	testName = "trigger-auth-bound-service-account-token-test"
)

var (
	testNamespace                          = fmt.Sprintf("%s-ns", testName)
	deploymentName                         = fmt.Sprintf("%s-deployment", testName)
	metricsServerDeploymentName            = fmt.Sprintf("%s-metrics-server", testName)
	triggerAuthName                        = fmt.Sprintf("%s-ta", testName)
	scaledObjectName                       = fmt.Sprintf("%s-so", testName)
	metricsServerServiceName               = fmt.Sprintf("%s-service", testName)
	metricsServerEndpoint                  = fmt.Sprintf("http://%s.%s.svc.cluster.local:8080/api/value", metricsServerServiceName, testNamespace)
	serviceAccountName                     = fmt.Sprintf("%s-sa", testName)
	serviceAccountTokenCreationRole        = fmt.Sprintf("%s-sa-role", testName)
	serviceAccountTokenCreationRoleBinding = fmt.Sprintf("%s-sa-role-binding", testName)
	minReplicaCount                        = 0
	maxReplicaCount                        = 1
)

type templateData struct {
	TestNamespace                          string
	ServiceAccountName                     string
	ServiceAccountTokenCreationRole        string
	ServiceAccountTokenCreationRoleBinding string
	DeploymentName                         string
	MetricsServerDeploymentName            string
	MetricsServerServiceName               string
	TriggerAuthName                        string
	ScaledObjectName                       string
	MetricsServerEndpoint                  string
	MetricValue                            int
	MinReplicaCount                        string
	MaxReplicaCount                        string
}

const (
	serviceAccountTemplate = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: {{.ServiceAccountName}}
  namespace: {{.TestNamespace}}
`
	// arbitrary k8s rbac permissions that the test metrics-api container requires requesters to have
	serviceAccountClusterRoleTemplate = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: {{.ServiceAccountName}}
rules:
- nonResourceURLs:
  - /api/value
  verbs:
  - get
`
	serviceAccountClusterRoleBindingTemplate = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: {{.ServiceAccountName}}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: {{.ServiceAccountName}}
subjects:
- kind: ServiceAccount
  name: {{.ServiceAccountName}}
  namespace: {{.TestNamespace}}
`
	serviceAccountTokenCreationRoleTemplate = `
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: {{.ServiceAccountTokenCreationRole}}
  namespace: {{.TestNamespace}}
rules:
- apiGroups:
  - ""
  resources:
  - serviceaccounts/token
  verbs:
  - create
  - get
`
	serviceAccountTokenCreationRoleBindingTemplate = `
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: {{.ServiceAccountTokenCreationRoleBinding}}
  namespace: {{.TestNamespace}}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: {{.ServiceAccountTokenCreationRole}}
subjects:
- kind: ServiceAccount
  name: keda-operator
  namespace: keda
`
	tokenReviewAndSubjectAccessReviewClusterRoleTemplate = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: token-review-and-subject-access-review-role
rules:
- apiGroups:
  - "authentication.k8s.io"
  resources:
  - tokenreviews
  verbs:
  - create
- apiGroups:
  - authorization.k8s.io
  resources:
  - subjectaccessreviews
  verbs:
  - create
`
	tokenReviewAndSubjectAccessReviewClusterRoleBindingTemplate = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: token-review-and-subject-access-review-role-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: token-review-and-subject-access-review-role
subjects:
- kind: ServiceAccount
  name: default
  namespace: {{.TestNamespace}}
`
	metricsServerDeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: {{.MetricsServerDeploymentName}}
  name: {{.MetricsServerDeploymentName}}
  namespace: {{.TestNamespace}}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: {{.MetricsServerDeploymentName}}
  template:
    metadata:
      labels:
        app: {{.MetricsServerDeploymentName}}
        type: keda-testing
    spec:
      containers:
      - name: k8s-protected-metrics-api
        image: ghcr.io/kedacore/tests-bound-service-account-token:latest
        imagePullPolicy: Always
        securityContext:
          allowPrivilegeEscalation: false
          runAsNonRoot: true
          capabilities:
            drop:
              - ALL
          seccompProfile:
            type: RuntimeDefault
`
	metricsServerService = `
apiVersion: v1
kind: Service
metadata:
  name: {{.MetricsServerServiceName}}
  namespace: {{.TestNamespace}}
spec:
  ports:
  - name: http
    port: 8080
    targetPort: 8080
  selector:
    app: {{.MetricsServerDeploymentName}}
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
  replicas: 0
  selector:
    matchLabels:
      app: {{.DeploymentName}}
  template:
    metadata:
      labels:
        app: {{.DeploymentName}}
        type: keda-testing
    spec:
      containers:
      - name: prom-test-app
        image: ghcr.io/kedacore/tests-prometheus:latest
        imagePullPolicy: Always
        securityContext:
          allowPrivilegeEscalation: false
          runAsNonRoot: true
          capabilities:
            drop:
              - ALL
          seccompProfile:
            type: RuntimeDefault
`
	triggerAuthTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthName}}
  namespace: {{.TestNamespace}}
spec:
  boundServiceAccountToken:
    - parameter: token
      serviceAccountName: {{.ServiceAccountName}}
`
	scaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  pollingInterval: 5
  minReplicaCount: 0
  maxReplicaCount: 1
  cooldownPeriod: 10
  triggers:
  - type: metrics-api
    metadata:
      targetValue: "10"
      url: "{{.MetricsServerEndpoint}}"
      valueLocation: 'value'
      authMode: "bearer"
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
        image: curlimages/curl
        imagePullPolicy: Always
        command: ["curl", "-X", "POST", "{{.MetricsServerEndpoint}}/{{.MetricValue}}"]
      restartPolicy: Never`
)

func TestScaler(t *testing.T) {
	// setup
	// ctx := context.Background()
	t.Log("--- setting up ---")

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	// wait for metrics server to be ready; scale target to start at 0 replicas
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, metricsServerDeploymentName, testNamespace, 1, 60, 1),
		"replica count should be 1 after 1 minute")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")

	// test scaling
	testScaleOut(t, kc, data)
	testScaleIn(t, kc, data)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:                          testNamespace,
			ServiceAccountName:                     serviceAccountName,
			ServiceAccountTokenCreationRole:        serviceAccountTokenCreationRole,
			ServiceAccountTokenCreationRoleBinding: serviceAccountTokenCreationRoleBinding,
			MetricsServerDeploymentName:            metricsServerDeploymentName,
			MetricsServerEndpoint:                  metricsServerEndpoint,
			MetricsServerServiceName:               metricsServerServiceName,
			DeploymentName:                         deploymentName,
			TriggerAuthName:                        triggerAuthName,
			ScaledObjectName:                       scaledObjectName,
			MinReplicaCount:                        fmt.Sprintf("%d", minReplicaCount),
			MaxReplicaCount:                        fmt.Sprintf("%d", maxReplicaCount),
			MetricValue:                            1,
		}, []Template{
			// required for the keda to act as the service account which has the necessary permissions
			{Name: "serviceAccountTemplate", Config: serviceAccountTemplate},
			{Name: "serviceAccountClusterRoleTemplate", Config: serviceAccountClusterRoleTemplate},
			{Name: "serviceAccountClusterRoleBindingTemplate", Config: serviceAccountClusterRoleBindingTemplate},
			// required for the keda to request token creations for the service account
			{Name: "serviceAccountTokenCreationRoleTemplate", Config: serviceAccountTokenCreationRoleTemplate},
			{Name: "serviceAccountTokenCreationRoleBindingTemplate", Config: serviceAccountTokenCreationRoleBindingTemplate},
			// required for the metrics-api container to delegate authenticate/authorize requests to k8s apiserver
			{Name: "tokenReviewAndSubjectAccessReviewClusterRoleTemplate", Config: tokenReviewAndSubjectAccessReviewClusterRoleTemplate},
			{Name: "tokenReviewAndSubjectAccessReviewClusterRoleBindingTemplate", Config: tokenReviewAndSubjectAccessReviewClusterRoleBindingTemplate},
			{Name: "metricsServerDeploymentTemplate", Config: metricsServerDeploymentTemplate},
			{Name: "metricsServerService", Config: metricsServerService},
			// scale target and trigger auths
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "triggerAuthTemplate", Config: triggerAuthTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	data.MetricValue = 50
	KubectlReplaceWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, maxReplicaCount),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale in ---")
	data.MetricValue = 0
	KubectlReplaceWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
}
