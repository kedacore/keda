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
	testNamespace                 = fmt.Sprintf("%s-ns", testName)
	triggerAuthEmptyIDName        = fmt.Sprintf("%s-ta-empty", testName)
	triggerAuthNilIDName          = fmt.Sprintf("%s-ta-nil", testName)
	clusterTriggerAuthEmptyIDName = fmt.Sprintf("%s-cta-empty", testName)
	clusterTriggerAuthNilIDName   = fmt.Sprintf("%s-cta-nil", testName)
)

type templateData struct {
	TestNamespace                 string
	TriggerAuthEmptyIDName        string
	TriggerAuthNilIDName          string
	ClusterTriggerAuthEmptyIDName string
	ClusterTriggerAuthNilIDName   string
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
		TestNamespace:                 testNamespace,
		TriggerAuthEmptyIDName:        triggerAuthEmptyIDName,
		TriggerAuthNilIDName:          triggerAuthNilIDName,
		ClusterTriggerAuthEmptyIDName: clusterTriggerAuthEmptyIDName,
		ClusterTriggerAuthNilIDName:   clusterTriggerAuthNilIDName,
	}, []Template{}
}

// expect triggerauthentication should not be created with empty identity id
func testTriggerAuthenticationWithEmptyID(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- create triggerauthentication with empty identity id  ---")

	err := KubectlApplyWithErrors(t, data, "triggerAuthEmptyIDTemplate", triggerAuthEmptyIDTemplate)
	assert.Errorf(t, err, "can deploy TriggerAuthtication - %s", err)
}

// expect triggerauthentication can be created without identity id property
func testTriggerAuthenticationWithNilID(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- create triggerauthentication with nil identity id  ---")

	kedaKc := GetKedaKubernetesClient(t)
	KubectlApplyWithTemplate(t, data, "triggerAuthNilITemplate", triggerAuthNilIDTemplate)

	triggerauthentication, _ := kedaKc.TriggerAuthentications(testNamespace).Get(context.Background(), triggerAuthNilIDName, v1.GetOptions{})
	assert.NotNil(t, triggerauthentication)
}

// expect clustertriggerauthentication should not be created with empty identity id
func testClusterTriggerAuthenticationWithEmptyID(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- create clustertriggerauthentication with empty identity id  ---")

	err := KubectlApplyWithErrors(t, data, "clusterTriggerAuthEmptyIDTemplate", clusterTriggerAuthEmptyIDTemplate)
	assert.Errorf(t, err, "can deploy ClusterTriggerAuthtication - %s", err)
}

// expect clustertriggerauthentication can be created without identity id property
func testClusterTriggerAuthenticationWithNilID(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- create clustertriggerauthentication with nil identity id  ---")

	kedaKc := GetKedaKubernetesClient(t)
	KubectlApplyWithTemplate(t, data, "clusterTriggerAuthNilIDTemplate", clusterTriggerAuthNilIDTemplate)

	clustertriggerauthentication, _ := kedaKc.ClusterTriggerAuthentications().Get(context.Background(), clusterTriggerAuthNilIDTemplate, v1.GetOptions{})
	assert.NotNil(t, clustertriggerauthentication)
}
