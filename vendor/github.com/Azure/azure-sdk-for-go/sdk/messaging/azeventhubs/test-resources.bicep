// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

@description('The base resource name.')
param baseName string = resourceGroup().name

#disable-next-line no-hardcoded-env-urls // it's flagging the help string.
@description('Storage endpoint suffix. The default value uses Azure Public Cloud (ie: core.windows.net)')
param storageEndpointSuffix string = environment().suffixes.storage

@description('The resource location')
param location string = resourceGroup().location

param tenantIsTME bool = false

var apiVersion = '2017-04-01'
var namespaceName = baseName
var storageAccountName = 'storage${baseName}'
var containerName = 'container'
var authorizationName = '${baseName}/RootManageSharedAccessKey'

resource namespace 'Microsoft.EventHub/namespaces@2024-01-01' = {
  name: namespaceName
  location: location
  sku: {
    name: 'Standard'
    tier: 'Standard'
    capacity: 5
  }
  properties: {
    disableLocalAuth: !tenantIsTME
    isAutoInflateEnabled: false
    maximumThroughputUnits: 0
  }
}

resource authorization 'Microsoft.EventHub/namespaces/AuthorizationRules@2017-04-01' = {
  name: authorizationName
  properties: {
    rights: [
      'Listen'
      'Manage'
      'Send'
    ]
  }
  dependsOn: [
    namespace
  ]
}

resource authorizedListenOnly 'Microsoft.EventHub/namespaces/AuthorizationRules@2017-04-01' = {
  name: 'ListenOnly'
  parent: namespace
  properties: {
    rights: [
      'Listen'
    ]
  }
}

resource authorizedSendOnly 'Microsoft.EventHub/namespaces/AuthorizationRules@2017-04-01' = {
  name: 'SendOnly'
  parent: namespace
  properties: {
    rights: [
      'Send'
    ]
  }
}

resource eventHub 'Microsoft.EventHub/namespaces/eventhubs@2017-04-01' = {
  name: 'eventhub'
  properties: {
    messageRetentionInDays: 1
    partitionCount: 4
  }
  parent: namespace
}

resource linksonly 'Microsoft.EventHub/namespaces/eventhubs@2017-04-01' = {
  name: 'linksonly'
  properties: {
    messageRetentionInDays: 1
    partitionCount: 1
  }
  parent: namespace
}

resource namespaceName_default 'Microsoft.EventHub/namespaces/networkRuleSets@2017-04-01' = {
  name: 'default'
  parent: namespace
  properties: {
    defaultAction: 'Deny'
    virtualNetworkRules: []
    ipRules: []
  }
}

resource eventHubNameFull_Default 'Microsoft.EventHub/namespaces/eventhubs/consumergroups@2017-04-01' = {
  name: '$Default'
  properties: {}
  parent: eventHub
}

resource storageAccount 'Microsoft.Storage/storageAccounts@2019-04-01' = {
  name: storageAccountName
  location: location
  sku: {
    name: 'Standard_RAGRS'
  }
  kind: 'StorageV2'
  properties: {
    allowSharedKeyAccess: false
    networkAcls: {
      bypass: 'AzureServices'
      virtualNetworkRules: []
      ipRules: []
      defaultAction: 'Allow'
    }
    supportsHttpsTrafficOnly: true
    encryption: {
      services: {
        file: {
          enabled: true
        }
        blob: {
          enabled: true
        }
      }
      keySource: 'Microsoft.Storage'
    }
    accessTier: 'Hot'
  }
}

resource storageAccountName_default_container 'Microsoft.Storage/storageAccounts/blobServices/containers@2019-04-01' = {
  name: '${storageAccountName}/default/${containerName}'
  dependsOn: [
    storageAccount
  ]
}

// used for TokenCredential tests
output EVENTHUB_NAMESPACE string = '${namespace.name}.servicebus.windows.net'
output CHECKPOINTSTORE_STORAGE_ENDPOINT string = storageAccount.properties.primaryEndpoints.blob
output EVENTHUB_NAME string = eventHub.name
output EVENTHUB_LINKSONLY_NAME string = linksonly.name

// connection strings
output EVENTHUB_CONNECTION_STRING string = tenantIsTME
  ? listKeys(
      resourceId('Microsoft.EventHub/namespaces/authorizationRules', namespaceName, 'RootManageSharedAccessKey'),
      apiVersion
    ).primaryConnectionString
  : ''

output EVENTHUB_CONNECTION_STRING_LISTEN_ONLY string = tenantIsTME
  ? listKeys(
      resourceId('Microsoft.EventHub/namespaces/authorizationRules', namespaceName, authorizedListenOnly.name),
      apiVersion
    ).primaryConnectionString
  : ''
output EVENTHUB_CONNECTION_STRING_SEND_ONLY string = tenantIsTME
  ? listKeys(
      resourceId('Microsoft.EventHub/namespaces/authorizationRules', namespaceName, authorizedSendOnly.name),
      apiVersion
    ).primaryConnectionString
  : ''

output RESOURCE_GROUP string = resourceGroup().name
output AZURE_SUBSCRIPTION_ID string = subscription().subscriptionId
