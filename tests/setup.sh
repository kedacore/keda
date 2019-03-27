#! /bin/bash

set -e

echo "Login to azure"
az login --service-principal -u "$AZURE_SP_ID" -p "$AZURE_SP_KEY" --tenant "$AZURE_SP_TENANT"

echo "Set subscription to $AZURE_SUBSCRIPTION"
az account set --subscription $AZURE_SUBSCRIPTION

az group create --name $AZURE_RESOURCE_GROUP --location westus

if [ -z "$SKIP_AKS_CREATE" ]; then
    echo "Check for AKS cluster $AKS_NAME"
    set +e
    az aks show --resource-group $AZURE_RESOURCE_GROUP --name $AKS_NAME
    result=$?
    set -e
    if [ $result -eq 3 ]; then
        echo "Create AKS cluster $AKS_NAME"
        az aks create --resource-group $AZURE_RESOURCE_GROUP --name $AKS_NAME --node-count 1 --service-principal $AZURE_SP_ID --client-secret $AZURE_SP_KEY --generate-ssh-keys
    elif [ $result -eq 0 ]; then
        echo "Cluster $AKS_NAME in resource group $AZURE_RESOURCE_GROUP exists."
    else
        echo "Unknown error running az aks show"
        exit 1
    fi
else
    echo "Skip AKS create"
fi

echo "Get kubectl credentials"
az aks get-credentials --resource-group $AZURE_RESOURCE_GROUP --name $AKS_NAME
