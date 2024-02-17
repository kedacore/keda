@description('The base resource name.')
param baseName string = resourceGroup().name

#disable-next-line no-hardcoded-env-urls // it's flagging the help string.
@description('Storage endpoint suffix. The default value uses Azure Public Cloud (ie: core.windows.net)')
param storageEndpointSuffix string = environment().suffixes.storage

@description('The resource location')
param location string = resourceGroup().location

var apiVersion = '2017-04-01'
var storageApiVersion = '2019-04-01'
var namespaceName = baseName
var storageAccountName = 'storage${baseName}'
var containerName = 'container'
var iotName = 'iot${baseName}'
var authorizationName = '${baseName}/RootManageSharedAccessKey'

resource namespace 'Microsoft.EventHub/namespaces@2017-04-01' = {
  name: namespaceName
  location: location
  sku: {
    name: 'Standard'
    tier: 'Standard'
    capacity: 5
  }
  properties: {
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
  properties: {
  }
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

resource iot 'Microsoft.Devices/IotHubs@2018-04-01' = {
  name: iotName
  location: location
  sku: {
    name: 'S1'
    capacity: 1
  }
  properties: {
    ipFilterRules: []
    eventHubEndpoints: {
      events: {
        retentionTimeInDays: 1
        partitionCount: 4
      }
    }
    routing: {
      endpoints: {
        serviceBusQueues: []
        serviceBusTopics: []
        eventHubs: []
        storageContainers: []
      }
      routes: []
      fallbackRoute: {
        name: '$fallback'
        source: 'DeviceMessages'
        condition: 'true'
        endpointNames: [
          'events'
        ]
        isEnabled: true
      }
    }
    storageEndpoints: {
      '$default': {
        sasTtlAsIso8601: 'PT1H'
        connectionString: 'DefaultEndpointsProtocol=https;AccountName=${storageAccountName};AccountKey=${listKeys(storageAccount.id, storageApiVersion).keys[0].value};EndpointSuffix=${storageEndpointSuffix}'
        containerName: containerName
      }
    }
    messagingEndpoints: {
      fileNotifications: {
        lockDurationAsIso8601: 'PT1M'
        ttlAsIso8601: 'PT1H'
        maxDeliveryCount: 10
      }
    }
    enableFileUploadNotifications: false
    cloudToDevice: {
      maxDeliveryCount: 10
      defaultTtlAsIso8601: 'PT1H'
      feedback: {
        lockDurationAsIso8601: 'PT1M'
        ttlAsIso8601: 'PT1H'
        maxDeliveryCount: 10
      }
    }
    features: 'None'
  }
}

output EVENTHUB_NAME string = eventHub.name
output EVENTHUB_LINKSONLY_NAME string = linksonly.name
output EVENTHUB_CONNECTION_STRING string = listKeys(resourceId('Microsoft.EventHub/namespaces/authorizationRules', namespaceName, 'RootManageSharedAccessKey'), apiVersion).primaryConnectionString
output EVENTHUB_CONNECTION_STRING_LISTEN_ONLY string = listKeys(resourceId('Microsoft.EventHub/namespaces/authorizationRules', namespaceName, authorizedListenOnly.name), apiVersion).primaryConnectionString
output EVENTHUB_CONNECTION_STRING_SEND_ONLY string = listKeys(resourceId('Microsoft.EventHub/namespaces/authorizationRules', namespaceName, authorizedSendOnly.name), apiVersion).primaryConnectionString
output IOTHUB_CONNECTION_STRING string = 'HostName=${reference(iot.id, providers('Microsoft.Devices', 'IoTHubs').apiVersions[0]).hostName};SharedAccessKeyName=iothubowner;SharedAccessKey=${listKeys(iot.id, providers('Microsoft.Devices', 'IoTHubs').apiVersions[0]).value[0].primaryKey}'
output CHECKPOINTSTORE_STORAGE_CONNECTION_STRING string = 'DefaultEndpointsProtocol=https;AccountName=${storageAccountName};AccountKey=${listKeys(storageAccount.id, storageApiVersion).keys[0].value};EndpointSuffix=${storageEndpointSuffix}'
output RESOURCE_GROUP string = resourceGroup().name
output AZURE_SUBSCRIPTION_ID string = subscription().subscriptionId
