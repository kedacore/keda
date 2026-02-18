#!/usr/bin/env bash

set -e

# test with -race
echo "with race:"
go test --timeout 5m -race ./...

# test with noerrtrace build tag
tag=noerrtrace
echo "with ${tag} build tag:"
go test -tags "${tag}" --timeout 5m -race ./...
