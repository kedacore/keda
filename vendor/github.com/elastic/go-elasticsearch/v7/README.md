# go-elasticsearch

The official Go client for [Elasticsearch](https://www.elastic.co/products/elasticsearch).

[![GoDoc](https://godoc.org/github.com/elastic/go-elasticsearch?status.svg)](https://pkg.go.dev/github.com/elastic/go-elasticsearch/v7)
[![Go Report Card](https://goreportcard.com/badge/github.com/elastic/go-elasticsearch)](https://goreportcard.com/report/github.com/elastic/go-elasticsearch)
[![codecov.io](https://codecov.io/github/elastic/go-elasticsearch/coverage.svg?branch=main)](https://codecov.io/gh/elastic/go-elasticsearch?branch=main)
[![Build](https://github.com/elastic/go-elasticsearch/workflows/Build/badge.svg)](https://github.com/elastic/go-elasticsearch/actions?query=branch%3Amain)
[![Unit](https://github.com/elastic/go-elasticsearch/workflows/Unit/badge.svg)](https://github.com/elastic/go-elasticsearch/actions?query=branch%3Amain)
[![Integration](https://github.com/elastic/go-elasticsearch/workflows/Integration/badge.svg)](https://github.com/elastic/go-elasticsearch/actions?query=branch%3Amain)
[![API](https://github.com/elastic/go-elasticsearch/workflows/API/badge.svg)](https://github.com/elastic/go-elasticsearch/actions?query=branch%3Amain)

## Compatibility

Language clients are forward compatible; meaning that clients support communicating with greater or equal minor versions of Elasticsearch.
Elasticsearch language clients are only backwards compatible with default distributions and without guarantees made.

When using Go modules, include the version in the import path, and specify either an explicit version or a branch:

    require github.com/elastic/go-elasticsearch/v7 7.16
    require github.com/elastic/go-elasticsearch/v7 7.0.0

It's possible to use multiple versions of the client in a single project:

    // go.mod
    github.com/elastic/go-elasticsearch/v6 6.x
    github.com/elastic/go-elasticsearch/v7 7.16

    // main.go
    import (
      elasticsearch6 "github.com/elastic/go-elasticsearch/v6"
      elasticsearch7 "github.com/elastic/go-elasticsearch/v7"
    )
    // ...
    es6, _ := elasticsearch6.NewDefaultClient()
    es7, _ := elasticsearch7.NewDefaultClient()

The `main` branch of the client is compatible with the current `master` branch of Elasticsearch.

<!-- ----------------------------------------------------------------------------------------------- -->

## Installation

Add the package to your `go.mod` file:

    require github.com/elastic/go-elasticsearch/v8 main

Or, clone the repository:

    git clone --branch main https://github.com/elastic/go-elasticsearch.git $GOPATH/src/github.com/elastic/go-elasticsearch

A complete example:

```bash
mkdir my-elasticsearch-app && cd my-elasticsearch-app

cat > go.mod <<-END
  module my-elasticsearch-app

  require github.com/elastic/go-elasticsearch/v8 main
END

cat > main.go <<-END
  package main

  import (
    "log"

    "github.com/elastic/go-elasticsearch/v7"
  )

  func main() {
    es, _ := elasticsearch.NewDefaultClient()
    log.Println(elasticsearch.Version)
    log.Println(es.Info())
  }
END

go run main.go
```


<!-- ----------------------------------------------------------------------------------------------- -->

## Usage

The `elasticsearch` package ties together two separate packages for calling the Elasticsearch APIs and transferring data over HTTP: `esapi` and `estransport`, respectively.

Use the `elasticsearch.NewDefaultClient()` function to create the client with the default settings.

```golang
es, err := elasticsearch.NewDefaultClient()
if err != nil {
  log.Fatalf("Error creating the client: %s", err)
}

res, err := es.Info()
if err != nil {
  log.Fatalf("Error getting response: %s", err)
}

defer res.Body.Close()
log.Println(res)

// [200 OK] {
//   "name" : "node-1",
//   "cluster_name" : "go-elasticsearch"
// ...
```

> NOTE: It is _critical_ to both close the response body _and_ to consume it, in order to re-use persistent TCP connections in the default HTTP transport. If you're not interested in the response body, call `io.Copy(ioutil.Discard, res.Body)`.

When you export the `ELASTICSEARCH_URL` environment variable,
it will be used to set the cluster endpoint(s). Separate multiple adresses by a comma.

To set the cluster endpoint(s) programatically, pass a configuration object
to the `elasticsearch.NewClient()` function.

```golang
cfg := elasticsearch.Config{
  Addresses: []string{
    "http://localhost:9200",
    "http://localhost:9201",
  },
  // ...
}
es, err := elasticsearch.NewClient(cfg)
```

To set the username and password, include them in the endpoint URL,
or use the corresponding configuration options.

```golang
cfg := elasticsearch.Config{
  // ...
  Username: "foo",
  Password: "bar",
}
```

To set a custom certificate authority used to sign the certificates of cluster nodes,
use the `CACert` configuration option.

```golang
cert, _ := ioutil.ReadFile(*cacert)

cfg := elasticsearch.Config{
  // ...
  CACert: cert,
}
```

To configure other HTTP settings, pass an [`http.Transport`](https://golang.org/pkg/net/http/#Transport)
object in the configuration object.

```golang
cfg := elasticsearch.Config{
  Transport: &http.Transport{
    MaxIdleConnsPerHost:   10,
    ResponseHeaderTimeout: time.Second,
    TLSClientConfig: &tls.Config{
      MinVersion: tls.VersionTLS12,
      // ...
    },
    // ...
  },
}
```

See the [`_examples/configuration.go`](_examples/configuration.go) and
[`_examples/customization.go`](_examples/customization.go) files for
more examples of configuration and customization of the client.
See the [`_examples/security`](_examples/security) for an example of a security configuration.

The following example demonstrates a more complex usage. It fetches the Elasticsearch version from the cluster, indexes a couple of documents concurrently, and prints the search results, using a lightweight wrapper around the response body.

```golang
// $ go run _examples/main.go

package main

import (
  "bytes"
  "context"
  "encoding/json"
  "log"
  "strconv"
  "strings"
  "sync"
  "bytes"

  "github.com/elastic/go-elasticsearch/v7"
  "github.com/elastic/go-elasticsearch/v7/esapi"
)

func main() {
  log.SetFlags(0)

  var (
    r  map[string]interface{}
    wg sync.WaitGroup
  )

  // Initialize a client with the default settings.
  //
  // An `ELASTICSEARCH_URL` environment variable will be used when exported.
  //
  es, err := elasticsearch.NewDefaultClient()
  if err != nil {
    log.Fatalf("Error creating the client: %s", err)
  }

  // 1. Get cluster info
  //
  res, err := es.Info()
  if err != nil {
    log.Fatalf("Error getting response: %s", err)
  }
  defer res.Body.Close()
  // Check response status
  if res.IsError() {
    log.Fatalf("Error: %s", res.String())
  }
  // Deserialize the response into a map.
  if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
    log.Fatalf("Error parsing the response body: %s", err)
  }
  // Print client and server version numbers.
  log.Printf("Client: %s", elasticsearch.Version)
  log.Printf("Server: %s", r["version"].(map[string]interface{})["number"])
  log.Println(strings.Repeat("~", 37))

  // 2. Index documents concurrently
  //
  for i, title := range []string{"Test One", "Test Two"} {
    wg.Add(1)

    go func(i int, title string) {
      defer wg.Done()

      // Build the request body.      
      data, err := json.Marshal(struct{ Title string }{Title: title})
      if err != nil {
        log.Fatalf("Error marshaling document: %s", err)
      }

      // Set up the request object.
      req := esapi.IndexRequest{
        Index:      "test",
        DocumentID: strconv.Itoa(i + 1),
        Body:       bytes.NewReader(data),
        Refresh:    "true",
      }

      // Perform the request with the client.
      res, err := req.Do(context.Background(), es)
      if err != nil {
        log.Fatalf("Error getting response: %s", err)
      }
      defer res.Body.Close()

      if res.IsError() {
        log.Printf("[%s] Error indexing document ID=%d", res.Status(), i+1)
      } else {
        // Deserialize the response into a map.
        var r map[string]interface{}
        if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
          log.Printf("Error parsing the response body: %s", err)
        } else {
          // Print the response status and indexed document version.
          log.Printf("[%s] %s; version=%d", res.Status(), r["result"], int(r["_version"].(float64)))
        }
      }
    }(i, title)
  }
  wg.Wait()

  log.Println(strings.Repeat("-", 37))

  // 3. Search for the indexed documents
  //
  // Build the request body.
  var buf bytes.Buffer
  query := map[string]interface{}{
    "query": map[string]interface{}{
      "match": map[string]interface{}{
        "title": "test",
      },
    },
  }
  if err := json.NewEncoder(&buf).Encode(query); err != nil {
    log.Fatalf("Error encoding query: %s", err)
  }

  // Perform the search request.
  res, err = es.Search(
    es.Search.WithContext(context.Background()),
    es.Search.WithIndex("test"),
    es.Search.WithBody(&buf),
    es.Search.WithTrackTotalHits(true),
    es.Search.WithPretty(),
  )
  if err != nil {
    log.Fatalf("Error getting response: %s", err)
  }
  defer res.Body.Close()

  if res.IsError() {
    var e map[string]interface{}
    if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
      log.Fatalf("Error parsing the response body: %s", err)
    } else {
      // Print the response status and error information.
      log.Fatalf("[%s] %s: %s",
        res.Status(),
        e["error"].(map[string]interface{})["type"],
        e["error"].(map[string]interface{})["reason"],
      )
    }
  }

  if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
    log.Fatalf("Error parsing the response body: %s", err)
  }
  // Print the response status, number of results, and request duration.
  log.Printf(
    "[%s] %d hits; took: %dms",
    res.Status(),
    int(r["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64)),
    int(r["took"].(float64)),
  )
  // Print the ID and document source for each hit.
  for _, hit := range r["hits"].(map[string]interface{})["hits"].([]interface{}) {
    log.Printf(" * ID=%s, %s", hit.(map[string]interface{})["_id"], hit.(map[string]interface{})["_source"])
  }

  log.Println(strings.Repeat("=", 37))
}

// Client: 7.0.0-SNAPSHOT
// Server: 7.0.0-SNAPSHOT
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
// [201 Created] updated; version=1
// [201 Created] updated; version=1
// -------------------------------------
// [200 OK] 2 hits; took: 5ms
//  * ID=1, map[title:Test One]
//  * ID=2, map[title:Test Two]
// =====================================
```

As you see in the example above, the `esapi` package allows to call the Elasticsearch APIs in two distinct ways: either by creating a struct, such as `IndexRequest`, and calling its `Do()` method by passing it a context and the client, or by calling the `Search()` function on the client directly, using the option functions such as `WithIndex()`. See more information and examples in the
[package documentation](https://godoc.org/github.com/elastic/go-elasticsearch/esapi).

The `estransport` package handles the transfer of data to and from Elasticsearch, including retrying failed requests, keeping a connection pool, discovering cluster nodes and logging.

Read more about the client internals and usage in the following blog posts:

* https://www.elastic.co/blog/the-go-client-for-elasticsearch-introduction
* https://www.elastic.co/blog/the-go-client-for-elasticsearch-configuration-and-customization
* https://www.elastic.co/blog/the-go-client-for-elasticsearch-working-with-data

<!-- ----------------------------------------------------------------------------------------------- -->

## Helpers

The `esutil` package provides convenience helpers for working with the client. At the moment, it provides the `esutil.JSONReader()` and the `esutil.BulkIndexer` helpers.

<!-- ----------------------------------------------------------------------------------------------- -->

## Examples

The **[`_examples`](./_examples)** folder contains a number of recipes and comprehensive examples to get you started with the client, including configuration and customization of the client, using a custom certificate authority (CA) for security (TLS), mocking the transport for unit tests, embedding the client in a custom type, building queries, performing requests individually and in bulk, and parsing the responses.

<!-- ----------------------------------------------------------------------------------------------- -->

## License

(c) 2019 Elasticsearch. Licensed under the Apache License, Version 2.0.
