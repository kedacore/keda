// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

@description('The base resource name.')
param baseName string = resourceGroup().name

@description('The resource location')
param location string = resourceGroup().location

@description('The client OID to grant access to test resources.')
param testApplicationOid string

output RESOURCE_GROUP string = resourceGroup().name
output AZURE_SUBSCRIPTION_ID string = subscription().subscriptionId

resource egTopic 'Microsoft.EventGrid/topics@2023-06-01-preview' = {
  name: '${baseName}-eg'
  location: location
  kind: 'Azure'
  properties: {
    inputSchema: 'EventGridSchema'
  }
}

resource ceTopic 'Microsoft.EventGrid/topics@2023-06-01-preview' = {
  name: '${baseName}-ce'
  location: location
  kind: 'Azure'
  properties: {
    inputSchema: 'CloudEventSchemaV1_0'
  }
}

resource egContributorRole 'Microsoft.Authorization/roleAssignments@2018-01-01-preview' = {
  name: guid('egContributorRoleId${baseName}')
  scope: resourceGroup()
  properties: {
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', '1e241071-0855-49ea-94dc-649edcd759de')
    //    roleDefinitionId: '/subscriptions/${subscription().subscriptionId}/providers/Microsoft.Authorization/roleDefinitions/1e241071-0855-49ea-94dc-649edcd759de'
    principalId: testApplicationOid
  }
}

resource egDataSenderRole 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid('egSenderRoleId${baseName}')
  scope: resourceGroup()
  properties: {
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', 'd5a91429-5739-47e2-a06b-3470a27159e7')
    principalId: testApplicationOid
  }
}

output EVENTGRID_TOPIC_NAME string = egTopic.name
#disable-next-line outputs-should-not-contain-secrets // (this is just how our test deployments work)
output EVENTGRID_TOPIC_KEY string = egTopic.listKeys().key1
output EVENTGRID_TOPIC_ENDPOINT string = egTopic.properties.endpoint

output EVENTGRID_CE_TOPIC_NAME string = ceTopic.name
#disable-next-line outputs-should-not-contain-secrets // (this is just how our test deployments work)
output EVENTGRID_CE_TOPIC_KEY string = ceTopic.listKeys().key1
output EVENTGRID_CE_TOPIC_ENDPOINT string = ceTopic.properties.endpoint
