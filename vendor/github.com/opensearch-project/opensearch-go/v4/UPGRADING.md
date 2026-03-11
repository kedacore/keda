- [Upgrading Opensearch GO Client](#upgrading-opensearch-go-client)
  - [Upgrading to >= 4.0.0](#upgrading-to->=-4.0.0)
    - [error types](#error-types)
  - [Upgrading to >= 3.0.0](#upgrading-to->=-3.0.0)
    - [client creation](#client-creation)
    - [requests](#requests)
    - [responses](#responses)
    - [error handing](#error-handling)
  - [Upgrading to >= 2.3.0](#upgrading-to->=-2.3.0)
    - [snapshot delete](#snapshot-delete)

# Upgrading Opensearch GO Client

## Upgrading to >= 5.0.0
Version 5.0.0 returns `*opensearch.StringError` error type instead of `*fmt.wrapError` when response received from the server is an unknown JSON. For example, consider delete document API which returns an unknown JSON body when document is not found.

Before 5.0.0:
```go
docDelResp, err = client.Document.Delete(ctx, opensearchapi.DocumentDeleteReq{Index: "movies", DocumentID: "3"})
if err != nil {
	fmt.Println(err)
	
	if !errors.Is(err, opensearch.ErrJSONUnmarshalBody) && docDelResp != nil {
		resp := docDelResp.Inspect().Response
		// get http status
		fmt.Println(resp.StatusCode)
		body := strings.TrimPrefix(err.Error(), "opensearch error response could not be parsed as error: ")
		errResp := opensearchapi.DocumentDeleteResp{}
		json.Unmarshal([]byte(body), &errResp) 
		// extract result field from the body 
		fmt.Println(errResp.Result)
	}
}
```

After 5.0.0:
```go
docDelResp, err = client.Document.Delete(ctx, opensearchapi.DocumentDeleteReq{Index: "movies", DocumentID: "3"})
if err != nil {
	// parse into *opensearch.StringError
	var myStringErr *opensearch.StringError
	if errors.As(err, &myStringErr) {
		// get http status
		fmt.Println(myStringErr.Status)
		errResp := opensearchapi.DocumentDeleteResp{}
		json.Unmarshal([]byte(myStringErr.Err), &errResp)
		// extract result field from the body
		fmt.Println(errResp.Result)
	}
}
```


## Upgrading to >= 4.0.0

Version 4.0.0 moved the error types, added with 3.0.0, from opensearchapi to opensearch, renamed them and added new error types.

### Error Types

Before 4.0.0:
Error types:
- `opensearchapi.Error`
- `opensearchapi.StringError`

With 4.0.0:
Error types
- `opensearch.Error`
- `opensearch.StringError`
- `opensearch.ReasonError`
- `opensearch.MessageError`
- `opensearch.StructError` (which was the `opensearchapi.Error`)

## Upgrading to >= 3.0.0

Version 3.0.0 is a major refactor of the client.

### Client Creation
You now create the client from the opensearchapi and not from the opensearch lib. This was done to make the different APIs independent from each other. Plugin APIs like Security will get there own folder and therefore its own sub-lib.

Before 3.0.0:
```go
// default client
client, err := opensearch.NewDefaultClient()

// with config
client, err := opensearch.NewClient(
    opensearch.Config{
	    Transport: &http.Transport{
		    TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Addresses: []string{"https://localhost:9200"},
		Username:  "admin",
		Password:  "admin",
	},
)
```

With 3.0.0:

```go
// default client
client, err := opensearchapi.NewDefaultClient()

// with config
client, err := opensearchapi.NewClient(
    opensearchapi.Config{
		Client: opensearch.Config{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // For testing only. Use certificate for validation.
			},
			Addresses: []string{"https://localhost:9200"},
			Username:  "admin", // For testing only. Don't store credentials in code.
			Password:  "admin",
		},
	},
)
```

### Requests

Prior version 3.0.0 there were two options on how to perform requests. You could either use the request struct of the wished function and execute it with the client .Do() function or use the client function and add wanted args with so called With<arg>() functions. With the new version you now use functions attached to the client and give a context and the wanted request body as argument.

Before 3.0.0:

```go
// using the client function and adding args by using the With<arg>() functions
createIndex, err := client.Indices.Create(
    "some-index",
    client.Indices.Create.WithContext(ctx),
    client.Indices.Create.WithBody(strings.NewReader(`{"settings":{"index":{"number_of_shards":4}}}`)),
)

// using the request struct
createIndex := opensearchapi.IndicesCreateRequest{
    Index: "some-index",
    Body:  strings.NewReader(`{"settings":{"index":{"number_of_shards":4}}}`),
}
createIndexResponse, err := createIndex.Do(ctx, client)
```

With 3.0.0:

```go
createIndexResponse, err := client.Indices.Create(
    ctx,
    opensearchapi.IndicesCreateReq{
        Index: "some-index",
        Body:  strings.NewReader(`{"settings":{"index":{"number_of_shards":4}}}`),
    },
)
```


### Responses

With the version 3.0.0 the lib no longer returns the opensearch.Response which is just a wrap up http.Response. Instead it will check the response for errors and try to parse the body into existing structs. Please note that some responses are so complex that we parse them as [json.RawMessage](https://pkg.go.dev/encoding/json#RawMessage) so you can parse them to your expected struct. If you need the opensearch.Response, then you can call .Inspect().

Before 3.0.0:

```go
// Create the request
createIndex := opensearchapi.IndicesCreateRequest{
    Index: "some-index",
    Body:  strings.NewReader(`{"settings":{"index":{"number_of_shards":4}}}`),
}
// Execute the requests
resp, err := createIndex.Do(ctx, client)
if err != nil {
	return err
}
// Close the body
defer resp.Body.Close()

// Check if the status code is >299
if resp.IsError() {
	return fmt.Errorf("Opensearch Returned an error: %#v", resp)
}

// Create a struct that represents the create index response
createResp := struct {
	Acknowledged       bool   `json:"acknowledged"`
	ShardsAcknowledged bool   `json:"shards_acknowledged"`
	Index              string `json:"index"`
}

// Try to parse the response into the created struct
if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
	return fmt.Errorf("Undexpected response body: %s, %#v, %s"resp.StatusCode, resp.Body, err)
}
// Print the created index name
fmt.Println(createResp.Index)
```

With 3.0.0:

```go
// Create and execute the requests
createResp, err := client.Indices.Create(
    ctx,
    opensearchapi.IndicesCreateReq{
        Index: "some-index",
        Body:  strings.NewReader(`{"settings":{"index":{"number_of_shards":4}}}`),
    },
)
if err != nil {
	return err
}
// Print the created index name
fmt.Println(createResp.Index)

// To get the opensearch.Response/http.Response
rawResp := createResp.Inspect().Response
```

### Error Handling

With opensearch-go >= 3.0.0 opensearchapi responses are now checked for errors. Checking for errors twice is no longer needed.

Prior versions only returned an error if the request failed to execute. For example if the client can't reach the server or the TLS handshake failed. With opensearch-go >= 3.0.0 each opensearchapi requests will return an error if the response http status code is > 299. The error can be parsed into the new `opensearchapi.Error` type by using `errors.As` to match for exceptions and get a more detailed view.

Before 3.0.0:

```go
// Create the request
createIndex := opensearchapi.IndicesCreateRequest{
    Index: "some-index",
    Body:  strings.NewReader(`{"settings":{"index":{"number_of_shards":4}}}`),
}

// Execute the requests
resp, err := createIndex.Do(ctx, client)
if err != nil {
	return err
}
// Close the body
defer resp.Body.Close()

// Check if the status code is >299
if createIndexResp.IsError() {
    fmt.Errorf("Opensearch returned an error. Status: %d", createIndexResp.StatusCode)
}
```

With 3.0.0:

```go
var opensearchError opensearchapi.Error
// Create and execute the requests
createResp, err := client.Indices.Create(
    ctx,
    opensearchapi.IndicesCreateReq{
        Index: "some-index",
        Body:  strings.NewReader(`{"settings":{"index":{"number_of_shards":4}}}`),
    },
)
// Load err into opensearchapi.Error to access the fields and tolerate if the index already exists
if err != nil {
	if errors.As(err, &opensearchError) {
		if opensearchError.Err.Type != "resource_already_exists_exception" {
			return err
		}
	} else {
		return err
	}
}
```

## Upgrading to >= 2.3.0

### Snapshot Delete

`SnapshotDeleteRequest` and `SnapshotDelete` changed the argument `Snapshot` type from `string` to `[]string`.

Before 2.3.0:

```go
// If you have a string containing your snapshot
stringSnapshotsToDelete := "snapshot-1,snapshot-2"
reqSnapshots := &opensearchapi.SnapshotDeleteRequest{
  Repository: repo,
	Snapshot: stringSnapshotsToDelete,
}

// If you have a slice of strings containing your snapshot
sliceSnapshotToDelete := []string{"snapshot-1","snapshot-2"}
reqSnapshots := &opensearchapi.SnapshotDeleteRequest{
  Repository: repo,
  Snapshot: strings.Join(sliceSnapshotsToDelete, ","),
}
```

With 2.3.0:

```go
// If you have a string containing your snapshots
stringSnapshotsToDelete := strings.Split("snapshot-1,snapshot-2", ",")
reqSnapshots := &opensearchapi.SnapshotDeleteRequest{
  Repository: repo,
  Snapshot:   stringSnapshotsToDelete,
}

// If you have a slice of strings containing your snapshots
sliceSnapshotToDelete := []string{"snapshot-1", "snapshot-2"}
reqSnapshots := &opensearchapi.SnapshotDeleteRequest{
  Repository: repo,
  Snapshot: sliceSnapshotsToDelete,
```
