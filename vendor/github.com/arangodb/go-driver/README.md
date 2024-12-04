# ArangoDB Go Driver

This project contains the official Go driver for the [ArangoDB database system](https://arangodb.com).

[![CircleCI](https://dl.circleci.com/status-badge/img/gh/arangodb/go-driver/tree/master.svg?style=svg)](https://dl.circleci.com/status-badge/redirect/gh/arangodb/go-driver/tree/master)
[![GoDoc](https://godoc.org/github.com/arangodb/go-driver?status.svg)](http://godoc.org/github.com/arangodb/go-driver)

Version 2:
- Tutorial coming soon
- [Code examples](v2/examples/)
- [Reference documentation](https://godoc.org/github.com/arangodb/go-driver/v2)

Version 1:
- ⚠️ This version is deprecated and will not receive any new features.
  Please use version 2 ([v2](v2/)) instead.
- [Tutorial](Tutorial_v1.md)
- [Code examples](examples/)
- [Reference documentation](https://godoc.org/github.com/arangodb/go-driver)

## Supported Go Versions

| Driver        | Go 1.19 | Go 1.20 | Go 1.21 |
|---------------|---------|---------|---------|
| `1.5.0-1.6.1` | ✓       | -       | -       |
| `1.6.2`       | ✓       | ✓       | ✓       |
| `2.1.0`       | ✓       | ✓       | ✓       |
| `master`      | ✓       | ✓       | ✓       |

## Supported ArangoDB Versions

| Driver   | ArangoDB 3.10 | ArangoDB 3.11 | ArangoDB 3.12 |
|----------|---------------|---------------|---------------|
| `1.5.0`  | ✓             | -             | -             |
| `1.6.0`  | ✓             | ✓             | -             |
| `2.1.0`  | ✓             | ✓             | ✓             |
| `master` | +             | +             | +             |

Key:

* `✓` Exactly the same features in both the driver and the ArangoDB version.
* `+` Features included in the driver may be not present in the ArangoDB API.
  Calls to ArangoDB may result in unexpected responses (404).
* `-` The ArangoDB version has features that are not supported by the driver.
