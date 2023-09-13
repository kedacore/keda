@description('The base resource name.')
param baseName string = resourceGroup().name

@description('The resource location')
param location string = resourceGroup().location

var namespaceName = '${baseName}-2'
var topicName = 'testtopic1'
var subscriptionName = 'testsubscription1'

resource ns_resource 'Microsoft.EventGrid/namespaces@2023-06-01-preview' = {
  name: namespaceName
  location: location
  sku: {
    name: 'Standard'
    capacity: 1
  }
  properties: {
    isZoneRedundant: true
    publicNetworkAccess: 'Enabled'
  }
}

resource ns_testtopic1 'Microsoft.EventGrid/namespaces/topics@2023-06-01-preview' = {
  parent: ns_resource
  name: topicName
  properties: {
    publisherType: 'Custom'
    inputSchema: 'CloudEventSchemaV1_0'
    eventRetentionInDays: 1
  }
}

resource ns_testtopic1_testsubscription1 'Microsoft.EventGrid/namespaces/topics/eventSubscriptions@2023-06-01-preview' = {
  parent: ns_testtopic1
  name: subscriptionName
  properties: {
    deliveryConfiguration: {
      deliveryMode: 'Queue'
      queue: {
        receiveLockDurationInSeconds: 60
        maxDeliveryCount: 10
        eventTimeToLive: 'P1D'
      }
    }
    eventDeliverySchema: 'CloudEventSchemaV1_0'
    filtersConfiguration: {
      includedEventTypes: []
    }
  }
}

// https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane-version2023-06-01-preview/namespaces/list-shared-access-keys?tabs=HTTP
output EVENTGRID_KEY string = listKeys(resourceId('Microsoft.EventGrid/namespaces', namespaceName), '2023-06-01-preview').key1
// TODO: get this formatted properly
output EVENTGRID_ENDPOINT string = 'https://${ns_resource.properties.topicsConfiguration.hostname}'

output EVENTGRID_TOPIC string = topicName
output EVENTGRID_SUBSCRIPTION string = subscriptionName
output RESOURCE_GROUP string = resourceGroup().name
output AZURE_SUBSCRIPTION_ID string = subscription().subscriptionId
