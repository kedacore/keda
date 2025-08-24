package azure

import (
	"testing"

	"github.com/go-logr/logr"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

type testTryAndGetAzureManagedPrometheusHTTPRoundTripperData struct {
	testName            string
	podIdentityProvider kedav1alpha1.PodIdentityProvider
	isError             bool
}

type testAzureManagedPrometheusResourceURLData struct {
	testName            string
	podIdentityProvider kedav1alpha1.PodIdentityProvider
	metadata            map[string]string
	resourceURL         string
	isError             bool
}

var testTryAndGetAzureManagedPrometheusHTTPRoundTripperTestData = []testTryAndGetAzureManagedPrometheusHTTPRoundTripperData{
	{"test azure WI trigger metadata absent", kedav1alpha1.PodIdentityProviderAzureWorkload, true},
	{"test azure pod identity trigger metadata absent", kedav1alpha1.PodIdentityProviderAzureWorkload, true},
	{"test not azure identity", kedav1alpha1.PodIdentityProviderNone, false},
}

var testAzureManagedPrometheusResourceURLTestData = []testAzureManagedPrometheusResourceURLData{
	// workload identity

	// with default cloud
	{"test default azure cloud with WI", kedav1alpha1.PodIdentityProviderAzureWorkload, map[string]string{"serverAddress": "http://dummy-azure-monitor-workspace", "metricName": "http_requests_total", "threshold": "100", "query": "up"}, "https://prometheus.monitor.azure.com/.default", false},
	// with public cloud
	{"test azure public cloud with WI", kedav1alpha1.PodIdentityProviderAzureWorkload, map[string]string{"serverAddress": "http://dummy-azure-monitor-workspace", "metricName": "http_requests_total", "threshold": "100", "query": "up", "cloud": "AZUREPUBLICCLOUD"}, "https://prometheus.monitor.azure.com/.default", false},
	// with china cloud
	{"test azure china cloud with WI", kedav1alpha1.PodIdentityProviderAzureWorkload, map[string]string{"serverAddress": "http://dummy-azure-monitor-workspace", "metricName": "http_requests_total", "threshold": "100", "query": "up", "cloud": "AZURECHINACLOUD"}, "https://prometheus.monitor.azure.cn/.default", false},
	// with US GOV cloud
	{"test azure US GOV cloud with WI", kedav1alpha1.PodIdentityProviderAzureWorkload, map[string]string{"serverAddress": "http://dummy-azure-monitor-workspace", "metricName": "http_requests_total", "threshold": "100", "query": "up", "cloud": "AZUREUSGOVERNMENTCLOUD"}, "https://prometheus.monitor.azure.us/.default", false},
	// with private cloud success
	{"test azure private cloud with WI", kedav1alpha1.PodIdentityProviderAzureWorkload, map[string]string{"serverAddress": "http://dummy-azure-monitor-workspace", "metricName": "http_requests_total", "threshold": "100", "query": "up", "cloud": "PRIVATE", "azureManagedPrometheusResourceURL": "blah-blah-resourceURL"}, "blah-blah-resourceURL", false},
	// with private cloud failure
	{"test default azure cloud with WI", kedav1alpha1.PodIdentityProviderAzureWorkload, map[string]string{"serverAddress": "http://dummy-azure-monitor-workspace", "metricName": "http_requests_total", "threshold": "100", "query": "up", "cloud": "PRIVATE"}, "", true},
}

func TestTryAndGetAzureManagedPrometheusHTTPRoundTripperForTriggerMetadataAbsent(t *testing.T) {
	for _, testData := range testTryAndGetAzureManagedPrometheusHTTPRoundTripperTestData {
		_, err := TryAndGetAzureManagedPrometheusHTTPRoundTripper(logr.Discard(), kedav1alpha1.AuthPodIdentity{Provider: testData.podIdentityProvider}, nil)
		if testData.isError {
			if err == nil {
				t.Errorf("Test: %v; Expected error but got success. testData: %v", testData.testName, testData)
			}
		} else if err != nil {
			t.Errorf("Test: %v; Expected success but got error: %v", testData.testName, err)
		}
	}
}

func TestTryAndGetAzureManagedPrometheusHTTPRoundTripperWithTriggerForResourceURL(t *testing.T) {
	for _, testData := range testAzureManagedPrometheusResourceURLTestData {
		rt, err := TryAndGetAzureManagedPrometheusHTTPRoundTripper(logr.Discard(), kedav1alpha1.AuthPodIdentity{Provider: testData.podIdentityProvider}, testData.metadata)
		if testData.isError {
			if err == nil {
				t.Errorf("Test: %v; Expected error but got success. testData: %v", testData.testName, testData)
			}
		} else {
			if err != nil {
				t.Errorf("Test: %v; Expected success but got error: %v", testData.testName, err)
			} else {
				azureRT := rt.(*azureManagedPrometheusHTTPRoundTripper)
				if azureRT == nil {
					t.Errorf("Test: %v; Expected azure round tripper but got nil", testData.testName)
				} else if azureRT.resourceURL != testData.resourceURL {
					t.Errorf("Test: %v; Expected resourceURL %v but got %v", testData.testName, testData.resourceURL, azureRT.resourceURL)
				}
			}
		}
	}
}
