//go:build e2e
// +build e2e

// Co-authored-by: Christopher Homberger <christopher.homberger@web.de>
//
// The Gitea bootstrap approach here — standing a real Gitea up in-cluster with
// `gitea admin user create --access-token` rather than shipping a pre-seeded
// database — comes from https://github.com/kedacore/keda/pull/6765.
//
// This version drives Gitea entirely through `kubectl exec` (the pattern already
// used by the elasticsearch/loki/mongodb tests) instead of port-forwarding, so it
// needs no additional vendored dependency and nothing outside the cluster.

package gitea_runner_test

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper" // For helper methods
)

var _ = godotenv.Load("../../.env")

const (
	testName = "gitea-runner-test"

	giteaUser = "test01"
	giteaRepo = "scaler-test"

	// Must match the label the workflow requests via `runs-on`.
	runnerLabel = "ubuntu-latest"
)

var (
	testNamespace   = fmt.Sprintf("%s-ns", testName)
	secretName      = fmt.Sprintf("%s-secret", testName)
	triggerAuthName = fmt.Sprintf("%s-auth", testName)
	scaledJobName   = fmt.Sprintf("%s-sj", testName)

	giteaAddress = fmt.Sprintf("http://gitea.%s.svc.cluster.local:3000", testNamespace)

	maxReplicaCount = 2
)

type templateData struct {
	TestNamespace   string
	SecretName      string
	TriggerAuthName string
	ScaledJobName   string
	ScaledJobLabel  string
	GiteaAddress    string
	GiteaPassword   string
	GiteaToken      string
	RunnerLabel     string
	MaxReplicaCount int
}

const (
	giteaDeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gitea
  namespace: {{ .TestNamespace }}
  labels:
    app: gitea
spec:
  replicas: 1
  selector:
    matchLabels:
      app: gitea
  template:
    metadata:
      labels:
        app: gitea
    spec:
      containers:
        - name: gitea
          image: gitea/gitea:1.27
          ports:
            - containerPort: 3000
          command:
            - sh
            - -c
            - |
              set -e
              mkdir -p /data/gitea/conf
              cat > /data/gitea/conf/app.ini << 'EOF'
              I_AM_BEING_UNSAFE_RUNNING_AS_ROOT = true
              [security]
              INSTALL_LOCK = true
              PASSWORD_COMPLEXITY = off
              [database]
              DB_TYPE = sqlite3
              PATH = /data/gitea.db
              [repository]
              ROOT = /data/repositories
              [actions]
              ENABLED = true
              [server]
              ROOT_URL = {{ .GiteaAddress }}
              EOF
              gitea migrate -c /data/gitea/conf/app.ini
              gitea admin user create -c /data/gitea/conf/app.ini \
                --username ` + giteaUser + ` --password '{{ .GiteaPassword }}' \
                --email test01@gitea.local --admin --must-change-password=false
              exec gitea web -c /data/gitea/conf/app.ini
          readinessProbe:
            httpGet:
              path: /api/healthz
              port: 3000
            initialDelaySeconds: 10
            periodSeconds: 5
            failureThreshold: 30
          volumeMounts:
            - name: data
              mountPath: /data
      volumes:
        - name: data
          emptyDir: {}
`

	giteaServiceTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: gitea
  namespace: {{ .TestNamespace }}
spec:
  selector:
    app: gitea
  ports:
    - port: 3000
      targetPort: 3000
`

	secretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{ .SecretName }}
  namespace: {{ .TestNamespace }}
data:
  token: {{ .GiteaToken }}
`

	triggerAuthenticationTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{ .TriggerAuthName }}
  namespace: {{ .TestNamespace }}
spec:
  secretTargetRef:
    - parameter: token
      name: {{ .SecretName }}
      key: token
`

	// The scaled workload is deliberately trivial. This test proves the SCALER
	// reads Gitea's queue correctly and drives Job creation; it is not a test of
	// docker-in-docker or of the runner image.
	scaledJobTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledJob
metadata:
  name: {{ .ScaledJobName }}
  namespace: {{ .TestNamespace }}
