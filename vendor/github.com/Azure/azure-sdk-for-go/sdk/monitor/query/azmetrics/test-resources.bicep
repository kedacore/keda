// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

param baseName string
param appSku string = 'standard'
param location string = resourceGroup().location

resource app_config 'Microsoft.AppConfiguration/configurationStores@2022-05-01' = {
  name: baseName
  location: location
  sku: {
    name: appSku
  }
}

output RESOURCE_URI string = app_config.id
