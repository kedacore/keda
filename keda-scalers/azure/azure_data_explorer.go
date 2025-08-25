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
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/Azure/azure-kusto-go/kusto"
	"github.com/Azure/azure-kusto-go/kusto/data/table"
	"github.com/Azure/azure-kusto-go/kusto/kql"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

type DataExplorerMetadata struct {
	ClientID                string
	ClientSecret            string
	DatabaseName            string
	Endpoint                string
	MetricName              string
	PodIdentity             kedav1alpha1.AuthPodIdentity
	Query                   string
	TenantID                string
	Threshold               float64
	ActivationThreshold     float64
	ActiveDirectoryEndpoint string
}

var azureDataExplorerLogger = logf.Log.WithName("azure_data_explorer_scaler")

func CreateAzureDataExplorerClient(metadata *DataExplorerMetadata, httpClient *http.Client) (*kusto.Client, error) {
	kcsb, err := getDataExplorerAuthConfig(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to get data explorer auth config: %w", err)
	}

	client, err := kusto.New(kcsb, kusto.WithHttpClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("failed to create kusto client: %w", err)
	}

	return client, nil
}

func getDataExplorerAuthConfig(metadata *DataExplorerMetadata) (*kusto.ConnectionStringBuilder, error) {
	kcsb := kusto.NewConnectionStringBuilder(metadata.Endpoint)

	switch metadata.PodIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		azureDataExplorerLogger.V(1).Info("Creating Azure Data Explorer Client using clientID, clientSecret and tenantID")
		if metadata.ClientID == "" {
			return nil, fmt.Errorf("missing credentials. please ensure that ClientID is provided")
		}
		if metadata.ClientSecret == "" {
			return nil, fmt.Errorf("missing credentials. please ensure that ClientSecret is provided")
		}
		if metadata.TenantID == "" {
			return nil, fmt.Errorf("missing credentials. please ensure that TenantID is provided")
		}
		kcsb.WithAadAppKey(metadata.ClientID, metadata.ClientSecret, metadata.TenantID)
		// This should be here because internally the SDK resets the configuration
		// after calling `WithAadAppKey`
		clientOptions := &policy.ClientOptions{
			Cloud: cloud.Configuration{
				ActiveDirectoryAuthorityHost: metadata.ActiveDirectoryEndpoint,
				Services:                     map[cloud.ServiceName]cloud.ServiceConfiguration{},
			},
		}
		kcsb.AttachPolicyClientOptions(clientOptions)

	case kedav1alpha1.PodIdentityProviderAzureWorkload:
		azureDataExplorerLogger.V(1).Info(fmt.Sprintf("Creating Azure Data Explorer Client using podIdentity %s", metadata.PodIdentity.Provider))
		creds, chainedErr := NewChainedCredential(azureDataExplorerLogger, metadata.PodIdentity)
		if chainedErr != nil {
			return nil, chainedErr
		}
		kcsb.WithTokenCredential(creds)

	default:
		return nil, fmt.Errorf("missing credentials. please reconfigure your scaled object metadata")
	}

	return kcsb, nil
}

func GetAzureDataExplorerMetricValue(ctx context.Context, client *kusto.Client, db string, query string) (float64, error) {
	azureDataExplorerLogger.V(1).Info("Querying Azure Data Explorer", "db", db, "query", query)

	iter, err := client.Query(ctx, db, kql.New("").AddUnsafe(query))
	if err != nil {
		return -1, fmt.Errorf("failed to get azure data explorer metric result from query %s: %w", query, err)
	}
	defer iter.Stop()

	row, inlineError, err := iter.NextRowOrError()
	if inlineError != nil {
		return -1, fmt.Errorf("failed to get query %s result: %v", query, inlineError)
	}
	if err != nil {
		return -1, fmt.Errorf("failed to get query %s result: %w", query, err)
	}

	if !row.ColumnTypes[0].Type.Valid() {
		return -1, fmt.Errorf("column type %s is not valid", row.ColumnTypes[0].Type)
	}

	// Return error if there is more than one row.
	_, _, err = iter.NextRowOrError()
	if err != io.EOF {
		return -1, fmt.Errorf("query %s result had more than a single result row", query)
	}

	metricValue, err := extractDataExplorerMetricValue(row)
	if err != nil {
		return -1, fmt.Errorf("failed to extract value from query %s: %w", query, err)
	}

	return metricValue, nil
}

func extractDataExplorerMetricValue(row *table.Row) (float64, error) {
	if row == nil || len(row.ColumnTypes) == 0 {
		return -1, fmt.Errorf("query has no results")
	}

	// Query result validation.
	dataType := row.ColumnTypes[0].Type
	if dataType != "real" && dataType != "int" && dataType != "long" {
		return -1, fmt.Errorf("data type %s is not valid", dataType)
	}

	value, err := strconv.ParseFloat(row.Values[0].String(), 64)
	if err != nil {
		return -1, fmt.Errorf("failed to convert result %s to int", row.Values[0].String())
	}
	if value < 0 {
		return -1, fmt.Errorf("query result must be >= 0 but received: %f", value)
	}

	azureDataExplorerLogger.V(1).Info("Query Result", "value", value, "dataType", dataType)
	return value, nil
}
