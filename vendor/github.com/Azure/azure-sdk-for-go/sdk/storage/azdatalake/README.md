# ADLS Gen2 Storage SDK for Go
[![PkgGoDev](https://pkg.go.dev/badge/github.com/Azure/azure-sdk-for-go/sdk/azdatalake)](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake)
[![Build Status](https://dev.azure.com/azure-sdk/public/_apis/build/status/go/go%20-%20azdatalake%20-%20ci?branchName=main)](https://dev.azure.com/azure-sdk/public/_build/latest?definitionId=6338&branchName=main)
[![Code Coverage](https://img.shields.io/azure-devops/coverage/azure-sdk/public/6338/main)](https://img.shields.io/azure-devops/coverage/azure-sdk/public/6338/main)

> Service Version: 2021-06-08

Azure Data Lake Storage Gen2 (ADLS Gen2) is Microsoft's hierarchical object storage solution for the cloud with converged capabilities with Azure Blob Storage. 
For example, Data Lake Storage Gen2 provides file system semantics, file-level security, and scale. 
Because these capabilities are built on Blob storage, you also get low-cost, tiered storage, with high availability/disaster recovery capabilities.
ADLS Gen2 makes Azure Storage the foundation for building enterprise data lakes on Azure. 
Designed from the start to service multiple petabytes of information while sustaining hundreds of gigabits of throughput, ADLS Gen2 allows you to easily manage massive amounts of data.

[Source code][source] | [API reference documentation][docs] | [REST API documentation][rest_docs]

## Getting started

### Install the package

Install the ADLS Gen2 Storage SDK for Go with [go get][goget]:

```Powershell
go get github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake
```

If you're going to authenticate with Azure Active Directory (recommended), install the [azidentity][azidentity] module.
```Powershell
go get github.com/Azure/azure-sdk-for-go/sdk/azidentity
```

### Prerequisites

A supported [Go][godevdl] version (the Azure SDK supports the two most recent Go releases).

You need an [Azure subscription][azure_sub] and a
[Storage Account][storage_account_docs] to use this package.

To create a new Storage Account, you can use the [Azure Portal][storage_account_create_portal],
[Azure PowerShell][storage_account_create_ps], or the [Azure CLI][storage_account_create_cli].
Here's an example using the Azure CLI:

```Powershell
az storage account create --name MyStorageAccount --resource-group MyResourceGroup --location westus --sku Standard_LRS
```

### Authenticate the client

In order to interact with the ADLS Gen2 Storage service, you'll need to create an instance of the `Client` type.  The [azidentity][azidentity] module makes it easy to add Azure Active Directory support for authenticating Azure SDK clients with their corresponding Azure services.

```go
// create a credential for authenticating with Azure Active Directory
cred, err := azidentity.NewDefaultAzureCredential(nil)
// TODO: handle err

// create a service.Client for the specified storage account that uses the above credential
client, err := service.NewClient("https://MYSTORAGEACCOUNT.dfs.core.windows.net/", cred, nil)
// TODO: handle err
// you can also create filesystem, file and directory clients
```

Learn more about enabling Azure Active Directory for authentication with Azure Storage in [our documentation][storage_ad] and [our samples](#next-steps).

## Key concepts

ADLS Gen2 provides:
- Hadoop-compatible access
- Hierarchical directory structure
- Optimized cost and performance
- Finer grain security model
- Massive scalability

ADLS Gen2 storage is designed for:

- Serving images or documents directly to a browser.
- Storing files for distributed access.
- Streaming video and audio.
- Writing to log files.
- Storing data for backup and restore, disaster recovery, and archiving.
- Storing data for analysis by an on-premises or Azure-hosted service.

ADLS Gen2 storage offers three types of resources:

- The _storage account_
- One or more _filesystems_ in a storage account
- One or more _files_ or _directories_ in a filesystem

Instances of the `Client` type provide methods for manipulating filesystems and paths within a storage account.
The storage account is specified when the `Client` is constructed. The clients available are referenced below.
Use the appropriate client constructor function for the authentication mechanism you wish to use.

### Goroutine safety
We guarantee that all client instance methods are goroutine-safe and independent of each other ([guideline](https://azure.github.io/azure-sdk/golang_introduction.html#thread-safety)). This ensures that the recommendation of reusing client instances is always safe, even across goroutines.

### About metadata
ADLS Gen2 metadata name/value pairs are valid HTTP headers and should adhere to all restrictions governing HTTP headers. Metadata names must be valid HTTP header names, may contain only ASCII characters, and should be treated as case-insensitive. Base64-encode or URL-encode metadata values containing non-ASCII characters.

### Additional concepts
<!-- CLIENT COMMON BAR -->
[Client options](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azcore/policy#ClientOptions) |
[Accessing the response](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime#WithCaptureResponse) |
[Handling failures](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azcore#ResponseError) |
[Logging](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azcore/log)
<!-- CLIENT COMMON BAR -->

## Examples

### Creating and uploading a file (assuming filesystem exists)

```go
const (
	path = "https://MYSTORAGEACCOUNT.dfs.core.windows.net/sample-fs/sample-file"
)

// authenticate with Azure Active Directory
cred, err := azidentity.NewDefaultAzureCredential(nil)
// TODO: handle error

// create a client for the specified storage account
client, err := file.NewClient(path, cred, nil)
// TODO: handle error

_, err = client.Create(context.TODO(), nil)
// TODO: handle error

// open the file for reading
fh, err := os.OpenFile(sampleFile, os.O_RDONLY, 0)
// TODO: handle error
defer fh.Close()

// upload the file to the specified filesystem with the specified file name
_, err = client.UploadFile(context.TODO(), fh, nil)
// TODO: handle error
```

### Downloading a file

```go
const (
    path = "https://MYSTORAGEACCOUNT.dfs.core.windows.net/sample-fs/cloud.jpg"
)

// authenticate with Azure Active Directory
cred, err := azidentity.NewDefaultAzureCredential(nil)
// TODO: handle error

// create a client for the specified storage account
client, err := file.NewClient(path, cred, nil)
// TODO: handle error

// create or open a local file where we can download the file
file, err := os.Create("cloud.jpg")
// TODO: handle error
defer file.Close()

// download the file
_, err = client.DownloadFile(context.TODO(), file, nil)
// TODO: handle error
```

### Creating and deleting a filesystem

```go
const (
	fs = "https://MYSTORAGEACCOUNT.dfs.core.windows.net/sample-fs"
)

// authenticate with Azure Active Directory
cred, err := azidentity.NewDefaultAzureCredential(nil)
// TODO: handle error

// create a client for the specified storage account
client, err := filesystem.NewClient(fs, cred, nil)
// TODO: handle error

_, err = client.Create(context.TODO(), nil)
// TODO: handle error

_, err = client.Delete(context.TODO(), nil)
// TODO: handle error
```

### Enumerating paths (assuming filesystem exists)

```go
const (
	fs = "https://MYSTORAGEACCOUNT.dfs.core.windows.net/sample-fs"
)

// authenticate with Azure Active Directory
cred, err := azidentity.NewDefaultAzureCredential(nil)
// TODO: handle error

// create a filesystem client for the specified storage account
client, err := filesystem.NewClient(fs, cred, nil)
// TODO: handle error

// path listings are returned across multiple pages
pager := client.NewListPathsPager(true, nil)

// continue fetching pages until no more remain
for pager.More() {
  // advance to the next page
	page, err := pager.NextPage(context.TODO())
	// TODO: handle error

	// print the path names for this page
	for _, path := range page.PathList.Paths {
		fmt.Println(*path.Name)
        fmt.Println(*path.IsDirectory)
	}
}
```

## Troubleshooting

All Datalake service operations will return an
[*azcore.ResponseError][azcore_response_error] on failure with a
populated `ErrorCode` field. Many of these errors are recoverable.
The [datalakeerror][datalake_error] package provides the possible Storage error codes
along with various helper facilities for error handling.


### Specialized clients

The ADLS Gen2 Storage SDK for Go provides specialized clients in various subpackages.

The [file][file] package contains APIs related to file path types.

The [directory][directory] package contains APIs related to directory path types.

The [lease][lease] package contains clients for managing leases on paths (paths represent both directory and file paths) and filesystems.  Please see the [reference docs](https://learn.microsoft.com/rest/api/storageservices/lease-blob#remarks) for general information on leases.

The [filesystem][filesystem] package contains APIs specific to filesystems.  This includes APIs setting access policies or properties, and more.

The [service][service] package contains APIs specific to Datalake service.  This includes APIs for manipulating filesystems, retrieving account information, and more.

The [sas][sas] package contains utilities to aid in the creation and manipulation of Shared Access Signature tokens.
See the package's documentation for more information.


You can find additional context and examples in our samples for each subpackage (named examples_test.go).

## Contributing

See the [Storage CONTRIBUTING.md][storage_contrib] for details on building,
testing, and contributing to this library.

This project welcomes contributions and suggestions.  Most contributions require
you to agree to a Contributor License Agreement (CLA) declaring that you have
the right to, and actually do, grant us the rights to use your contribution. For
details, visit [cla.microsoft.com][cla].

This project has adopted the [Microsoft Open Source Code of Conduct][coc].
For more information see the [Code of Conduct FAQ][coc_faq]
or contact [opencode@microsoft.com][coc_contact] with any
additional questions or comments.

<!-- LINKS -->
[source]: https://github.com/Azure/azure-sdk-for-go/tree/main/sdk/storage/azdatalake
[docs]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake
[rest_docs]: https://learn.microsoft.com/rest/api/storageservices/data-lake-storage-gen2
[godevdl]: https://go.dev/dl/
[goget]: https://pkg.go.dev/cmd/go#hdr-Add_dependencies_to_current_module_and_install_them
[storage_account_docs]: https://learn.microsoft.com/azure/storage/common/storage-account-overview
[storage_account_create_ps]: https://learn.microsoft.com/azure/storage/common/storage-quickstart-create-account?tabs=azure-powershell
[storage_account_create_cli]: https://learn.microsoft.com/azure/storage/common/storage-quickstart-create-account?tabs=azure-cli
[storage_account_create_portal]: https://learn.microsoft.com/azure/storage/common/storage-quickstart-create-account?tabs=azure-portal
[azure_sub]: https://azure.microsoft.com/free/
[azidentity]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity
[storage_ad]: https://learn.microsoft.com/azure/storage/common/storage-auth-aad
[azcore_response_error]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azcore#ResponseError
[datalake_error]: https://github.com/Azure/azure-sdk-for-go/tree/main/sdk/storage/azdatalake/datalakeerror/error_codes.go
[filesystem]: https://github.com/Azure/azure-sdk-for-go/tree/main/sdk/storage/azdatalake/filesystem/client.go
[lease]: https://github.com/Azure/azure-sdk-for-go/tree/main/sdk/storage/azdatalake/lease
[file]: https://github.com/Azure/azure-sdk-for-go/tree/main/sdk/storage/azdatalake/file/client.go
[directory]: https://github.com/Azure/azure-sdk-for-go/tree/main/sdk/storage/azdatalake/directory/client.go
[sas]: https://github.com/Azure/azure-sdk-for-go/tree/main/sdk/storage/azdatalake/sas
[service]: https://github.com/Azure/azure-sdk-for-go/tree/main/sdk/storage/azdatalake/service/client.go
[storage_contrib]: https://github.com/Azure/azure-sdk-for-go/blob/main/CONTRIBUTING.md
[cla]: https://cla.microsoft.com
[coc]: https://opensource.microsoft.com/codeofconduct/
[coc_faq]: https://opensource.microsoft.com/codeofconduct/faq/
[coc_contact]: mailto:opencode@microsoft.com
