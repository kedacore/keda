#! /bin/bash

# Removes finalizers from every KEDA custom resource left in the cluster and
# deletes the external metrics apiservice, in case KEDA was removed before its
# resources were properly finalized (e.g. operator deleted mid e2e run),
# leaving namespaces stuck in "Terminating".

set -euo pipefail

echo "Force cleaning up leftover KEDA resources"

while read -r namespace
do
    resources=$(kubectl get so,sj,ta,cloudeventsource -n "$namespace" -o name --ignore-not-found=true)
    if [[ -n "$resources" ]]
    then
        while read -r resource
        do
            kubectl patch "$resource" -n "$namespace" --type=merge -p '{"metadata":{"finalizers":null}}' || true
            kubectl delete "$resource" -n "$namespace" --ignore-not-found=true --timeout=60s || true
        done <<< "$resources"
    fi
done < <(kubectl get namespaces -o jsonpath="{range .items[*]}{.metadata.name}{'\n'}{end}")

cluster_resources=$(kubectl get cta,clustercloudeventsource -o name --ignore-not-found=true)
if [[ -n "$cluster_resources" ]]
then
    while read -r resource
    do
        kubectl patch "$resource" --type=merge -p '{"metadata":{"finalizers":null}}' || true
        kubectl delete "$resource" --ignore-not-found=true --timeout=60s || true
    done <<< "$cluster_resources"
fi

echo "Removing external metrics apiservice if present"
kubectl delete apiservice v1beta1.external.metrics.k8s.io --ignore-not-found=true --timeout=60s
