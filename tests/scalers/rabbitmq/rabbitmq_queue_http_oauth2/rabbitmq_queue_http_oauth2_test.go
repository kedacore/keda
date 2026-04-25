//go:build e2e
// +build e2e

package rabbitmq_queue_http_oauth2_test

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
	. "github.com/kedacore/keda/v2/tests/scalers/rabbitmq"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../../../.env")

const (
	testName = "rmq-queue-http-oauth2"
)

var (
	testNamespace        = fmt.Sprintf("%s-ns", testName)
	rmqNamespace         = fmt.Sprintf("%s-rmq", testName)
	keycloakNamespace    = fmt.Sprintf("%s-kc", testName)
	deploymentName       = fmt.Sprintf("%s-deployment", testName)
	secretName           = fmt.Sprintf("%s-secret", testName)
	triggerAuthName      = fmt.Sprintf("%s-ta", testName)
	triggerSecretName    = fmt.Sprintf("%s-ta-secret", testName)
	scaledObjectName     = fmt.Sprintf("%s-so", testName)
	queueName            = "hello"
	user                 = fmt.Sprintf("%s-user", testName)
	password             = fmt.Sprintf("%s-password", testName)
	vhost                = "/"
	connectionString     = fmt.Sprintf("amqp://%s:%s@rabbitmq.%s.svc.cluster.local/", user, password, rmqNamespace)
	httpConnectionString = fmt.Sprintf("http://%s:%s@rabbitmq.%s.svc.cluster.local/", user, password, rmqNamespace)
	messageCount         = 100

	// Keycloak / OAuth2 settings
	realmName     = "rabbitmq"
	oauthClientID = "rabbitmq-keda"
	clientSecret  = "rabbitmq-keda-secret"
	tokenURL      = fmt.Sprintf("http://keycloak.%s.svc.cluster.local:8080/realms/%s/protocol/openid-connect/token", keycloakNamespace, realmName)
)

