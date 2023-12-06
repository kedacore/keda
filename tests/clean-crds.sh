#! /bin/bash

echo "Cleaning up CRDs before undeploying KEDA"
while read -r namespace
do
    resources=$(kubectl get so,sj,ta,cta,cloudeventsource -n $namespace -o name)
    if [[ -n  "$resources" ]]
    then
        kubectl delete $resources -n $namespace
    fi
done < <(kubectl get namespaces -o jsonpath="{range .items[*]}{.metadata.name}{'\n'}{end}")
