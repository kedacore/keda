#! /bin/bash

# Remove Keda
helm delete --purge keda-test-release
kubectl delete crd scaledobjects.kore.k8s.io

# Remove Tiller
helm reset

# Remove Tiller's service account
kubectl delete ClusterRoleBinding tiller
kubectl delete ServiceAccount tiller --namespace kube-system


# https://stackoverflow.com/a/33510531/3234163
for each in $(kubectl get ns -o jsonpath="{.items[*].metadata.name}" | grep -v kube-system);
do
  kubectl delete ns $each
done

az aks delete --resource-group $AZURE_RESOURCE_GROUP --name $AKS_NAME --no-wait --yes