const (
	keycloakDeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: keycloak
  namespace: {{.KeycloakNamespace}}
  labels:
    app: keycloak
spec:
  replicas: 1
  selector:
    matchLabels:
      app: keycloak
  template:
    metadata:
      labels:
        app: keycloak
    spec:
      initContainers:
      - name: tls-gen
        image: docker.io/library/alpine:3.19
        command:
        - sh
        - -c
        - |
          apk add --no-cache openssl
          openssl req -x509 -newkey rsa:2048 -keyout /tls/tls.key -out /tls/tls.crt \
            -days 365 -nodes -subj "/CN=keycloak.{{.KeycloakNamespace}}.svc.cluster.local" \
            -addext "subjectAltName=DNS:keycloak.{{.KeycloakNamespace}}.svc.cluster.local,DNS:keycloak"
          chmod 644 /tls/tls.key /tls/tls.crt
        volumeMounts:
        - name: tls-certs
          mountPath: /tls
      containers:
      - name: keycloak
        image: quay.io/keycloak/keycloak:26.0
        args:
        - start-dev
        - --import-realm
        - --https-certificate-file=/tls/tls.crt
        - --https-certificate-key-file=/tls/tls.key
        - --hostname-strict=false
        env:
        - name: KC_HEALTH_ENABLED
          value: "true"
        - name: KC_BOOTSTRAP_ADMIN_USERNAME
          value: admin
        - name: KC_BOOTSTRAP_ADMIN_PASSWORD
          value: admin
        ports:
        - containerPort: 8443
        - containerPort: 8080
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 9000
            scheme: HTTPS
          initialDelaySeconds: 30
          periodSeconds: 10
        volumeMounts:
        - name: realm-config
          mountPath: /opt/keycloak/data/import
        - name: tls-certs
          mountPath: /tls
          readOnly: true
      volumes:
      - name: realm-config
        configMap:
          name: keycloak-realm
      - name: tls-certs
        emptyDir: {}
---
apiVersion: v1
kind: Service
metadata:
  name: keycloak
  namespace: {{.KeycloakNamespace}}
  labels:
    app: keycloak
spec:
  ports:
  - name: https
    port: 8443
    targetPort: 8443
  - name: http
    port: 8080
    targetPort: 8080
  selector:
    app: keycloak
`

	keycloakRealmConfigMapTemplate = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: keycloak-realm
  namespace: {{.KeycloakNamespace}}
data:
  realm.json: |
    {
      "realm": "{{.RealmName}}",
      "enabled": true,
      "clients": [
        {
          "clientId": "{{.OAuthClientID}}",
          "enabled": true,
          "clientAuthenticatorType": "client-secret",
          "secret": "{{.ClientSecret}}",
          "serviceAccountsEnabled": true,
          "directAccessGrantsEnabled": true,
          "publicClient": false,
          "protocol": "openid-connect",
          "defaultClientScopes": [
            "{{.OAuthClientID}}"
          ]
        }
      ],
      "clientScopes": [
        {
          "name": "{{.OAuthClientID}}",
          "protocol": "openid-connect",
          "attributes": {
            "include.in.token.scope": "true"
          },
          "protocolMappers": [
            {
              "name": "audience-mapper",
              "protocol": "openid-connect",
              "protocolMapper": "oidc-audience-mapper",
              "config": {
                "included.client.audience": "{{.OAuthClientID}}",
                "id.token.claim": "true",
                "access.token.claim": "true"
              }
            },
            {
              "name": "rabbitmq-permissions",
              "protocol": "openid-connect",
              "protocolMapper": "oidc-hardcoded-claim-mapper",
              "config": {
                "claim.name": "rabbitmq_permissions",
                "claim.value": "[\"rabbitmq.tag:administrator\", \"rabbitmq.read:*/*\", \"rabbitmq.write:*/*\", \"rabbitmq.configure:*/*\"]",
                "jsonType.label": "JSON",
                "id.token.claim": "true",
                "access.token.claim": "true",
                "userinfo.token.claim": "true"
              }
            }
          ]
        }
      ]
    }
`

	keycloakVerifyRealmJobTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  name: keycloak-verify-realm
  namespace: {{.KeycloakNamespace}}
spec:
  backoffLimit: 10
  template:
    spec:
      containers:
      - name: curl
        image: docker.io/curlimages/curl
        command: ["curl", "-ksf", "https://keycloak.{{.KeycloakNamespace}}.svc.cluster.local:8443/realms/{{.RealmName}}/.well-known/openid-configuration"]
      restartPolicy: Never
`

	keycloakVerifyTokenJobTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  name: keycloak-verify-token
  namespace: {{.KeycloakNamespace}}
spec:
  backoffLimit: 10
  template:
    spec:
      containers:
      - name: curl
        image: docker.io/curlimages/curl
        command:
        - sh
        - -c
        - |
          TOKEN=$(curl -ksf -X POST \
            "https://keycloak.{{.KeycloakNamespace}}.svc.cluster.local:8443/realms/{{.RealmName}}/protocol/openid-connect/token" \
            -H "Content-Type: application/x-www-form-urlencoded" \
            -d "grant_type=client_credentials" \
            -d "client_id={{.OAuthClientID}}" \
            -d "client_secret={{.ClientSecret}}" \
            -d "scope={{.OAuthClientID}}")
          echo "Token response: $TOKEN"
          echo "$TOKEN" | grep -q "access_token"
      restartPolicy: Never
`

	oauthSecretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.TriggerSecretName}}
  namespace: {{.TestNamespace}}
type: Opaque
data:
  clientSecret: {{.Base64ClientSecret}}
`

	triggerAuthTemplate = `
apiVersion: keda.sh/v1alpha1
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
    tokenUrl: {{.TokenURL}}
    scopes:
    - {{.OAuthClientID}}
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
  maxReplicaCount: 4
  triggers:
    - type: rabbitmq
      metadata:
        queueName: {{.QueueName}}
        host: {{.HttpNoAuthConnection}}
        protocol: http
        mode: QueueLength
        value: '10'
      authenticationRef:
        name: {{.TriggerAuthName}}
