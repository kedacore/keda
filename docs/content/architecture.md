+++
title = "Architecture"
date = "2017-10-05"
fragment = "content"
weight = 100
+++

KEDA performs two key roles within Kubernetes.  First, it acts as an agent to activate and deactivate a deployment to scale to and from zero on no events.  Second, it acts as a [Kubernetes metrics server](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/#support-for-custom-metrics) to expose rich event data like queue length or stream lag to the horizontal pod autoscaler to drive scale out.  It is up to the deployment to then consume the events directly from the source.  This preserves rich event integration and enables gestures like completing or abandoning queue messages to work out of the box.

<p align="center"><img src="./../images/keda-arch.png" width="550"/></p>

## Event sources and scalers

KEDA has a number of "scalers" that can both detect if a deployment should be activated or deactivated, and feed custom metrics for a specific event source.  Today there is scaler support for:

* AWS CloudWatch
* AWS Simple Queue Service
* Azure Event Hub†
* Azure Service Bus Queues and Topics
* Azure Storage Queues
* GCP PubSub
* Kafka
* Liiklus
* Nats Streaming
* Prometheus
* RabbitMQ
* Redis Lists

You can view other planned scalers [in our wiki and issue backlog](https://github.com/kedacore/keda/wiki/Scaler-prioritization).

_†: As of now, the Event Hub scaler only supports reading from Blob Storage, as well as scaling only Event Hub applications written in C#, Python or created with Azure Functions._