#! /bin/sh

set -e

DIR="$(dirname $0)"


$DIR/../vendor/k8s.io/code-generator/generate-groups.sh all \
    github.com/kedacore/keda/pkg/client \
    github.com/kedacore/keda/pkg/apis \
    keda:v1alpha1 \
    --go-header-file $DIR/boilerplate.go.txt
