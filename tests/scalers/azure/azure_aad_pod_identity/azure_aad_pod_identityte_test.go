//go:build e2e
// +build e2e

package azure_aad_pod_identity_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../../.env")

const (
	testName = "azure-aad-pod-identity-test"
)

var (
	testNamespace                 = fmt.Sprintf("%s-ns", testName)
	triggerAuthEmptyIdName        = fmt.Sprintf("%s-ta-empty", testName)
	triggerAuthNilIdName          = fmt.Sprintf("%s-ta-nil", testName)
	clusterTriggerAuthEmptyIdName = fmt.Sprintf("%s-cta-empty", testName)
	clusterTriggerAuthNilIdName   = fmt.Sprintf("%s-cta-nil", testName)
)

type templateData struct {
	TestNamespace                 string
	TriggerAuthEmptyIdName        string
	TriggerAuthNilIdName          string
	ClusterTriggerAuthEmptyIdName string
	ClusterTriggerAuthNilIdName   string
}

const (
	triggerAuthEmptyIdTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthEmptyIdName}}
  namespace: {{.TestNamespace}}
spec:
  podIdentity:
    provider: azure
    identityId: ""
`

	triggerAuthNilIdTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthNilIdName}}
  namespace: {{.TestNamespace}}
spec:
  podIdentity:
    provider: azure
`
	clusterTriggerAuthEmptyIdTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ClusterTriggerAuthentication
metadata:
  name: {{.ClusterTriggerAuthEmptyIdName}}
spec:
  podIdentity:
    provider: azure
    identityId: ""
`

	clusterTriggerAuthNilIdTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ClusterTriggerAuthentication
metadata:
  name: {{.ClusterTriggerAuthNilIdName}}
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
		TriggerAuthEmptyIdName:        triggerAuthEmptyIdName,
		TriggerAuthNilIdName:          triggerAuthNilIdName,
		ClusterTriggerAuthEmptyIdName: clusterTriggerAuthEmptyIdName,
		ClusterTriggerAuthNilIdName:   clusterTriggerAuthNilIdName,
	}, []Template{}
}

// expect triggerauthentication should not be created with empty identity id
func testTriggerAuthenticationWithEmptyID(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- create triggerauthentication with empty identity id  ---")

	err := KubectlApplyWithErrors(t, data, "triggerAuthEmptyIdTemplate", triggerAuthEmptyIdTemplate)
	assert.Errorf(t, err, "can deploy TriggerAuthtication - %s", err)
}

// expect triggerauthentication can be created without identity id property
func testTriggerAuthenticationWithNilID(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- create triggerauthentication with nil identity id  ---")

	kedaKc := GetKedaKubernetesClient(t)
	KubectlApplyWithTemplate(t, data, "triggerAuthNilIdTemplate", triggerAuthNilIdTemplate)

	triggerauthentication, _ := kedaKc.TriggerAuthentications(testNamespace).Get(context.Background(), triggerAuthNilIdName, v1.GetOptions{})
	assert.NotNil(t, triggerauthentication)
}

// expect clustertriggerauthentication should not be created with empty identity id
func testClusterTriggerAuthenticationWithEmptyID(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- create clustertriggerauthentication with empty identity id  ---")

	err := KubectlApplyWithErrors(t, data, "clusterTriggerAuthEmptyIdTemplate", clusterTriggerAuthEmptyIdTemplate)
	assert.Errorf(t, err, "can deploy ClusterTriggerAuthtication - %s", err)
}

// expect clustertriggerauthentication can be created without identity id property
func testClusterTriggerAuthenticationWithNilID(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- create clustertriggerauthentication with nil identity id  ---")

	kedaKc := GetKedaKubernetesClient(t)
	KubectlApplyWithTemplate(t, data, "clusterTriggerAuthNilIdTemplate", clusterTriggerAuthNilIdTemplate)

	clustertriggerauthentication, _ := kedaKc.ClusterTriggerAuthentications().Get(context.Background(), clusterTriggerAuthNilIdTemplate, v1.GetOptions{})
	assert.NotNil(t, clustertriggerauthentication)
}
