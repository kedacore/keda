#!/bin/bash
export GOOS=linux
export CGO_ENABLED=0
set -eux
mkdir -p ../dist
go build -o keda ../cmd
mv keda ../dist
