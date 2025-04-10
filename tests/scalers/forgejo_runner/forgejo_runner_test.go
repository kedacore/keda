//go:build e2e
// +build e2e

package forgejo_runner_test

import (
	"fmt"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"
	"os"
	"testing"

	. "github.com/kedacore/keda/v2/tests/helper" // For helper methods
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "forgejo-test"
)

var (
	testNamespace  = fmt.Sprintf("%s-ns", testName)
	deploymentName = fmt.Sprintf("%s-deployment", testName)
	scaledJobName  = fmt.Sprintf("%s-so", testName)
	configName     = fmt.Sprintf("%s-configmap", testName)

	forgejoRunnerName  = "forgejo-runner"
	forgejoToken       = os.Getenv("FORGEJO_TOKEN")
	forgejoGlobal      = "true"
	forgejoOwner       = os.Getenv("FORGEJO_OWNER")
	forgejoRepo        = os.Getenv("FORGEJO_REPO")
	forgejoLabel       = "ubuntu-latest"
	forgejoAccessToken = os.Getenv("FORGEJO_ACCESS_TOKEN")

	forgejoPodName = "forgejo"

	minReplicaCount = 0
	maxReplicaCount = 1
)

type templateData struct {
	TestNamespace    string
	DeploymentName   string
	ScaledObjectName string
	ConfigName       string

	ForgejoRunnerName  string
	ForgejoToken       string
	ForgejoGlobal      string
	ForgejoOwner       string
	ForgejoRepo        string
	ForgejoLabel       string
	ForgejoAccessToken string
}

const (
	forgejoService = `
apiVersion: v1
kind: Service
metadata:
  labels:
    app: forgejo
  name: forgejo-service
  namespace: {{.TestNamespace}}
spec:
  ports:
  - port: 3000
    protocol: TCP
    targetPort: 3000
  selector:
    app: forgejo
  type: ClusterIP
`

	forgejoDeployment = `
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: forgejo
  name: forgejo
  namespace: {{.TestNamespace}}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: forgejo
  template:
    metadata:
      labels:
        app: forgejo
    spec:
      containers:
      - image: codeberg.org/forgejo-experimental/forgejo:11
        imagePullPolicy: IfNotPresent
        livenessProbe:
          failureThreshold: 10
          httpGet:
            path: /
            port: 3000
            scheme: HTTP
          initialDelaySeconds: 30
          periodSeconds: 10
          successThreshold: 1
          timeoutSeconds: 5
        name: forgejo
        ports:
        - containerPort: 3000
          name: http
          protocol: TCP
        readinessProbe:
          failureThreshold: 10
          httpGet:
            path: /
            port: 3000
            scheme: HTTP
          initialDelaySeconds: 30
          periodSeconds: 30
          successThreshold: 1
          timeoutSeconds: 5
        volumeMounts:
        - mountPath: /var/lib/gitea
          name: forgejo
          subPath: forgejo/var/lib/gitea
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      securityContext:
        fsGroup: 1000
        runAsGroup: 1000
        runAsNonRoot: true
        runAsUser: 1000
      volumes:
      - emptyDir: {}
        name: forgejo
 `
	forgejoConfigMap = `apiVersion: v1
kind: ConfigMap
metadata:
  name: forgejo-config
  namespace: {{.TestNamespace}}
data:
  app.ini: |
    APP_NAME = Forgejo: Beyond coding. We Forge.
    RUN_USER = merino_mora
    WORK_PATH = /
    RUN_MODE = prod
    
    [database]
    DB_TYPE = sqlite3
    HOST = 127.0.0.1:3306
    NAME = forgejo
    USER = forgejo
    PASSWD =
    SCHEMA =
    SSL_MODE = disable
    PATH = /data/forgejo.db
    LOG_SQL = false
    
    [repository]
    ROOT = /data/forgejo-repositories
    
    [server]
    SSH_DOMAIN = localhost
    DOMAIN = localhost
    HTTP_PORT = 3000
    ROOT_URL = http://localhost:3000/
    APP_DATA_PATH = /data
    DISABLE_SSH = false
    SSH_PORT = 22
    LFS_START_SERVER = true
    OFFLINE_MODE = true
    
    [lfs]
    PATH = /data/lfs
    
    [mailer]
    ENABLED = false
    
    [service]
    REGISTER_EMAIL_CONFIRM = false
    ENABLE_NOTIFY_MAIL = false
    DISABLE_REGISTRATION = false
    ALLOW_ONLY_EXTERNAL_REGISTRATION = false
    ENABLE_CAPTCHA = false
    REQUIRE_SIGNIN_VIEW = false
    DEFAULT_KEEP_EMAIL_PRIVATE = false
    DEFAULT_ALLOW_CREATE_ORGANIZATION = true
    DEFAULT_ENABLE_TIMETRACKING = true
    NO_REPLY_ADDRESS = noreply.localhost
    
    [openid]
    ENABLE_OPENID_SIGNIN = true
    ENABLE_OPENID_SIGNUP = true
    
    [cron.update_checker]
    ENABLED = true
    
    [session]
    PROVIDER = file
    
    [log]
    MODE = console
    LEVEL = info
    ROOT_PATH = /log
    
    [repository.pull-request]
    DEFAULT_MERGE_STYLE = merge
    
    [repository.signing]
    DEFAULT_TRUST_MODEL = committer
    
    [security]
    INSTALL_LOCK = true
    PASSWORD_HASH_ALGO = pbkdf2_hi
    
    [actions]
    ENABLED = true
    DEFAULT_ACTIONS_URL = https://github.com
    
    [stackitgitsettings]
    ENABLE_USER_PASS_SIGNIN = true
`

	runnerConfigMap = `apiVersion: v1
kind: ConfigMap
metadata:
  name: {{.ConfigName}}
  namespace: {{.TestNamespace}}
data:
  .runner: |
    {
      "WARNING": "This file is automatically generated by act-runner. Do not edit it manually unless you know what you are doing. Removing this file will cause act runner to re-register as a new runner.",
      "id": 368,
      "uuid": "ec85849d-bba3-4abc-9303-1445d06943ff",
      "name": "{{.ForgejoRunnerName}}",
      "token": "{{.ForgejoToken}}",
      "address": "http://forgejo.{{.TestNamespace}}.svc.cluster.local",
      "labels": [
        "{{.ForgejoLabel}}"
      ]
    }
`

	scaledJob = `apiVersion: keda.sh/v1alpha1
kind: ScaledJob
metadata:
  labels:
    app: forgejo-runner
  name: forgejo-runner
  namespace: runners
spec:
  jobTargetRef:
    template:
      metadata:
        labels:
          app: forgejo-runner
      spec:
        volumes:
        - name: runner-data
          emptyDir: {}
        - name: runner-config
          configMap:
            name: runner-config
        containers:
        - name: runner
          image: localhost:5001/runner:latest
          command: ["sh", "-c", "forgejo-runner job"]
          volumeMounts:
          - name: runner-data
            mountPath: /data
          - name: runner-config
            mountPath: /config
            readOnly: false
  minReplicaCount: 0
  maxReplicaCount: 20
  pollingInterval: 30
  triggers:
  - type: forgejo-runner
    metadata:
      name: "{{.ForgejoRunnerName}}"
      token: "{{.ForgejoAccessToken}}"
      address: "http://forgejo.{{.TestNamespace}}.svc.cluster.local"
      global: "{{.ForgejoGlobal}}"
      labels: "{{.ForgejoLabel}}"
      repo: "{{.ForgejoRepo}}"
      owner: "{{.ForgejoOwner}}"
`
)

