//go:build e2e
// +build e2e

package redis_azure_managed_aad_wi_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"

	. "github.com/kedacore/keda/v2/tests/helper"
)

var _ = godotenv.Load("../../../.env")

const testName = "redis-azure-managed-aad-wi-test"

var (
	managedRedisHost = os.Getenv("TF_AZURE_MANAGED_REDIS_HOST")
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	triggerAuthName  = fmt.Sprintf("%s-ta", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
	jobName          = fmt.Sprintf("%s-job", testName)
	redisList        = fmt.Sprintf("keda-list-%d", GetRandomNumber())
)

type templateData struct {
	TestNamespace    string
	KedaNamespace    string
	DeploymentName   string
	TriggerAuthName  string
	ScaledObjectName string
	JobName          string
	RedisAction      string
	ManagedRedisHost string
	RedisList        string
}

const (
	deploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
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
    spec:
      containers:
      - name: nginx
        image: ghcr.io/nginx/nginx-unprivileged:1.26
`
	triggerAuthTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthName}}
  namespace: {{.TestNamespace}}
spec:
  podIdentity:
    provider: azure-workload
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
  cooldownPeriod: 10
  minReplicaCount: 0
  maxReplicaCount: 1
  triggers:
  - type: redis
    metadata:
      address: {{.ManagedRedisHost}}:10000
      enableTLS: "true"
      listName: {{.RedisList}}
      listLength: "5"
    authenticationRef:
      name: {{.TriggerAuthName}}
`
	// The job runs as the keda-operator service account because the e2e
	// infrastructure federates identity_1 (the Managed Redis data contributor)
	// only with system:serviceaccount:keda:keda-operator.
	redisJobTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  name: {{.JobName}}
  namespace: {{.KedaNamespace}}
spec:
  backoffLimit: 1
  template:
    metadata:
      labels:
        azure.workload.identity/use: "true"
    spec:
      serviceAccountName: keda-operator
      restartPolicy: Never
      containers:
      - name: redis
        image: mcr.microsoft.com/azure-cli:2.79.0
        env:
        - name: REDIS_HOST
          value: {{.ManagedRedisHost}}
        - name: REDIS_LIST
          value: {{.RedisList}}
        - name: REDIS_ACTION
          value: {{.RedisAction}}
        command: ["/bin/sh", "-c"]
        args:
        - |
          set -eu
          token="$(az login --service-principal -u "$AZURE_CLIENT_ID" -t "$AZURE_TENANT_ID" --federated-token "$(cat "$AZURE_FEDERATED_TOKEN_FILE")" >/dev/null && az account get-access-token --resource https://redis.azure.com/ --query accessToken -o tsv)"
          python3 - "$token" <<'PY'
          import base64, json, os, socket, ssl, sys

          token = sys.argv[1]
          payload = token.split('.')[1] + '==='
          oid = json.loads(base64.urlsafe_b64decode(payload))['oid']
          sock = socket.create_connection((os.environ['REDIS_HOST'], 10000), 10)
          sock = ssl.create_default_context().wrap_socket(sock, server_hostname=os.environ['REDIS_HOST'])

          def command(*parts):
              data = b'*%d\r\n' % len(parts)
              for part in parts:
                  part = str(part).encode()
                  data += b'$%d\r\n%s\r\n' % (len(part), part)
              sock.sendall(data)
              response = b''
              while b'\r\n' not in response:
                  chunk = sock.recv(4096)
                  if not chunk:
                      raise RuntimeError('Redis connection closed before the reply was complete')
                  response += chunk
              if response.startswith(b'-'):
                  raise RuntimeError(response.decode())

          command('AUTH', oid, token)
          if os.environ['REDIS_ACTION'] == 'push':
              command('RPUSH', os.environ['REDIS_LIST'], 'item-1', 'item-2', 'item-3', 'item-4', 'item-5')
          else:
              command('DEL', os.environ['REDIS_LIST'])
          sock.close()
          PY
`
)

func TestScaler(t *testing.T) {
	require.NotEmpty(t, managedRedisHost, "TF_AZURE_MANAGED_REDIS_HOST env variable is required for Azure Managed Redis tests")

	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	t.Cleanup(func() {
		deleteRedisJob(t, data, jobName)
		deleteRedisJob(t, data, fmt.Sprintf("%s-drain", testName))
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})

	CreateKubernetesResources(t, kc, testNamespace, data, templates)
	require.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1))

	t.Log("--- testing scale out ---")
	data.RedisAction = "push"
	KubectlApplyWithTemplate(t, data, "redisJobTemplate", redisJobTemplate)
	waitForRedisJob(t, data)
	require.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 1))

	t.Log("--- testing scale in ---")
	data.JobName = fmt.Sprintf("%s-drain", testName)
	data.RedisAction = "drain"
	KubectlApplyWithTemplate(t, data, "redisJobTemplate", redisJobTemplate)
	waitForRedisJob(t, data)
	require.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1))
}

func waitForRedisJob(t *testing.T, data templateData) {
	_, err := ExecuteCommand(fmt.Sprintf("kubectl wait --for=condition=complete --timeout=120s job/%s -n %s", data.JobName, data.KedaNamespace))
	if err != nil {
		logs, _ := ExecuteCommand(fmt.Sprintf("kubectl logs job/%s -n %s --all-containers", data.JobName, data.KedaNamespace))
		t.Logf("redis job %s logs:\n%s", data.JobName, logs)
	}
	require.NoError(t, err)
}

func deleteRedisJob(t *testing.T, data templateData, name string) {
	_, err := ExecuteCommand(fmt.Sprintf("kubectl delete job/%s -n %s --ignore-not-found --cascade=foreground", name, data.KedaNamespace))
	if err != nil {
		t.Logf("failed to delete redis job %s: %v", name, err)
	}
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:    testNamespace,
			KedaNamespace:    KEDANamespace,
			DeploymentName:   deploymentName,
			TriggerAuthName:  triggerAuthName,
			ScaledObjectName: scaledObjectName,
			JobName:          jobName,
			RedisAction:      "push",
			ManagedRedisHost: managedRedisHost,
			RedisList:        redisList,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "triggerAuthTemplate", Config: triggerAuthTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}
