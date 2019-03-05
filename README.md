| Branch | Status |
|--------|--------|
| master |[![CircleCI](https://circleci.com/gh/Azure/Kore.svg?style=svg&circle-token=1c70b5074bceb569aa5e4ac9a1b43836ffe25f54)](https://circleci.com/gh/Azure/Kore)|

# Kore -  Event driven autoscaler and scale to zero for Kubernetes

Kore allows for fine grained autoscaling (including to/from zero) for event driven Kubernetes workloads.
Kore serves as a Kubernetes Metrics Server and allows users to define autoscaling rules using a dedicated CRD.

Kore can run on both the cloud and the edge, integrates natively with Kubernetes components such as the Horizontal Pod Autoscaler, and has no external dependencies.

![k](https://user-images.githubusercontent.com/645740/51940231-46cf5380-23c6-11e9-9433-39cdd4055b4c.gif)

## Setup

### Prerequisites

1. A Kubernetes cluster [(instructions)](https://kubernetes.io/docs/tutorials/kubernetes-basics/).

    Make sure your Kubernetes cluster is RBAC enabled.
    For AKS cluster ensure that you download the AKS cluster credentials with the following CLI

  ```cli
    az aks get-credentials -n <cluster-name> -g <resource-group>
  ```

2. *Kubectl* has been installed and configured to work with your cluster [(instructions)](https://kubernetes.io/docs/tasks/tools/install-kubectl/).

3. `docker login -u "b514b60c-68cc-4f12-b361-3858878b2479" -p '4jX5vkPTSrUQ96UBbU/B7CQrBoJwT62WSs5WfZtFbB8=' projectkore.azurecr.io`

### Deploy Kore

Clone the repo:

```bash
git clone https://github.com/Azure/Kore.git
```

Deploy CRD to your cluster:

```bash
kubectl apply -f ./Kore/deploy
```

## Getting Started


## Development

### Prerequisites

1. The Go language environment [(instructions)](https://golang.org/doc/install).

    Make sure you've already configured your GOPATH and GOROOT environment variables.
2. Dep [(instructions)](https://github.com/golang/dep).

### Environment set up

First, clone the repo into your GOPATH:

```bash
cd $GOPATH/src
mkdir -p github.com/Azure/Kore
git clone https://github.com/Azure/Kore
```

Run dep:

```bash
cd $GOPATH/src/github.com/Azure/Kore
dep ensure
```

Run the code locally:

```bash
# bash
CONFIG=/path/to/.kube/config go run cmd/main.go

#Powershell
$Env:CONFIG=/path/to/.kube/config
go run cmd/main.go
```

### Create a functions project:

1. Create a standard functions project:

* [Using vscode](https://docs.microsoft.com/en-us/azure/azure-functions/functions-create-first-function-vs-code)
* [Using Visual Studio](https://docs.microsoft.com/en-us/azure/azure-functions/functions-create-your-first-function-visual-studio)
* [Using `func` cli](https://docs.microsoft.com/en-us/azure/azure-functions/functions-create-first-function-python)

2. Build a docker container for your functions:
<details>

Add a `.dockerignore`

```
local.settings.json
deploy.yaml
```

Add a `Dockerfile` depending on the language of your functions

**dotnet:**
```dockerfile
FROM microsoft/dotnet:2.1-sdk AS installer-env

COPY . /src/dotnet-function-app
RUN cd /src/dotnet-function-app && \
    mkdir -p /home/site/wwwroot && \
    dotnet publish *.csproj --output /home/site/wwwroot

FROM mcr.microsoft.com/azure-functions/dotnet:2.0
ENV AzureWebJobsScriptRoot=/home/site/wwwroot

COPY --from=installer-env ["/home/site/wwwroot", "/home/site/wwwroot"]
```

**javascript:**
```dockerfile
FROM mcr.microsoft.com/azure-functions/node:2.0

ENV AzureWebJobsScriptRoot=/home/site/wwwroot
COPY . /home/site/wwwroot
RUN cd /home/site/wwwroot && \
    npm install
```
**python:**
```dockerfile
FROM mcr.microsoft.com/azure-functions/python:2.0

COPY . /home/site/wwwroot

RUN cd /home/site/wwwroot && \
    pip install -r requirements.txt
```

Build your container
```bash
docker build -t {IMAGE_NAME} .
```

Push your container to a container registry
```bash
docker push {IMAGE_NAME}
```
</details>

3. Add your connection strings in `local.settings.json`

e.g:
```json
{
  "IsEncrypted": false,
  "Values": {
    ...
    "AzureWebJobsStorage": "DefaultEndpointsProtocol=https;AccountName={name};AccountKey=......",
    ...
  }
}
```

4. Download this build of `core-tools`:
   1. [windows](https://ahmelsayed.blob.core.windows.net/public/Azure.Functions.Cli.win-x86.2.4.9999.zip)
   2. [linux](https://ahmelsayed.blob.core.windows.net/public/Azure.Functions.Cli.linux-x64.2.4.9999.zip)
   3. [mac](https://ahmelsayed.blob.core.windows.net/public/Azure.Functions.Cli.osx-x64.2.4.9999.zip)

5. Run
```bash
func kdeploy --image-name {image_name_from_above}
```

6. Deploy to your k8s cluster
```bash
kubectl create -f deploy.yaml
```

# Contributing

This project welcomes contributions and suggestions.  Most contributions require you to agree to a
Contributor License Agreement (CLA) declaring that you have the right to, and actually do, grant us
the rights to use your contribution. For details, visit https://cla.microsoft.com.

When you submit a pull request, a CLA-bot will automatically determine whether you need to provide
a CLA and decorate the PR appropriately (e.g., label, comment). Simply follow the instructions
provided by the bot. You will only need to do this once across all repos using our CLA.

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/).
For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or
contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.
