#!/usr/bin/env bash

set -e

go test -v -coverpkg=./... -coverprofile=profile.out ./...
go tool cover -func profile.out
