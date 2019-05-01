| Branch | Status |
|--------|--------|
| master |[![CircleCI](https://circleci.com/gh/kedacore/keda.svg?style=svg&circle-token=1c70b5074bceb569aa5e4ac9a1b43836ffe25f54)](https://circleci.com/gh/kedacore/keda)|

# KEDA - Kubernetes-based Event Driven Autoscaling

KEDA allows for fine grained autoscaling (including to/from zero) for event driven Kubernetes workloads.  KEDA serves as a Kubernetes Metrics Server and allows users to define autoscaling rules using a dedicated Kubernetes custom resource definition.

KEDA can run on both the cloud and the edge, integrates natively with Kubernetes components such as the Horizontal Pod Autoscaler, and has no external dependencies.

## Getting started

* [QuickStart - RabbitMQ and Go](https://github.com/kedacore/sample-go-rabbitmq)
* [QuickStart - Azure Functions and Queues](https://github.com/kedacore/sample-hello-world-azure-functions)

## Setup

### Deploying with a Helm chart

#### Add Helm repo
```cli
helm repo add kedacore https://kedacore.azureedge.net/helm
```

#### Update Helm repo
```cli
helm repo update
```

#### Install keda-edge chart
```cli
helm install kedacore/keda-edge --devel --set logLevel=debug
```

### Deploying with the [Azure Functions Core Tools](https://github.com/Azure/azure-functions-core-tools)
```
func kubernetes install --namespace keda
```


## How KEDA works



* [Using Azure Functions with KEDA and Osiris](https://github.com/kedacore/keda/wiki/Using-Azure-Functions-with-Keda-and-Osiris)
* [`ScaledObject` spec](https://github.com/kedacore/keda/wiki/ScaledObject-spec)

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
