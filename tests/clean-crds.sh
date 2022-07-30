#! /bin/bash

echo "Cleaning up scaled objects and jobs before undeploying KEDA"
while read -r namespace
do
    resources=$(kubectl get so,sj -n $namespace -o name)
    if [[ -n  "$resources" ]]
    then
        kubectl delete $resources -n $namespace
    fi
done < <(kubectl get namespaces -o jsonpath="{range .items[*]}{.metadata.name}{'\n'}{end}")
