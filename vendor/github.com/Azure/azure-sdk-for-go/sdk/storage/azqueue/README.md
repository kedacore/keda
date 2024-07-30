# Azure Queue Storage SDK for Go

> Service Version: 2018-03-28

Azure Queue storage is a service for storing large numbers of messages that can be accessed from anywhere in 
the world via authenticated calls using HTTP or HTTPS. 
A single queue message can be up to 64 KiB in size, and a queue can contain millions of messages, 
up to the total capacity limit of a storage account.

[Source code][source] | [API reference documentation][docs] | [REST API documentation][rest_docs]

## Getting started

### Install the package

Install the Azure Queue Storage SDK for Go with [go get][goget]:

```Powershell
go get github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue
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

In order to interact with the Azure Queue Storage service, you'll need to create an instance of the `azqueue.ServiceClient` type.  The [azidentity][azidentity] module makes it easy to add Azure Active Directory support for authenticating Azure SDK clients with their corresponding Azure services.

```go
// create a credential for authenticating with Azure Active Directory
cred, err := azidentity.NewDefaultAzureCredential(nil)
// TODO: handle err

// create an azqueue.ServiceClient for the specified storage account that uses the above credential
client, err := azqueue.NewServiceClient("https://MYSTORAGEACCOUNT.queue.core.windows.net/", cred, nil)
// TODO: handle err
```

Learn more about enabling Azure Active Directory for authentication with Azure Storage in [our documentation][storage_ad] and [our samples](#next-steps).

## Key concepts
The following components make up the Azure Queue Service:
* The storage account itself
* A queue within the storage account, which contains a set of messages
* A message within a queue, in any format, of up to 64 KiB

The Azure Storage Queues client library for GO allows you to interact with each of these components through the
use of a dedicated client object.

### Clients
Two different clients are provided to interact with the various components of the Queue Service:
1. ServiceClient -
   this client represents interaction with the Azure storage account itself, and allows you to acquire preconfigured
   client instances to access the queues within. It provides operations to retrieve and configure the account
   properties as well as list, create, and delete queues within the account. To perform operations on a specific queue,
   retrieve a client using the `NewQueueClient` method.
2. QueueClient -
   this client represents interaction with a specific queue (which need not exist yet). It provides operations to
   create, delete, or configure a queue and includes operations to enqueue, dequeue, peek, delete, and update messages
   within it.

### Messages
* **Enqueue** - Adds a message to the queue and optionally sets a visibility timeout for the message.
* **Dequeue** - Retrieves a message from the queue and makes it invisible to other consumers.
* **Peek** - Retrieves a message from the front of the queue, without changing the message visibility.
* **Update** - Updates the visibility timeout of a message and/or the message contents.
* **Delete** - Deletes a specified message from the queue.
* **Clear** - Clears all messages from the queue.

### Goroutine safety
We guarantee that all client instance methods are goroutine-safe and independent of each other ([guideline](https://azure.github.io/azure-sdk/golang_introduction.html#thread-safety)). This ensures that the recommendation of reusing client instances is always safe, even across goroutines.

### About Queue metadata
Queue metadata name/value pairs are valid HTTP headers and should adhere to all restrictions governing HTTP headers. Metadata names must be valid HTTP header names, may contain only ASCII characters, and should be treated as case-insensitive. Base64-encode or URL-encode metadata values containing non-ASCII characters.

### Additional concepts
<!-- CLIENT COMMON BAR -->
[Client options](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azcore/policy#ClientOptions) |
[Accessing the response](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime#WithCaptureResponse) |
[Handling failures](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azcore#ResponseError) |
[Logging](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azcore/log)
<!-- CLIENT COMMON BAR -->

## Examples

### Queue Manipulation

```go
const (
	accountName   = "MYSTORAGEACCOUNT"
	accountKey    = "ACCOUNT_KEY"
	queueName     = "samplequeue"
)
```

### Exploring Queue Service APIs

```go
// shared key credential set up
cred := azqueue.NewSharedKeyCredential(accountName, accountKey)

// instantiate service client
serviceClient, err := azqueue.NewServiceClientWithSharedKeyCredential(account, cred, nil)
// TODO: handle error

// 1. create queue
queueClient := serviceClient.NewQueueClient(queueName)
_, err = queueClient.Create(context.TODO(), nil)
// TODO: handle error

// 2. enqueue message
_, err = queueClient.EnqueueMessage(context.TODO(), message, nil)
// TODO: handle error

// 3. dequeue message
_, err = queueClient.DequeueMessage(context.TODO(), nil)
// TODO: handle error

// 4. delete queue
_, err =queueClient.Delete(context.TODO(), nil)
// TODO: handle error
```

### Enumerating queues

```go
const (
	account = "https://MYSTORAGEACCOUNT.queue.core.windows.net/"
)

