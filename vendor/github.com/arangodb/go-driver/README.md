![ArangoDB-Logo](https://www.arangodb.com/docs/assets/arangodb_logo_2016_inverted.png)

# ArangoDB GO Driver

This project contains the official Go driver for the [ArangoDB database](https://arangodb.com).

[![Build Status](https://travis-ci.org/arangodb/go-driver.svg?branch=master)](https://travis-ci.org/arangodb/go-driver)
[![GoDoc](https://godoc.org/github.com/arangodb/go-driver?status.svg)](http://godoc.org/github.com/arangodb/go-driver)


- [Getting Started](https://www.arangodb.com/docs/stable/drivers/go-getting-started.html)
- [Example Requests](https://www.arangodb.com/docs/stable/drivers/go-example-requests.html)
- [Connection Management](https://www.arangodb.com/docs/stable/drivers/go-connection-management.html)
- [Reference](https://godoc.org/github.com/arangodb/go-driver)

# Supported Go Versions

|               | Go 1.13 | Go 1.14 | Go 1.16 |
|---------------|---------|---------|---------|
| `1.0.0-1.3.0` | ✓       | ✓       | ✓       |
| `master`      | ✓       | ✓       | ✓       |

# Supported Versions

|          | < ArangoDB 3.6 | ArangoDB 3.6 | ArangoDB 3.7 | ArangoDB 3.8 | ArangoDB 3.9 |
|----------|----------------|--------------|--------------|--------------|--------------|
| `1.0.0`  | ✓              | ✓            | -            | -            | -            |
| `1.1.0`  | +              | +            | ✓            | -            | -            |
| `1.2.1`  | +              | +            | ✓            | ✓            | -            |
| `1.3.0`  | +              | +            | ✓            | ✓            | ✓            |
| `master` | +              | +            | +            | +            | +            |

Key:

* `✓` Exactly the same features in both driver and the ArangoDB version.
* `+` Features included in driver may be not present in the ArangoDB API. Calls to the ArangoDB may result in unexpected responses (404).
* `-` The ArangoDB has features which are not supported by driver.
