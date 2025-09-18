# Code Generation - Azure Datalake SDK for Golang

### Settings

```yaml
go: true
clear-output-folder: false
version: "^3.0.0"
license-header: MICROSOFT_MIT_NO_VERSION
input-file: "https://raw.githubusercontent.com/Azure/azure-rest-api-specs/d18a495685ccec837b72891b4deea017f62e8190/specification/storage/data-plane/Azure.Storage.Files.DataLake/stable/2025-05-05/DataLakeStorage.json"
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
use: "@autorest/go@4.0.0-preview.65"
```

### Add ListBlobsShowOnly value 'directories'
```yaml
directive:
- from: swagger-document
  where: $.parameters.ListBlobsShowOnly
  transform: >
    if (!$.enum.includes("directories")) {
        $.enum.push("directories");
    }

```

### Remove FileSystem and PathName from parameter list since they are not needed
``` yaml
directive:
- from: swagger-document
  where: $["x-ms-paths"]
  transform: >
    for (const property in $)
    {
        if (property.includes('/{filesystem}/{path}'))
        {
            $[property]["parameters"] = $[property]["parameters"].filter(function(param) { return (typeof param['$ref'] === "undefined") || (false == param['$ref'].endsWith("#/parameters/FileSystem") && false == param['$ref'].endsWith("#/parameters/Path"))});
        }
        else if (property.includes('/{filesystem}'))
        {
            $[property]["parameters"] = $[property]["parameters"].filter(function(param) { return (typeof param['$ref'] === "undefined") || (false == param['$ref'].endsWith("#/parameters/FileSystem"))});
        }
    }
```

### Turn Path eTag into etag

``` yaml
directive:
- from: swagger-document
  where: $.definitions.Path
  transform: >
    $.properties.etag = $.properties.eTag;
    delete $.properties.eTag;
    $.properties.etag["x-ms-client-name"] = "eTag";

```

### Remove pager methods and export various generated methods in filesystem client

