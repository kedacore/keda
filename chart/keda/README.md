<p align="center"><img src="https://raw.githubusercontent.com/kedacore/keda/master/images/keda-logo-transparent.png" width="300"/></p>
<p style="font-size: 25px" align="center"><b>Kubernetes-based Event Driven Autoscaling</b></p>

KEDA allows for fine grained autoscaling (including to/from zero) for event driven Kubernetes workloads.  KEDA serves as a Kubernetes Metrics Server and allows users to define autoscaling rules using a dedicated Kubernetes custom resource definition.

KEDA can run on both the cloud and the edge, integrates natively with Kubernetes components such as the Horizontal Pod Autoscaler, and has no external dependencies.

---
<p align="center">
In partnership with
</p>
<p align="center">
<img src="https://raw.githubusercontent.com/kedacore/keda/master/images/partner-logos.png" width="500"/>
  </p>

---

## TL;DR

```console
$ helm repo add kedacore https://kedacore.azureedge.net/helm
$ helm install kedacore/keda-edge --namespace keda --name keda
```

## Introduction

This chart bootstraps KEDA deployment on a Kubernetes cluster using the Helm package manager.

## Prerequisites

None.

## Installing the Chart

To install the chart with the release name `keda`:

```cli
$ helm repo add kedacore https://kedacore.azureedge.net/helm
$ helm repo update
$ helm install kedacore/keda-edge --devel --set logLevel=debug --namespace keda --name keda
```

### Deploying with the [Azure Functions Core Tools](https://github.com/Azure/azure-functions-core-tools)
```
func kubernetes install --namespace keda
```

## Configuration

| Parameter                         | Description                         | Default              |
|:----------------------------------|:------------------------------------|:---------------------|
| `image.repository`                | Repository which provides the image | `kedacore/keda`      |
| `image.tag`                       | Tag of image to use | `lastest`            |
| `image.pullPolicy`                | Policy to pull image | `Always`            |
| `replicaCount`                    | Amount of replicas to run | `1`            |
| `customResourceDefinition.create` | Indication to whether or not to create the custom resource definition | `true`            |
| `rbac.create`                     | Indication to whether or not to use role-based access control | `true`            |
| `serviceAccount.create`           | Indication to whether or not to a serivce account should be used | `true`            |
| `serviceAccount.name`             | Name of the service account to use | ``            |
| `logLevel`                        | Granularity of KEDA logs to use which includes scale controller & metric adapter | `info`          |
| `glogLevel`                       | Granularity of logs to use for metric adapter which is beyond KEDA scope | `2`            |

## Uninstalling the Chart

To uninstall/delete the `keda` deployment:

```console
$ helm delete keda
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## How KEDA works

KEDA performs two key roles within Kubernetes.  First, it acts as an agent to activate and deactivate a deployment to scale to and from zero on no events.  Second, it acts as a [Kubernetes metrics server](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/#support-for-custom-metrics) to expose rich event data like queue length or stream lag to the horizontal pod autoscaler to drive scale out.  It is up to the deployment to then consume the events directly from the source.  This preserves rich event integration and enables gestures like completing or abandoning queue messages to work out of the box.

<p align="center"><img src="https://raw.githubusercontent.com/kedacore/keda/master/images/keda-arch.png" width="550"/></p>

### Event sources and scalers

KEDA has a number of "scalers" that can both detect if a deployment should be activated or deactivated, and feed custom metrics for a specific event source.  Today there is scaler support for:

* Kafka
* RabbitMQ
* Azure Storage Queues
* Azure Service Bus Queues and Topics

You can view other planned scalers [in our wiki and issue backlog](https://github.com/kedacore/keda/wiki/Scaler-prioritization).

#### ScaledObject custom resource definition

In order to sync a deployment with an event source, a `ScaledObject` custom resource needs to be deployed.  The `ScaledObject` contains information on the deployment to scale.  The `ScaledObject` will result in corresponding autoscaling resource to scale the deployment.  `ScaledObjects` contain information on the deployment to scale, metadata on the event source (e.g. connection string secret, queue name), polling interval, and cooldown period.

ScaledObject examples and schemas [can be found in our wiki](https://github.com/kedacore/keda/wiki/ScaledObject-spec).

### HTTP scaling integration

KEDA enables scaling based on event sources where the event resides somewhere to be pulled.  For events like HTTP where the event is pushed to the container, KEDA works side by side with HTTP scale-to-zero components like [Osiris](https://github.com/deislabs/osiris) or [Knative serving](https://github.com/knative/serving).  By pairing KEDA with an HTTP scale-to-zero component you can provide rich event scaling for both HTTP and non-HTTP.

### Azure Functions Integration

While KEDA can be used with any container or deployment, the Azure Functions tooling natively integrates with KEDA for a seamless developer experience and event-driven programming model.  With functions, developers only need to write the code that should run on an event, and not have to worry about the event consuming scaffolding.  [Azure Functions is open source](https://github.com/azure/azure-functions-host), and all of the existing tooling and developer experience works regardless of the hosting option.

You can containerize and deploy an existing or new Azure Function using the [Azure Functions core tools](https://github.com/azure/azure-functions-core-tools)

```cli
func kubernetes deploy --name my-function --registry my-container-registry
```

<p><img src="https://raw.githubusercontent.com/kedacore/keda/master/images//kedascale.gif" width="650"/></p>

[Using Azure Functions with KEDA and Osiris](https://github.com/kedacore/keda/wiki/Using-Azure-Functions-with-Keda-and-Osiris)