//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

/*

Package azdatalake can access an Azure Data Lake Service Gen2 (ADLS Gen2).

The azdatalake package is capable of :-
    - Creating, deleting, and querying filesystems in an account
    - Creating, deleting, and querying files and directories in a filesystem
    - Creating Shared Access Signature for authentication

Types of Resources

The azdatalake package allows you to interact with three types of resources :-

* Azure storage accounts.
* filesystems within those storage accounts.
* files and directories within those filesystems.

ADLS Gen2 client library for Go allows you to interact with each of these components through the use of a dedicated client object.
To create a client object, you will need the account's ADLS Gen2 service endpoint URL and a credential that allows you to access the account.

Types of Credentials

The clients support different forms of authentication.
The azdatalake library supports any of the `azcore.TokenCredential` interfaces, authorization via a Connection String,
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

	serviceURL := fmt.Sprintf("https://%s.dfs.core.windows.net/", accountName)

	cred, err := azdatalake.NewSharedKeyCredential(accountName, accountKey)
	handle(err)

	serviceClient, err := service.NewClientWithSharedKey(serviceURL, cred, nil)
	handle(err)

	// get the underlying dfs endpoint
	fmt.Println(serviceClient.DFSURL())

	// get the underlying blob endpoint
	fmt.Println(serviceClient.BlobURL())

Using a Connection String

Depending on your use case and authorization method, you may prefer to initialize a client instance with a connection string instead of providing the account URL and credential separately.
To do this, pass the connection string to the service client's `NewClientFromConnectionString` method.
The connection string can be found in your storage account in the Azure Portal under the "Access Keys" section.

	connStr := "DefaultEndpointsProtocol=https;AccountName=<my_account_name>;AccountKey=<my_account_key>;EndpointSuffix=core.windows.net"
	serviceClient, err := service.NewClientFromConnectionString(connStr, nil)

Using a Shared Access Signature (SAS) Token

To use a shared access signature (SAS) token, provide the token at the end of your service URL.
You can generate a SAS token from the Azure Portal under Shared Access Signature or use the ServiceClient.GetSASToken() functions.

	accountName, ok := os.LookupEnv("AZURE_STORAGE_ACCOUNT_NAME")
	if !ok {
		panic("AZURE_STORAGE_ACCOUNT_NAME could not be found")
	}
	accountKey, ok := os.LookupEnv("AZURE_STORAGE_ACCOUNT_KEY")
	if !ok {
		panic("AZURE_STORAGE_ACCOUNT_KEY could not be found")
	}
	serviceURL := fmt.Sprintf("https://%s.dfs.core.windows.net/", accountName)

	cred, err := azdatalake.NewSharedKeyCredential(accountName, accountKey)
	handle(err)
	serviceClient, err := service.NewClientWithSharedKey(serviceURL, cred, nil)
	handle(err)
	fmt.Println(serviceClient.DFSURL())

	// Alternatively, you can create SAS on the fly

	resources := sas.AccountResourceTypes{Service: true}
	permission := sas.AccountPermissions{Read: true}
	start := time.Now()
	expiry := start.AddDate(0, 0, 1)
	serviceURLWithSAS, err := serviceClient.GetSASURL(resources, permission, start, expiry)
	handle(err)

	serviceClientWithSAS, err := service.NewClientWithNoCredential(serviceURLWithSAS, nil)
	handle(err)

	fmt.Println(serviceClientWithSAS.DFSURL())
	fmt.Println(serviceClientWithSAS.BlobURL())

Types of Clients

There are three different clients provided to interact with the various components of the ADLS Gen2 Service:

1. **`ServiceClient`**
    * Get and set account settings.
    * Query, create, list and delete filesystems within the account.

2. **`FileSystemClient`**
    * Get and set filesystem access settings, properties, and metadata.
    * Create, delete, and query files/directories within the filesystem.
    * `FileSystemLeaseClient` to support filesystem lease management.

3. **`FileClient`**
    * Get and set file properties.
    * Perform CRUD operations on a given file.
	* Set ACLs on a given file.
    * `PathLeaseClient` to support file lease management.

4. **`DirectoryClient`**
    * Get and set directory properties.
    * Perform CRUD operations on a given directory.
	* Set ACLs on a given directory and recursively on all subdirectories and files.
    * `PathLeaseClient` to support directory lease management.

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
	cred, err := azdatalake.NewSharedKeyCredential(accountName, accountKey)
	handle(err)

	// The service URL for dfs endpoints is usually in the form: http(s)://<account>.dfs.core.windows.net/
	serviceClient, err := service.NewClientWithSharedKey(fmt.Sprintf("https://%s.dfs.core.windows.net/", accountName), cred, nil)
	handle(err)

	// ===== 1. Create a filesystem =====

	// First, create a filesystem client, and use the Create method to create a new filesystem in your account
	fsClient, err := serviceClient.NewFileSystemClient("testfs")
	handle(err)

	// All APIs have an options' bag struct as a parameter.
	// The options' bag struct allows you to specify optional parameters such as metadata, public access types, etc.
	// If you want to use the default options, pass in nil.
	_, err = fsClient.Create(context.TODO(), nil)
	handle(err)

	// ===== 2. Upload and Download a file =====
	uploadData := "Hello world!"

	// Create a new file from the fsClient
	fileClient, err := fsClient.NewFileClient("HelloWorld.txt")
	handle(err)


	_, err = fileClient.UploadStream(context.TODO(), streaming.NopCloser(strings.NewReader(uploadData)), nil)
	handle(err)

	// Download the file's contents and ensure that the download worked properly
	fileDownloadResponse, err := fileClient.DownloadStream(context.TODO(), nil)
	handle(err)

	// Use the bytes.Buffer object to read the downloaded data.
	// RetryReaderOptions has a lot of in-depth tuning abilities, but for the sake of simplicity, we'll omit those here.
	reader := fileDownloadResponse.Body(nil)
	downloadData, err := io.ReadAll(reader)
	handle(err)
	if string(downloadData) != uploadData {
		handle(errors.New("Uploaded data should be same as downloaded data"))
	}

	if err = reader.Close(); err != nil {
		handle(err)
		return
	}

	// ===== 3. List paths =====
	// List methods returns a pager object which can be used to iterate over the results of a paging operation.
	// To iterate over a page use the NextPage(context.Context) to fetch the next page of results.
	// PageResponse() can be used to iterate over the results of the specific page.
	// Always check the Err() method after paging to see if an error was returned by the pager. A pager will return either an error or the page of results.

	pager := fsClient.NewListPathsPager(nil)
	page, err := pager.NextPage(context.TODO())
	handle(err)

	// print the path names for this page
	for _, path := range page.PathList.Paths {
		fmt.Println(*path.Name)
        fmt.Println(*path.IsDirectory)
	}

	// Delete the file.
	_, err = fileClient.Delete(context.TODO(), nil)
	handle(err)

	// Delete the filesystem.
	_, err = fsClient.Delete(context.TODO(), nil)
	handle(err)
*/

package azdatalake
