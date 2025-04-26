//go:build e2e
// +build e2e

package solr_test

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper" // For helper methods
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "solr-test"
)

var (
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
	secretName       = fmt.Sprintf("%s-secret", testName)
	solrUsername     = "solr"
	solrPassword     = "SolrRocks"
	solrCollection   = "my_core"
	solrPodName      = "solr-0"
	solrPath         = "bin/solr"

	minReplicaCount = 0
	maxReplicaCount = 2
)

type templateData struct {
	TestNamespace      string
	DeploymentName     string
	ScaledObjectName   string
	SecretName         string
	SolrUsernameBase64 string
	SolrPasswordBase64 string
}

const (
	secretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  solr_username: {{.SolrUsernameBase64}}
  solr_password: {{.SolrPasswordBase64}}
`

	triggerAuthenticationTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: keda-trigger-auth-solr-secret
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
    - parameter: username
      name: {{.SecretName}}
      key: solr_username
    - parameter: password
      name: {{.SecretName}}
      key: solr_password
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

	solrDeploymentTemplate = `
apiVersion: apps/v1
kind: StatefulSet
metadata:
  labels:
    app: solr-app
  name: solr
  namespace: {{.TestNamespace}}
spec:
  serviceName: {{.DeploymentName}}
  replicas: 1
  selector:
    matchLabels:
      app: solr-app
  template:
    metadata:
      labels:
        app: solr-app
    spec:
      containers:
        - name: solr
          image: solr:8
          ports:
            - containerPort: 8983
          volumeMounts:
            - name: data
              mountPath: /var/solr
      volumes:
        - name: data
          emptyDir: {}
`

	serviceTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
spec:
  selector:
    app: solr-app
  type: ClusterIP
  ports:
  - protocol: TCP
    port: 8983
    targetPort: 8983
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
  minReplicaCount: 0
  maxReplicaCount: 2
  pollingInterval: 1
  cooldownPeriod:  1
  triggers:
  - type: solr
    metadata:
      host: "http://{{.DeploymentName}}.{{.TestNamespace}}.svc.cluster.local:8983"
      collection: "my_core"
      query: "*:*"
      targetQueryValue: "1"
      activationTargetQueryValue: "5"
    authenticationRef:
      name: keda-trigger-auth-solr-secret
`
)

func TestSolrScaler(t *testing.T) {
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})

	// Create kubernetes resources
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	// setup solr
	setupSolr(t, kc)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)

	// test scaling
	testActivation(t, kc)
	testScaleOut(t, kc)
	testScaleIn(t, kc)
}

func setupSolr(t *testing.T, kc *kubernetes.Clientset) {
	require.True(t, WaitForStatefulsetReplicaReadyCount(t, kc, "solr", testNamespace, 1, 60, 3),
		"solr should be up")
	err := checkIfSolrStatusIsReady(t, solrPodName)
	require.NoErrorf(t, err, "%s", err)

	// Create the collection
	out, errOut, err := ExecCommandOnSpecificPod(t, solrPodName, testNamespace, fmt.Sprintf("%s create_core -c %s", solrPath, solrCollection))
	require.NoErrorf(t, err, "%s", err)
	t.Logf("Output: %s, Error: %s", out, errOut)

	// Enable BasicAuth
	out, errOut, err = ExecCommandOnSpecificPod(t, solrPodName, testNamespace, "echo '{\"authentication\":{\"class\":\"solr.BasicAuthPlugin\",\"credentials\":{\"solr\":\"IV0EHq1OnNrj6gvRCwvFwTrZ1+z1oBbnQdiVC3otuq0= Ndd7LKvVBAaZIF0QAVi1ekCfAJXr1GGfLtRUXhgrF8c=\"}},\"authorization\":{\"class\":\"solr.RuleBasedAuthorizationPlugin\",\"permissions\":[{\"name\":\"security-edit\",\"role\":\"admin\"}],\"user-role\":{\"solr\":\"admin\"}}}' > /var/solr/data/security.json")
	require.NoErrorf(t, err, "%s", err)
	t.Logf("Output: %s, Error: %s", out, errOut)

	// Restart solr to apply auth
	out, errOut, _ = ExecCommandOnSpecificPod(t, solrPodName, testNamespace, fmt.Sprintf("%s restart", solrPath))
	t.Logf("Output: %s, Error: %s", out, errOut)

	err = checkIfSolrStatusIsReady(t, solrPodName)
	require.NoErrorf(t, err, "%s", err)
	t.Log("--- BasicAuth plugin activated ---")

	t.Log("--- solr is ready ---")
}

