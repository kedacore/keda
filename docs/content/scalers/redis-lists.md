+++
fragment = "content"
weight = 100
title = "Redis Lists"
background = "light"
+++

Scale applications based on Redis Lists.

* **Availability:** v1.0 and above
* **Maintainer:** Community

<!--more-->

### Trigger Specification

This specification describes the `redis` trigger that scales based on the length of a list in Redis.

```yaml
  triggers:
  - type: redis
    metadata:
      address: REDIS_HOST # Required host:port format
      password: REDIS_PASSWORD
      listName: mylist # Required
      listLength: "5" # Required
```

The `address` field in the spec holds the host and port of the redis server. This could be an external redis server or one running in the kubernetes cluster.

Provide the `password` field if the redis server requires a password. Both the hostname and password fields need to be set to the names of the environment variables in the target deployment that contain the host name and password respectively.

The `listName` parameter in the spec points to the Redis List that you want to monitor. The `listLength` parameter defines the average target value for the Horizontal Pod Autoscaler (HPA).

### Authentication Parameters

Not supported yet.

### Example

```yaml
apiVersion: keda.k8s.io/v1alpha1
kind: ScaledObject
metadata:
  name: redis-scaledobject
  namespace: keda-redis-test
  labels:
    deploymentName: keda-redis-node
spec:
  scaleTargetRef:
    deploymentName: keda-redis-node
  triggers:
  - type: redis
    metadata:
      address: REDIS_HOST # Required host:port format
      password: REDIS_PASSWORD
      listName: mylist # Required
      listLength: "5" # Required
```