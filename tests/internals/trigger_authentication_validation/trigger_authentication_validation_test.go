//go:build e2e
// +build e2e

package trigger_authentication_validation_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../../.env")

const (
	testName = "azure-aad-pod-identity-test"
)

var (
	testNamespace                         = fmt.Sprintf("%s-ns", testName)
	triggerAuthEmptyIDName                = fmt.Sprintf("%s-ta-empty", testName)
	triggerAuthNilIDName                  = fmt.Sprintf("%s-ta-nil", testName)
	clusterTriggerAuthEmptyIDName         = fmt.Sprintf("%s-cta-empty", testName)
	clusterTriggerAuthNilIDName           = fmt.Sprintf("%s-cta-nil", testName)
	triggerAuthWorkloadEmptyIDName        = fmt.Sprintf("%s-ta-workload-empty", testName)
	triggerAuthWorkloadNilIDName          = fmt.Sprintf("%s-ta-workload-nil", testName)
	clusterTriggerAuthWorkloadEmptyIDName = fmt.Sprintf("%s-cta-workload-empty", testName)
	clusterTriggerAuthWorkloadNilIDName   = fmt.Sprintf("%s-cta-workload-nil", testName)
)

type templateData struct {
	TestNamespace                         string
	TriggerAuthEmptyIDName                string
	TriggerAuthNilIDName                  string
	ClusterTriggerAuthEmptyIDName         string
	ClusterTriggerAuthNilIDName           string
	TriggerAuthWorkloadEmptyIDName        string
	TriggerAuthWorkloadNilIDName          string
	ClusterTriggerAuthWorkloadEmptyIDName string
	ClusterTriggerAuthWorkloadNilIDName   string
}

const (
	triggerAuthEmptyIDTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthEmptyIDName}}
  namespace: {{.TestNamespace}}
spec:
  podIdentity:
    provider: azure
    identityId: ""
`

	triggerAuthNilIDTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthNilIDName}}
  namespace: {{.TestNamespace}}
spec:
  podIdentity:
    provider: azure
`
	clusterTriggerAuthEmptyIDTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ClusterTriggerAuthentication
metadata:
  name: {{.ClusterTriggerAuthEmptyIDName}}
spec:
  podIdentity:
    provider: azure
    identityId: ""
`

	clusterTriggerAuthNilIDTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ClusterTriggerAuthentication
metadata:
  name: {{.ClusterTriggerAuthNilIDName}}
spec:
  podIdentity:
    provider: azure
`

	triggerAuthWorkloadEmptyIDTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthEmptyIDName}}
  namespace: {{.TestNamespace}}
spec:
  podIdentity:
    provider: azure-workload
    identityId: ""
`

	triggerAuthWorkloadNilIDTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthNilIDName}}
  namespace: {{.TestNamespace}}
spec:
  podIdentity:
    provider: azure-workload
`
	clusterTriggerAuthWorkloadEmptyIDTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ClusterTriggerAuthentication
metadata:
  name: {{.ClusterTriggerAuthEmptyIDName}}
spec:
  podIdentity:
    provider: azure-workload
    identityId: ""
`

	clusterTriggerAuthWorkloadNilIDTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ClusterTriggerAuthentication
metadata:
  name: {{.ClusterTriggerAuthNilIDName}}
spec:
  podIdentity:
    provider: azure-workload
`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	// test auth
	testTriggerAuthenticationWithEmptyID(t, kc, data)
	testTriggerAuthenticationWithNilID(t, kc, data)
	testClusterTriggerAuthenticationWithEmptyID(t, kc, data)
	testClusterTriggerAuthenticationWithNilID(t, kc, data)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
		TestNamespace:                         testNamespace,
		TriggerAuthEmptyIDName:                triggerAuthEmptyIDName,
		TriggerAuthNilIDName:                  triggerAuthNilIDName,
		ClusterTriggerAuthEmptyIDName:         clusterTriggerAuthWorkloadEmptyIDName,
		ClusterTriggerAuthNilIDName:           clusterTriggerAuthWorkloadNilIDName,
		TriggerAuthWorkloadEmptyIDName:        triggerAuthWorkloadEmptyIDName,
		TriggerAuthWorkloadNilIDName:          triggerAuthWorkloadNilIDName,
		ClusterTriggerAuthWorkloadEmptyIDName: clusterTriggerAuthWorkloadEmptyIDName,
		ClusterTriggerAuthWorkloadNilIDName:   clusterTriggerAuthWorkloadNilIDName,
	}, []Template{}
}

