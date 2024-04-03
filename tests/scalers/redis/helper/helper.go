//go:build e2e
// +build e2e

package helper

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	"github.com/kedacore/keda/v2/tests/helper"
)

type templateData struct {
	Namespace     string
	RedisName     string
	RedisPassword string
}

var (
	redisStandaloneTemplates = []helper.Template{
		{Name: "standaloneRedisTemplate", Config: standaloneRedisTemplate},
		{Name: "standaloneRedisServiceTemplate", Config: standaloneRedisServiceTemplate},
	}
)

const (
	standaloneRedisTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.RedisName}}
  namespace: {{.Namespace}}
spec:
  selector:
    matchLabels:
      app: {{.RedisName}}
  replicas: 1
  template:
    metadata:
      labels:
        app: {{.RedisName}}
    spec:
      containers:
      - name: master
        image: redis:7.0
        command: ["redis-server", "--requirepass", {{.RedisPassword}}]
        ports:
        - containerPort: 6379`

	standaloneRedisServiceTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: redis
  namespace: {{.Namespace}}
  labels:
    app: {{.RedisName}}
spec:
  ports:
  - port: 6379
    targetPort: 6379
  selector:
    app: {{.RedisName}}`
)

func InstallStandalone(t *testing.T, kc *kubernetes.Clientset, name, namespace, password string) {
	helper.CreateNamespace(t, kc, namespace)
	var data = templateData{
		Namespace:     namespace,
		RedisName:     name,
		RedisPassword: password,
	}
	helper.KubectlApplyMultipleWithTemplate(t, data, redisStandaloneTemplates)
}

func RemoveStandalone(t *testing.T, name, namespace string) {
	var data = templateData{
		Namespace: namespace,
		RedisName: name,
	}
	helper.KubectlApplyMultipleWithTemplate(t, data, redisStandaloneTemplates)
	helper.DeleteNamespace(t, namespace)
}

func InstallSentinel(t *testing.T, kc *kubernetes.Clientset, name, namespace, password string) {
	helper.CreateNamespace(t, kc, namespace)
	_, err := helper.ExecuteCommand("helm repo add bitnami https://charts.bitnami.com/bitnami")
	require.NoErrorf(t, err, "cannot execute command - %s", err)
	_, err = helper.ExecuteCommand("helm repo update")
	require.NoErrorf(t, err, "cannot execute command - %s", err)
	_, err = helper.ExecuteCommand(fmt.Sprintf(`helm install --wait --timeout 900s %s --namespace %s --set sentinel.enabled=true --set master.persistence.enabled=false --set replica.persistence.enabled=false --set global.redis.password=%s bitnami/redis`,
		name,
		namespace,
		password))
	require.NoErrorf(t, err, "cannot execute command - %s", err)
}

func RemoveSentinel(t *testing.T, name, namespace string) {
	_, err := helper.ExecuteCommand(fmt.Sprintf(`helm uninstall --wait --timeout 900s %s --namespace %s`,
		name,
		namespace))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
	helper.DeleteNamespace(t, namespace)
}

func InstallCluster(t *testing.T, kc *kubernetes.Clientset, name, namespace, password string) {
	helper.CreateNamespace(t, kc, namespace)
	_, err := helper.ExecuteCommand("helm repo add bitnami https://charts.bitnami.com/bitnami")
	require.NoErrorf(t, err, "cannot execute command - %s", err)
	_, err = helper.ExecuteCommand("helm repo update")
	require.NoErrorf(t, err, "cannot execute command - %s", err)
	_, err = helper.ExecuteCommand(fmt.Sprintf(`helm install --wait --timeout 900s %s --namespace %s --set persistence.enabled=false --set password=%s --timeout 10m0s bitnami/redis-cluster`,
		name,
		namespace,
		password))
	require.NoErrorf(t, err, "cannot execute command - %s", err)
}

func RemoveCluster(t *testing.T, name, namespace string) {
	_, err := helper.ExecuteCommand(fmt.Sprintf(`helm uninstall --wait --timeout 900s %s --namespace %s`,
		name,
		namespace))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
	helper.DeleteNamespace(t, namespace)
}
