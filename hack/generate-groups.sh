#! /bin/sh

set -e

DIR="$(dirname $0)"


$DIR/../vendor/k8s.io/code-generator/generate-groups.sh all \
    github.com/Azure/Kore/pkg/client \
    github.com/Azure/Kore/pkg/apis \
    kore:v1alpha1