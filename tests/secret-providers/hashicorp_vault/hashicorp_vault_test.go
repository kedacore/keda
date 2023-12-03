//go:build e2e
// +build e2e

package hashicorp_vault_test

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
	"github.com/kedacore/keda/v2/tests/scalers/prometheus"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "hashicorp-vault-test"
)

var (
	testNamespace              = fmt.Sprintf("%s-ns", testName)
	vaultNamespace             = "hashicorp-ns"
	vaultPromDomain            = "e2e.vault.keda.sh"
	deploymentName             = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName           = fmt.Sprintf("%s-so", testName)
	publishDeploymentName      = fmt.Sprintf("%s-publish", testName)
	monitoredAppName           = fmt.Sprintf("%s-monitored-app", testName)
	triggerAuthenticationName  = fmt.Sprintf("%s-ta", testName)
	secretName                 = fmt.Sprintf("%s-secret", testName)
	postgreSQLStatefulSetName  = "postgresql"
	postgresqlPodName          = fmt.Sprintf("%s-0", postgreSQLStatefulSetName)
	postgreSQLUsername         = "test-user"
	postgreSQLPassword         = "test-password"
	postgreSQLDatabase         = "test_db"
	postgreSQLConnectionString = fmt.Sprintf("postgresql://%s:%s@postgresql.%s.svc.cluster.local:5432/%s?sslmode=disable",
		postgreSQLUsername, postgreSQLPassword, testNamespace, postgreSQLDatabase)
	prometheusServerName = fmt.Sprintf("%s-prom-server", testName)
	minReplicaCount      = 0
	maxReplicaCount      = 2
)

type templateData struct {
	TestNamespace                    string
	DeploymentName                   string
	VaultNamespace                   string
	ScaledObjectName                 string
	TriggerAuthenticationName        string
	VaultSecretPath                  string
	VaultPromDomain                  string
	SecretName                       string
	HashiCorpToken                   string
	PostgreSQLStatefulSetName        string
	PostgreSQLConnectionStringBase64 string
	PostgreSQLUsername               string
	PostgreSQLPassword               string
	PostgreSQLDatabase               string
	MinReplicaCount                  int
	MaxReplicaCount                  int
	PublishDeploymentName            string
	MonitoredAppName                 string
	PrometheusServerName             string
	VaultPkiCommonName               string
}

const (
	deploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: postgresql-update-worker
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
spec:
  replicas: 0
  selector:
    matchLabels:
      app: postgresql-update-worker
  template:
    metadata:
      labels:
        app: postgresql-update-worker
    spec:
      containers:
      - image: ghcr.io/kedacore/tests-postgresql
        imagePullPolicy: Always
        name: postgresql-processor-test
        command:
          - /app
          - update
        env:
          - name: TASK_INSTANCES_COUNT
            value: "6000"
          - name: CONNECTION_STRING
            valueFrom:
              secretKeyRef:
                name: {{.SecretName}}
                key: postgresql_conn_str
`

	secretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
type: Opaque
data:
  postgresql_conn_str: {{.PostgreSQLConnectionStringBase64}}
`

	triggerAuthenticationTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthenticationName}}
  namespace: {{.TestNamespace}}
spec:
  hashiCorpVault:
    address: http://vault.{{.VaultNamespace}}:8200
    authentication: token
    credential:
      token: {{.HashiCorpToken}}
    secrets:
    - parameter: connection
      key: connectionString
      path: {{.VaultSecretPath}}
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
  cooldownPeriod:  10
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  triggers:
  - type: postgresql
    metadata:
      targetQueryValue: "4"
      activationTargetQueryValue: "5"
      query: "SELECT CEIL(COUNT(*) / 5) FROM task_instance WHERE state='running' OR state='queued'"
    authenticationRef:
      name: {{.TriggerAuthenticationName}}
`

	postgreSQLStatefulSetTemplate = `
