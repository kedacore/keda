<p align="center"><img src="images/keda-logo-transparent.png" width="300"/></p>
<p style="font-size: 25px" align="center"><b>Kubernetes-based Event Driven Autoscaling</b></p>
<p style="font-size: 25px" align="center">
<a href="https://github.com/kedacore/keda/actions"><img src="https://github.com/kedacore/keda/workflows/master%20build/badge.svg" alt="master build"></a>
<a href="https://github.com/kedacore/keda/actions"><img src="https://github.com/kedacore/keda/workflows/nightly%20e2e%20test/badge.svg" alt="nightly e2e"></a>
<a href="https://bestpractices.coreinfrastructure.org/projects/3791"><img src="https://bestpractices.coreinfrastructure.org/projects/3791/badge"></a>
<a href="https://twitter.com/kedaorg"><img src="https://img.shields.io/twitter/follow/kedaorg?style=social" alt="Twitter"></a></p>

KEDA allows for fine grained autoscaling (including to/from zero) for event driven Kubernetes workloads. KEDA serves 
as a Kubernetes Metrics Server and allows users to define autoscaling rules using a dedicated Kubernetes custom 
resource definition.

KEDA can run on both the cloud and the edge, integrates natively with Kubernetes components such as the Horizontal 
Pod Autoscaler, and has no external dependencies.

We are a Cloud Native Computing Foundation (CNCF) sandbox project.
![CNCF Logo](./images/logo-cncf.svg)
  
## Getting started