// expect triggerauthentication should not be created with empty identity id
func testTriggerAuthenticationWithEmptyID(t *testing.T, _ *kubernetes.Clientset, data templateData) {
	t.Log("--- create triggerauthentication with empty identity id  ---")

	err := KubectlApplyWithErrors(t, data, "triggerAuthEmptyIDTemplate", triggerAuthEmptyIDTemplate)
	assert.Errorf(t, err, "can deploy TriggerAuthtication - %s", err)

	err = KubectlApplyWithErrors(t, data, "triggerAuthWorkloadEmptyIDTemplate", triggerAuthWorkloadEmptyIDTemplate)
	assert.Errorf(t, err, "can deploy TriggerAuthtication with azureworkload - %s", err)
}

// expect triggerauthentication can be created without identity id property
func testTriggerAuthenticationWithNilID(t *testing.T, _ *kubernetes.Clientset, data templateData) {
	t.Log("--- create triggerauthentication with nil identity id  ---")

	kedaKc := GetKedaKubernetesClient(t)
	KubectlApplyWithTemplate(t, data, "triggerAuthNilITemplate", triggerAuthNilIDTemplate)

	triggerauthentication, _ := kedaKc.TriggerAuthentications(testNamespace).Get(context.Background(), triggerAuthNilIDName, v1.GetOptions{})
	assert.NotNil(t, triggerauthentication)

	KubectlApplyWithTemplate(t, data, "triggerAuthWorkloadNilITemplate", triggerAuthWorkloadNilIDTemplate)

	triggerauthentication, _ = kedaKc.TriggerAuthentications(testNamespace).Get(context.Background(), triggerAuthWorkloadNilIDName, v1.GetOptions{})
	assert.NotNil(t, triggerauthentication)
}

// expect clustertriggerauthentication should not be created with empty identity id
func testClusterTriggerAuthenticationWithEmptyID(t *testing.T, _ *kubernetes.Clientset, data templateData) {
	t.Log("--- create clustertriggerauthentication with empty identity id  ---")

	err := KubectlApplyWithErrors(t, data, "clusterTriggerAuthEmptyIDTemplate", clusterTriggerAuthEmptyIDTemplate)
	assert.Errorf(t, err, "can deploy ClusterTriggerAuthtication - %s", err)

	err = KubectlApplyWithErrors(t, data, "clusterTriggerAuthWorkloadEmptyIDName", clusterTriggerAuthWorkloadEmptyIDName)
	assert.Errorf(t, err, "can deploy ClusterTriggerAuthtication with azureworkload - %s", err)
}

// expect clustertriggerauthentication can be created without identity id property
func testClusterTriggerAuthenticationWithNilID(t *testing.T, _ *kubernetes.Clientset, data templateData) {
	t.Log("--- create clustertriggerauthentication with nil identity id  ---")

	kedaKc := GetKedaKubernetesClient(t)
	KubectlApplyWithTemplate(t, data, "clusterTriggerAuthNilIDTemplate", clusterTriggerAuthNilIDTemplate)

	clustertriggerauthentication, _ := kedaKc.ClusterTriggerAuthentications().Get(context.Background(), clusterTriggerAuthNilIDTemplate, v1.GetOptions{})
	assert.NotNil(t, clustertriggerauthentication)

	KubectlApplyWithTemplate(t, data, "clusterTriggerAuthWorkloadNilIDTemplate", clusterTriggerAuthWorkloadNilIDTemplate)

	clustertriggerauthentication, _ = kedaKc.ClusterTriggerAuthentications().Get(context.Background(), clusterTriggerAuthWorkloadNilIDTemplate, v1.GetOptions{})
	assert.NotNil(t, clustertriggerauthentication)
}