apiVersion: apps/v1
kind: StatefulSet
metadata:
  labels:
    app: {{.PostgreSQLStatefulSetName}}
  name: {{.PostgreSQLStatefulSetName}}
  namespace: {{.TestNamespace}}
spec:
  replicas: 1
  serviceName: {{.PostgreSQLStatefulSetName}}
  selector:
    matchLabels:
      app: {{.PostgreSQLStatefulSetName}}
  template:
    metadata:
      labels:
        app: {{.PostgreSQLStatefulSetName}}
    spec:
      containers:
      - image: postgres:10.5
        name: postgresql
        env:
          - name: POSTGRES_USER
            value: {{.PostgreSQLUsername}}
          - name: POSTGRES_PASSWORD
            value: {{.PostgreSQLPassword}}
          - name: POSTGRES_DB
            value: {{.PostgreSQLDatabase}}
        ports:
          - name: postgresql
            protocol: TCP
            containerPort: 5432
`

	postgreSQLServiceTemplate = `
apiVersion: v1
kind: Service
metadata:
  labels:
    app: {{.PostgreSQLStatefulSetName}}
  name: {{.PostgreSQLStatefulSetName}}
  namespace: {{.TestNamespace}}
spec:
  ports:
  - port: 5432
    protocol: TCP
    targetPort: 5432
  selector:
    app: {{.PostgreSQLStatefulSetName}}
  type: ClusterIP
`

	lowLevelRecordsJobTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  labels:
    app: postgresql-insert-low-level-job
  name: postgresql-insert-low-level-job
  namespace: {{.TestNamespace}}
spec:
  template:
    metadata:
      labels:
        app: postgresql-insert-low-level-job
    spec:
      containers:
      - image: ghcr.io/kedacore/tests-postgresql
        imagePullPolicy: Always
        name: postgresql-processor-test
        command:
          - /app
          - insert
        env:
          - name: TASK_INSTANCES_COUNT
            value: "20"
          - name: CONNECTION_STRING
            valueFrom:
              secretKeyRef:
                name: {{.SecretName}}
                key: postgresql_conn_str
      restartPolicy: Never
  backoffLimit: 4
`

	insertRecordsJobTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  labels:
    app: postgresql-insert-job
  name: postgresql-insert-job
  namespace: {{.TestNamespace}}
spec:
  template:
    metadata:
      labels:
        app: postgresql-insert-job
    spec:
      containers:
      - image: ghcr.io/kedacore/tests-postgresql
        imagePullPolicy: Always
        name: postgresql-processor-test
        command:
          - /app
          - insert
        env:
          - name: TASK_INSTANCES_COUNT
            value: "10000"
          - name: CONNECTION_STRING
            valueFrom:
              secretKeyRef:
                name: {{.SecretName}}
                key: postgresql_conn_str
      restartPolicy: Never
  backoffLimit: 4
`

	prometheusDeploymentTemplate = `apiVersion: apps/v1
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
---
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
---
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

	prometheusTriggerAuthenticationTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthenticationName}}
  namespace: {{.TestNamespace}}
spec:
  hashiCorpVault:
    address: http://vault.{{.VaultNamespace}}:8200
    authentication: token
    credential:
      token: {{.HashiCorpToken}}
    secrets:
      - key: "ca_chain"
        parameter: "ca"
        path: {{ .VaultSecretPath }}
        type: pki
        pkiData:
          commonName: {{ .VaultPkiCommonName }}
      - key: "private_key"
        parameter: "key"
        path: {{ .VaultSecretPath }}
        type: pki
        pkiData:
          commonName: {{ .VaultPkiCommonName }}
      - key: "certificate"
        parameter: "cert"
        path: {{ .VaultSecretPath }}
        type: pki
        pkiData:
          commonName: {{ .VaultPkiCommonName }}`
	prometheusScaledObjectTemplate = `apiVersion: keda.sh/v1alpha1
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
      serverAddress: https://{{.PrometheusServerName}}.{{.TestNamespace}}.svc:80
      authModes: "tls"
      metricName: http_requests_total
      threshold: '20'
      activationThreshold: '20'
      query: sum(rate(http_requests_total{app="{{.MonitoredAppName}}"}[2m]))
    authenticationRef:
      name: {{.TriggerAuthenticationName}}
