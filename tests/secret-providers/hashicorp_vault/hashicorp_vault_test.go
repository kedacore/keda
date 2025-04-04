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
	prometheusServerName                   = fmt.Sprintf("%s-prom-server", testName)
	minReplicaCount                        = 0
	maxReplicaCount                        = 1
	serviceAccountTokenCreationRole        = fmt.Sprintf("%s-sa-role", testName)
	serviceAccountTokenCreationRoleBinding = fmt.Sprintf("%s-sa-role-binding", testName)
)

type templateData struct {
	TestNamespace                          string
	DeploymentName                         string
	VaultNamespace                         string
	ScaledObjectName                       string
	TriggerAuthenticationName              string
	VaultSecretPath                        string
	VaultPromDomain                        string
	SecretName                             string
	HashiCorpAuthentication                string
	HashiCorpToken                         string
	PostgreSQLStatefulSetName              string
	PostgreSQLConnectionStringBase64       string
	PostgreSQLUsername                     string
	PostgreSQLPassword                     string
	PostgreSQLDatabase                     string
	MinReplicaCount                        int
	MaxReplicaCount                        int
	PublishDeploymentName                  string
	MonitoredAppName                       string
	PrometheusServerName                   string
	VaultPkiCommonName                     string
	VaultRole                              string
	VaultServiceAccountName                string
	ServiceAccountTokenCreationRole        string
	ServiceAccountTokenCreationRoleBinding string
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
    authentication: {{.HashiCorpAuthentication}}
    role: {{.VaultRole}}
    mount: kubernetes
    credential:
      token: {{.HashiCorpToken}}
      serviceAccountName: {{.VaultServiceAccountName}}
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
    authentication: {{.HashiCorpAuthentication}}
    role: keda
    mount: kubernetes
    credential:
      token: {{.HashiCorpToken}}
      serviceAccount: /var/run/secrets/kubernetes.io/serviceaccount/token
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
      threshold: '20'
      query: http_requests_total{app="{{.MonitoredAppName}}"}
    authenticationRef:
      name: {{.TriggerAuthenticationName}}
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
      - image: ghcr.io/kedacore/tests-hey:latest
        name: test
        command: ["/bin/sh"]
        args: ["-c", "for i in $(seq 1 60);do echo $i;/hey -c 15 -n 240 http://{{.MonitoredAppName}}.{{.TestNamespace}}.svc;sleep 1;done"]
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
	pkiPolicyTemplate = `path "pki*" {
  capabilities = [ "create", "read", "update", "delete", "list", "sudo" ]
}`

	secretReadPolicyTemplate = `path "secret/data/keda" {
    capabilities = ["read"]
}
path "secret/metadata/keda" {
    capabilities = ["read", "list"]
}`

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
)

func TestPkiSecretsEngine(t *testing.T) {
	tests := []struct {
		authentication string
	}{
		{
			authentication: "kubernetes",
		},
		{
			authentication: "token",
		},
	}

	for _, test := range tests {
		t.Run(test.authentication, func(t *testing.T) {
			// Create kubernetes resources
			kc := GetKubernetesClient(t)
			useKubernetesAuth := test.authentication == "kubernetes"
			hashiCorpToken, promPkiData := setupHashiCorpVault(t, kc, 2, useKubernetesAuth, true, false)
			prometheus.Install(t, kc, prometheusServerName, testNamespace, promPkiData)

			// Create kubernetes resources for testing
			data, templates := getPrometheusTemplateData()
			data.HashiCorpAuthentication = test.authentication
			data.HashiCorpToken = RemoveANSI(hashiCorpToken)
			data.VaultSecretPath = fmt.Sprintf("pki/issue/%s", testNamespace)
			KubectlApplyMultipleWithTemplate(t, data, templates)
			assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, monitoredAppName, testNamespace, 1, 60, 3),
				"replica count should be %d after 3 minutes", minReplicaCount)
			assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
				"replica count should be %d after 3 minutes", minReplicaCount)

			testPromScaleOut(t, kc, data)

			// cleanup
			KubectlDeleteMultipleWithTemplate(t, data, templates)
			prometheus.Uninstall(t, prometheusServerName, testNamespace, nil)
		})
	}
}

