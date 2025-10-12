// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package azmetrics

import "github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"

// Cloud Service Names for Monitor Query Metrics, used to identify the respective cloud.ServiceConfiguration
const ServiceName cloud.ServiceName = "query/azmetrics"

func init() {
	cloud.AzureChina.Services[ServiceName] = cloud.ServiceConfiguration{
		Audience: "https://metrics.monitor.azure.cn",
	}
	cloud.AzureGovernment.Services[ServiceName] = cloud.ServiceConfiguration{
		Audience: "https://metrics.monitor.azure.us",
	}
	cloud.AzurePublic.Services[ServiceName] = cloud.ServiceConfiguration{
		Audience: "https://metrics.monitor.azure.com",
	}
}
