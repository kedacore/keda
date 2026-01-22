#!/usr/bin/env bash

set -e

go test -race -v -coverpkg=./... -coverprofile=profile.out ./...
go tool cover -func profile.out