`
)

type templateData struct {
	TestNamespace        string
	KeycloakNamespace    string
	DeploymentName       string
	TriggerAuthName      string
	TriggerSecretName    string
	ScaledObjectName     string
	SecretName           string
	QueueName            string
	Connection           string
	Base64Connection     string
	HttpNoAuthConnection string
	RealmName            string
	OAuthClientID        string
	ClientSecret         string
	Base64ClientSecret   string
	TokenURL             string
}

func TestScaler(t *testing.T) {
	t.Log("--- setting up ---")
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	// Setup Keycloak
	CreateNamespace(t, kc, keycloakNamespace)
	kcData := data // reuse templateData for keycloak templates
	KubectlApplyWithTemplate(t, kcData, "keycloakRealmConfigMapTemplate", keycloakRealmConfigMapTemplate)
	KubectlApplyWithTemplate(t, kcData, "keycloakDeploymentTemplate", keycloakDeploymentTemplate)
	require.True(t, WaitForDeploymentReplicaReadyCount(t, kc, "keycloak", keycloakNamespace, 1, 180, 1),
		"keycloak should be ready after 3 minutes")

	// Verify Keycloak realm is accessible
	t.Log("--- verifying keycloak realm ---")
	KubectlApplyWithTemplate(t, data, "keycloakVerifyRealmJobTemplate", keycloakVerifyRealmJobTemplate)
	require.True(t, WaitForJobSuccess(t, kc, "keycloak-verify-realm", keycloakNamespace, 30, 10),
		"keycloak realm should be accessible")

	// Verify OAuth2 client credentials flow works
	t.Log("--- verifying keycloak token endpoint ---")
	KubectlApplyWithTemplate(t, data, "keycloakVerifyTokenJobTemplate", keycloakVerifyTokenJobTemplate)
	require.True(t, WaitForJobSuccess(t, kc, "keycloak-verify-token", keycloakNamespace, 30, 10),
		"keycloak client credentials token request should succeed")

	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
		RMQUninstall(t, rmqNamespace, user, password, vhost, WithKeycloakOAuth(oauthClientID, keycloakNamespace, realmName))
		KubectlDeleteWithTemplate(t, kcData, "keycloakDeploymentTemplate", keycloakDeploymentTemplate)
		KubectlDeleteWithTemplate(t, kcData, "keycloakRealmConfigMapTemplate", keycloakRealmConfigMapTemplate)
		DeleteNamespace(t, keycloakNamespace)
	})

	// Setup RabbitMQ with OAuth2 pointing to Keycloak JWKS
	RMQInstall(t, kc, rmqNamespace, user, password, vhost, WithKeycloakOAuth(oauthClientID, keycloakNamespace, realmName))

	// Create KEDA resources (secret, trigger auth, scaled object, target deployment)
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")

	testScaling(t, kc)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:        testNamespace,
			KeycloakNamespace:    keycloakNamespace,
			DeploymentName:       deploymentName,
			ScaledObjectName:     scaledObjectName,
			SecretName:           secretName,
			QueueName:            queueName,
			Connection:           connectionString,
			Base64Connection:     base64.StdEncoding.EncodeToString([]byte(httpConnectionString)),
			HttpNoAuthConnection: fmt.Sprintf("http://rabbitmq.%s.svc.cluster.local/", rmqNamespace),
			TriggerAuthName:      triggerAuthName,
			TriggerSecretName:    triggerSecretName,
			RealmName:            realmName,
			OAuthClientID:        oauthClientID,
			ClientSecret:         clientSecret,
			Base64ClientSecret:   base64.StdEncoding.EncodeToString([]byte(clientSecret)),
			TokenURL:             tokenURL,
		}, []Template{
			{Name: "deploymentTemplate", Config: RMQTargetDeploymentTemplate},
			{Name: "oauthSecretTemplate", Config: oauthSecretTemplate},
			{Name: "triggerAuthTemplate", Config: triggerAuthTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

func testScaling(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale out ---")
	RMQPublishMessages(t, rmqNamespace, connectionString, queueName, messageCount, 0)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 4, 60, 3),
		"replica count should be 4 after 3 minutes")

	t.Log("--- testing scale in ---")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")
}
