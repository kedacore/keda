package azure

import (
	"testing"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

type testTryAndGetAzureManagedPrometheusHTTPRoundTripperData struct {
	testName            string
	podIdentityProvider kedav1alpha1.PodIdentityProvider
	isError             bool
}

var testTryAndGetAzureManagedPrometheusHTTPRoundTripperTestData = []testTryAndGetAzureManagedPrometheusHTTPRoundTripperData{
	{"test azure workload identity trigger metadata absent", kedav1alpha1.PodIdentityProviderAzureWorkload, true},
	{"test azure pod identity trigger metadata absent", kedav1alpha1.PodIdentityProviderAzureWorkload, true},
	{"test not azure identity", kedav1alpha1.PodIdentityProviderNone, false},
}

func TestTryAndGetAzureManagedPrometheusHTTPRoundTripper(t *testing.T) {
	for _, testData := range testTryAndGetAzureManagedPrometheusHTTPRoundTripperTestData {
		_, err := TryAndGetAzureManagedPrometheusHTTPRoundTripper(kedav1alpha1.AuthPodIdentity{Provider: testData.podIdentityProvider}, nil)
		if testData.isError {
			if err == nil {
				t.Errorf("Test: %v; Expected error but got success. testData: %v", testData.testName, testData)
			}
		} else if err != nil {
			t.Errorf("Test: %v; Expected success but got error: %v", testData.testName, err)
		}
	}
}