spec:
  jobTargetRef:
    parallelism: 1
    completions: 1
    backoffLimit: 0
    template:
      metadata:
        labels:
          app: {{ .ScaledJobLabel }}
      spec:
        restartPolicy: Never
        containers:
          - name: worker
            image: busybox:1.36
            command: ["sh", "-c", "sleep 15"]
  minReplicaCount: 0
  maxReplicaCount: {{ .MaxReplicaCount }}
  pollingInterval: 5
  successfulJobsHistoryLimit: 5
  failedJobsHistoryLimit: 5
  scalingStrategy:
    strategy: accurate
  triggers:
    - type: gitea-runner
      metadata:
        address: {{ .GiteaAddress }}
        global: "true"
        labels: "{{ .RunnerLabel }}"
      authenticationRef:
        name: {{ .TriggerAuthName }}
`
)

func TestGiteaRunnerScaler(t *testing.T) {
	kc := GetKubernetesClient(t)

	// Generated per run: no credential literal lives in this source file.
	password := randomPassword(t)

	data := getTemplateData("", password)
	CreateKubernetesResources(t, kc, testNamespace, data, giteaTemplates())
	defer DeleteKubernetesResources(t, testNamespace, data, giteaTemplates())

	require.True(t, WaitForDeploymentReplicaReadyCount(t, kc, "gitea", testNamespace, 1, 12, 15),
		"gitea should be ready within 3 minutes")

	pod := giteaPodName(t)
	token := mintToken(t, pod, password)
	require.NotEmpty(t, token, "should have minted a Gitea API token")

	seedRepo(t, pod, token)

	data = getTemplateData(base64.StdEncoding.EncodeToString([]byte(token)), password)
	KubectlApplyMultipleWithTemplate(t, data, scalerTemplates())
	defer KubectlDeleteMultipleWithTemplate(t, data, scalerTemplates())

	testNotActivated(t, kc)
	testScaleOut(t, kc, pod, token)
	testScaleIn(t, kc, pod, token)
}

func giteaTemplates() []Template {
	return []Template{
		{Name: "giteaDeploymentTemplate", Config: giteaDeploymentTemplate},
		{Name: "giteaServiceTemplate", Config: giteaServiceTemplate},
	}
}

func scalerTemplates() []Template {
	return []Template{
		{Name: "secretTemplate", Config: secretTemplate},
		{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
		{Name: "scaledJobTemplate", Config: scaledJobTemplate},
	}
}

func getTemplateData(token, password string) templateData {
	return templateData{
		TestNamespace:   testNamespace,
		SecretName:      secretName,
		TriggerAuthName: triggerAuthName,
		ScaledJobName:   scaledJobName,
		ScaledJobLabel:  scaledJobName,
		GiteaAddress:    giteaAddress,
		GiteaPassword:   password,
		GiteaToken:      token,
		RunnerLabel:     runnerLabel,
		MaxReplicaCount: maxReplicaCount,
	}
}

// randomPassword generates the throwaway admin password for this test run, so
// no credential literal is committed to the repository.
func randomPassword(t *testing.T) string {
	t.Helper()
	buf := make([]byte, 24)
	_, err := rand.Read(buf)
	require.NoError(t, err, "should be able to generate a test password")
	return "Aa1-" + base64.RawURLEncoding.EncodeToString(buf)
}

func giteaPodName(t *testing.T) string {
	out, err := ExecuteCommand(fmt.Sprintf(
		"kubectl get pod -n %s -l app=gitea -o jsonpath={.items[0].metadata.name}", testNamespace))
	require.NoError(t, err, "should be able to find the gitea pod")
	return strings.TrimSpace(string(out))
}

// giteaCurl runs curl inside the Gitea pod, so the test needs no port-forward
// and no route from the test runner into the cluster.
func giteaCurl(t *testing.T, pod, args string) string {
	stdout, stderr, err := ExecCommandOnSpecificPodWithoutTTY(t, pod, testNamespace,
		fmt.Sprintf("curl -sS --fail-with-body %s", args))
	require.NoError(t, err, "curl inside gitea pod failed: %s", stderr)
	return stdout
}

// mintToken creates an API token for the admin user. read:admin is the scope the
// scaler needs for `global: true` — the /admin/* endpoints are not reachable with
// a repo-scoped token.
func mintToken(t *testing.T, pod, password string) string {
	out := giteaCurl(t, pod, fmt.Sprintf(
		`-X POST -u '%s:%s' -H 'Content-Type: application/json' `+
			`-d '{"name":"keda-e2e","scopes":["read:admin","write:repository","write:user"]}' `+
			`http://localhost:3000/api/v1/users/%s/tokens`,
		giteaUser, password, giteaUser))

	token := extractJSONString(out, "sha1")
	t.Logf("minted Gitea token (%d chars)", len(token))
	return token
}

