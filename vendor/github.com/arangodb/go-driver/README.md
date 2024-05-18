![ArangoDB-Logo](https://user-images.githubusercontent.com/3998723/207981337-79d49127-48fc-4c7c-9411-8a688edca1dd.png)


# ArangoDB GO Driver

This project contains the official Go driver for the [ArangoDB database](https://arangodb.com).

[![CircleCI](https://dl.circleci.com/status-badge/img/gh/arangodb/go-driver/tree/master.svg?style=svg)](https://dl.circleci.com/status-badge/redirect/gh/arangodb/go-driver/tree/master)
[![GoDoc](https://godoc.org/github.com/arangodb/go-driver?status.svg)](http://godoc.org/github.com/arangodb/go-driver)


- [Getting Started](https://www.arangodb.com/docs/stable/drivers/go-getting-started.html)
- [Example Requests](https://www.arangodb.com/docs/stable/drivers/go-example-requests.html)
- [Connection Management](https://www.arangodb.com/docs/stable/drivers/go-connection-management.html)
- [Reference](https://godoc.org/github.com/arangodb/go-driver)

# Supported Go Versions

|               | Go 1.19 | Go 1.20 | Go 1.21 |
|---------------|---------|---------|---------|
| `1.5.0-1.6.1` | ✓       | -       | -       |
| `1.6.2`       | ✓       | ✓       | ✓       |
| `2.1.0`       | ✓       | ✓       | ✓       |
| `master`      | ✓       | ✓       | ✓       |

# Supported Versions

|          | ArangoDB 3.10 | ArangoDB 3.11 | ArangoDB 3.12 |
|----------|---------------|---------------|---------------|
| `1.5.0`  | ✓             | -             | -             |
| `1.6.0`  | ✓             | ✓             | -             |
| `2.1.0`  | ✓             | ✓             | ✓             |
| `master` | +             | +             | +             |

Key:

* `✓` Exactly the same features in both driver and the ArangoDB version.
* `+` Features included in driver may be not present in the ArangoDB API. Calls to the ArangoDB may result in unexpected responses (404).
* `-` The ArangoDB has features which are not supported by driver.
