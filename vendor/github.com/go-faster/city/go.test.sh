#!/usr/bin/env bash

set -e

echo "test"
go test --timeout 5m ./...

echo "test -race"
go test --timeout 5m -race ./...
