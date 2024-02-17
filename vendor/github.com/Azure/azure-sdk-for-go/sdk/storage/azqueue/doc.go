//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

/*

Package azqueue can access an Azure Queue Storage.

The azqueue package is capable of :-
   - Creating, deleting, and clearing queues in an account
   - Enqueuing, dequeuing, and editing messages in a queue
   - Creating Shared Access Signature for authentication

Types of Resources

The azqueue package allows you to interact with three types of resources :-

* Azure storage accounts.
* Queues within those storage accounts.
* Messages within those queues.

The Azure Queue Storage (azqueue) client library for Go allows you to interact with each of these components through the use of a dedicated client object.
To create a client object, you will need the account's queue service endpoint URL and a credential that allows you to access the account.

Types of Credentials

The clients support different forms of authentication.
The azqueue library supports any of the `azcore.TokenCredential` interfaces, authorization via a Connection String,
or authorization with a Shared Access Signature token.

Using a Shared Key

To use an account shared key (aka account key or access key), provide the key as a string.
This can be found in your storage account in the Azure Portal under the "Access Keys" section.

Use the key as the credential parameter to authenticate the client:

	accountName, ok := os.LookupEnv("AZURE_STORAGE_ACCOUNT_NAME")
	if !ok {
		panic("AZURE_STORAGE_ACCOUNT_NAME could not be found")
	}
	accountKey, ok := os.LookupEnv("AZURE_STORAGE_ACCOUNT_KEY")
	if !ok {
		panic("AZURE_STORAGE_ACCOUNT_KEY could not be found")
	}

	serviceURL := fmt.Sprintf("https://%s.queue.core.windows.net/", accountName)

	cred, err := azqueue.NewSharedKeyCredential(accountName, accountKey)
	handle(err)

	serviceClient, err := azqueue.NewServiceClientWithSharedKey(serviceURL, cred, nil)
	handle(err)

	fmt.Println(serviceClient.URL())

Using a Connection String

Depending on your use case and authorization method, you may prefer to initialize a client instance with a connection string instead of providing the account URL and credential separately.
To do this, pass the connection string to the service client's `NewServiceClientFromConnectionString` method.
The connection string can be found in your storage account in the Azure Portal under the "Access Keys" section.

	connStr := "DefaultEndpointsProtocol=https;AccountName=<my_account_name>;AccountKey=<my_account_key>;EndpointSuffix=core.windows.net"
	serviceClient, err := azqueue.NewServiceClientFromConnectionString(connStr, nil)

Using a Shared Access Signature (SAS) Token

To use a shared access signature (SAS) token, provide the token at the end of your service URL.
You can generate a SAS token from the Azure Portal under Shared Access Signature or use the ServiceClient.GetSASURL() functions.

	accountName, ok := os.LookupEnv("AZURE_STORAGE_ACCOUNT_NAME")
	if !ok {
		panic("AZURE_STORAGE_ACCOUNT_NAME could not be found")
	}
	accountKey, ok := os.LookupEnv("AZURE_STORAGE_ACCOUNT_KEY")
	if !ok {
		panic("AZURE_STORAGE_ACCOUNT_KEY could not be found")
	}
	serviceURL := fmt.Sprintf("https://%s.queue.core.windows.net/", accountName)

	cred, err := azqueue.NewSharedKeyCredential(accountName, accountKey)
	handle(err)
	serviceClient, err := azqueue.NewServiceClientWithSharedKey(serviceURL, cred, nil)
	handle(err)
	fmt.Println(serviceClient.URL())

	// Alternatively, you can create SAS on the fly

	resources := azqueue.AccountResourceTypes{Service: true}
	permission := azqueue.AccountSASPermissions{Read: true}
	expiry := start.AddDate(0, 0, 1)
	serviceURLWithSAS, err := serviceClient.GetSASURL(resources, permission, expiry, nil)
	handle(err)

	serviceClientWithSAS, err := azqueue.NewServiceClientWithNoCredential(serviceURLWithSAS, nil)
	handle(err)

	fmt.Println(serviceClientWithSAS.URL())

Types of Clients

There are two different clients provided to interact with the various components of the Queue Service:

1. **`ServiceClient`**
   * Get and set account settings.
   * Query, create, and delete queues within the account.

2. **`QueueClient`**
   * Get and set queue access settings and metadata.
   * Enqueue, Dequeue and Peek messages within a queue.
   * Update and Delete messages.

Examples

	// Your account name and key can be obtained from the Azure Portal.
	accountName, ok := os.LookupEnv("AZURE_STORAGE_ACCOUNT_NAME")
	if !ok {
		panic("AZURE_STORAGE_ACCOUNT_NAME could not be found")
	}

	accountKey, ok := os.LookupEnv("AZURE_STORAGE_ACCOUNT_KEY")
	if !ok {
		panic("AZURE_STORAGE_ACCOUNT_KEY could not be found")
	}
	cred, err := azqueue.NewSharedKeyCredential(accountName, accountKey)
	handle(err)

	// The service URL for queue endpoints is usually in the form: http(s)://<account>.queue.core.windows.net/
	serviceClient, err := azqueue.NewServiceClientWithSharedKey(fmt.Sprintf("https://%s.queue.core.windows.net/", accountName), cred, nil)
	handle(err)

	// ===== 1. Create a queue =====

	// First, create a queue client, and use the Create method to create a new queue in your account
	queueClient, err := serviceClient.NewQueueClient("testqueue")
	handle(err)

	// All APIs have an options' bag struct as a parameter.
	// The options' bag struct allows you to specify optional parameters such as metadata, access, etc.
	// If you want to use the default options, pass in nil.
	_, err = queueClient.Create(context.TODO(), nil)
	handle(err)

	// ===== 2. Enqueue and Dequeue a message =====
	message := "Hello world!"

	// send message to queue
	_, err = queueClient.EnqueueMessage(context.TODO(), message, nil)
	handle(err)

	// dequeue message from queue, you can also use `DequeueMessage()` to dequeue more than one message (up to 32)
	_, err = queueClient.DequeueMessage(context.TODO(), nil)
	handle(err)

	// ===== 3. Peek messages =====
	// You can also peek messages from the queue (without removing them), you can peek a maximum of 32 messages.

	opts := azqueue.PeekMessagesOptions{NumberOfMessages: to.Ptr(int32(4))}
	resp, err := queueClient.PeekMessages(context.TODO(), &opts)

	// Delete the queue.
	_, err = queueClient.Delete(context.TODO(), nil)
	handle(err)
*/

package azqueue
