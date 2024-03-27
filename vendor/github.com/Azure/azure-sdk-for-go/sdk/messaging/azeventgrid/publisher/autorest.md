## Go

``` yaml
title: EventGridPublisherClient
description: Azure Event Grid client
generated-metadata: false
clear-output-folder: false
go: true
input-file: 
    - https://raw.githubusercontent.com/Azure/azure-rest-api-specs/main/specification/eventgrid/data-plane/Microsoft.EventGrid/stable/2018-01-01/EventGrid.json
license-header: MICROSOFT_MIT_NO_VERSION
openapi-type: "data-plane"
output-folder: ../publisher
override-client-name: Client
security: "AADToken"
use: "@autorest/go@4.0.0-preview.52"
version: "^3.0.0"
slice-elements-byval: true
remove-non-reference-schema: true
directive:
  # make the endpoint a parameter of the client constructor
  - from: swagger-document
    where: $["x-ms-parameterized-host"]
    transform: $.parameters[0]["x-ms-parameter-location"] = "client"
  # reference azcore/messaging/CloudEvent
  - from: client.go
    where: $
    transform: return $.replace(/\[\]CloudEvent/g, "[]messaging.CloudEvent");
  - from: client.go
    where: $
    transform: return $.replace(/func \(client \*Client\) PublishCloudEventEvents\(/g, "func (client *Client) internalPublishCloudEventEvents(");  
  - from: swagger-document
    where: $.definitions.CloudEventEvent
    transform: $["x-ms-external"] = true
  # delete client name prefix from method options and response types
  - from:
      - client.go
      - models.go
      - response_types.go
      - options.go
    where: $
    transform: return $.replace(/Client(\w+)((?:Options|Response))/g, "$1$2");
  # delete some models that look like they're system events...
  - from: models.go
    where: $
    transform: return $.replace(/\/\/ (SubscriptionDeletedEventData|SubscriptionValidationEventData|SubscriptionValidationResponse).+?\n}/gs, "")    
  - from: models_serde.go
    where: $    
    transform: |
      return $
        .replace(/\/\/ MarshalJSON implements the json.Marshaller interface for type (SubscriptionDeletedEventData|SubscriptionValidationEventData|SubscriptionValidationResponse).+?\n}/gs, "")
        .replace(/\/\/ UnmarshalJSON implements the json.Unmarshaller interface for type (SubscriptionDeletedEventData|SubscriptionValidationEventData|SubscriptionValidationResponse).+?\n}/gs, "");
  - from: 
      - models.go
      - client.go
      - response_types.go
      - options.go
    where: $
    transform: return $.replace(/CloudEventEvent/g, "CloudEvent");
  - from: 
      - models.go
      - models_serde.go
      - client.go
      - response_types.go
      - options.go
    where: $
    transform: return $.replace(/EventGridEvent/g, "Event");
  - from: 
      - client.go
    where: $
    transform: | 
      return $.replace(
        /(func \(client \*Client\) publishCloudEventsCreateRequest.+?)return req, nil/s, 
        '$1\nreq.Raw().Header.Set("Content-type", "application/cloudevents-batch+json; charset=utf-8")\nreturn req, nil');

```
