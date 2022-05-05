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
	"strconv"

	"github.com/Azure/azure-kusto-go/kusto"
	"github.com/Azure/azure-kusto-go/kusto/data/table"
	"github.com/Azure/azure-kusto-go/kusto/unsafe"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
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
	Threshold               int64
	ActiveDirectoryEndpoint string
}

var azureDataExplorerLogger = logf.Log.WithName("azure_data_explorer_scaler")

func CreateAzureDataExplorerClient(metadata *DataExplorerMetadata) (*kusto.Client, error) {
	authConfig, err := getDataExplorerAuthConfig(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to get data explorer auth config: %v", err)
	}

	client, err := kusto.New(metadata.Endpoint, kusto.Authorization{Config: *authConfig})
	if err != nil {
		return nil, fmt.Errorf("failed to create kusto client: %v", err)
	}

	return client, nil
}

func getDataExplorerAuthConfig(metadata *DataExplorerMetadata) (*auth.AuthorizerConfig, error) {
	var authConfig auth.AuthorizerConfig

	if metadata.PodIdentity.Provider != "" {
		config := auth.NewMSIConfig()
		config.Resource = metadata.Endpoint
		azureDataExplorerLogger.V(1).Info("Creating Azure Data Explorer Client using Pod Identity")

		authConfig = config
		return &authConfig, nil
	}

	if metadata.ClientID != "" && metadata.ClientSecret != "" && metadata.TenantID != "" {
		config := auth.NewClientCredentialsConfig(metadata.ClientID, metadata.ClientSecret, metadata.TenantID)
		config.Resource = metadata.Endpoint
		config.AADEndpoint = metadata.ActiveDirectoryEndpoint
		azureDataExplorerLogger.V(1).Info("Creating Azure Data Explorer Client using clientID, clientSecret and tenantID")

		authConfig = config
		return &authConfig, nil
	}

	return nil, fmt.Errorf("missing credentials. please reconfigure your scaled object metadata")
}

func GetAzureDataExplorerMetricValue(ctx context.Context, client *kusto.Client, db string, query string) (int64, error) {
	azureDataExplorerLogger.V(1).Info("Querying Azure Data Explorer", "db", db, "query", query)

	iter, err := client.Query(ctx, db, kusto.NewStmt("", kusto.UnsafeStmt(unsafe.Stmt{Add: true, SuppressWarning: false})).UnsafeAdd(query))
	if err != nil {
		return -1, fmt.Errorf("failed to get azure data explorer metric result from query %s: %v", query, err)
	}
	defer iter.Stop()

	row, inlineError, err := iter.NextRowOrError()
	if inlineError != nil {
		return -1, fmt.Errorf("failed to get query %s result: %v", query, inlineError)
	}
	if err != nil {
		return -1, fmt.Errorf("failed to get query %s result: %v", query, err)
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
		return -1, fmt.Errorf("failed to extract value from query %s: %v", query, err)
	}

	return metricValue, nil
}

func extractDataExplorerMetricValue(row *table.Row) (int64, error) {
	if row == nil || len(row.ColumnTypes) == 0 {
		return -1, fmt.Errorf("query has no results")
	}

	// Query result validation.
	dataType := row.ColumnTypes[0].Type
	if dataType != "real" && dataType != "int" && dataType != "long" {
		return -1, fmt.Errorf("data type %s is not valid", dataType)
	}

	value, err := strconv.Atoi(row.Values[0].String())
	if err != nil {
		return -1, fmt.Errorf("failed to convert result %s to int", row.Values[0].String())
	}
	if value < 0 {
		return -1, fmt.Errorf("query result must be >= 0 but received: %d", value)
	}

	azureDataExplorerLogger.V(1).Info("Query Result", "value", value, "dataType", dataType)
	return int64(value), nil
}
