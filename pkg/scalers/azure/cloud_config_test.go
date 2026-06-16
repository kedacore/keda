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

package azure

import (
	"encoding/json"
	"os"
	"testing"
)

type environmentFromNameTestData struct {
	name	string
	expected	AzEnvironment
	isError	bool
}

var environmentFromNameTestDataset = []environmentFromNameTestData{
	{"AzurePublicCloud", PublicCloud, false},
	{"AZURECLOUD", PublicCloud, false},
	{"azurecloud", PublicCloud, false},
	{"AzureUSGovernmentCloud", USGovernmentCloud, false},
	{"AZUREUSGOVERNMENT", USGovernmentCloud, false},
	{"AzureChinaCloud", ChinaCloud, false},
	{"AzureGermanCloud", GermanCloud, false},
	{"InvalidCloud", AzEnvironment{}, true},
	{"", AzEnvironment{}, true},
}

func TestEnvironmentFromName(t *testing.T) {
	for _, testData := range environmentFromNameTestDataset {
		env, err := EnvironmentFromName(testData.name)
		if testData.isError && err == nil {
			t.Errorf("For cloud name %q: expected error but got success", testData.name)
			continue
		}
		if !testData.isError && err != nil {
			t.Errorf("For cloud name %q: expected success but got error: %v", testData.name, err)
			continue
		}
		if !testData.isError && env.Name != testData.expected.Name {
			t.Errorf("For cloud name %q: expected env.Name=%q but got %q", testData.name, testData.expected.Name, env.Name)
		}
	}
}

func TestEnvironmentFromFile(t *testing.T) {
	expected := AzEnvironment{
		Name:	"CustomCloud",
		ResourceManagerEndpoint:	"https://management.custom.cloud/",
		ActiveDirectoryEndpoint:	"https://login.custom.cloud/",
		StorageEndpointSuffix:	"core.custom.cloud",
	}

	data, err := json.Marshal(expected)
	if err != nil {
		t.Fatalf("failed to marshal test environment: %v", err)
	}

	f, err := os.CreateTemp("", "azure-env-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(f.Name())

	if _, err = f.Write(data); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	f.Close()

	env, err := EnvironmentFromFile(f.Name())
	if err != nil {
		t.Fatalf("EnvironmentFromFile returned unexpected error: %v", err)
	}
	if env.Name != expected.Name {
		t.Errorf("expected env.Name=%q but got %q", expected.Name, env.Name)
	}
	if env.ResourceManagerEndpoint != expected.ResourceManagerEndpoint {
		t.Errorf("expected ResourceManagerEndpoint=%q but got %q", expected.ResourceManagerEndpoint, env.ResourceManagerEndpoint)
	}
	if env.StorageEndpointSuffix != expected.StorageEndpointSuffix {
		t.Errorf("expected StorageEndpointSuffix=%q but got %q", expected.StorageEndpointSuffix, env.StorageEndpointSuffix)
	}
}

func TestEnvironmentFromFileNotFound(t *testing.T) {
	_, err := EnvironmentFromFile("/nonexistent/path/env.json")
	if err == nil {
		t.Error("expected error for nonexistent file but got success")
	}
}

func TestSetEnvironment(t *testing.T) {
	customEnv := AzEnvironment{
		Name:	"TestCustomCloud",
		ResourceManagerEndpoint:	"https://management.test.cloud/",
	}
	SetEnvironment("AzureTestCustomCloud", customEnv)

	env, err := EnvironmentFromName("AzureTestCustomCloud")
	if err != nil {
		t.Fatalf("expected to find registered custom environment but got error: %v", err)
	}
	if env.Name != customEnv.Name {
		t.Errorf("expected env.Name=%q but got %q", customEnv.Name, env.Name)
	}

	// cleanup
	delete(environments, "AZURETESTCUSTOMCLOUD")
}
