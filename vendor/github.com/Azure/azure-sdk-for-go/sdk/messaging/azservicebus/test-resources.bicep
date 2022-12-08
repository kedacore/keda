@description('The base resource name.')
param baseName string = resourceGroup().name

@description('The client OID to grant access to test resources.')
param testApplicationOid string

var apiVersion = '2017-04-01'
var location = resourceGroup().location
var authorizationRuleName_var = '${baseName}/RootManageSharedAccessKey'
var authorizationRuleNameNoManage_var = '${baseName}/NoManage'
var serviceBusDataOwnerRoleId = '/subscriptions/${subscription().subscriptionId}/providers/Microsoft.Authorization/roleDefinitions/090c5cfd-751d-490a-894a-3ce6f1109419'

var sbPremiumName = 'sb-premium-${baseName}'

resource servicebus 'Microsoft.ServiceBus/namespaces@2018-01-01-preview' = {
  name: baseName
  location: location
  sku: {
    name: 'Standard'
    tier: 'Standard'
  }
  properties: {
    zoneRedundant: false
  }
}

resource servicebusPremium 'Microsoft.ServiceBus/namespaces@2018-01-01-preview' = {
  name: sbPremiumName
  location: location
  sku: {
    name: 'Premium'
    tier: 'Premium'
  }
}


resource authorizationRuleName 'Microsoft.ServiceBus/namespaces/AuthorizationRules@2015-08-01' = {
  name: authorizationRuleName_var
  location: location
  properties: {
    rights: [
      'Listen'
      'Manage'
      'Send'
    ]
  }
  dependsOn: [
    servicebus
  ]
}

resource authorizationRuleNameNoManage 'Microsoft.ServiceBus/namespaces/AuthorizationRules@2015-08-01' = {
  name: authorizationRuleNameNoManage_var
  location: location
  properties: {
    rights: [
      'Listen'
      'Send'
    ]
  }
  dependsOn: [
    servicebus
  ]
}



resource dataOwnerRoleId 'Microsoft.Authorization/roleAssignments@2018-01-01-preview' = {
  name: guid('dataOwnerRoleId${baseName}')
  properties: {
    roleDefinitionId: serviceBusDataOwnerRoleId
    principalId: testApplicationOid
  }
  dependsOn: [
    servicebus
  ]
}

resource testQueue 'Microsoft.ServiceBus/namespaces/queues@2017-04-01' = {
  parent: servicebus
  name: 'testQueue'
  properties: {
    lockDuration: 'PT5M'
    maxSizeInMegabytes: 1024
    requiresDuplicateDetection: false
    requiresSession: false
    defaultMessageTimeToLive: 'P10675199DT2H48M5.4775807S'
    deadLetteringOnMessageExpiration: false
    duplicateDetectionHistoryTimeWindow: 'PT10M'
    maxDeliveryCount: 10
    autoDeleteOnIdle: 'P10675199DT2H48M5.4775807S'
    enablePartitioning: false
    enableExpress: false
  }
}

resource testQueueWithSessions 'Microsoft.ServiceBus/namespaces/queues@2017-04-01' = {
  parent: servicebus
  name: 'testQueueWithSessions'
  properties: {
    lockDuration: 'PT5M'
    maxSizeInMegabytes: 1024
    requiresDuplicateDetection: false
    requiresSession: true
    defaultMessageTimeToLive: 'P10675199DT2H48M5.4775807S'
    deadLetteringOnMessageExpiration: false
    duplicateDetectionHistoryTimeWindow: 'PT10M'
    maxDeliveryCount: 10
    autoDeleteOnIdle: 'P10675199DT2H48M5.4775807S'
    enablePartitioning: false
    enableExpress: false
  }
}

output SERVICEBUS_CONNECTION_STRING string = listKeys(resourceId('Microsoft.ServiceBus/namespaces/authorizationRules', baseName, 'RootManageSharedAccessKey'), apiVersion).primaryConnectionString
output SERVICEBUS_CONNECTION_STRING_NO_MANAGE string = listKeys(resourceId('Microsoft.ServiceBus/namespaces/authorizationRules', baseName, 'NoManage'), apiVersion).primaryConnectionString
output SERVICEBUS_CONNECTION_STRING_PREMIUM string = listKeys(resourceId('Microsoft.ServiceBus/namespaces/authorizationRules', sbPremiumName, 'RootManageSharedAccessKey'), apiVersion).primaryConnectionString
output SERVICEBUS_ENDPOINT string = replace(replace(servicebus.properties.serviceBusEndpoint, ':443/', ''), 'https://', '')
output QUEUE_NAME string = 'testQueue'
output QUEUE_NAME_WITH_SESSIONS string = 'testQueueWithSessions'
