[![Build Status](https://travis-ci.org/go-kivik/couchdb.svg?branch=master)](https://travis-ci.org/go-kivik/couchdb) [![Codecov](https://img.shields.io/codecov/c/github/go-kivik/couchdb.svg?style=flat)](https://codecov.io/gh/go-kivik/couchdb) [![GoDoc](https://godoc.org/github.com/go-kivik/couchdb?status.svg)](http://godoc.org/github.com/go-kivik/couchdb)

# Kivik CouchDB

CouchDB driver for [Kivik](https://github.com/go-kivik/kivik).

## Usage

This package provides an implementation of the
[`github.com/go-kivik/kivik/v3/driver`](http://godoc.org/github.com/go-kivik/kivik/driver)
interface. You must import the driver and can then use the full
[`Kivik`](http://godoc.org/github.com/go-kivik/kivik) API. Please consult the
[Kivik wiki](https://github.com/go-kivik/kivik/wiki) for complete documentation
and coding examples.

```go
package main

import (
    "context"

    kivik "github.com/go-kivik/kivik/v3"
    _ "github.com/go-kivik/couchdb/v3" // The CouchDB driver
)

func main() {
    client, err := kivik.New(context.TODO(), "couch", "")
    // ...
}
```

## License

This software is released under the terms of the Apache 2.0 license. See
LICENCE.md, or read the [full license](http://www.apache.org/licenses/LICENSE-2.0).