// authenticate with Azure Active Directory
cred, err := azidentity.NewDefaultAzureCredential(nil)
// TODO: handle error

// create a client for the specified storage account
client, err := azqueue.NewServiceClient(account, cred, nil)
// TODO: handle error

// queue listings are returned across multiple pages
pager := client.NewListQueuesPager(nil)

// continue fetching pages until no more remain
for pager.More() {
   resp, err := pager.NextPage(context.Background())
   _require.Nil(err)
   // print queue name
   for _, queue := range resp.Queues {
		fmt.Println(*queue.Name)
	}
}
```

## Troubleshooting

All queue service operations will return an
[*azcore.ResponseError][azcore_response_error] on failure with a
populated `ErrorCode` field. Many of these errors are recoverable.
The [queueerror][queue_error] package provides the possible Storage error codes
along with various helper facilities for error handling.

```go
const (
	connectionString = "<connection_string>"
	queueName        = "samplequeue"
)

// create a client with the provided connection string
client, err := azqueue.NewServiceClientFromConnectionString(connectionString, nil)
// TODO: handle error

// try to delete the queue, avoiding any potential race conditions with an in-progress or completed deletion
_, err = client.DeleteQueue(context.TODO(), queueName, nil)

if queueerror.HasCode(err, queueerror.QueueBeingDeleted, queueerror.QueueNotFound) {
	// ignore any errors if the queue is being deleted or already has been deleted
} else if err != nil {
	// TODO: some other error
}
```

## Next steps

Get started with our [Queue samples][samples].  They contain complete examples of the above snippets and more.

## Contributing

See the [Storage CONTRIBUTING.md][storage_contrib] for details on building,
testing, and contributing to this library.

This project welcomes contributions and suggestions. Most contributions require you to agree to a [Contributor License Agreement (CLA)][cla] declaring that you have the right to, and actually do, grant us the rights to use your contribution.
 
If you'd like to contribute to this library, please read the [contributing guide] [contributing_guide] to learn more about how to build and test the code.
 
When you submit a pull request, a CLA-bot will automatically determine whether you need to provide a CLA and decorate the PR appropriately (e.g., label, comment). Simply follow the instructions provided by the bot. You will only need to do this once across all repos using our CLA.
 
This project has adopted the [Microsoft Open Source Code of Conduct][coc]. For more information, see the [Code of Conduct FAQ][coc_faq] or contact [opencode@microsoft.com][coc_contact] with any additional questions or comments.


<!-- LINKS -->
[source]: https://github.com/Azure/azure-sdk-for-go/tree/main/sdk/storage/azqueue
[docs]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue
[rest_docs]: https://docs.microsoft.com/rest/api/storageservices/queue-service-rest-api
[godevdl]: https://go.dev/dl/
[goget]: https://pkg.go.dev/cmd/go#hdr-Add_dependencies_to_current_module_and_install_them
[storage_account_docs]: https://docs.microsoft.com/azure/storage/common/storage-account-overview
[storage_account_create_ps]: https://docs.microsoft.com/azure/storage/common/storage-quickstart-create-account?tabs=azure-powershell
[storage_account_create_cli]: https://docs.microsoft.com/azure/storage/common/storage-quickstart-create-account?tabs=azure-cli
[storage_account_create_portal]: https://docs.microsoft.com/azure/storage/common/storage-quickstart-create-account?tabs=azure-portal
[azure_cli]: https://docs.microsoft.com/cli/azure
[azure_sub]: https://azure.microsoft.com/free/
[azidentity]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity
[storage_ad]: https://docs.microsoft.com/azure/storage/common/storage-auth-aad
[azcore_response_error]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azcore#ResponseError
[samples]: https://github.com/Azure/azure-sdk-for-go/tree/main/sdk/storage/azqueue/samples_test.go
[queue_error]: https://github.com/Azure/azure-sdk-for-go/tree/main/sdk/storage/azqueue/queueerror/error_codes.go
[queue]: https://github.com/Azure/azure-sdk-for-go/tree/main/sdk/storage/azqueue/queue_client.go
[sas]: https://github.com/Azure/azure-sdk-for-go/tree/main/sdk/storage/azqueue/sas
[service]: https://github.com/Azure/azure-sdk-for-go/tree/main/sdk/storage/azqueue/service_client.go
[storage_contrib]: https://github.com/Azure/azure-sdk-for-go/blob/main/CONTRIBUTING.md
[contributing_guide]: https://github.com/Azure/azure-sdk-for-go/blob/main/CONTRIBUTING.md
[cla]: https://cla.microsoft.com
[coc]: https://opensource.microsoft.com/codeofconduct/
[coc_faq]: https://opensource.microsoft.com/codeofconduct/faq/
[coc_contact]: mailto:opencode@microsoft.com