* [QuickStart - RabbitMQ and Go](https://github.com/kedacore/sample-go-rabbitmq)
* [QuickStart - Azure Functions and Queues](https://github.com/kedacore/sample-hello-world-azure-functions)
* [QuickStart - Azure Functions and Kafka on Openshift 4](https://github.com/kedacore/sample-azure-functions-on-ocp4)

## Deploying KEDA

There are many ways to [deploy KEDA including Helm, Operator Hub and YAML files](https://keda.sh/docs/latest/deploy/).

## Documentation

Interested to learn more? Head over to [keda.sh](https://keda.sh).

## FAQ

You can find a [FAQ here](https://keda.sh/docs/latest/faq/) with some common questions.

## Samples

You can find several samples for various event sources [here](https://github.com/kedacore/samples).

## Releases

You can find the latest releases [here](https://github.com/kedacore/keda/releases)

## Contributing

You can find Contributing guide [here](./CONTRIBUTING.md)

## Community

If interested in contributing or participating in the direction of KEDA, you can join our community meetings.

* **Meeting time:** Bi-weekly Thurs 17:00 UTC (does follow US daylight savings). ([Subscribe to Google Agenda](https://calendar.google.com/calendar?cid=bjE0bjJtNWM0MHVmam1ob2ExcTgwdXVkOThAZ3JvdXAuY2FsZW5kYXIuZ29vZ2xlLmNvbQ) | [Convert to your timezone](https://www.thetimezoneconverter.com/?t=10%3A00%20am&tz=Seattle&))
* **Zoom link:** [https://zoom.us/j/150360492 ](https://zoom.us/j/150360492 )
* **Meeting agenda:** [https://hackmd.io/s/r127ErYiN](https://hackmd.io/s/r127ErYiN)

Just want to learn or chat about KEDA? Feel free to join the conversation in 
**[#KEDA](https://kubernetes.slack.com/messages/CKZJ36A5D)** on the **[Kubernetes Slack](https://slack.k8s.io/)**!

## Building: Quick start with [Visual Studio Code Remote - Containers](https://code.visualstudio.com/docs/remote/containers)

This helps you pull and build quickly - dev containers launch the project inside a container with all the tooling 
required for a consistent and seamless developer experience. 

This means you don't have to install and configure your dev environment as the container handles this for you.

To get started install [VSCode](https://code.visualstudio.com/) and the [Remote Containers extensions](
https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)

Clone the repo and launch code:

```bash
git clone git@github.com:kedacore/keda.git
cd keda
code .
```

Once VSCode launches run `CTRL+SHIFT+P -> Remote-Containers: Reopen in container` and then use the integrated
terminal to run:

```bash 
make build
```

> Note: The first time you run the container it will take some time to build and install the tooling. The image 
> will be cached so this is only required the first time.

## Building: Locally directly
This project is using [Operator SDK framework](https://github.com/operator-framework/operator-sdk), make sure you
have installed the right version. To check the current version used for KEDA check the `RELEASE_VERSION` in file
[tools/build-tools.Dockerfile](https://github.com/kedacore/keda/blob/master/tools/build-tools.Dockerfile).

```bash
git clone git@github.com:kedacore/keda.git
cd keda
make build
```

If the build process fails due to some "checksum mismatch" errors, make sure that `GOPROXY` and `GOSUMDB` environment variables are set properly.
With Go installation on Fedora, for example, it could happen they are wrong.

```bash
go env GOPROXY GOSUMDB
direct
off
```

If not set properly you can just run.

```bash
go env -w GOPROXY=https://proxy.golang.org,direct GOSUMDB=sum.golang.org
```

## Deploying: Custom KEDA locally outside cluster
The Operator SDK framework allows you to [run the operator/controller locally](https://github.com/operator-framework/operator-sdk/blob/master/doc/user-guide.md#2-run-locally-outside-the-cluster)
outside the cluster without a need of building an image. This should help during development/debugging of KEDA Operator or Scalers. 
> Note: This approach works only on Linux or macOS. 

To be KEDA to be fully operational we need to deploy Metrics Server first.

1. Deploy CRDs and KEDA into `keda` namespace
   ```bash
   kubectl apply -f deploy/crds/keda.k8s.io_scaledobjects_crd.yaml
   kubectl apply -f deploy/crds/keda.k8s.io_triggerauthentications_crd.yaml
   kubectl apply -f deploy/
   ```
2. Scale down `keda-operator` Deployment
   ```bash
   kubectl scale deployment/keda-operator --replicas=0 -n keda
   ```
3. Run the operator locally with the default Kubernetes config file present at `$HOME/.kube/config` and change the operator log level via `--zap-level=` if needed
   ```bash
   operator-sdk run --local --namespace="" --operator-flags="--zap-level=info"
   ``` 
   > Note: On older operator-sdk versions you need to use command `up` instead of `run`.

   > Note: Please run `operator-sdk -h` to see all possible commands and options (eg. for debugging: `--enable-delve`)

## Deploying: Custom KEDA as an image

If you want to change KEDA's behaviour, or if you have created a new scaler (more docs on this to come) and you want 
to deploy it as part of KEDA. Do the following:

1. Make your change in the code.
2. In terminal, create an environment variable `VERSION` and assign it a value for your preference, this tag will 
    be used when creating the operator image that will run KEDA.
    ***Note***: make sure it doesn't clash with the official tags of KEDA containers in DockerHub.
3. Still in terminal, run `make build` at the root of the source code. This will also build the docker image for 
    the KEDA operator that you can deploy to your local cluster. This should build 2 docker images: `kedacore/keda` 
    and `kedacore/keda-metrics-adapter` tagged with the tag you set in step 2
4. If you haven't downloaded them before, clone the charts repository: `git clone git@github.com:kedacore/charts.git` 
5. Still in terminal, navigate to the `charts/keda` folder (downloaded in step 4), and run the following command 
    (don't forget to replace the placeholder text in the command):
    ```bash
    helm install . --set image.keda=kedacore/keda:$VERSION,image.metricsAdapter=kedacore/keda-metrics-adapter:$VERSION,image.pullPolicy=IfNotPresent
    ```
    This will use the images built at step 3. Notice the need to override the image pullPolicy to `IfNotPresent` in 
    order to use the locally built images and not try to pull the images from remote repo on Docker Hub (and complain 
    about not finding them).
6. Once the keda pods are up, check the logs to verify everything running ok, eg: 
    ```bash
    kubectl get pods --no-headers -n keda | awk '{print $1}' | grep keda-operator | xargs kubectl -n keda logs -f

    kubectl get pods --no-headers -n keda | awk '{print $1}' | grep keda-metrics-apiserver | xargs kubectl -n keda logs -f
    ```

## Setting log levels
You can change default log levels for both KEDA Operator and Metrics Server. KEDA Operator uses [Operator SDK logging](https://github.com/operator-framework/operator-sdk/blob/master/doc/user/logging.md) mechanism.

### KEDA Operator logging
To change the logging level, find `--zap-level=` argument in Operator Deployment section in `deploy/12-operator.yaml` file, modify it's value and redeploy.

Allowed values are `debug`, `info`, `error`, or an integer value greater than `0`, specified as string

Default value: `info`

To change the logging time format, find `--zap-time-encoding=` argument in Operator Deployment section in `deploy/12-operator.yaml` file, modify it's value and redeploy.

Allowed values are `epoch`, `millis`, `nano`, or `iso8601`

### Metrics Server logging
Find `--v=0` argument in Operator Deployment section in `deploy/22-metrics-deployment.yaml` file, modify it's value and redeploy.

Allowed values are `"0"` for info, `"4"` for debug, or an integer value greater than `0`, specified as string

Default value: `"0"`
