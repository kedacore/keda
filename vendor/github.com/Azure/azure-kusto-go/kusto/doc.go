// Copyright 2020 Microsoft Corporation. All rights reserved.
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//
// Additional licenses for third-pary packages can be found in third_party_licenses/

/*
Package kusto provides a client for accessing Azure Data Explorer, also known as Kusto.

For details on the Azure Kusto service, see: https://azure.microsoft.com/en-us/services/data-explorer/

For general documentation on APIs and the Kusto query language, see: https://docs.microsoft.com/en-us/azure/data-explorer/

## Examples

Examples for various scenarios can be found on [pkg.go.dev](https://pkg.go.dev/github.com/Azure/azure-kusto-go#readme-examples) or in the example*_test.go files in our GitHub repo for [azure-kusto-go](https://github.com/Azure/azure-kusto-go/tree/master/kusto).

### Create the connection string

Azure Data Explorer (Kusto) connection strings are created using a connection string builder for an existing Azure Data Explorer (Kusto) cluster endpoint of the form `https://<cluster name>.<location>.kusto.windows.net`.

```go
kustoConnectionStringBuilder := kusto.NewConnectionStringBuilder(endpoint)
```

### Create and authenticate the client

Azure Data Explorer (Kusto) clients are created from a connection string and authenticated using a credential from the [Azure Identity package][azure_identity_pkg], like [DefaultAzureCredential][default_azure_credential].
You can also authenticate a client using a system- or user-assigned managed identity with Azure Active Directory (AAD) credentials.

#### Using the `DefaultAzureCredential`

```go
// kusto package is: github.com/Azure/azure-kusto-go/kusto

// Initialize a new kusto client using the default Azure credential
kustoConnectionString := kustoConnectionStringBuilder.WithDefaultAzureCredential()
client, err = kusto.New(kustoConnectionString)

	if err != nil {
		panic("add error handling")
	}

// Be sure to close the client when you're done. (Error handling omitted for brevity.)
defer client.Close()
```

#### Using the `az cli`

```go
kustoConnectionString := kustoConnectionStringBuilder.WithAzCli()
client, err = kusto.New(kustoConnectionString)
```

#### Using a system-assigned managed identity

```go
kustoConnectionString := kustoConnectionStringBuilder.WithSystemManagedIdentity()
client, err = kusto.New(kustoConnectionString)
```

#### Using a user-assigned managed identity

```go
kustoConnectionString := kustoConnectionStringBuilder.WithUserManagedIdentity(clientID)
client, err = kusto.New(kustoConnectionString)
```

#### Using a bearer token

```go
kustoConnectionString := kustoConnectionStringBuilder.WithApplicationToken(appId, token)
client, err = kusto.New(kustoConnectionString)
```

#### Using an app id and secret

```go
kustoConnectionString := kustoConnectionStringBuilder.WithAadAppKey(clientID, clientSecret, tenantID)
client, err = kusto.New(kustoConnectionString)
```

#### Using an application certificate

```go
kustoConnectionString := kustoConnectionStringBuilder.WithAppCertificate(appId, certificate, thumbprint, sendCertChain, authorityID)
client, err = kusto.New(kustoConnectionString)
```

### Querying

#### Simple queries

* Works for queries and management commands.
* Limited to queries that can be built using a string literal known at compile time.

The simplest queries can be built using `kql.New`:

```go
query := kql.New("systemNodes | project CollectionTime, NodeId")
```

Queries can only be built using a string literals known at compile time, and special methods for specific parts of the query.
The reason for this is to discourage the use of string concatenation to build queries, which can lead to security vulnerabilities.

#### Queries with parameters

* Can re-use the same query with different parameters.
* Only work for queries, management commands are not supported.

It is recommended to use parameters for queries that contain user input.
Management commands can not use parameters, and therefore should be built using the builder (see next section).

Parameters can be implicitly referenced in a query:

```go
query := kql.New("systemNodes | project CollectionTime, NodeId | where CollectionTime > startTime and NodeId == nodeIdValue")
```

Here, `startTime` and `nodeIdValue` are parameters that can be passed to the query.

To Pass the parameters values to the query, create `kql.Parameters`:

```
params :=  kql.NewParameters().AddDateTime("startTime", dt).AddInt("nodeIdValue", 1)
```

And then pass it to the `Query` method, as an option:
```go
results, err := client.Query(ctx, database, query, QueryParameters(params))

	if err != nil {
	    panic("add error handling")
	}

// You can see the generated parameters using the ToDeclarationString() method:
fmt.Println(params.ToDeclarationString()) // declare query_parameters(startTime:datetime, nodeIdValue:int);
```

#### Queries with inline parameters
* Works for queries and management commands.
* More involved building of queries, but allows for more flexibility.

Queries with runtime data can be built using `kql.New`.
The builder will only accept the correct types for each part of the query, and will escape any special characters in the data.

For example, here is a query that dynamically accepts values for the table name, and the comparison parameters for the columns:

```go
dt, _ := time.Parse(time.RFC3339Nano, "2020-03-04T14:05:01.3109965Z")
tableName := "system nodes"
value := 1

query := kql.New("")

	.AddTable(tableName)
	.AddLiteral(" | where CollectionTime == ").AddDateTime(dt)
	.AddLiteral(" and ")
	.AddLiteral("NodeId == ").AddInt(value)

// To view the query string, use the String() method:
fmt.Println(query.String())
// Output: ['system nodes'] | where CollectionTime == datetime(2020-03-04T14:05:01.3109965Z) and NodeId == int(1)
```

Building queries like this is useful for queries that are built from user input, or for queries that are built from a template, and are valid for management commands too.

#### Query For Rows

The kusto `table` package queries data into a ***table.Row** which can be printed or have the column data extracted.

```go
// table package is: github.com/Azure/azure-kusto-go/kusto/data/table

// Query our database table "systemNodes" for the CollectionTimes and the NodeIds.
iter, err := client.Query(ctx, "database", query)

	if err != nil {
		panic("add error handling")
	}

defer iter.Stop()

// .Do() will call the function for every row in the table.
err = iter.DoOnRowOrError(

	    func(row *table.Row, e *kustoErrors.Error) error {
	        if e != nil {
	            return e
	        }
	        if row.Replace {
	            fmt.Println("---") // Replace flag indicates that the query result should be cleared and replaced with this row
	        }
	        fmt.Println(row) // As a convenience, printing a *table.Row will output csv
	        return nil
		},

)

	if err != nil {
		panic("add error handling")
	}

```

#### Query Into Structs

Users will often want to turn the returned data into Go structs that are easier to work with.  The ***table.Row** object
that is returned supports this via the `.ToStruct()` method.

```go
// NodeRec represents our Kusto data that will be returned.

	type NodeRec struct {
		// ID is the table's NodeId. We use the field tag here to instruct our client to convert NodeId to ID.
		ID int64 `kusto:"NodeId"`
		// CollectionTime is Go representation of the Kusto datetime type.
		CollectionTime time.Time
	}

iter, err := client.Query(ctx, "database", query)

	if err != nil {
		panic("add error handling")
	}

defer iter.Stop()

recs := []NodeRec{}
err = iter.DoOnRowOrError(

	    func(row *table.Row, e *kustoErrors.Error) error {
	        if e != nil {
	        return e
	        }
			rec := NodeRec{}
			if err := row.ToStruct(&rec); err != nil {
				return err
			}
			if row.Replace {
				recs = recs[:0] // Replace flag indicates that the query result should be cleared and replaced with this row
			}
			recs = append(recs, rec)
			return nil
		},

)

	if err != nil {
		panic("add error handling")
	}

```

### Ingestion

The `ingest` package provides access to Kusto's ingestion service for importing data into Kusto. This requires
some prerequisite knowledge of acceptable data formats, mapping references, etc.

That documentation can be found [here](https://docs.microsoft.com/en-us/azure/kusto/management/data-ingestion/)

If ingesting data from memory, it is suggested that you stream the data in via `FromReader()` passing in the reader
from an `io.Pipe()`. The data will not begin ingestion until the writer closes.

#### Creating a queued ingestion client

Setup is quite simple, simply pass a `*kusto.Client`, the name of the database and table you wish to ingest into.

```go
in, err := ingest.New(kustoClient, "database", "table")

	if err != nil {
		panic("add error handling")
	}

// Be sure to close the ingestor when you're done. (Error handling omitted for brevity.)
defer in.Close()
```

#### Other Ingestion Clients

There are other ingestion clients that can be used for different ingestion scenarios.  The `ingest` package provides
the following clients:
  - Queued Ingest - `ingest.New()` - the default client, uses queues and batching to ingest data. Most reliable.
  - Streaming Ingest - `ingest.NewStreaming()` - Directly streams data into the engine. Fast, but is limited with size and can fail.
  - Managed Streaming Ingest - `ingest.NewManaged()` - Combines a streaming ingest client with a queued ingest client to provide a reliable ingestion method that is fast and can ingest large amounts of data.
    Managed Streaming will try to stream the data, and if it fails multiple times, it will fall back to a queued ingestion.

#### Ingestion From a File

Ingesting a local file requires simply passing the path to the file to be ingested:

```go

	if _, err := in.FromFile(ctx, "/path/to/a/local/file"); err != nil {
		panic("add error handling")
	}

```

`FromFile()` will accept Unix path names on Unix platforms and Windows path names on Windows platforms.
The file will not be deleted after upload (there is an option that will allow that though).

#### From a Blob Storage File

This package will also accept ingestion from an Azure Blob Storage file:

```go

	if _, err := in.FromFile(ctx, "https://myaccount.blob.core.windows.net/$root/myblob"); err != nil {
		panic("add error handling")
	}

```

This will ingest a file from Azure Blob Storage. We only support `https://` paths and your domain name may differ than what is here.

#### Ingestion from an io.Reader

Sometimes you want to ingest a stream of data that you have in memory without writing to disk.  You can do this simply by chunking the
data via an `io.Reader`.

```go
r, w := io.Pipe()

enc := json.NewEncoder(w)

	go func() {
		defer w.Close()
		for _, data := range dataSet {
			if err := enc.Encode(data); err != nil {
				panic("add error handling")
			}
		}
	}()

	if _, err := in.FromReader(ctx, r); err != nil {
		panic("add error handling")
	}

```

It is important to remember that `FromReader()` will terminate when it receives an `io.EOF` from the `io.Reader`.  Use `io.Readers` that won't
return `io.EOF` until the `io.Writer` is closed (such as `io.Pipe`).

# Mocking

To support mocking for this client in your code for hermetic testing purposes, this client supports mocking the data
returned by our RowIterator object. Please see the MockRows documentation for code examples.

# Package Examples

Below you will find a simple and complex example of doing Query() the represent compiled code:
*/
package kusto
