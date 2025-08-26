//go:build e2e
// +build e2e

package utils

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/kedacore/keda/v2/tests/helper"
	"github.com/kedacore/keda/v2/tests/utils/helper"
)

func TestVerifyCommands(t *testing.T) {
	commands := []string{"kubectl"}
	for _, cmd := range commands {
		_, err := exec.LookPath(cmd)
		require.NoErrorf(t, err, "%s is required for setup - %s", cmd, err)
	}
}

func TestKubernetesConnection(t *testing.T) {
	KubeClient = GetKubernetesClient(t)
}

func TestKubernetesVersion(t *testing.T) {
	out, err := ExecuteCommand("kubectl version")
	require.NoErrorf(t, err, "error getting kubernetes version - %s", err)

	t.Logf("kubernetes version: %s", string(out))
}

func TestSetupHelm(t *testing.T) {
	_, err := exec.LookPath("helm")
	if err == nil {
		t.Skip("helm is already installed. skipping setup.")
	}

	_, err = ExecuteCommand("curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3")
	require.NoErrorf(t, err, "cannot download helm installation shell script - %s", err)

	_, err = ExecuteCommand("chmod 700 get_helm.sh")
	require.NoErrorf(t, err, "cannot change permissions for helm installation script - %s", err)

	_, err = ExecuteCommand("./get_helm.sh")
	require.NoErrorf(t, err, "cannot download helm - %s", err)

	_, err = ExecuteCommand("helm version")
	require.NoErrorf(t, err, "cannot get helm version - %s", err)
}

// doing early in the sequence of tests so that config map update has time to be effective before the azure tests get executed.
func TestSetupAzureManagedPrometheusComponents(t *testing.T) {
	// this will install config map in kube-system namespace, as needed by azure manage prometheus collector agent
	KubectlApplyWithTemplate(t, helper.EmptyTemplateData{}, "azureManagedPrometheusConfigMapTemplate", helper.AzureManagedPrometheusConfigMapTemplate)
}

func TestSetupCertManager(t *testing.T) {
	if !InstallCertManager {
		t.Skip("skipping cert manager is not required")
	}

	_, err := ExecuteCommand("helm version")
	require.NoErrorf(t, err, "helm is not installed - %s", err)

	_, err = ExecuteCommand("helm repo add jetstack https://charts.jetstack.io")
	require.NoErrorf(t, err, "cannot add jetstack helm repo - %s", err)

	_, err = ExecuteCommand("helm repo update jetstack")
	require.NoErrorf(t, err, "cannot update jetstack helm repo - %s", err)

	KubeClient = GetKubernetesClient(t)
	CreateNamespace(t, KubeClient, CertManagerNamespace)

	_, err = ExecuteCommand(fmt.Sprintf("helm upgrade --install cert-manager jetstack/cert-manager --namespace %s --set installCRDs=true",
		CertManagerNamespace))
	require.NoErrorf(t, err, "cannot install cert-manager - %s", err)
}

func TestSetupWorkloadIdentityComponents(t *testing.T) {
	if AzureRunWorkloadIdentityTests == "" || AzureRunWorkloadIdentityTests == StringFalse {
		t.Skip("skipping as workload identity tests are disabled")
	}

	_, err := ExecuteCommand("helm version")
	require.NoErrorf(t, err, "helm is not installed - %s", err)

	_, err = ExecuteCommand("helm repo add azure-workload-identity https://azure.github.io/azure-workload-identity/charts")
	require.NoErrorf(t, err, "cannot add workload identity helm repo - %s", err)

	_, err = ExecuteCommand("helm repo update azure-workload-identity")
	require.NoErrorf(t, err, "cannot update workload identity helm repo - %s", err)

	KubeClient = GetKubernetesClient(t)
	CreateNamespace(t, KubeClient, AzureWorkloadIdentityNamespace)

	_, err = ExecuteCommand(fmt.Sprintf("helm upgrade --install workload-identity-webhook azure-workload-identity/workload-identity-webhook --namespace %s --set azureTenantID=%s",
		AzureWorkloadIdentityNamespace, AzureADTenantID))
	require.NoErrorf(t, err, "cannot install workload identity webhook - %s", err)
}

