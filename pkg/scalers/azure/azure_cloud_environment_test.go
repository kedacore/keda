package azure

import (
	"fmt"
	"testing"
)

type parseEnvironmentPropertyTestData struct {
	metadata            map[string]string
	endpointSuffix      string
	endpointKey         string
	envPropertyProvider EnvironmentPropertyProvider
	isError             bool
}

var testPropertyProvider EnvironmentPropertyProvider = func(env AzEnvironment) (string, error) {
	if env == USGovernmentCloud {
		return "", fmt.Errorf("test endpoint is not available in %s", env.Name)
	}
	return fmt.Sprintf("%s.suffix", env.Name), nil
}

var parseEnvironmentPropertyTestDataset = []parseEnvironmentPropertyTestData{
	{map[string]string{}, "AzurePublicCloud.suffix", DefaultEndpointSuffixKey, testPropertyProvider, false},
	{map[string]string{"cloud": "Invalid"}, "", DefaultEndpointSuffixKey, testPropertyProvider, true},
	{map[string]string{"cloud": "AzureUSGovernmentCloud"}, "", DefaultEndpointSuffixKey, testPropertyProvider, true},
	{map[string]string{"cloud": "AzureGermanCloud"}, "AzureGermanCloud.suffix", DefaultEndpointSuffixKey, testPropertyProvider, false},
	{map[string]string{"cloud": "Private"}, "", DefaultEndpointSuffixKey, testPropertyProvider, true},
	{map[string]string{"cloud": "Private", "endpointSuffix": "suffix.private.cloud"}, "suffix.private.cloud", DefaultEndpointSuffixKey, testPropertyProvider, false},
	{map[string]string{"endpointSuffix": "ignored"}, "AzurePublicCloud.suffix", DefaultEndpointSuffixKey, testPropertyProvider, false},
	{map[string]string{"cloud": "Private", "endpointSuffixDiff": "suffix.private.cloud"}, "suffix.private.cloud", "endpointSuffixDiff", testPropertyProvider, false},
	{map[string]string{"cloud": "Private", "storageEndpointSuffix": "suffix.private.cloud"}, "suffix.private.cloud", DefaultStorageSuffixKey, testPropertyProvider, false},
	{map[string]string{"cloud": "Private"}, "suffix.private.cloud", DefaultStorageSuffixKey, testPropertyProvider, true},
}

func TestParseEnvironmentProperty(t *testing.T) {
	for _, testData := range parseEnvironmentPropertyTestDataset {
		endpointSuffix, err := ParseEnvironmentProperty(testData.metadata, testData.endpointKey, testData.envPropertyProvider)
		if !testData.isError && err != nil {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
		if err == nil {
			if endpointSuffix != testData.endpointSuffix {
				t.Error(
					"For", testData.metadata,
					"expected endpointSuffix=", testData.endpointSuffix,
					"but got", endpointSuffix)
			}
		}
	}
}

const testADEndpoint = "testADEndpoint"

type parseActiveDirectoryEndpointTestData struct {
	metadata           map[string]string
	expectedADEndpoint string
	isError            bool
}

var parseActiveDirectoryEndpointTestDataset = []parseActiveDirectoryEndpointTestData{
	{metadata: map[string]string{}, isError: false, expectedADEndpoint: PublicCloud.ActiveDirectoryEndpoint},
	{metadata: map[string]string{"cloud": "AzureChinaCloud"}, isError: false, expectedADEndpoint: ChinaCloud.ActiveDirectoryEndpoint},
	{metadata: map[string]string{"cloud": "private", "activeDirectoryEndpoint": testADEndpoint}, isError: false,
		expectedADEndpoint: testADEndpoint},
	{metadata: map[string]string{"cloud": "private"}, isError: true},
	{metadata: map[string]string{"cloud": "invalid"}, isError: true},
}

func TestParseActiveDirectoryEndpoint(t *testing.T) {
	for _, testData := range parseActiveDirectoryEndpointTestDataset {
		activeDirectoryEndpoint, err := ParseActiveDirectoryEndpoint(testData.metadata)
		if !testData.isError && err != nil {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
		if err == nil {
			if activeDirectoryEndpoint != testData.expectedADEndpoint {
				t.Error(
					"For", testData.metadata,
					"expected activeDirectoryEndpoint=", testData.expectedADEndpoint,
					"but got", activeDirectoryEndpoint)
			}
		}
	}
}
