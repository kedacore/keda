//go:build e2e
// +build e2e

package utils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/kedacore/keda/v2/tests/helper"
	"github.com/kedacore/keda/v2/tests/utils/helper"
)

func TestRemoveKEDA(t *testing.T) {
	// default to true
	if InstallKeda == StringFalse {
		t.Skip("skipping as requested -- KEDA not installed via these tests")
	}
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

	DeleteNamespace(t, AzureWorkloadIdentityNamespace)
}

func TestRemoveAwsIdentityComponents(t *testing.T) {
	if AwsIdentityTests == "" || AwsIdentityTests == StringFalse {
		t.Skip("skipping as workload identity tests are disabled")
	}

	_, err := ExecuteCommand(fmt.Sprintf("helm uninstall aws-identity-webhook --namespace %s", AwsIdentityNamespace))
	require.NoErrorf(t, err, "cannot uninstall workload identity webhook - %s", err)

	DeleteNamespace(t, AwsIdentityNamespace)
}

func TestRemoveGcpIdentityComponents(t *testing.T) {
	if GcpIdentityTests == "" || GcpIdentityTests == StringFalse {
		t.Skip("skipping as workload identity tests are disabled")
	}

	_, err := ExecuteCommand(fmt.Sprintf("helm uninstall gcp-identity-webhook --namespace %s", GcpIdentityNamespace))
	require.NoErrorf(t, err, "cannot uninstall workload identity webhook - %s", err)
	DeleteNamespace(t, GcpIdentityNamespace)
}

func TestRemoveOpentelemetryComponents(t *testing.T) {
	if EnableOpentelemetry == "" || EnableOpentelemetry == StringFalse {
		t.Skip("skipping uninstall of opentelemetry")
	}

	_, err := ExecuteCommand(fmt.Sprintf("helm uninstall opentelemetry-collector --namespace %s", OpentelemetryNamespace))
	require.NoErrorf(t, err, "cannot uninstall opentelemetry-collector - %s", err)
	DeleteNamespace(t, OpentelemetryNamespace)
}

func TestRemoveArgoRollouts(t *testing.T) {
	// default to true
	if InstallArgoRollouts == StringFalse {
		t.Skip("skipping as requested -- Argo Rollouts not installed via these tests")
	}

	_, err := ExecuteCommand(fmt.Sprintf("kubectl delete -n %s -f https://github.com/argoproj/argo-rollouts/releases/latest/download/install.yaml", ArgoRolloutsNamespace))
	require.NoErrorf(t, err, "cannot uninstall argo rollouts - %s", err)
	DeleteNamespace(t, ArgoRolloutsNamespace)
}

func TestRemoveCertManager(t *testing.T) {
	if !InstallCertManager {
		t.Skip("skipping as cert manager isn't required")
	}

	_, err := ExecuteCommand(fmt.Sprintf("helm uninstall cert-manager --namespace %s", CertManagerNamespace))
	require.NoErrorf(t, err, "cannot uninstall cert-manager - %s", err)
	DeleteNamespace(t, CertManagerNamespace)
}

func TestRemoveAzureManagedPrometheusComponents(t *testing.T) {
	KubectlDeleteWithTemplate(t, helper.EmptyTemplateData{}, "azureManagedPrometheusConfigMapTemplate", helper.AzureManagedPrometheusConfigMapTemplate)
}

func TestRemoveStrimzi(t *testing.T) {
	// default to true
	if InstallKafka == StringFalse {
		t.Skip("skipping as requested -- Kafka not managed by E2E tests")
	}
	_, err := ExecuteCommand(fmt.Sprintf(`helm uninstall --namespace %s %s`,
		StrimziNamespace,
		StrimziChartName))
	require.NoErrorf(t, err, "cannot uninstall strimzi - %s", err)
	DeleteNamespace(t, StrimziNamespace)
}
