//go:build e2e
// +build e2e

package utils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/kedacore/keda/v2/tests/helper"
)

func TestRemoveKEDA(t *testing.T) {
	out, err := ExecuteCommandWithDir("make undeploy", "../..")
	require.NoErrorf(t, err, "error removing KEDA - %s", err)

	t.Log(string(out))
	t.Log("KEDA removed successfully using 'make undeploy' command")
}

func TestRemoveWorkloadIdentityComponents(t *testing.T) {
	if AzureRunWorkloadIdentityTests == "" || AzureRunWorkloadIdentityTests == StringFalse {
		t.Skip("skipping as workload identity tests are disabled")
	}

	_, err := ExecuteCommand(fmt.Sprintf("helm uninstall workload-identity-webhook --namespace %s", AzureWorkloadIdentityNamespace))
	require.NoErrorf(t, err, "cannot uninstall workload identity webhook - %s", err)

	KubeClient = GetKubernetesClient(t)

	DeleteNamespace(t, KubeClient, AzureWorkloadIdentityNamespace)
}

func TestRemoveAwsIdentityComponents(t *testing.T) {
	if AwsIdentityTests == "" || AwsIdentityTests == StringFalse {
		t.Skip("skipping as workload identity tests are disabled")
	}

	_, err := ExecuteCommand(fmt.Sprintf("helm uninstall aws-identity-webhook --namespace %s", AwsIdentityNamespace))
	require.NoErrorf(t, err, "cannot uninstall workload identity webhook - %s", err)

	_, err = ExecuteCommand(fmt.Sprintf("helm uninstall cert-manager --namespace %s", AwsIdentityNamespace))
	require.NoErrorf(t, err, "cannot uninstall cert-manager - %s", err)

	KubeClient = GetKubernetesClient(t)

	DeleteNamespace(t, KubeClient, AwsIdentityNamespace)
}
