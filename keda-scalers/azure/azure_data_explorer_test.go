/*
Copyright 2021 The KEDA Authors

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
	"testing"

	"github.com/Azure/azure-kusto-go/kusto/data/errors"
	"github.com/Azure/azure-kusto-go/kusto/data/table"
	"github.com/Azure/azure-kusto-go/kusto/data/types"
	"github.com/Azure/azure-kusto-go/kusto/data/value"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

type testExtractDataExplorerMetricValue struct {
	testRow *table.Row
	isError bool
}

type testGetDataExplorerAuthConfig struct {
	testMetadata *DataExplorerMetadata
	isError      bool
}

var (
	clientID              = "test_client_id"
	rowName               = "result"
	rowType  types.Column = "long"
	rowValue int64        = 3
	secret                = "test_secret"
	tenantID              = "test_tenant_id"
)

var testExtractDataExplorerMetricValues = []testExtractDataExplorerMetricValue{
	// pass
	{testRow: &table.Row{ColumnTypes: table.Columns{{Name: rowName, Type: rowType}}, Values: value.Values{value.Long{Value: rowValue, Valid: true}}, Op: errors.OpQuery}, isError: false},
	// nil row - fail
	{testRow: nil, isError: true},
	// Empty row - fail
	{testRow: &table.Row{}, isError: true},
	// Metric value is not bigger than 0 - fail
	{testRow: &table.Row{ColumnTypes: table.Columns{{Name: rowName, Type: rowType}}, Values: value.Values{value.Long{Value: -1, Valid: true}}, Op: errors.OpQuery}, isError: true},
	// Metric result is invalid - fail
	{testRow: &table.Row{ColumnTypes: table.Columns{{Name: rowName, Type: rowType}}, Values: value.Values{value.String{Value: "invalid", Valid: true}}, Op: errors.OpQuery}, isError: true},
	// Metric Type is not valid - fail
	{testRow: &table.Row{ColumnTypes: table.Columns{{Name: rowName, Type: "String"}}, Values: value.Values{value.Long{Value: rowValue, Valid: true}}, Op: errors.OpQuery}, isError: true},
}

var testGetDataExplorerAuthConfigs = []testGetDataExplorerAuthConfig{
	// Auth with aad app - pass
	{testMetadata: &DataExplorerMetadata{ClientID: clientID, ClientSecret: secret, TenantID: tenantID, Endpoint: "https://test.kusto.windows.net", ActiveDirectoryEndpoint: "https://test.kusto.windows.net"}, isError: false},
	// Auth with workload identity - pass
	{testMetadata: &DataExplorerMetadata{PodIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAzureWorkload}, Endpoint: "https://test.kusto.windows.net", ActiveDirectoryEndpoint: "https://test.kusto.windows.net"}, isError: false},
	// Empty metadata - fail
	{testMetadata: &DataExplorerMetadata{Endpoint: "https://test.kusto.windows.net", ActiveDirectoryEndpoint: "https://test.kusto.windows.net"}, isError: true},
	// Empty tenantID - fail
	{testMetadata: &DataExplorerMetadata{ClientID: clientID, ClientSecret: secret, Endpoint: "https://test.kusto.windows.net", ActiveDirectoryEndpoint: "https://test.kusto.windows.net"}, isError: true},
	// Empty clientID - fail
	{testMetadata: &DataExplorerMetadata{ClientSecret: secret, TenantID: tenantID, Endpoint: "https://test.kusto.windows.net", ActiveDirectoryEndpoint: "https://test.kusto.windows.net"}, isError: true},
	// Empty clientSecret - fail
	{testMetadata: &DataExplorerMetadata{ClientID: clientID, TenantID: tenantID, Endpoint: "https://test.kusto.windows.net", ActiveDirectoryEndpoint: "https://test.kusto.windows.net"}, isError: true},
}

func TestExtractDataExplorerMetricValue(t *testing.T) {
	for _, testData := range testExtractDataExplorerMetricValues {
		_, err := extractDataExplorerMetricValue(testData.testRow)
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

func TestGetDataExplorerAuthConfig(t *testing.T) {
	for _, testData := range testGetDataExplorerAuthConfigs {
		_, err := getDataExplorerAuthConfig(testData.testMetadata)
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}
