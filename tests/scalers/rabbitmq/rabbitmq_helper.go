//go:build e2e
// +build e2e

package rabbitmq

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	"github.com/kedacore/keda/v2/tests/helper"
)

const (
	publishTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  name: rabbitmq-publish-{{.QueueName}}
  namespace: {{.Namespace}}
spec:
  template:
    spec:
      containers:
      - name: rabbitmq-client
        image: ghcr.io/kedacore/tests-rabbitmq
        imagePullPolicy: Always
        command:
          - send
        args:
          - '{{.Connection}}'
          - '{{.MessageCount}}'
          - '{{.QueueName}}'
          - '{{.Interval}}'
      restartPolicy: Never
`

	consumeTemplateJobCount = 4
	consumeTemplate         = `
apiVersion: batch/v1
kind: Job
metadata:
  name: rabbitmq-consume-0-{{.QueueName}}
  namespace: {{.Namespace}}
spec:
  template:
    spec:
      containers:
      - name: rabbitmq-client
        image: ghcr.io/kedacore/tests-rabbitmq
        imagePullPolicy: Always
        command:
          - receive
        args:
          - '{{.Connection}}'
      restartPolicy: Never
---
apiVersion: batch/v1
kind: Job
metadata:
  name: rabbitmq-consume-1-{{.QueueName}}
  namespace: {{.Namespace}}
spec:
  template:
    spec:
      containers:
      - name: rabbitmq-client
        image: ghcr.io/kedacore/tests-rabbitmq
        imagePullPolicy: Always
        command:
          - receive
        args:
          - '{{.Connection}}'
      restartPolicy: Never
---
apiVersion: batch/v1
kind: Job
metadata:
  name: rabbitmq-consume-2-{{.QueueName}}
  namespace: {{.Namespace}}
spec:
  template:
    spec:
      containers:
      - name: rabbitmq-client
        image: ghcr.io/kedacore/tests-rabbitmq
        imagePullPolicy: Always
        command:
          - receive
        args:
          - '{{.Connection}}'
      restartPolicy: Never
---
apiVersion: batch/v1
kind: Job
metadata:
  name: rabbitmq-consume-3-{{.QueueName}}
  namespace: {{.Namespace}}
spec:
  template:
    spec:
      containers:
      - name: rabbitmq-client
        image: ghcr.io/kedacore/tests-rabbitmq
        imagePullPolicy: Always
        command:
          - receive
        args:
          - '{{.Connection}}'
      restartPolicy: Never
`

	vHostTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  name: rabbitmq-create-vhost-{{.VHostName}}
  namespace: {{.Namespace}}
spec:
  template:
    spec:
      containers:
      - name: curl-client
        image: docker.io/curlimages/curl
        imagePullPolicy: Always
        command: ["curl", "-u", "{{.Username}}:{{.Password}}", "-X", "PUT", "http://{{.HostName}}/api/vhosts/{{.VHostName}}"]
      restartPolicy: Never
`

	deploymentTemplate = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: rabbitmq-config
  namespace: {{.Namespace}}
data:
  rabbitmq.conf: |
    default_user = {{.Username}}
    default_pass = {{.Password}}
    default_vhost = {{.VHostName}}
    management.tcp.port = 15672
    management.tcp.ip = 0.0.0.0
    {{if .EnableOAuth}}
    auth_backends.1 = rabbit_auth_backend_internal
    auth_backends.2 = rabbit_auth_backend_oauth2
    auth_backends.3 = rabbit_auth_backend_amqp
    auth_oauth2.resource_server_id = {{.OAuthClientID}}
    auth_oauth2.scope_prefix = rabbitmq.
    auth_oauth2.additional_scopes_key = {{.OAuthScopesKey}}
    auth_oauth2.jwks_url = {{.OAuthJwksURI}}
    {{end}}
  enabled_plugins: |
    [rabbitmq_management].
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: rabbitmq
  name: rabbitmq
  namespace: {{.Namespace}}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: rabbitmq
  template:
    metadata:
      labels:
        app: rabbitmq
      namespace: {{.Namespace}}
    spec:
      containers:
      - image: docker.io/library/rabbitmq:3.12-management
        name: rabbitmq
        volumeMounts:
          - mountPath: /etc/rabbitmq
            name: rabbitmq-config
        readinessProbe:
          tcpSocket:
            port: 5672
          initialDelaySeconds: 5
          periodSeconds: 10
      volumes:
        - name: rabbitmq-config
          configMap:
            name: rabbitmq-config
---
apiVersion: v1
kind: Service
metadata:
  labels:
    app: rabbitmq
  name: rabbitmq
  namespace: {{.Namespace}}
spec:
  ports:
  - name: amqp
    port: 5672
    protocol: TCP
    targetPort: 5672
  - name: http
    port: 80
    protocol: TCP
    targetPort: 15672
  selector:
    app: rabbitmq
`

	RMQTargetDeploymentTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  RabbitApiHost: {{.Base64Connection}}
---
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
      - name: rabbitmq-consumer
        image: ghcr.io/kedacore/tests-rabbitmq
        imagePullPolicy: Always
        command:
          - receive
        args:
          - '{{.Connection}}'
        envFrom:
        - secretRef:
            name: {{.SecretName}}
`

	RMQPublisherTargetDeploymentTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  RabbitApiHost: {{.Base64Connection}}
---
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
      - name: rabbitmq-publisher
        image: ghcr.io/kedacore/tests-rabbitmq
        imagePullPolicy: Always
        command:
          - send
        args:
          - '{{.Connection}}'
          - '4'
          - '{{.QueueName}}'
          - '1'
        envFrom:
        - secretRef:
            name: {{.SecretName}}
`

	RMQTargetDeploymentWithAuthEnvTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  RabbitApiHost: {{.Base64Connection}}
  RabbitUsername: {{.Base64Username}}
  RabbitPassword: {{.Base64Password}}
---
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
      - name: rabbitmq-consumer
        image: ghcr.io/kedacore/tests-rabbitmq
        imagePullPolicy: Always
        command:
          - receive
        args:
          - '{{.Connection}}'
        envFrom:
        - secretRef:
            name: {{.SecretName}}
`
)

