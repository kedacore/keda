## Go

``` yaml
title: EventGridClient
description: Azure Event Grid client
generated-metadata: false
clear-output-folder: false
go: true
input-file: 
    # This was the commit that everyone used to generate their first official betas.
    - https://raw.githubusercontent.com/Azure/azure-rest-api-specs/c07d9898ed901330e5ac4996b1bc641adac2e6fd/specification/eventgrid/data-plane/Microsoft.EventGrid/preview/2023-06-01-preview/EventGrid.json
    # - https://raw.githubusercontent.com/Azure/azure-rest-api-specs/947c9ce9b20900c6cbc8e95bc083e723d09a9c2c/specification/eventgrid/data-plane/Microsoft.EventGrid/preview/2023-06-01-preview/EventGrid.json
license-header: MICROSOFT_MIT_NO_VERSION
module: github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventgrid
openapi-type: "data-plane"
output-folder: ../azeventgrid
override-client-name: Client
security: "AADToken"
use: "@autorest/go@4.0.0-preview.52"
version: "^3.0.0"
slice-elements-byval: true
remove-non-reference-schema: true
directive:
  # we have to write a little wrapper code for this so we'll hide the public function
  # for now.
  - from: client.go
    where: $
    transform: return $.replace(/PublishCloudEvents\(/g, "internalPublishCloudEvents(");
  - from: swagger-document
    where: $.definitions.CloudEvent.properties.specversion
    transform: $["x-ms-client-name"] = "SpecVersion"
  - from: swagger-document
    where: $.definitions.CloudEvent.properties.datacontenttype
    transform: $["x-ms-client-name"] = "DataContentType"
  - from: swagger-document
    where: $.definitions.CloudEvent.properties.dataschema
    transform: $["x-ms-client-name"] = "DataSchema"
  # mark models as external so they're just omitted
  - from: swagger-document
    where: $.definitions.CloudEvent
    transform: $["x-ms-external"] = true
  - from: swagger-document
    where: $.definitions.["Azure.Core.Foundations.Error"]
    transform: $["x-ms-external"] = true
  - from: swagger-document
    where: $.definitions.["Azure.Core.Foundations.ErrorResponse"]
    transform: $["x-ms-external"] = true
  - from: swagger-document
    where: $.definitions.["Azure.Core.Foundations.InnerError"]
    transform: $["x-ms-external"] = true
  # make the endpoint a parameter of the client constructor
  - from: swagger-document
    where: $["x-ms-parameterized-host"]
    transform: $.parameters[0]["x-ms-parameter-location"] = "client"
  # delete client name prefix from method options and response types
  - from:
      - client.go
      - models.go
      - response_types.go
      - options.go
    where: $
    transform: return $.replace(/Client(\w+)((?:Options|Response))/g, "$1$2");
  # replace references to the "generated" CloudEvent to the actual version in azcore/messaging
  - from:
      - client.go
      - models.go
      - response_types.go
      - options.go
    where: $
    transform: |
      return $.replace(/\[\]CloudEvent/g, "[]messaging.CloudEvent")
              .replace(/\*CloudEvent/g, "messaging.CloudEvent");

  # remove the 'Interface any' that's generated for an empty response object.
  - from:
      - swagger-document
    where: $["x-ms-paths"]["/topics/{topicName}:publish?api-version={apiVersion}"].post.responses["200"]
    transform: delete $["schema"];
```
