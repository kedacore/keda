//go:build e2e
// +build e2e

package rabbitmq

import (
	"fmt"
	"testing"

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
        command: ["send",  "{{.Connection}}", "{{.MessageCount}}", "{{.QueueName}}"]
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
        image: curlimages/curl
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
      - image: rabbitmq:3-management
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
)

type templateData struct {
	Namespace           string
	Connection          string
	QueueName           string
	HostName, VHostName string
	Username, Password  string
	MessageCount        int
}

func RMQInstall(t *testing.T, kc *kubernetes.Clientset, namespace, user, password, vhost string) {
	helper.CreateNamespace(t, kc, namespace)
	data := templateData{
		Namespace: namespace,
		VHostName: vhost,
		Username:  user,
		Password:  password,
	}

	helper.KubectlApplyWithTemplate(t, data, "rmqDeploymentTemplate", deploymentTemplate)
}

func RMQUninstall(t *testing.T, kc *kubernetes.Clientset, namespace, user, password, vhost string) {
	data := templateData{
		Namespace: namespace,
		VHostName: vhost,
		Username:  user,
		Password:  password,
	}

	helper.KubectlDeleteWithTemplate(t, data, "rmqDeploymentTemplate", deploymentTemplate)
	helper.DeleteNamespace(t, kc, namespace)
}

func RMQPublishMessages(t *testing.T, namespace, connectionString, queueName string, messageCount int) {
	data := templateData{
		Namespace:    namespace,
		Connection:   connectionString,
		QueueName:    queueName,
		MessageCount: messageCount,
	}

	// Before push messages remove previous jobs if any
	_, _ = helper.ExecuteCommand(fmt.Sprintf("kubectl delete jobs/rabbitmq-publish-%s --namespace %s", queueName, namespace))

	helper.KubectlApplyWithTemplate(t, data, "rmqPublishTemplate", publishTemplate)
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
