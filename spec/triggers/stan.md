# How to test the Nats streaming scaler

## First start a nats streaming server with monitoring endpoint.


Get the chart from [here](https://github.com/balchua/gonuts/tree/master/natss-chart)

Execute the command:

`helm install --namespace stan -n stan . `

You should have the following resources in the `stan` namespace

```
kubectl -n stan get all
NAME                 READY   STATUS    RESTARTS   AGE
pod/stan-nats-ss-0   1/1     Running   11         37h

NAME                   TYPE       CLUSTER-IP      EXTERNAL-IP   PORT(S)                         AGE
service/stan-nats-ss   NodePort   10.152.183.67   <none>        4222:31592/TCP,8222:32484/TCP   37h

NAME                            READY   AGE
statefulset.apps/stan-nats-ss   1/1     37h

```

Take note of the service name `stan-nats-ss`, you will use this to populate the `ScaledObject`

You should also enable the monitoring endpoint of nats-streaming.

## Start a Nats Streaming publisher

See the example [publisher](https://github.com/balchua/gonuts/tree/master/pub) code.

## Start a Nats Streaming consumer

See the example [consumer](https://github.com/balchua/gonuts/tree/master/sub) code.


## Apply `stan_scaledobject.yaml`

Example:

```yaml
apiVersion: keda.k8s.io/v1alpha1
kind: ScaledObject
metadata:
  name: stan-scaledobject
  namespace: gonuts
  labels:
    deploymentName: gonuts-sub
spec:
  pollingInterval: 10   # Optional. Default: 30 seconds
  cooldownPeriod: 30   # Optional. Default: 300 seconds
  minReplicaCount: 0   # Optional. Default: 0
  maxReplicaCount: 30  # Optional. Default: 100  
  scaleTargetRef:
    deploymentName: gonuts-sub
  triggers:
  - type: stan
    metadata:
      natsServerMonitoringEndpoint: "stan-nats-ss.stan.svc.cluster.local:8222"
      queueGroup: "grp1"
      durableName: "ImDurable"
      subject: "Test"
      lagThreshold: "10"
```

Where:

* `natsServerMonitoringEndpoint` : Is the location of the Nats Streaming monitoring endpoint.  In this example it is the FQDN of nats streaming deployed.
* `queuGroup` : The queue group name of the subscribers.
* `durableName` :  Must identify the durability name used by the subscribers.
* `subject` : Sometimes called the channel name.
* `lagThreshold` : This value is used to tell the Horizontal Pod Autoscaler to use as TargetAverageValue.


Example [`examples/stan_scaledobject.yaml`](./../../examples/stan_scaledobject.yaml)