func TestSetupAwsIdentityComponents(t *testing.T) {
	if AwsIdentityTests == "" || AwsIdentityTests == StringFalse {
		t.Skip("skipping aws identity tests are disabled")
	}

	_, err := ExecuteCommand("helm version")
	require.NoErrorf(t, err, "helm is not installed - %s", err)

	_, err = ExecuteCommand("helm repo add jkroepke https://jkroepke.github.io/helm-charts")
	require.NoErrorf(t, err, "cannot add jkroepke helm repo - %s", err)

	_, err = ExecuteCommand("helm repo update jkroepke")
	require.NoErrorf(t, err, "cannot update jkroepke helm repo - %s", err)

	KubeClient = GetKubernetesClient(t)
	CreateNamespace(t, KubeClient, AwsIdentityNamespace)

	_, err = ExecuteCommand(fmt.Sprintf("helm upgrade --install aws-identity-webhook jkroepke/amazon-eks-pod-identity-webhook --namespace %s --set config.defaultAwsRegion=eu-west-2 --set readinessProbe.httpGet.scheme=HTTPS --set livenessProbe.httpGet.scheme=HTTPS --set fullnameOverride=aws-identity-webhook",
		AwsIdentityNamespace))
	require.NoErrorf(t, err, "cannot install workload identity webhook - %s", err)
}

func TestSetupGcpIdentityComponents(t *testing.T) {
	if GcpIdentityTests == "" || GcpIdentityTests == StringFalse {
		t.Skip("skipping gcp identity tests are disabled")
	}

	_, err := ExecuteCommand("helm version")
	require.NoErrorf(t, err, "helm is not installed - %s", err)

	_, err = ExecuteCommand("helm repo add gcp-workload-identity-federation-webhook https://pfnet-research.github.io/gcp-workload-identity-federation-webhook")
	require.NoErrorf(t, err, "cannot add gcp-workload-identity-federation-webhook helm repo - %s", err)

	_, err = ExecuteCommand("helm repo update gcp-workload-identity-federation-webhook")
	require.NoErrorf(t, err, "cannot update gcp-workload-identity-federation-webhook helm repo - %s", err)

	KubeClient = GetKubernetesClient(t)
	CreateNamespace(t, KubeClient, GcpIdentityNamespace)

	_, err = ExecuteCommand(fmt.Sprintf("helm upgrade --install gcp-identity-webhook gcp-workload-identity-federation-webhook/gcp-workload-identity-federation-webhook --namespace %s --set fullnameOverride=gcp-identity-webhook --set controllerManager.manager.args[0]=--token-default-mode=0444",
		GcpIdentityNamespace))
	require.NoErrorf(t, err, "cannot install workload identity webhook - %s", err)
}

func TestVerifyPodsIdentity(t *testing.T) {
	if AzureRunWorkloadIdentityTests == StringTrue {
		assert.True(t, WaitForDeploymentReplicaReadyCount(t, KubeClient, "azure-wi-webhook-controller-manager", "azure-workload-identity-system", 2, 30, 6),
			"replica count should be 1 after 3 minutes")
	}

	if AwsIdentityTests == StringTrue {
		assert.True(t, WaitForDeploymentReplicaReadyCount(t, KubeClient, "aws-identity-webhook", "aws-identity-system", 1, 30, 6),
			"replica count should be 1 after 3 minutes")
	}

	if GcpIdentityTests == StringTrue {
		assert.True(t, WaitForDeploymentReplicaReadyCount(t, KubeClient, "gcp-identity-webhook-controller-manager", "gcp-identity-system", 1, 30, 6),
			"replica count should be 1 after 3 minutes")
	}
}

