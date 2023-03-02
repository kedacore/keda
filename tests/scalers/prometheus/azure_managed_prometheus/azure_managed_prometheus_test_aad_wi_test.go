//go:build e2e
// +build e2e

package azure_managed_prometheus_test

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/joho/godotenv"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testNameWorkloadIdentity = "azure-managed-prom-wi-test"
)

// Workload Identity test vars
var (
	randomNumberWI           = rand.Int()
	testNamespaceWI          = fmt.Sprintf("%s-ns-%d", testNameWorkloadIdentity, randomNumberWI)
	deploymentNameWI         = fmt.Sprintf("%s-deployment-%d", testNameWorkloadIdentity, randomNumberWI)
	monitoredAppNameWI       = fmt.Sprintf("%s-monitored-app-%d", testNameWorkloadIdentity, randomNumberWI)
	publishDeploymentNameWI  = fmt.Sprintf("%s-publish-%d", testNameWorkloadIdentity, randomNumberWI)
	scaledObjectNameWI       = fmt.Sprintf("%s-so-%d", testNameWorkloadIdentity, randomNumberWI)
	workloadIdentityProvider = "azure-workload"
)

// TestAzureManagedPrometheusScalerWithWorkloadIdentity creates deployments - there are two deployments - both using the same image but one deployment
// is directly tied to the KEDA HPA while the other is isolated that can be used for metrics
// even when the KEDA deployment is at zero - the service points to both deployments
func TestAzureManagedPrometheusScalerWithWorkloadIdentity(t *testing.T) {
	testAzureManagedPrometheusScaler(t, getTemplateDataForWorkloadIdentityTest())
}

func getTemplateDataForWorkloadIdentityTest() templateData {
	return templateData{
		TestNamespace:           testNamespaceWI,
		DeploymentName:          deploymentNameWI,
		PublishDeploymentName:   publishDeploymentNameWI,
		ScaledObjectName:        scaledObjectNameWI,
		MonitoredAppName:        monitoredAppNameWI,
		PodIdentityProvider:     workloadIdentityProvider,
		PrometheusQueryEndpoint: prometheusQueryEndpoint,
		MinReplicaCount:         minReplicaCount,
		MaxReplicaCount:         maxReplicaCount,
	}
}
