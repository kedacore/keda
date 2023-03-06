//go:build e2e
// +build e2e

package utils

import (
	"context"
	"fmt"
	"os/exec"
	"testing"
	"time"

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

	workloadIdentityDeploymentName := "azure-wi-webhook-controller-manager"
	success := false
	for i := 0; i < 20; i++ {
		deployment, err := KubeClient.AppsV1().Deployments(AzureWorkloadIdentityNamespace).Get(context.Background(), workloadIdentityDeploymentName, v1.GetOptions{})
		require.NoErrorf(t, err, "unable to get workload identity webhook deployment - %s", err)

		readyReplicas := deployment.Status.ReadyReplicas
		if readyReplicas != 2 {
			t.Log("workload identity webhook is not ready. sleeping")
			time.Sleep(5 * time.Second)
		} else {
			t.Log("workload identity webhook is ready")
			success = true

			time.Sleep(2 * time.Minute) // sleep for some time for webhook to setup properly
			break
		}
	}

	require.True(t, success, "expected workload identity webhook deployment to start 2 pods successfully")
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

	_, err = ExecuteCommand(fmt.Sprintf("helm upgrade --install cert-manager jetstack/cert-manager --namespace %s --set installCRDs=true --wait",
		CertManagerNamespace))
	require.NoErrorf(t, err, "cannot install cert-manager - %s", err)
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

	_, err = ExecuteCommand(fmt.Sprintf("helm upgrade --install aws-identity-webhook jkroepke/amazon-eks-pod-identity-webhook --namespace %s --set fullnameOverride=aws-identity-webhook --wait",
		AwsIdentityNamespace))
	require.NoErrorf(t, err, "cannot install workload identity webhook - %s", err)
	time.Sleep(1 * time.Minute) // sleep for some time for webhook to setup properly
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

	_, err = ExecuteCommand(fmt.Sprintf("helm upgrade --install gcp-identity-webhook gcp-workload-identity-federation-webhook/gcp-workload-identity-federation-webhook --namespace %s --set fullnameOverride=gcp-identity-webhook --set controllerManager.manager.args[0]=--token-default-mode=0444 --wait",
		GcpIdentityNamespace))
	require.NoErrorf(t, err, "cannot install workload identity webhook - %s", err)
	time.Sleep(1 * time.Minute) // sleep for some time for webhook to setup properly
}

func TestDeployKEDA(t *testing.T) {
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
	success := false
	for i := 0; i < 20; i++ {
		operatorDeployment, err := KubeClient.AppsV1().Deployments(KEDANamespace).Get(context.Background(), KEDAOperator, v1.GetOptions{})
		require.NoErrorf(t, err, "unable to get %s deployment - %s", KEDAOperator, err)

		metricsServerDeployment, err := KubeClient.AppsV1().Deployments(KEDANamespace).Get(context.Background(), KEDAMetricsAPIServer, v1.GetOptions{})
		require.NoErrorf(t, err, "unable to get %s deployment - %s", KEDAMetricsAPIServer, err)

		webhooksDeployment, err := KubeClient.AppsV1().Deployments(KEDANamespace).Get(context.Background(), KEDAAdmissionWebhooks, v1.GetOptions{})
		require.NoErrorf(t, err, "unable to get %s deployment - %s", KEDAAdmissionWebhooks, err)

		operatorReadyReplicas := operatorDeployment.Status.ReadyReplicas
		metricsServerReadyReplicas := metricsServerDeployment.Status.ReadyReplicas
		webhooksReadyReplicas := webhooksDeployment.Status.ReadyReplicas

		if operatorReadyReplicas != 1 || metricsServerReadyReplicas != 1 || webhooksReadyReplicas != 1 {
			t.Log("KEDA is not ready. sleeping")
			time.Sleep(10 * time.Second)
		} else {
			t.Logf("KEDA is running 1 pod for %s, 1 pod for %s and 1 pod for %s", KEDAOperator, KEDAMetricsAPIServer, KEDAAdmissionWebhooks)
			success = true
			break
		}
	}

	require.True(t, success, "expected KEDA deployments to start 3 pods successfully")
}

func TestSetupAadPodIdentityComponents(t *testing.T) {
	if AzureRunAadPodIdentityTests == "" || AzureRunAadPodIdentityTests == StringFalse {
		t.Skip("skipping as aad pod identity tests are disabled")
	}

	_, err := ExecuteCommand("helm version")
	require.NoErrorf(t, err, "helm is not installed - %s", err)

	_, err = ExecuteCommand("helm repo add aad-pod-identity https://raw.githubusercontent.com/Azure/aad-pod-identity/master/charts")
	require.NoErrorf(t, err, "cannot add pod identity helm repo - %s", err)

	_, err = ExecuteCommand("helm repo update aad-pod-identity")
	require.NoErrorf(t, err, "cannot update aad pod identity helm repo - %s", err)

	KubeClient = GetKubernetesClient(t)
	CreateNamespace(t, KubeClient, AzureAdPodIdentityNamespace)

	_, err = ExecuteCommand(fmt.Sprintf("helm upgrade --install "+
		"aad-pod-identity aad-pod-identity/aad-pod-identity "+
		"--namespace %s --wait "+
		"--set azureIdentities.keda.type=0 "+
		"--set azureIdentities.keda.namespace=keda "+
		"--set azureIdentities.keda.clientID=%s "+
		"--set azureIdentities.keda.resourceID=%s "+
		"--set azureIdentities.keda.binding.selector=keda "+
		"--set azureIdentities.keda.binding.name=keda",
		AzureAdPodIdentityNamespace, AzureADMsiClientID, AzureADMsiID))
	require.NoErrorf(t, err, "cannot install aad pod identity webhook - %s", err)
}
