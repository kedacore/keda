| Branch | Status |
|--------|--------|
| master |[![CircleCI](https://circleci.com/gh/kedacore/keda.svg?style=svg&circle-token=1c70b5074bceb569aa5e4ac9a1b43836ffe25f54)](https://circleci.com/gh/kedacore/keda)|

# KEDA - Kubernetes-based Event Driven Autoscaling

<p align="center"><img src="images/keda-wordmark.png" width="300"/></p>

KEDA allows for fine grained autoscaling (including to/from zero) for event driven Kubernetes workloads.  KEDA serves as a Kubernetes Metrics Server and allows users to define autoscaling rules using a dedicated Kubernetes custom resource definition.

KEDA can run on both the cloud and the edge, integrates natively with Kubernetes components such as the Horizontal Pod Autoscaler, and has no external dependencies.

---
<p align="center">
In partnership with
</p>
<p align="center">
<img src="images/partner-logos.png" width="500"/>
  </p>

---

## Getting started

* [QuickStart - RabbitMQ and Go](https://github.com/kedacore/sample-go-rabbitmq)
* [QuickStart - Azure Functions and Queues](https://github.com/kedacore/sample-hello-world-azure-functions)
* [QuickStart - Azure Functions and Kafka on Openshift 4](https://github.com/kedacore/keda/wiki/Using-Keda-and-Azure-Functions-on-Openshift-4)

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

KEDA performs two key roles within Kubernetes.  First, it acts as an agent to activate and deactivate a deployment to scale to and from zero on no events.  Second, it acts as a [Kubernetes metrics server](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/#support-for-custom-metrics) to expose rich event data like queue length or stream lag to the horizontal pod autoscaler to drive scale out.  It is up to the deployment to then consume the events directly from the source.  This preserves rich event integration and enables gestures like completing or abandoning queue messages to work out of the box.

![KEDA visualization](images/keda-arch.png)

### Event sources and scalers

KEDA has a number of "scalers" that can both detect if a deployment should be activated or deactivated, and feed custom metrics for a specific event source.  Today there is scalar support for:

* Kafka
* RabbitMQ
* Azure Storage Queues
* Azure Service Bus Queues

You can view other planned scalars [in our wiki and issue backlog](https://github.com/kedacore/keda/wiki/Scaler-prioritization).

#### ScaledObject custom resource definition

In order to sync a deployment with an event source, a `ScaledObject` custom resource needs to be deployed.  The `ScaledObject` contains information on the deployment to scale.  The `ScaledObject` will result in corresponding autoscaling resource to scale the deployment.  `ScaledObjects` contain information on the deployment to scale, metadata on the event source (e.g. connection string secret, queue name), polling interval, and cooldown period.

ScaledObject examples and schemas [can be found in our wiki](https://github.com/kedacore/keda/wiki/ScaledObject-spec).

### HTTP scaling integration

KEDA enables scaling based on event sources where the event resides somewhere to be pulled.  For events like HTTP where the event is pushed to the container, KEDA works side by side with HTTP scale-to-zero components like [Osiris](https://github.com/deislabs/osiris) or [Knative serving](https://github.com/knative/serving).  By pairing KEDA with an HTTP scale-to-zero component you can provide rich event scaling for both HTTP and non-HTTP.

### Azure Functions Integration

While KEDA can be used with any container or deployment, the Azure Functions tooling natively integrates with KEDA for a fully managed event-driven programming model.  With functions, developers only need to write the code that should run on an event, and not have to worry about the event consuming scaffolding.  [Azure Functions is open source](https://github.com/azure/azure-functions-host), and all of the existing tooling and developer experience works regardless of the hosting option.

```javascript
module.exports = async function (context, myQueueItem) {
    context.log('JavaScript queue trigger function processed work item', myQueueItem);
};
```

You can containerize and deploy an existing or new Azure Function using the [Azure Functions core tools](https://github.com/azure/azure-functions-core-tools)

```cli
func kubernetes deploy --name my-function --registry my-container-registry
```

[Using Azure Functions with KEDA and Osiris](https://github.com/kedacore/keda/wiki/Using-Azure-Functions-with-Keda-and-Osiris)

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
