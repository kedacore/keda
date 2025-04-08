// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

@description('The base resource name.')
param baseName string = resourceGroup().name

@description('The client OID to grant access to test resources.')
param testApplicationOid string

@description('The resource location')
param location string = resourceGroup().location

@description('Enable deploying a premium service bus')
param enablePremium bool = true

@description('Disable applying any RBAC rules')
param disableAddingRBACRole bool = false

param tenantIsTME bool = false

var apiVersion = '2017-04-01'
var serviceBusDataOwnerRoleId = '/subscriptions/${subscription().subscriptionId}/providers/Microsoft.Authorization/roleDefinitions/090c5cfd-751d-490a-894a-3ce6f1109419'

var sbPremiumName = 'sb-premium-${baseName}'

resource servicebus 'Microsoft.ServiceBus/namespaces@2024-01-01' = {
  name: baseName
  location: location 
  sku: {
    name: 'Standard'
    tier: 'Standard'
  }
  properties: {
    disableLocalAuth: !tenantIsTME
    zoneRedundant: false
  }
}

resource servicebusPremium 'Microsoft.ServiceBus/namespaces@2024-01-01' = if (enablePremium) {
  name: sbPremiumName
  location: location
  sku: {
    name: 'Premium'
    tier: 'Premium'
  }
  properties: {
    disableLocalAuth: !tenantIsTME    
  }
}

resource authorizationRuleName 'Microsoft.ServiceBus/namespaces/AuthorizationRules@2015-08-01' = {
  parent: servicebus
  name: 'RootManageSharedAccessKey'
  location: location
  properties: {
    rights: [
      'Listen'
      'Manage'
      'Send'
    ]
  }
}

resource authorizationRuleNameNoManage 'Microsoft.ServiceBus/namespaces/AuthorizationRules@2015-08-01' = {
  parent: servicebus
  name: 'NoManage'
  location: location
  properties: {
    rights: [
      'Listen'
      'Send'
    ]
  }
}

resource authorizationRuleNameSendOnly 'Microsoft.ServiceBus/namespaces/AuthorizationRules@2015-08-01' = {
  parent: servicebus
  name: 'SendOnly'
  location: location
  properties: {
    rights: [
      'Send'
    ]
  }
}

resource authorizationRuleNameListenOnly 'Microsoft.ServiceBus/namespaces/AuthorizationRules@2015-08-01' = {
  parent: servicebus
  name: 'ListenOnly'
  location: location
  properties: {
    rights: [
      'Listen'
    ]
  }
}

resource dataOwnerRoleId 'Microsoft.Authorization/roleAssignments@2018-01-01-preview' = if (!disableAddingRBACRole) {
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

output SERVICEBUS_CONNECTION_STRING string = tenantIsTME ? listKeys(
  resourceId('Microsoft.ServiceBus/namespaces/authorizationRules', baseName, 'RootManageSharedAccessKey'),
  apiVersion
).primaryConnectionString : ''

// connection strings with fewer rights - no manage rights, listen only (ie, receive) and send only.
output SERVICEBUS_CONNECTION_STRING_NO_MANAGE string = tenantIsTME ? listKeys(
  resourceId('Microsoft.ServiceBus/namespaces/authorizationRules', baseName, 'NoManage'),
  apiVersion
).primaryConnectionString : ''
output SERVICEBUS_CONNECTION_STRING_SEND_ONLY string = tenantIsTME ? listKeys(
  resourceId('Microsoft.ServiceBus/namespaces/authorizationRules', baseName, 'SendOnly'),
  apiVersion
).primaryConnectionString : ''
output SERVICEBUS_CONNECTION_STRING_LISTEN_ONLY string = tenantIsTME ? listKeys(
  resourceId('Microsoft.ServiceBus/namespaces/authorizationRules', baseName, 'ListenOnly'),
  apiVersion
).primaryConnectionString: ''

output SERVICEBUS_CONNECTION_STRING_PREMIUM string = enablePremium && tenantIsTME
  ? listKeys(
      resourceId('Microsoft.ServiceBus/namespaces/authorizationRules', sbPremiumName, 'RootManageSharedAccessKey'),
      apiVersion
    ).primaryConnectionString
  : ''

output SERVICEBUS_ENDPOINT string = replace(
  replace(servicebus.properties.serviceBusEndpoint, ':443/', ''),
  'https://',
  ''
)
output SERVICEBUS_ENDPOINT_PREMIUM string = enablePremium
  ? replace(replace(servicebusPremium.properties.serviceBusEndpoint, ':443/', ''), 'https://', '')
  : ''

output QUEUE_NAME string = 'testQueue'
output QUEUE_NAME_WITH_SESSIONS string = 'testQueueWithSessions'