// seedRepo creates a repository containing a single workflow that requests the
// runner label under test. No runner is ever registered, so dispatched jobs stay
// queued — which is exactly the state the scaler must report.
func seedRepo(t *testing.T, pod, token string) {
	auth := fmt.Sprintf("-H 'Authorization: token %s' -H 'Content-Type: application/json'", token)

	giteaCurl(t, pod, fmt.Sprintf(
		`-X POST %s -d '{"name":"%s","auto_init":true,"default_branch":"main"}' `+
			`http://localhost:3000/api/v1/user/repos`, auth, giteaRepo))

	workflow := "name: e2e\non: [workflow_dispatch]\njobs:\n  build:\n    runs-on: " + runnerLabel +
		"\n    steps:\n      - run: echo hello\n"
	encoded := base64.StdEncoding.EncodeToString([]byte(workflow))

	giteaCurl(t, pod, fmt.Sprintf(
		`-X POST %s -d '{"content":"%s","message":"add workflow","branch":"main"}' `+
			`http://localhost:3000/api/v1/repos/%s/%s/contents/.gitea/workflows/e2e.yaml`,
		auth, encoded, giteaUser, giteaRepo))

	t.Log("seeded repository with a workflow requesting label", runnerLabel)
}

func dispatchWorkflow(t *testing.T, pod, token string) {
	giteaCurl(t, pod, fmt.Sprintf(
		`-X POST -H 'Authorization: token %s' -H 'Content-Type: application/json' `+
			`-d '{"ref":"main"}' `+
			`http://localhost:3000/api/v1/repos/%s/%s/actions/workflows/e2e.yaml/dispatches`,
		token, giteaUser, giteaRepo))
}

// queuedJobs asks Gitea the same question the scaler asks, so a failure can be
// attributed to either the scaler or the fixture rather than being ambiguous.
func queuedJobs(t *testing.T, pod, token string) string {
	out := giteaCurl(t, pod, fmt.Sprintf(
		`-H 'Authorization: token %s' `+
			`'http://localhost:3000/api/v1/admin/actions/jobs?status=queued&status=in_progress&limit=1'`,
		token))
	return extractJSONNumber(out, "total_count")
}

func testNotActivated(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing not activated ---")
	// Nothing dispatched yet, so the queue is empty and no Job should appear.
	// Job count, not replica count: a ScaledJob has no Deployment to inspect.
	assert.True(t, WaitForJobCountUntilIteration(t, kc, testNamespace, 0, 6, 5),
		"no jobs should be created while the Gitea queue is empty")
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, pod, token string) {
	t.Log("--- testing scale out ---")

	for i := 0; i < maxReplicaCount; i++ {
		dispatchWorkflow(t, pod, token)
	}
	t.Logf("dispatched %d workflow runs; gitea reports total_count=%s",
		maxReplicaCount, queuedJobs(t, pod, token))

	assert.True(t, WaitForJobCount(t, kc, testNamespace, maxReplicaCount, 60, 2),
		fmt.Sprintf("job count should reach %d within 2 minutes", maxReplicaCount))
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, pod, token string) {
	t.Log("--- testing scale in ---")

	// Deleting the repository clears its queued jobs, so the metric returns to 0.
	giteaCurl(t, pod, fmt.Sprintf(
		`-X DELETE -H 'Authorization: token %s' http://localhost:3000/api/v1/repos/%s/%s`,
		token, giteaUser, giteaRepo))

	t.Logf("repo deleted; gitea now reports total_count=%s", queuedJobs(t, pod, token))

	assert.True(t, WaitForPodsCompleted(t, kc, fmt.Sprintf("app=%s", scaledJobName), testNamespace, 60, 2),
		"in-flight jobs should complete rather than be killed")
}

// extractJSONString pulls a string field out of a JSON object without needing a
// struct, keeping the exec-based fixture dependency-free.
func extractJSONString(body, key string) string {
	marker := fmt.Sprintf(`"%s":"`, key)
	idx := strings.Index(body, marker)
	if idx < 0 {
		return ""
	}
	rest := body[idx+len(marker):]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return ""
	}
	return rest[:end]
}

func extractJSONNumber(body, key string) string {
	marker := fmt.Sprintf(`"%s":`, key)
	idx := strings.Index(body, marker)
	if idx < 0 {
		return "?"
	}
	rest := body[idx+len(marker):]
	end := strings.IndexAny(rest, ",}")
	if end < 0 {
		return "?"
	}
	return strings.TrimSpace(rest[:end])
}