func TestSetupOpentelemetryComponents(t *testing.T) {
	if EnableOpentelemetry == "" || EnableOpentelemetry == StringFalse {
		t.Skip("skipping installing opentelemetry")
	}

	otlpTempFileName := "otlp.yml"
	otlpServiceTempFileName := "otlpServicePatch.yml"
	defer os.Remove(otlpTempFileName)
	defer os.Remove(otlpServiceTempFileName)
	err := os.WriteFile(otlpTempFileName, []byte(helper.OtlpConfig), 0755)
	assert.NoErrorf(t, err, "cannot create otlp config file - %s", err)

	err = os.WriteFile(otlpServiceTempFileName, []byte(helper.OtlpServicePatch), 0755)
	assert.NoErrorf(t, err, "cannot create otlp service patch file - %s", err)

	_, err = ExecuteCommand("helm version")
	require.NoErrorf(t, err, "helm is not installed - %s", err)

	_, err = ExecuteCommand("helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts")
	require.NoErrorf(t, err, "cannot add open-telemetry helm repo - %s", err)

	_, err = ExecuteCommand("helm repo update open-telemetry")
	require.NoErrorf(t, err, "cannot update open-telemetry helm repo - %s", err)

	KubeClient = GetKubernetesClient(t)
	CreateNamespace(t, KubeClient, OpentelemetryNamespace)

	_, err = ExecuteCommand(fmt.Sprintf("helm upgrade --install opentelemetry-collector open-telemetry/opentelemetry-collector -f %s --namespace %s", otlpTempFileName, OpentelemetryNamespace))

	require.NoErrorf(t, err, "cannot install opentelemetry - %s", err)

	_, err = ExecuteCommand(fmt.Sprintf("kubectl apply -f %s -n %s", otlpServiceTempFileName, OpentelemetryNamespace))
	require.NoErrorf(t, err, "cannot update opentelemetry ports - %s", err)
}

func TestDeployKEDA(t *testing.T) {
	// default to true
	if InstallKeda == StringFalse {
		t.Skip("skipping as requested -- KEDA assumed to be already installed")
	}
	KubeClient = GetKubernetesClient(t)
	CreateNamespace(t, KubeClient, KEDANamespace)

	caCtr, _ := GetTestCA(t)
	secret := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      "custom-cas",
			Namespace: KEDANamespace,
		},
		StringData: map[string]string{
			"test-ca.crt": string(caCtr),
		},
	}

	_, err := KubeClient.CoreV1().Secrets(KEDANamespace).Create(context.Background(), secret, v1.CreateOptions{})
	require.NoErrorf(t, err, "error deploying custom CA - %s", err)

	out, err := ExecuteCommandWithDir("make deploy", "../..")
	require.NoErrorf(t, err, "error deploying KEDA - %s", err)

	t.Log(string(out))
	t.Log("KEDA deployed successfully using 'make deploy' command")
}

func TestVerifyKEDA(t *testing.T) {
	// default to true
	if InstallKeda == StringFalse {
		t.Skip("skipping as requested -- KEDA assumed to be already installed")
	}
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, KubeClient, KEDAOperator, KEDANamespace, 1, 30, 6),
		"replica count should be 1 after 3 minutes")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, KubeClient, KEDAMetricsAPIServer, KEDANamespace, 1, 30, 6),
		"replica count should be 1 after 3 minutes")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, KubeClient, KEDAAdmissionWebhooks, KEDANamespace, 1, 30, 6),
		"replica count should be 1 after 3 minutes")
}

func TestSetUpStrimzi(t *testing.T) {
	// default to true
	if InstallKafka == StringFalse {
		t.Skip("skipping as requested -- Kafka assumed to be unneeded or already installed")
	}
	t.Log("--- installing kafka operator ---")
	_, err := ExecuteCommand("helm repo add strimzi https://strimzi.io/charts/")
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
	_, err = ExecuteCommand("helm repo update")
	assert.NoErrorf(t, err, "cannot execute command - %s", err)

	KubeClient = GetKubernetesClient(t)

	CreateNamespace(t, KubeClient, StrimziNamespace)

	_, err = ExecuteCommand(fmt.Sprintf(`helm upgrade --install --namespace %s --wait %s strimzi/strimzi-kafka-operator --version %s --set watchAnyNamespace=true`,
		StrimziNamespace,
		StrimziChartName,
		StrimziVersion))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)

	t.Log("--- kafka operator installed ---")
}

func TestVerifyStrimzi(t *testing.T) {
	// default to true
	if InstallKafka == StringFalse {
		t.Skip("skipping as requested -- Kafka assumed to be unneeded or already installed")
	}
	t.Log("--- verifying kafka operator is ready ---")

	// Wait for the Strimzi cluster operator deployment to be ready
	// This ensures the operator is fully initialized before tests proceed
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, KubeClient, "strimzi-cluster-operator", StrimziNamespace, 1, 120, 5),
		"Strimzi cluster operator should be ready after 10 minutes")

	t.Log("--- kafka operator verified and ready ---")
}
