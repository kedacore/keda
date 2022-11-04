/*
Copyright 2022 The KEDA Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package resolver

import (
	"testing"

	az "github.com/Azure/go-autorest/autorest/azure"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

const (
	testResourceURL             = "testResourceURL"
	testActiveDirectoryEndpoint = "testActiveDirectoryEndpoint"
)

type testData struct {
	name                  string
	isError               bool
	vault                 kedav1alpha1.AzureKeyVault
	expectedKVResourceURL string
	expectedADEndpoint    string
}

var testDataset = []testData{
	{
		name:    "known Azure cloud",
		isError: false,
		vault: kedav1alpha1.AzureKeyVault{
			Cloud: &kedav1alpha1.AzureKeyVaultCloudInfo{
				Type: "azurePublicCloud",
			},
		},
		expectedKVResourceURL: az.PublicCloud.ResourceIdentifiers.KeyVault,
		expectedADEndpoint:    az.PublicCloud.ActiveDirectoryEndpoint,
	},
	{
		name:    "private cloud",
		isError: false,
		vault: kedav1alpha1.AzureKeyVault{
			Cloud: &kedav1alpha1.AzureKeyVaultCloudInfo{
				Type:                    "private",
				KeyVaultResourceURL:     testResourceURL,
				ActiveDirectoryEndpoint: testActiveDirectoryEndpoint,
			},
		},
		expectedKVResourceURL: testResourceURL,
		expectedADEndpoint:    testActiveDirectoryEndpoint,
	},
	{
		name:    "nil cloud info",
		isError: false,
		vault: kedav1alpha1.AzureKeyVault{
			Cloud: nil,
		},
		expectedKVResourceURL: az.PublicCloud.ResourceIdentifiers.KeyVault,
		expectedADEndpoint:    az.PublicCloud.ActiveDirectoryEndpoint,
	},
	{
		name:    "invalid cloud",
		isError: true,
		vault: kedav1alpha1.AzureKeyVault{
			Cloud: &kedav1alpha1.AzureKeyVaultCloudInfo{
				Type: "invalid cloud",
			},
		},
		expectedKVResourceURL: "",
		expectedADEndpoint:    "",
	},
	{
		name:    "private cloud missing keyvault resource URL",
		isError: true,
		vault: kedav1alpha1.AzureKeyVault{
			Cloud: &kedav1alpha1.AzureKeyVaultCloudInfo{
				Type:                    "private",
				ActiveDirectoryEndpoint: testActiveDirectoryEndpoint,
			},
		},
		expectedKVResourceURL: "",
		expectedADEndpoint:    "",
	},
	{
		name:    "private cloud missing active directory endpoint",
		isError: true,
		vault: kedav1alpha1.AzureKeyVault{
			Cloud: &kedav1alpha1.AzureKeyVaultCloudInfo{
				Type:                "private",
				KeyVaultResourceURL: testResourceURL,
			},
		},
		expectedKVResourceURL: "",
		expectedADEndpoint:    "",
	},
}

func TestGetPropertiesForCloud(t *testing.T) {
	for _, testData := range testDataset {
		vh := NewAzureKeyVaultHandler(&testData.vault)

		kvResourceURL, adEndpoint, err := vh.getPropertiesForCloud()

		if err != nil && !testData.isError {
			t.Fatalf("test %s: expected success but got error - %s", testData.name, err)
		}

		if err == nil && testData.isError {
			t.Fatalf("test %s: expected error but got success, testData - %+v", testData.name, testData)
		}

		if kvResourceURL != testData.expectedKVResourceURL {
			t.Errorf("test %s: keyvault resource URl does not match. expected - %s, got - %s",
				testData.name, testData.expectedKVResourceURL, kvResourceURL)
		}

		if adEndpoint != testData.expectedADEndpoint {
			t.Errorf("test %s: active directory endpoint does not match. expected - %s, got - %s",
				testData.name, testData.expectedADEndpoint, adEndpoint)
		}
	}
}
