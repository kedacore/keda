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

func InstallStandalone(t *testing.T, kc *kubernetes.Clientset, name, namespace, password string) {
	helper.CreateNamespace(t, kc, namespace)
	_, err := helper.ExecuteCommand(fmt.Sprintf(`helm install --wait --timeout 900s %s --namespace %s --set architecture=standalone --set master.persistence.enabled=false --set auth.password=%s oci://registry-1.docker.io/cloudpirates/redis`,
		name,
		namespace,
		password))
	require.NoErrorf(t, err, "cannot execute command - %s", err)
}

func RemoveStandalone(t *testing.T, name, namespace string) {
	_, err := helper.ExecuteCommand(fmt.Sprintf(`helm uninstall --wait --timeout 900s %s --namespace %s`,
		name,
		namespace))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
	helper.DeleteNamespace(t, namespace)
}

func InstallSentinel(t *testing.T, kc *kubernetes.Clientset, name, namespace, password string) {
	helper.CreateNamespace(t, kc, namespace)
	_, err := helper.ExecuteCommand(fmt.Sprintf(`helm install --wait --timeout 900s %s --namespace %s --set architecture=replication --set sentinel.enabled=true --set master.persistence.enabled=false --set replica.persistence.enabled=false  --set auth.password=%s oci://registry-1.docker.io/cloudpirates/redis`,
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
	_, err = helper.ExecuteCommand(fmt.Sprintf(`helm install --wait --timeout 900s %s --namespace %s --set password=%s --set persistence.enabled=false --set image.repository=bitnamilegacy/redis-cluster --set image.tag=latest --set global.security.allowInsecureImages=true --timeout 10m0s bitnami/redis-cluster`,
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
