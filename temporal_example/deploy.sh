# Check we are logged into Azure
az account show 

source .env

# Get the Kubernetes client context for the AKS cluster
az aks get-credentials --resource-group $RG_NAME --name ${AKS_NAME}

# Log into the Azure Container Registry
TOKEN=$(az acr login --name ${ACR_NAME} --expose-token --output tsv --query accessToken)
docker login ${ACR_NAME}.azurecr.io --username 00000000-0000-0000-0000-000000000000 --password-stdin <<< $TOKEN
#az acr login -n 

# Build KEDA
cd .. && IMAGE_REGISTRY=${ACR_NAME}.azurecr.io IMAGE_REPO=${ACR_NAME} make publish

IMAGE_REGISTRY=${ACR_NAME}.azurecr.io IMAGE_REPO=${ACR_NAME} make deploy

kubectl apply -f temporal_example/temporal_scaledObject.yml

kubectl get pods --namespace keda

#seq 1000 |  parallel -n0 -j5 "curl http://${ENDPOINT}/async?name=v2"

seq 1000 |  parallel -n0 -j4 "curl http://${ENDPOINT}/delay"