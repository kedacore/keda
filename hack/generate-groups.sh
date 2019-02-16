#! /bin/sh

$GOPATH/src/k8s.io/code-generator/generate-groups.sh all \
    github.com/Azure/Kore/pkg/client \
    github.com/Azure/Kore/pkg/apis \
    kesc:v1alpha1