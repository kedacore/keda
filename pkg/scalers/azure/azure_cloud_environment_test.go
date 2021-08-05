package azure

import (
	"fmt"
	"testing"

	az "github.com/Azure/go-autorest/autorest/azure"
)

type parseEndpointSuffixTestData struct {
	metadata       map[string]string
	endpointSuffix string
	suffixProvider EnvironmentSuffixProvider
	isError        bool
}

var testSuffixProvider EnvironmentSuffixProvider = func(env az.Environment) (string, error) {
	if env == az.USGovernmentCloud {
		return "", fmt.Errorf("test endpoint is not available in %s", env.Name)
	}
	return fmt.Sprintf("%s.suffix", env.Name), nil
}

var parseEndpointSuffixTestDataset = []parseEndpointSuffixTestData{
	{map[string]string{}, "AzurePublicCloud.suffix", testSuffixProvider, false},
	{map[string]string{"cloud": "Invalid"}, "", testSuffixProvider, true},
	{map[string]string{"cloud": "AzureUSGovernmentCloud"}, "", testSuffixProvider, true},
	{map[string]string{"cloud": "AzureGermanCloud"}, "AzureGermanCloud.suffix", testSuffixProvider, false},
	{map[string]string{"cloud": "Private"}, "", testSuffixProvider, true},
	{map[string]string{"cloud": "Private", "endpointSuffix": "suffix.private.cloud"}, "suffix.private.cloud", testSuffixProvider, false},
	{map[string]string{"endpointSuffix": "ignored"}, "AzurePublicCloud.suffix", testSuffixProvider, false},
}

func TestParseEndpointSuffix(t *testing.T) {
	for _, testData := range parseEndpointSuffixTestDataset {
		endpointSuffix, err := ParseEndpointSuffix(testData.metadata, testData.suffixProvider)
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
