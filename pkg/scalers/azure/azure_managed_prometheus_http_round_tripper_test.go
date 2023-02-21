package azure

import (
	"testing"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

type testAzureManagedPrometheusCloudResourceURLData struct {
	testName                          string
	cloud                             string
	azureManagedPrometheusResourceURL string
	isError                           bool
	expectedValue                     string
}

var testAzureManagedPrometheusCloudResourceURLTestData = []testAzureManagedPrometheusCloudResourceURLData{
	{"test public cloud", "AZUREPUBLICCLOUD", "", false, "https://prometheus.monitor.azure.com/.default"},
	{"test usgov cloud", "AZUREUSGOVERNMENTCLOUD", "", false, "https://prometheus.monitor.usgovcloudapi.net/.default"},
	{"test china cloud", "AZURECHINACLOUD", "", false, "https://prometheus.monitor.chinacloudapp.cn/.default"},
	{"test private cloud success", "PRIVATE", "https://some.private.url", false, "https://some.private.url"},
	{"test private cloud failure", "PRIVATE", "", true, ""},
	{"test non existing cloud", "BOGUS_CLOUD", "", true, ""},
}

func TestAzureManagedPrometheusGetResourceUrlBasedOnCloud(t *testing.T) {
	for _, testData := range testAzureManagedPrometheusCloudResourceURLTestData {
		value, err := getResourceUrlBasedOnCloud(map[string]string{"cloud": testData.cloud, "azureManagedPrometheusResourceURL": testData.azureManagedPrometheusResourceURL})
		if testData.isError {
			if err == nil {
				t.Errorf("Test: %v; Expected error but got success. testData: %v", testData.testName, testData)
			}
		} else {
			if err == nil {
				if testData.expectedValue != value {
					t.Errorf("Test: %v; Expected value %v but got %v testData: %v", testData.testName, testData.expectedValue, value, testData)
				}
			} else {
				t.Errorf("Test: %v; Expected success but got error: %v", testData.testName, err)
			}
		}
	}
}

type testTryAndGetAzureManagedPrometheusHttpRoundTripperData struct {
	testName            string
	podIdentityProvider kedav1alpha1.PodIdentityProvider
	isError             bool
}

var testTryAndGetAzureManagedPrometheusHttpRoundTripperTestData = []testTryAndGetAzureManagedPrometheusHttpRoundTripperData{
	{"test azure workload identity trigger metadata absent", kedav1alpha1.PodIdentityProviderAzureWorkload, true},
	{"test azure pod identity trigger metadata absent", kedav1alpha1.PodIdentityProviderAzureWorkload, true},
	{"test not azure identity", kedav1alpha1.PodIdentityProviderNone, false},
}

func TestTryAndGetAzureManagedPrometheusHttpRoundTripper(t *testing.T) {
	for _, testData := range testTryAndGetAzureManagedPrometheusHttpRoundTripperTestData {
		_, err := TryAndGetAzureManagedPrometheusHttpRoundTripper(kedav1alpha1.AuthPodIdentity{Provider: testData.podIdentityProvider}, nil)
		if testData.isError {
			if err == nil {
				t.Errorf("Test: %v; Expected error but got success. testData: %v", testData.testName, testData)
			}
		} else if err != nil {
			t.Errorf("Test: %v; Expected success but got error: %v", testData.testName, err)
		}
	}
}