func TestForgejoScaler(t *testing.T) {
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})

	// Create kubernetes resources
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	// setup forgejo
	setupForgejo(t, kc)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)

	// test scaling
	testActivation(t, kc)
	testScaleOut(t, kc)
	testScaleIn(t, kc)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:      testNamespace,
			DeploymentName:     deploymentName,
			ScaledObjectName:   scaledJobName,
			ConfigName:         configName,
			ForgejoRunnerName:  forgejoRunnerName,
			ForgejoToken:       forgejoToken,
			ForgejoGlobal:      forgejoGlobal,
			ForgejoOwner:       forgejoOwner,
			ForgejoRepo:        forgejoRepo,
			ForgejoLabel:       forgejoLabel,
			ForgejoAccessToken: forgejoAccessToken,
		}, []Template{
			{Name: "forgejoConfigMapTemplate", Config: forgejoConfigMap},
			{Name: "runnerConfigMapTemplate", Config: runnerConfigMap},
			{Name: "forgejoDeploymentTemplate", Config: forgejoDeployment},
			{Name: "forgejoServiceTemplate", Config: forgejoService},
			{Name: "scaledObjectTemplate", Config: scaledJob},
		}
}

func setupForgejo(t *testing.T, kc *kubernetes.Clientset) {
	require.True(t,
		WaitForDeploymentReplicaReadyCount(t, kc, "forgejo", testNamespace, 1, 3, 20),
		"Forgejo should be running after 1 minute")

}

// add 3 documents to solr -> activation should not happen (activationTargetValue = 5)
func testActivation(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing activation ---")

}

// add 3 more documents to solr, which in total is 6 -> should be scaled up
func testScaleOut(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale out ---")

}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")

}
