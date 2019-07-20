# ScaledObject specification

[`types.go`](./../pkg/apis/keda/v1alpha1/types.go)

```yaml
apiVersion: keda.k8s.io/v1alpha1
kind: ScaledObject
metadata:
  name: {scaled-object-name}
  labels:
    deploymentName: {deployment-name} # must be in the same namespace as the ScaledObject
spec:
  scaleTargetRef:
    deploymentName: {deployment-name} # must be in the same namespace as the ScaledObject
    containerName: azure-functions-container  #Optional. Default: deployment.spec.template.spec.containers[0]
  pollingInterval: 30  # Optional. Default: 30 seconds
  cooldownPeriod:  300 # Optional. Default: 300 seconds
  minReplicaCount: 0   # Optional. Default: 0
  maxReplicaCount: 100 # Optional. Default: 100
  triggers:
  # {list of triggers to activate the deployment}
```

## Details
```yaml
  scaleTargetRef:
    deploymentName: {deployment-name} # must be in the same namespace
    containerName: azure-functions-container  #Optional. Default: deployment.spec.template.spec.containers[0]
```

The name of the deployment this scaledObject is for. This is the deployment keda will scale up and setup an HPA for based on the triggers defined in `triggers:`. Make sure to include the deployment name in the label as well, otherwise the metrics provider will not be able to query the metrics for the scaled object and 1-n scale will be broken.

**Assumptions:** `deploymentName` is in the same namespace as the scaledObject

---

```yaml
  pollingInterval: 30  # Optional. Default: 30 seconds
```

This is the interval to check each trigger on. By default Keda will check each trigger source on every ScaledObject every 30 seconds.

**Example:** in a queue scenario, keda will check the queueLength every `pollingInterval`, and scale the deployment up or down accordingly.

---

```yaml
  cooldownPeriod:  300 # Optional. Default: 300 seconds
```

The period to wait after the last trigger reported active before scaling the deployment back to 0. By default it's 5 minutes (300 seconds)

**Example:** wait 5 minutes after the last time keda checked the queue and it was empty. (this is obviously dependent on `pollingInterval`)

```yaml
  minReplicaCount: 0   # Optional. Default: 0
```

Minimum number of replicas keda will scale the deployment down to. By default it's scale to zero, but you can use it with some other value as well. Keda will not enforce that value, meaning you can manually scale the deployment to 0 and keda will not scale it back up. However, when keda itself is scaling the deployment it will respect the value set there.

---

```yaml
  maxReplicaCount: 100 # Optional. Default: 100
```

This setting is passed to the HPA definition that keda will create for a given deployment.