func TestSecretsEngine(t *testing.T) {
	tests := []struct {
		name               string
		vaultEngineVersion uint
		vaultSecretPath    string
		useKubernetesAuth  bool
		useDelegatesSAAuth bool
	}{
		{
			name:               "vault kv engine v1",
			vaultEngineVersion: 1,
			vaultSecretPath:    "secret/keda",
			useKubernetesAuth:  false,
			useDelegatesSAAuth: false,
		},
		{
			name:               "vault kv engine v2",
			vaultEngineVersion: 2,
			vaultSecretPath:    "secret/data/keda",
			useKubernetesAuth:  false,
			useDelegatesSAAuth: false,
		},
		{
			name:               "vault kv engine v2",
			vaultEngineVersion: 2,
			vaultSecretPath:    "secret/data/keda",
			useKubernetesAuth:  true,
			useDelegatesSAAuth: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create kubernetes resources for PostgreSQL server
			kc := GetKubernetesClient(t)
			data, postgreSQLtemplates := getPostgreSQLTemplateData()

			CreateKubernetesResources(t, kc, testNamespace, data, postgreSQLtemplates)
			hashiCorpToken, _ := setupHashiCorpVault(t, kc, test.vaultEngineVersion, test.useKubernetesAuth, false, test.useDelegatesSAAuth)

			assert.True(t, WaitForStatefulsetReplicaReadyCount(t, kc, postgreSQLStatefulSetName, testNamespace, 1, 60, 3),
				"replica count should be %d after 3 minutes", 1)

			createTableSQL := "CREATE TABLE task_instance (id serial PRIMARY KEY,state VARCHAR(10));"
			psqlCreateTableCmd := fmt.Sprintf("psql -U %s -d %s -c \"%s\"", postgreSQLUsername, postgreSQLDatabase, createTableSQL)

			ok, out, errOut, err := WaitForSuccessfulExecCommandOnSpecificPod(t, postgresqlPodName, testNamespace, psqlCreateTableCmd, 60, 3)
			assert.True(t, ok, "executing a command on PostreSQL Pod should work; Output: %s, ErrorOutput: %s, Error: %s", out, errOut, err)

			// Create kubernetes resources for testing
			data, templates := getTemplateData()
			data.VaultSecretPath = test.vaultSecretPath
			data.VaultRole = "keda"
			if test.useKubernetesAuth {
				data.HashiCorpAuthentication = "kubernetes"
			} else {
				data.HashiCorpAuthentication = "token"
				data.HashiCorpToken = RemoveANSI(hashiCorpToken)
			}

			if test.useDelegatesSAAuth {
				data.VaultRole = "vault-delegated-sa"
				data.VaultServiceAccountName = "default"
			}

			KubectlApplyMultipleWithTemplate(t, data, templates)
			assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
				"replica count should be %d after 3 minutes", minReplicaCount)

			testScaleOut(t, kc, data)

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

func setupHashiCorpVault(t *testing.T, kc *kubernetes.Clientset, kvVersion uint, useKubernetesAuth, pki, delegatedAuth bool) (string, *prometheus.VaultPkiData) {
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

	// Enable Kubernetes auth
	if useKubernetesAuth {
		if pki {
			remoteFile := "/tmp/pki_policy.hcl"
			KubectlCopyToPod(t, pkiPolicyTemplate, remoteFile, podName, vaultNamespace)
			assert.NoErrorf(t, err, "cannot create policy file in hashicorp vault - %s", err)
			_, _, err = ExecCommandOnSpecificPod(t, podName, vaultNamespace, fmt.Sprintf("vault policy write pkiPolicy %s", remoteFile))
			assert.NoErrorf(t, err, "cannot create policy in hashicorp vault - %s", err)
		}
		_, _, err = ExecCommandOnSpecificPod(t, podName, vaultNamespace, "vault auth enable kubernetes")
		assert.NoErrorf(t, err, "cannot enable kubernetes in hashicorp vault - %s", err)
		_, _, err = ExecCommandOnSpecificPod(t, podName, vaultNamespace, "vault write auth/kubernetes/config kubernetes_host=https://$KUBERNETES_SERVICE_HOST:$KUBERNETES_SERVICE_PORT")
		assert.NoErrorf(t, err, "cannot set kubernetes host in hashicorp vault - %s", err)
		_, _, err = ExecCommandOnSpecificPod(t, podName, vaultNamespace, "vault write auth/kubernetes/role/keda bound_service_account_names=keda-operator bound_service_account_namespaces=keda policies=pkiPolicy ttl=1h")
		assert.NoErrorf(t, err, "cannot cerate keda role in hashicorp vault - %s", err)
		if delegatedAuth {
			remoteFile := "/tmp/secret_read_policy.hcl"
			KubectlCopyToPod(t, secretReadPolicyTemplate, remoteFile, podName, vaultNamespace)
			assert.NoErrorf(t, err, "cannot create policy file in hashicorp vault - %s", err)
			_, _, err = ExecCommandOnSpecificPod(t, podName, vaultNamespace, fmt.Sprintf("vault policy write secretReadPolicy %s", remoteFile))
			assert.NoErrorf(t, err, "cannot create policy in hashicorp vault - %s", err)

			_, _, err = ExecCommandOnSpecificPod(t, podName, vaultNamespace, fmt.Sprintf("vault write auth/kubernetes/role/vault-delegated-sa bound_service_account_names=default bound_service_account_namespaces=%s policies=secretReadPolicy ttl=1h", testNamespace))
			assert.NoErrorf(t, err, "cannot cerate keda role in hashicorp vault - %s", err)
		}
	}

	// Create kv secret
	if !pki {
		_, _, err = ExecCommandOnSpecificPod(t, podName, vaultNamespace, fmt.Sprintf("vault kv put secret/keda connectionString=%s", postgreSQLConnectionString))
		assert.NoErrorf(t, err, "cannot put connection string in hashicorp vault - %s", err)
	}

	// Create PKI Backend
	var pkiData *prometheus.VaultPkiData
	if pki {
		pkiData = setupHashiCorpVaultPki(t, podName, vaultNamespace)
	}

	// Generate Hashicorp Token
	token := "INVALID"
	if !useKubernetesAuth {
		token, _, err = ExecCommandOnSpecificPod(t, podName, vaultNamespace, "vault token create -field token")
		assert.NoErrorf(t, err, "cannot create hashicorp vault token - %s", err)
	}
	return token, pkiData
}

func cleanupHashiCorpVault(t *testing.T) {
	_, err := ExecuteCommand(fmt.Sprintf("helm uninstall vault --namespace %s", vaultNamespace))
	assert.NoErrorf(t, err, "cannot uninstall hashicorp vault - %s", err)

	_, err = ExecuteCommand("helm repo remove hashicorp")
	assert.NoErrorf(t, err, "cannot remove hashicorp repo - %s", err)

	DeleteNamespace(t, vaultNamespace)
}

func testPromScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	KubectlReplaceWithTemplate(t, data, "generateLoadJobTemplate", generatePromLoadJobTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	KubectlReplaceWithTemplate(t, data, "insertRecordsJobTemplate", insertRecordsJobTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 5),
		"replica count should be %d after 5 minutes", maxReplicaCount)
}

var data = templateData{
	TestNamespace:                          testNamespace,
	PostgreSQLStatefulSetName:              postgreSQLStatefulSetName,
	DeploymentName:                         deploymentName,
	ScaledObjectName:                       scaledObjectName,
	MinReplicaCount:                        minReplicaCount,
	MaxReplicaCount:                        maxReplicaCount,
	TriggerAuthenticationName:              triggerAuthenticationName,
	SecretName:                             secretName,
	PostgreSQLUsername:                     postgreSQLUsername,
	PostgreSQLPassword:                     postgreSQLPassword,
	PostgreSQLDatabase:                     postgreSQLDatabase,
	PostgreSQLConnectionStringBase64:       b64.StdEncoding.EncodeToString([]byte(postgreSQLConnectionString)),
	PrometheusServerName:                   prometheusServerName,
	MonitoredAppName:                       monitoredAppName,
	PublishDeploymentName:                  publishDeploymentName,
	VaultNamespace:                         vaultNamespace,
	VaultPromDomain:                        vaultPromDomain,
	VaultPkiCommonName:                     fmt.Sprintf("keda.%s.svc", testNamespace),
	ServiceAccountTokenCreationRole:        serviceAccountTokenCreationRole,
	ServiceAccountTokenCreationRoleBinding: serviceAccountTokenCreationRoleBinding,
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
		// required for the keda to request token creations for the service account
		{Name: "serviceAccountTokenCreationRoleTemplate", Config: serviceAccountTokenCreationRoleTemplate},
		{Name: "serviceAccountTokenCreationRoleBindingTemplate", Config: serviceAccountTokenCreationRoleBindingTemplate},
	}
}
