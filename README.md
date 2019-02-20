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

### Deploy Kore

Clone the repo:

```
git clone https://github.com/Azure/Kore.git
```

Deploy:

```
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

```
cd $GOPATH/src
mkdir -p github.com/Azure/Kore
git clone https://github.com/Azure/Kore
```

Run dep:

```
cd $GOPATH/src/github.com/Azure/Kore
dep ensure
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