func checkIfSolrStatusIsReady(t *testing.T, name string) error {
	t.Log("--- checking solr status ---")

	for i := 0; i < 12; i++ {
		out, errOut, _ := ExecCommandOnSpecificPod(t, name, testNamespace, fmt.Sprintf("%s status", solrPath))
		t.Logf("Output: %s, Error: %s", out, errOut)
		if !strings.Contains(out, "running on port") {
			time.Sleep(time.Second * 5)
			continue
		}
		return nil
	}
	return errors.New("solr is not ready")
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:      testNamespace,
			DeploymentName:     deploymentName,
			ScaledObjectName:   scaledObjectName,
			SecretName:         secretName,
			SolrUsernameBase64: base64.StdEncoding.EncodeToString([]byte(solrUsername)),
			SolrPasswordBase64: base64.StdEncoding.EncodeToString([]byte(solrPassword)),
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
			{Name: "serviceTemplate", Config: serviceTemplate},
			{Name: "solrDeploymentTemplate", Config: solrDeploymentTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

// add 3 documents to solr -> activation should not happen (activationTargetValue = 5)
func testActivation(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing activation ---")

	// Add documents
	out, errOut, _ := ExecCommandOnSpecificPod(t, solrPodName, testNamespace, fmt.Sprintf("curl -u %s:%s -X POST -H 'Content-Type: application/json' 'http://localhost:8983/solr/%s/update' --data-binary '[{\"id\": \"1\",\"title\": \"Doc 1\"},,{\"id\": \"2\",\"title\": \"Doc 2\"},{\"id\": \"3\",\"title\":\"Doc 3\"}]'", solrUsername, solrPassword, solrCollection))
	t.Logf("Output: %s, Error: %s", out, errOut)
	// Update documents
	out, errOut, _ = ExecCommandOnSpecificPod(t, solrPodName, testNamespace, fmt.Sprintf("curl -u %s:%s -X POST 'http://localhost:8983/solr/%s/update' --data-binary '{\"commit\":{}}' -H 'Content-type:application/json'", solrUsername, solrPassword, solrCollection))
	t.Logf("Output: %s, Error: %s", out, errOut)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

// add 3 more documents to solr, which in total is 6 -> should be scaled up
func testScaleOut(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale out ---")

	// Add documents
	out, errOut, _ := ExecCommandOnSpecificPod(t, solrPodName, testNamespace, fmt.Sprintf("curl -u %s:%s -X POST -H 'Content-Type: application/json' 'http://localhost:8983/solr/%s/update' --data-binary '[{ \"id\": \"10\",\"title\": \"Doc 10\"},{ \"id\": \"20\",\"title\": \"Doc 20\"},{ \"id\": \"30\",\"title\": \"Doc 30\"}]'", solrUsername, solrPassword, solrCollection))
	t.Logf("Output: %s, Error: %s", out, errOut)
	// Update documents
	out, errOut, _ = ExecCommandOnSpecificPod(t, solrPodName, testNamespace, fmt.Sprintf("curl -u %s:%s -X POST 'http://localhost:8983/solr/%s/update' --data-binary '{\"commit\":{}}' -H 'Content-type:application/json'", solrUsername, solrPassword, solrCollection))
	t.Logf("Output: %s, Error: %s", out, errOut)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")

	// Delete documents
	out, errOut, err := ExecCommandOnSpecificPod(t, solrPodName, testNamespace, fmt.Sprintf("curl -u %s:%s -X POST 'http://localhost:8983/solr/%s/update' --data '<delete><query>*:*</query></delete>' -H 'Content-type:text/xml; charset=utf-8'", solrUsername, solrPassword, solrCollection))
	t.Logf("Output: %s, Error: %s", out, errOut)
	// Commit changes
	out, errOut, _ = ExecCommandOnSpecificPod(t, solrPodName, testNamespace, fmt.Sprintf("curl -u %s:%s -X POST 'http://localhost:8983/solr/%s/update' --data-binary '{\"commit\":{}}' -H 'Content-type:application/json'", solrUsername, solrPassword, solrCollection))
	t.Logf("Output: %s, Error: %s", out, errOut)

	assert.NoErrorf(t, err, "cannot enqueue messages - %s", err)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
}