const RabbitServerName string = "rabbitmq"

type RabbitOAuthConfig struct {
	Enable    bool
	ClientID  string
	ScopesKey string
	JwksURI   string
}

func WithoutOAuth() RabbitOAuthConfig {
	return RabbitOAuthConfig{
		Enable: false,
	}
}

func WithAzureADOAuth(tenantID string, clientID string) RabbitOAuthConfig {
	return RabbitOAuthConfig{
		Enable:    true,
		ClientID:  clientID,
		ScopesKey: "roles",
		JwksURI:   fmt.Sprintf("https://login.microsoftonline.com/%s/discovery/keys", tenantID),
	}
}

type templateData struct {
	Namespace           string
	Connection          string
	QueueName           string
	HostName, VHostName string
	Username, Password  string
	MessageCount        int
	Interval            int
	EnableOAuth         bool
	OAuthClientID       string
	OAuthScopesKey      string
	OAuthJwksURI        string
}

func RMQInstall(t *testing.T, kc *kubernetes.Clientset, namespace, user, password, vhost string, oauth RabbitOAuthConfig) {
	helper.CreateNamespace(t, kc, namespace)
	data := templateData{
		Namespace:      namespace,
		VHostName:      vhost,
		Username:       user,
		Password:       password,
		EnableOAuth:    oauth.Enable,
		OAuthClientID:  oauth.ClientID,
		OAuthScopesKey: oauth.ScopesKey,
		OAuthJwksURI:   oauth.JwksURI,
	}

	helper.KubectlApplyWithTemplate(t, data, "rmqDeploymentTemplate", deploymentTemplate)
	require.True(t, helper.WaitForDeploymentReplicaReadyCount(t, kc, RabbitServerName, namespace, 1, 180, 1),
		"replica count should be 1 after 3 minute")
}

func RMQUninstall(t *testing.T, namespace, user, password, vhost string, oauth RabbitOAuthConfig) {
	data := templateData{
		Namespace:      namespace,
		VHostName:      vhost,
		Username:       user,
		Password:       password,
		EnableOAuth:    oauth.Enable,
		OAuthClientID:  oauth.ClientID,
		OAuthScopesKey: oauth.ScopesKey,
		OAuthJwksURI:   oauth.JwksURI,
	}

	helper.KubectlDeleteWithTemplate(t, data, "rmqDeploymentTemplate", deploymentTemplate)
	helper.DeleteNamespace(t, namespace)
}

func RMQPublishMessages(t *testing.T, namespace, connectionString, queueName string, messageCount, interval int) {
	data := templateData{
		Namespace:    namespace,
		Connection:   connectionString,
		QueueName:    queueName,
		MessageCount: messageCount,
		Interval:     interval,
	}

	// Before pushing new messages, remove all previous publishing jobs, if any.
	RMQStopPublishingMessages(namespace, queueName)

	helper.KubectlApplyWithTemplate(t, data, "rmqPublishTemplate", publishTemplate)
}

func RMQStopPublishingMessages(namespace, queueName string) {
	_, _ = helper.ExecuteCommand(fmt.Sprintf("kubectl delete jobs/rabbitmq-publish-%s --namespace %s", queueName, namespace))
}

func RMQConsumeMessages(t *testing.T, namespace, connectionString, queueName string) {
	data := templateData{
		Namespace:  namespace,
		Connection: connectionString,
		QueueName:  queueName,
	}

	// Before consuming messages, remove all previous consumer jobs, if any.
	RMQStopConsumingMessages(namespace, queueName)

	helper.KubectlApplyWithTemplate(t, data, "rmqConsumerTemplate", consumeTemplate)
}

func RMQStopConsumingMessages(namespace, queueName string) {
	for i := 0; i < consumeTemplateJobCount; i++ {
		_, _ = helper.ExecuteCommand(fmt.Sprintf("kubectl delete jobs/rabbitmq-consume-%d-%s --namespace %s", i, queueName, namespace))
	}
}

func RMQCreateVHost(t *testing.T, namespace, host, user, password, vhost string) {
	data := templateData{
		Namespace: namespace,
		HostName:  host,
		VHostName: vhost,
		Username:  user,
		Password:  password,
	}

	helper.KubectlApplyWithTemplate(t, data, "rmqVHostTemplate", vHostTemplate)
}
