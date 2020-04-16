# Creating a new scaler

## Developing a scaler

In order to develop a scaler, a developer should do the following:
1. Download KEDA's code
2. Define the main pieces of data that you expect the user to supply so the scaler runs properly. For example, if your scaler needs to connect to an external source based on a connection string, you expect the user to supply this connection string in the configuration within the ScaledObject under `trigger`. This data will be passed to your constructing function as map[string]string.
2. Create the new scaler struct under the `pkg/scalers` folder.
3. Implement the methods defined in the [scaler interface](#scaler-interface) section.
4. Create a constructor according to [this](#constructor).
5. Change the `getScaler` function in `pkg/scaling/scale_handler.go` by adding another switch case that matches your scaler.
6. Run `make build` from the root of KEDA and your scaler is ready.

If you want to deploy locally
1. Run `export VERSION=local`
2. Open the terminal and go to the root of the source code
3. Run `make build`
5. If you haven't done it yet clone the charts repository: `git  clone git@github.com:kedacore/charts.git`
6. In the terminal, navigate to the `chart/keda` folder (the charts downloaded in step 3), and run the following command (don't forget to replace the placeholder text in the command) `helm install . --set image.keda=kedacore/keda:[tag used in step 1],image.pullPolicy=IfNotPresent`.

The last step assumes that you have `helm` already installed in the cluster. In this step we install the helm chart, and we substitute the image with the image we built in step 1. Notice that we are also overriding the image PullPolice to `IfNotPresent` since this is a local cluster.

## Scaler interface

The scalers in KEDA are implementations of a KEDA `Scaler` Go interface declared in `pkg/scalers/scaler.go`.  This documentation describes how scalers works and is targeted towards contributors and maintainers.

### GetMetrics

This is the key function of a scaler; it returns a value that represents a current state of an external metric (e.g. length of a queue). The return type is an `ExternalMetricValue` struct which has the following fields:
- `MetricName`: this is the name of the metric that we are returning.
- `Timestamp`: indicates the time at which the metrics were produced.
- `WindowSeconds`: //TODO
- `Value`: A numerical value that represents the state of the metric. It could be the length of a queue, or it can be the amount of lag in a stream, but it can also be a simple representation of the state.

Kubernetes HPA (Horizontal Pod Autoscaler) will poll `GetMetrics` reulgarly through KEDA's metric server (as long as there is at least one pod), and compare the returned value to a configured value in the ScaledObject configuration. Kubernetes will use the following formula to decide whether to scale the pods up and down:

`desiredReplicas = ceil[currentReplicas * ( currentMetricValue / desiredMetricValue )]`.

For more details check [Kubernetes HPA documentation](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/).

### GetMetricSpecForScaling

KEDA works in conjunction with Kubernetes Horizontal Pod Autoscaler (HPA). When KEDA notices a new ScaledObject, it creates an HPA object that has basic information about the metric it needs to poll and scale the pods accordingly. To create this HPA object, KEDA invokes `GetMetricSpecForScaling`.

The return type of this function is `MetricSpec`, but in KEDA's case we will mostly write External metrics. So the property that should be filled is `ExternalMetricSource`, where the:
- `MetricName`: the name of our metric we are returning in this scaler
- `MetricSelector`: //TODO
- `TargetValue`: is the value of the metric we want to reach at all times at all costs. As long as the current metric doesn't match TargetValue, HPA will increase the number of the pods until it reaches the maximum number of pods allowed to scale to.
- `TargetAverageValue`: the value of the metric for which we require one pod to handle. e.g. if we are have a scaler based on the length of a message queue, and we specificy 10 for `TargetAverageValue`, we are saying that each pod will handle 10 messages. So if the length of the queue becomes 30, we expect that we have 3 pods in our cluster. (`TargetAveryage` and `TargetValue` are mutually exclusive)

### IsActive

For some reason, the scaler might need to declare itself as in-active, and the way it can do this is through implementing the function `IsActive`.

KEDA polls ScaledObject object according to the `pollingInterval` confiugred in the ScaledObject; it checks the last time it was polled, it checks if the number of replicas is greater than 0, and if the scaler itself is active. So if the scaler returns false for `IsActive`, and if current number of replicas is greater than 0, and there is no configured minimum pods, then KEDA scales down to 0.

### Close
After each poll on the scaler to retrieve the metrics, KEDA calls this function for each scaler to give the scaler the opportunity to close any resources, like http clients for example.

### Constructor
What is missing from the `scaler` interface is a function that constructs the scaler itself. Up until the moment of writing this document, KEDA does not have a dynamic way to load scalers (at least not officially)[***]; instead scalers are part of KEDA's code-base, and they are shipped with KEDA's binary.

Thus, each scaler should have a constructing function, KEDA will [explicitly invoke](https://github.com/kedacore/keda/blob/4d0cf5ef09ef348cf3a158634910f00741ae5258/pkg/handler/scale_handler.go#L565) the construction function based on the `trigger` property configured in the ScaledObject.

The constructor should have the following parameters:

- `resolvedEnv`: of type `map[string]string`. This is a map of all the environment variables that are exist for the target Deploymnet.
- `metadata`: of type `map[string]string`. This is a map for all the `trigger` attributes of the ScaledObject.


## Lifecycle of a scaler

The scaler is created and closed everytime KEDA or HPA wants to call `GetMetrics`, and everytime a new ScaledObject is created or updated that has a trigger for that scaler. Thus, a developer of a scaler should not assume that the scaler will maintain any state between these calls.
