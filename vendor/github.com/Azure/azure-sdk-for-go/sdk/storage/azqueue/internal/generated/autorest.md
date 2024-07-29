# Code Generation - Azure Queue SDK for Golang

### Settings

```yaml
go: true
clear-output-folder: false
version: "^3.0.0"
license-header: MICROSOFT_MIT_NO_VERSION
input-file: "https://raw.githubusercontent.com/Azure/azure-rest-api-specs/main/specification/storage/data-plane/Microsoft.QueueStorage/preview/2018-03-28/queue.json"
credential-scope: "https://storage.azure.com/.default"
output-folder: ../generated
file-prefix: "zz_"
openapi-type: "data-plane"
verbose: true
security: AzureKey
modelerfour:
  group-parameters: false
  seal-single-value-enum-by-default: true
  lenient-model-deduplication: true
export-clients: true
use: "@autorest/go@4.0.0-preview.45"
```

### Remove QueueName from parameter list since it is not needed

``` yaml
directive:
- from: swagger-document
  where: $["x-ms-paths"]
  transform: >
    for (const property in $)
    {
        if (property.includes('/{queueName}/messages/{messageid}'))
        {
            $[property]["parameters"] = $[property]["parameters"].filter(function(param) { return (typeof param['$ref'] === "undefined") || (false == param['$ref'].endsWith("#/parameters/QueueName") && false == param['$ref'].endsWith("#/parameters/MessageId"))});
        }
        else if (property.includes('/{queueName}'))
        {
            $[property]["parameters"] = $[property]["parameters"].filter(function(param) { return (typeof param['$ref'] === "undefined") || (false == param['$ref'].endsWith("#/parameters/QueueName"))});
        }
    }
```

### Fix GeoReplication

``` yaml
directive:
- from: swagger-document
  where: $.definitions
  transform: >
    delete $.GeoReplication.properties.Status["x-ms-enum"];
    $.GeoReplication.properties.Status["x-ms-enum"] = {
        "name": "QueueGeoReplicationStatus",
        "modelAsString": false
    };
```

### Remove pager method (since we implement it ourselves on the client layer) and export various generated methods in service client to utilize them in higher layers

``` yaml
directive:
  - from: zz_service_client.go
    where: $
    transform: >-
      return $.
        replace(/func \(client \*ServiceClient\) NewListQueuesSegmentPager\(.+\/\/ listQueuesSegmentCreateRequest creates the ListQueuesSegment request/s, `// ListQueuesSegmentCreateRequest creates the ListQueuesFlatSegment ListQueuesSegment`).
        replace(/\(client \*ServiceClient\) listQueuesSegmentCreateRequest\(/, `(client *ServiceClient) ListQueuesSegmentCreateRequest(`).
        replace(/\(client \*ServiceClient\) listQueuesSegmentHandleResponse\(/, `(client *ServiceClient) ListQueuesSegmentHandleResponse(`);
```

### Change `VisibilityTimeout` parameter in queues to be options

``` yaml
directive:
- from: swagger-document
  where: $.parameters.VisibilityTimeoutRequired
  transform: >
    $.required = false;
```

### Change CORS acronym to be all caps

``` yaml
directive:
  - from: source-file-go
    where: $
    transform: >-
      return $.
        replace(/Cors/g, "CORS");
```

### Change cors xml to be correct

``` yaml
directive:
  - from: source-file-go
    where: $
    transform: >-
      return $.
        replace(/xml:"CORS>CORSRule"/g, "xml:\"Cors>CorsRule\"");
```

### Remove `Item` suffix

``` yaml
directive:
- rename-model:
    from: DequeuedMessageItem
    to: DequeuedMessage
- rename-model:
    from: QueueItem
    to: Queue
- rename-model:
    from: PeekedMessageItem
    to: PeekedMessage
```

### Remove `List` suffix

``` yaml
directive:
  - from: source-file-go
    where: $
    transform: >-
      return $.
        replace(/QueueMessagesList/g, "Messages");
```

### Remove `Item` suffix

``` yaml
directive:
  - from: source-file-go
    where: $
    transform: >-
      return $.
        replace(/QueueItems/g, "Queues");
```

### Remove `Queue` prefix

``` yaml
directive:
  - from: source-file-go
    where: $
    transform: >-
      return $.
        replace(/QueueGeoReplicationStatus/g, "GeoReplicationStatus");
```