`

	generatePromLowLevelLoadJobTemplate = `apiVersion: batch/v1
kind: Job
metadata:
  name: generate-low-level-requests-job
  namespace: {{.TestNamespace}}
spec:
  template:
    spec:
      containers:
      - image: quay.io/zroubalik/hey
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

	generatePromLoadJobTemplate = `apiVersion: batch/v1
kind: Job
metadata:
  name: generate-requests-job
  namespace: {{.TestNamespace}}
spec:
  template:
    spec:
      containers:
      - image: quay.io/zroubalik/hey
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

func TestPrometheusScalerWithMtls(t *testing.T) {
	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	hashiCorpToken, promPkiData := setupHashiCorpVault(t, kc, 2, true)
	prometheus.Install(t, kc, prometheusServerName, testNamespace, promPkiData)

	// Create kubernetes resources for testing
	data, templates := getPrometheusTemplateData()
	data.HashiCorpToken = RemoveANSI(hashiCorpToken)
	data.VaultSecretPath = fmt.Sprintf("pki/issue/%s", testNamespace)
	KubectlApplyMultipleWithTemplate(t, data, templates)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, monitoredAppName, testNamespace, 1, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)

	testPromActivation(t, kc, data)
	testPromScaleOut(t, kc, data)
	testScaleIn(t, kc)

	// cleanup
	KubectlDeleteMultipleWithTemplate(t, data, templates)
	prometheus.Uninstall(t, prometheusServerName, testNamespace, nil)
}

func TestPostreSQLScaler(t *testing.T) {
	tests := []struct {
		name               string
		vaultEngineVersion uint
		vaultSecretPath    string
	}{
		{
			name:               "vault kv engine v1",
			vaultEngineVersion: 1,
			vaultSecretPath:    "secret/keda",
		},
		{
			name:               "vault kv engine v2",
			vaultEngineVersion: 2,
			vaultSecretPath:    "secret/data/keda",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create kubernetes resources for PostgreSQL server
			kc := GetKubernetesClient(t)
			data, postgreSQLtemplates := getPostgreSQLTemplateData()

			CreateKubernetesResources(t, kc, testNamespace, data, postgreSQLtemplates)
			hashiCorpToken, _ := setupHashiCorpVault(t, kc, test.vaultEngineVersion, false)

			assert.True(t, WaitForStatefulsetReplicaReadyCount(t, kc, postgreSQLStatefulSetName, testNamespace, 1, 60, 3),
				"replica count should be %d after 3 minutes", 1)

			createTableSQL := "CREATE TABLE task_instance (id serial PRIMARY KEY,state VARCHAR(10));"
			psqlCreateTableCmd := fmt.Sprintf("psql -U %s -d %s -c \"%s\"", postgreSQLUsername, postgreSQLDatabase, createTableSQL)

			ok, out, errOut, err := WaitForSuccessfulExecCommandOnSpecificPod(t, postgresqlPodName, testNamespace, psqlCreateTableCmd, 60, 3)
			assert.True(t, ok, "executing a command on PostreSQL Pod should work; Output: %s, ErrorOutput: %s, Error: %s", out, errOut, err)

			// Create kubernetes resources for testing
			data, templates := getTemplateData()
			data.HashiCorpToken = RemoveANSI(hashiCorpToken)
			data.VaultSecretPath = test.vaultSecretPath

			KubectlApplyMultipleWithTemplate(t, data, templates)
			assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
				"replica count should be %d after 3 minutes", minReplicaCount)

			testActivation(t, kc, data)
			testScaleOut(t, kc, data)
			testScaleIn(t, kc)

			// cleanup
			KubectlDeleteMultipleWithTemplate(t, data, templates)
			cleanupHashiCorpVault(t)
			DeleteKubernetesResources(t, testNamespace, data, postgreSQLtemplates)
		})
	}
}

func setupHashiCorpVaultPki(t *testing.T, podName string, nameSpace string) *prometheus.VaultPkiData {
	vaultCommands := []string{
		"vault secrets enable pki",
		"vault secrets tune -max-lease-ttl=8760h pki",
		fmt.Sprintf("vault write pki/root/generate/internal common_name=%s.svc ttl=8760h", testNamespace),
		"vault write pki/config/urls issuing_certificates=\"http://127.0.0.1:8200/v1/pki/ca\" crl_distribution_points=\"http://127.0.0.1:8200/v1/pki/crl\"",
		fmt.Sprintf("vault write pki/roles/%s require_cn=false allowed_domains=%s.svc allow_subdomains=true max_ttl=72h", testNamespace, testNamespace),
	}
	for _, vaultCommand := range vaultCommands {
		_, _, err := ExecCommandOnSpecificPod(
			t,
			podName,
			nameSpace,
			vaultCommand,
		)
		assert.NoErrorf(t, err, "cannot set vault pki command %s - %s", vaultCommand, err)
	}
	rawPkiSecret, _, err := ExecCommandOnSpecificPod(
		t,
		podName,
		nameSpace,
		fmt.Sprintf("vault write pki/issue/%s common_name=%s.%s.svc -format=json", testNamespace, prometheusServerName, testNamespace),
	)
	assert.NoErrorf(t, err, "cannot issue certificate - %s", err)
	var pkiSecret vaultapi.Secret
	err = json.Unmarshal([]byte(rawPkiSecret), &pkiSecret)
	assert.NoErrorf(t, err, "cannot read certificate raw secret - %s", err)
	serverKey := b64.StdEncoding.EncodeToString([]byte(pkiSecret.Data["private_key"].(string)))
	serverCertificate := b64.StdEncoding.EncodeToString([]byte(pkiSecret.Data["certificate"].(string)))
	caCertificate := b64.StdEncoding.EncodeToString([]byte((pkiSecret.Data["ca_chain"].([]interface{}))[0].(string)))
	pkiData := prometheus.VaultPkiData{
		ServerKey:         serverKey,
		ServerCertificate: serverCertificate,
		CaCertificate:     caCertificate,
	}
	return &pkiData
}

func setupHashiCorpVault(t *testing.T, kc *kubernetes.Clientset, kvVersion uint, pki bool) (string, *prometheus.VaultPkiData) {
	CreateNamespace(t, kc, vaultNamespace)

	_, err := ExecuteCommand("helm repo add hashicorp https://helm.releases.hashicorp.com")
	assert.NoErrorf(t, err, "cannot add hashicorp repo - %s", err)

	_, err = ExecuteCommand("helm repo update")
	assert.NoErrorf(t, err, "cannot update repos - %s", err)

	var helmValues strings.Builder
	helmValues.WriteString("--set server.dev.enabled=true")

	if kvVersion == 1 {
		helmValues.WriteString(" --set server.extraArgs=-dev-kv-v1")
	}

	_, err = ExecuteCommand(fmt.Sprintf(`helm upgrade --install %s --namespace %s --wait vault hashicorp/vault`, helmValues.String(), vaultNamespace))
	assert.NoErrorf(t, err, "cannot install hashicorp vault - %s", err)

	podName := "vault-0"

	// Create kv secret
	_, _, err = ExecCommandOnSpecificPod(t, podName, vaultNamespace, fmt.Sprintf("vault kv put secret/keda connectionString=%s", postgreSQLConnectionString))
	assert.NoErrorf(t, err, "cannot put connection string in hashicorp vault - %s", err)

	// Create PKI Backend
	var pkiData *prometheus.VaultPkiData
	if pki {
		pkiData = setupHashiCorpVaultPki(t, podName, vaultNamespace)
	}

	out, _, err := ExecCommandOnSpecificPod(t, podName, vaultNamespace, "vault token create -field token")
	assert.NoErrorf(t, err, "cannot create hashicorp vault token - %s", err)

	return out, pkiData
}

func cleanupHashiCorpVault(t *testing.T) {
	_, err := ExecuteCommand(fmt.Sprintf("helm uninstall vault --namespace %s", vaultNamespace))
	assert.NoErrorf(t, err, "cannot uninstall hashicorp vault - %s", err)

	_, err = ExecuteCommand("helm repo remove hashicorp")
	assert.NoErrorf(t, err, "cannot remove hashicorp repo - %s", err)

	DeleteNamespace(t, vaultNamespace)
}

func testPromActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")
	KubectlReplaceWithTemplate(t, data, "generateLowLevelLoadJobTemplate", generatePromLowLevelLoadJobTemplate)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testPromScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	KubectlReplaceWithTemplate(t, data, "generateLoadJobTemplate", generatePromLoadJobTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")
	KubectlReplaceWithTemplate(t, data, "lowLevelRecordsJobTemplate", lowLevelRecordsJobTemplate)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	KubectlReplaceWithTemplate(t, data, "insertRecordsJobTemplate", insertRecordsJobTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 5),
		"replica count should be %d after 5 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 5),
		"replica count should be %d after 5 minutes", minReplicaCount)
}

var data = templateData{
	TestNamespace:                    testNamespace,
	PostgreSQLStatefulSetName:        postgreSQLStatefulSetName,
	DeploymentName:                   deploymentName,
	ScaledObjectName:                 scaledObjectName,
	MinReplicaCount:                  minReplicaCount,
	MaxReplicaCount:                  maxReplicaCount,
	TriggerAuthenticationName:        triggerAuthenticationName,
	SecretName:                       secretName,
	PostgreSQLUsername:               postgreSQLUsername,
	PostgreSQLPassword:               postgreSQLPassword,
	PostgreSQLDatabase:               postgreSQLDatabase,
	PostgreSQLConnectionStringBase64: b64.StdEncoding.EncodeToString([]byte(postgreSQLConnectionString)),
	PrometheusServerName:             prometheusServerName,
	MonitoredAppName:                 monitoredAppName,
	PublishDeploymentName:            publishDeploymentName,
	VaultNamespace:                   vaultNamespace,
	VaultPromDomain:                  vaultPromDomain,
	VaultPkiCommonName:               fmt.Sprintf("keda.%s.svc", testNamespace),
}

func getPostgreSQLTemplateData() (templateData, []Template) {
	return data, []Template{
		{Name: "postgreSQLStatefulSetTemplate", Config: postgreSQLStatefulSetTemplate},
		{Name: "postgreSQLServiceTemplate", Config: postgreSQLServiceTemplate},
	}
}

func getPrometheusTemplateData() (templateData, []Template) {
	return data, []Template{
		{Name: "triggerAuthenticationTemplate", Config: prometheusTriggerAuthenticationTemplate},
		{Name: "deploymentTemplate", Config: prometheusDeploymentTemplate},
		{Name: "monitoredAppDeploymentTemplate", Config: monitoredAppDeploymentTemplate},
		{Name: "monitoredAppServiceTemplate", Config: monitoredAppServiceTemplate},
		{Name: "scaledObjectTemplate", Config: prometheusScaledObjectTemplate},
	}
}

func getTemplateData() (templateData, []Template) {
	return data, []Template{
		{Name: "secretTemplate", Config: secretTemplate},
		{Name: "deploymentTemplate", Config: deploymentTemplate},
		{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
		{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
	}
}