``` yaml
directive:
  - from: zz_filesystem_client.go
    where: $
    transform: >-
      return $.
        replace(/func \(client \*FileSystemClient\) NewListBlobHierarchySegmentPager\(.+\/\/ listBlobHierarchySegmentCreateRequest creates the ListBlobHierarchySegment request/s, `//\n// ListBlobHierarchySegmentCreateRequest creates the ListBlobHierarchySegment request`).
        replace(/\(client \*FileSystemClient\) listBlobHierarchySegmentCreateRequest\(/, `(client *FileSystemClient) ListBlobHierarchySegmentCreateRequest(`).
        replace(/\(client \*FileSystemClient\) listBlobHierarchySegmentHandleResponse\(/, `(client *FileSystemClient) ListBlobHierarchySegmentHandleResponse(`);
```

### Remove pager methods and export various generated methods in filesystem client

``` yaml
directive:
  - from: zz_filesystem_client.go
    where: $
    transform: >-
      return $.
        replace(/func \(client \*FileSystemClient\) NewListPathsPager\(.+\/\/ listPathsCreateRequest creates the ListPaths request/s, `//\n// ListPathsCreateRequest creates the ListPaths request`).
        replace(/\(client \*FileSystemClient\) listPathsCreateRequest\(/, `(client *FileSystemClient) ListPathsCreateRequest(`).
        replace(/\(client \*FileSystemClient\) listPathsHandleResponse\(/, `(client *FileSystemClient) ListPathsHandleResponse(`);
```

### Remove pager methods and export various generated methods in service client

``` yaml
directive:
  - from: zz_service_client.go
    where: $
    transform: >-
      return $.
        replace(/func \(client \*ServiceClient\) NewListFileSystemsPager\(.+\/\/ listFileSystemsCreateRequest creates the ListFileSystems request/s, `//\n// ListFileSystemsCreateRequest creates the ListFileSystems request`).
        replace(/\(client \*ServiceClient\) listFileSystemsCreateRequest\(/, `(client *ServiceClient) ListFileSystemsCreateRequest(`).
        replace(/\(client \*ServiceClient\) listFileSystemsHandleResponse\(/, `(client *ServiceClient) ListFileSystemsHandleResponse(`);
```


### Remove pager methods and export various generated methods in path client

``` yaml
directive:
  - from: zz_path_client.go
    where: $
    transform: >-
      return $.
        replace(/\(client \*PathClient\) setAccessControlRecursiveCreateRequest\(/, `(client *PathClient) SetAccessControlRecursiveCreateRequest(`).
        replace(/\(client \*PathClient\) setAccessControlRecursiveHandleResponse\(/, `(client *PathClient) SetAccessControlRecursiveHandleResponse(`).
        replace(/setAccessControlRecursiveCreateRequest/g, 'SetAccessControlRecursiveCreateRequest').
        replace(/setAccessControlRecursiveHandleResponse/g, 'SetAccessControlRecursiveHandleResponse');
```

### Fix EncryptionAlgorithm

``` yaml
directive:
- from: swagger-document
  where: $.parameters
  transform: >
    delete $.EncryptionAlgorithm.enum;
    $.EncryptionAlgorithm.enum = [
      "None",
      "AES256"
    ];
```

### Add Missing Imports to zz_service_client.go

``` yaml
directive:
- from: zz_service_client.go
  where: $
  transform: >-
      return $.
        replace(/"strconv"/, `"strconv"\n\t"strings"`);
```
### Add Missing Imports to zz_models_serde.go

``` yaml
directive:
- from: zz_models_serde.go
  where: $
  transform: >-
      return $.
        replace(/"reflect"/, `"reflect"\n\t"strconv"`);
```

### Clean up some const type names so they don't stutter

``` yaml
directive:
- from: swagger-document
  where: $.parameters['PathExpiryOptions']
  transform: >
    $["x-ms-enum"].name = "ExpiryOptions";
    $["x-ms-client-name"].name = "ExpiryOptions";

```

### use azcore.ETag

``` yaml
directive:
- from: 
    - zz_options.go
    - zz_models.go       
  where: $
  transform: >-
    return $.
      replace(/import "time"/, `import (\n\t"time"\n\t"github.com/Azure/azure-sdk-for-go/sdk/azcore"\n)`).
      replace(/Etag\s+\*string/g, `ETag *azcore.ETag`).
      replace(/IfMatch\s+\*string/g, `IfMatch *azcore.ETag`).
      replace(/IfNoneMatch\s+\*string/g, `IfNoneMatch *azcore.ETag`).
      replace(/SourceIfMatch\s+\*string/g, `SourceIfMatch *azcore.ETag`).
      replace(/SourceIfNoneMatch\s+\*string/g, `SourceIfNoneMatch *azcore.ETag`);

- from: zz_responses.go
  where: $
  transform: >-
    return $.
      replace(/"time"/, `"time"\n\t"github.com/Azure/azure-sdk-for-go/sdk/azcore"`).
      replace(/ETag\s+\*string/g, `ETag *azcore.ETag`);

- from:
  - zz_filesystem_client.go
  - zz_path_client.go
  where: $
  transform: >-
    return $.
      replace(/"github\.com\/Azure\/azure\-sdk\-for\-go\/sdk\/azcore\/policy"/, `"github.com/Azure/azure-sdk-for-go/sdk/azcore"\n\t"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"`).
      replace(/result\.ETag\s+=\s+&val/g, `result.ETag = (*azcore.ETag)(&val)`).
      replace(/\*modifiedAccessConditions.IfMatch/g, `string(*modifiedAccessConditions.IfMatch)`).
      replace(/\*modifiedAccessConditions.IfNoneMatch/g, `string(*modifiedAccessConditions.IfNoneMatch)`).
      replace(/\*sourceModifiedAccessConditions.SourceIfMatch/g, `string(*sourceModifiedAccessConditions.SourceIfMatch)`).
      replace(/\*sourceModifiedAccessConditions.SourceIfNoneMatch/g, `string(*sourceModifiedAccessConditions.SourceIfNoneMatch)`);

```

### Fix up x-ms-content-crc64 header response name

``` yaml
directive:
- from: swagger-document
  where: $.x-ms-paths.*.*.responses.*.headers.x-ms-content-crc64
  transform: >
    $["x-ms-client-name"] = "ContentCRC64"
```

### Updating encoding URL, Golang adds '+' which disrupts encoding with service

``` yaml
directive:
- from: zz_service_client.go
  where: $
  transform: >-
    return $.
      replace(/req.Raw\(\).URL.RawQuery \= reqQP.Encode\(\)/, `req.Raw().URL.RawQuery = strings.Replace(reqQP.Encode(), "+", "%20", -1)`);
```

### Change `Duration` parameter in leases to be required

``` yaml
directive:
- from: swagger-document
  where: $.parameters.LeaseDuration
  transform: >
    $.required = true;
```

### Change CPK acronym to be all caps

``` yaml
directive:
  - from: source-file-go
    where: $
    transform: >-
      return $.
        replace(/Cpk/g, "CPK");
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

### Convert time to GMT for If-Modified-Since and If-Unmodified-Since request headers

``` yaml
directive:
- from: 
  - zz_filesystem_client.go
  - zz_path.go
  where: $
  transform: >-
    return $.
      replace (/req\.Raw\(\)\.Header\[\"If-Modified-Since\"\]\s+=\s+\[\]string\{modifiedAccessConditions\.IfModifiedSince\.Format\(time\.RFC1123\)\}/g, 
      `req.Raw().Header["If-Modified-Since"] = []string{(*modifiedAccessConditions.IfModifiedSince).In(gmt).Format(time.RFC1123)}`).
      replace (/req\.Raw\(\)\.Header\[\"If-Unmodified-Since\"\]\s+=\s+\[\]string\{modifiedAccessConditions\.IfUnmodifiedSince\.Format\(time\.RFC1123\)\}/g, 
      `req.Raw().Header["If-Unmodified-Since"] = []string{(*modifiedAccessConditions.IfUnmodifiedSince).In(gmt).Format(time.RFC1123)}`).
      replace (/req\.Raw\(\)\.Header\[\"x-ms-source-if-modified-since\"\]\s+=\s+\[\]string\{sourceModifiedAccessConditions\.SourceIfModifiedSince\.Format\(time\.RFC1123\)\}/g, 
      `req.Raw().Header["x-ms-source-if-modified-since"] = []string{(*sourceModifiedAccessConditions.SourceIfModifiedSince).In(gmt).Format(time.RFC1123)}`).
      replace (/req\.Raw\(\)\.Header\[\"x-ms-source-if-unmodified-since\"\]\s+=\s+\[\]string\{sourceModifiedAccessConditions\.SourceIfUnmodifiedSince\.Format\(time\.RFC1123\)\}/g, 
      `req.Raw().Header["x-ms-source-if-unmodified-since"] = []string{(*sourceModifiedAccessConditions.SourceIfUnmodifiedSince).In(gmt).Format(time.RFC1123)}`).
      replace (/req\.Raw\(\)\.Header\[\"x-ms-immutability-policy-until-date\"\]\s+=\s+\[\]string\{options\.ImmutabilityPolicyExpiry\.Format\(time\.RFC1123\)\}/g, 
      `req.Raw().Header["x-ms-immutability-policy-until-date"] = []string{(*options.ImmutabilityPolicyExpiry).In(gmt).Format(time.RFC1123)}`);
      
```

### Change container prefix to filesystem
``` yaml
directive:
  - from: source-file-go
    where: $
    transform: >-
      return $.
        replace(/PublicAccessTypeBlob/g, 'PublicAccessTypeFile').
        replace(/PublicAccessTypeContainer/g, 'PublicAccessTypeFileSystem').
        replace(/FileSystemClientListBlobHierarchySegmentResponse/g, 'FileSystemClientListPathHierarchySegmentResponse').
        replace(/ListBlobsHierarchySegmentResponse/g, 'ListPathsHierarchySegmentResponse').
        replace(/ContainerName\s*\*string/g, 'FileSystemName *string').
        replace(/BlobHierarchyListSegment/g, 'PathHierarchyListSegment').
        replace(/BlobItems/g, 'PathItems').
        replace(/BlobItem/g, 'PathItem').
        replace(/BlobPrefix/g, 'PathPrefix').
        replace(/BlobPrefixes/g, 'PathPrefixes').
        replace(/BlobProperties/g, 'PathProperties').
        replace(/ContainerProperties/g, 'FileSystemProperties');
```

### 
``` yaml
directive:
- from: 
  - zz_models_serde.go
  where: $
  transform: >-
    return $.
        replace(/err = unpopulate\((.*), "ContentLength", &p\.ContentLength\)/g, 'var rawVal string\nerr = unpopulate(val, "ContentLength", &rawVal)\nintVal, _ := strconv.ParseInt(rawVal, 10, 64)\np.ContentLength = &intVal').
        replace(/err = unpopulate\((.*), "IsDirectory", &p\.IsDirectory\)/g, 'var rawVal string\nerr = unpopulate(val, "IsDirectory", &rawVal)\nboolVal, _ := strconv.ParseBool(rawVal)\np.IsDirectory = &boolVal');
```

### Updating service version to 2025-07-05
```yaml
directive:
- from: 
  - zz_service_client.go
  - zz_filesystem_client.go 
  - zz_path_client.go
  where: $
  transform: >-
    return $.
      replaceAll(`[]string{"2025-05-05"}`, `[]string{ServiceVersion}`);
```