// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

param baseName string
param sku string = 'pergb2018'
param appSku string = 'standard'
param retentionInDays int = 30
param resourcePermissions bool = false
param location string = resourceGroup().location

resource log_analytics1 'Microsoft.OperationalInsights/workspaces@2020-08-01' = {
  name: '${baseName}1'
  location: location
  properties: {
    sku: {
      name: sku
    }
    retentionInDays: retentionInDays
    features: {
      searchVersion: 1
      legacy: 0
      enableLogAccessUsingOnlyResourcePermissions: resourcePermissions
    }
  }
}

resource log_analytics2 'Microsoft.OperationalInsights/workspaces@2020-08-01' = {
  name: '${baseName}2'
  location: location
  properties: {
    sku: {
      name: sku
    }
    retentionInDays: retentionInDays
    features: {
      searchVersion: 1
      legacy: 0
      enableLogAccessUsingOnlyResourcePermissions: resourcePermissions
    }
  }
}

resource app_config 'Microsoft.AppConfiguration/configurationStores@2022-05-01' = {
  name: baseName
  location: location
  sku: {
    name: appSku
  }
  properties: {
    disableLocalAuth: true 
  }
}

output WORKSPACE_ID string = log_analytics1.properties.customerId
output WORKSPACE_ID2 string = log_analytics2.properties.customerId
output RESOURCE_URI string = app_config.id
