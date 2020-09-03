<p style="font-size: 30px" align="center"><b>This branch contains unstable KEDA v2.0.0-alpha1, currently under development</b></p>

## How can I try KEDA v2 alpha version?
Make sure to remove previous KEDA (including CRD) from the cluster. Switch to the `v2` branch and deploy yaml files:
```bash
   git fetch --all
   git checkout v2
   make deploy
```


<p align="center"><img src="images/keda-logo-transparent.png" width="300"/></p>
<p style="font-size: 25px" align="center"><b>Kubernetes-based Event Driven Autoscaling</b></p>
<p style="font-size: 25px" align="center">
<a href="https://github.com/kedacore/keda/actions"><img src="https://github.com/kedacore/keda/workflows/master%20build/badge.svg" alt="master build"></a>
<a href="https://github.com/kedacore/keda/actions"><img src="https://github.com/kedacore/keda/workflows/nightly%20e2e%20test/badge.svg" alt="nightly e2e"></a>
<a href="https://bestpractices.coreinfrastructure.org/projects/3791"><img src="https://bestpractices.coreinfrastructure.org/projects/3791/badge"></a>
<a href="https://twitter.com/kedaorg"><img src="https://img.shields.io/twitter/follow/kedaorg?style=social" alt="Twitter"></a></p>

KEDA allows for fine-grained autoscaling (including to/from zero) for event driven Kubernetes workloads. KEDA serves
as a Kubernetes Metrics Server and allows users to define autoscaling rules using a dedicated Kubernetes custom
resource definition.

KEDA can run on both the cloud and the edge, integrates natively with Kubernetes components such as the Horizontal
Pod Autoscaler, and has no external dependencies.

We are a Cloud Native Computing Foundation (CNCF) sandbox project.
<img src="https://raw.githubusercontent.com/kedacore/keda/master/images/logo-cncf.svg" height="75px">

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of contents**

- [Getting started](#getting-started)
- [Deploying KEDA](#deploying-keda)
- [Documentation](#documentation)
- [FAQ](#faq)
- [Samples](#samples)
- [Releases](#releases)
- [Contributing](#contributing)
- [Community](#community)
- [Building: Quick start with Visual Studio Code Remote - Containers](#building-quick-start-with-visual-studio-code-remote---containers)
- [Building: Locally directly](#building-locally-directly)
- [Deploying: Custom KEDA locally outside cluster](#deploying-custom-keda-locally-outside-cluster)
- [Deploying: Custom KEDA as an image](#deploying-custom-keda-as-an-image)
- [Setting log levels](#setting-log-levels)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->


## Getting started

* [QuickStart - RabbitMQ and Go](https://github.com/kedacore/sample-go-rabbitmq)
* [QuickStart - Azure Functions and Queues](https://github.com/kedacore/sample-hello-world-azure-functions)
* [QuickStart - Azure Functions and Kafka on Openshift 4](https://github.com/kedacore/sample-azure-functions-on-ocp4)
* [QuickStart - Azure Storage Queue with ScaledJob](https://github.com/kedacore/sample-go-storage-queue)

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

* **Meeting time:** Bi-weekly Thurs 16:00 UTC (does follow US daylight savings).
([Subscribe to Google Agenda](https://calendar.google.com/calendar?cid=bjE0bjJtNWM0MHVmam1ob2ExcTgwdXVkOThAZ3JvdXAuY2FsZW5kYXIuZ29vZ2xlLmNvbQ) |
 [Convert to your timezone](https://www.thetimezoneconverter.com/?t=04%3A00%20pm&tz=UTC))
* **Zoom link:** [https://us02web.zoom.us/j/150360492?pwd=eUVtQzBPMzFoQUR2K1dqUWhENjJJdz09](https://us02web.zoom.us/j/150360492?pwd=eUVtQzBPMzFoQUR2K1dqUWhENjJJdz09)  (Password: keda)
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

If the build process fails due to some "checksum mismatch" errors, make sure that `GOPROXY` and `GOSUMDB`
 environment variables are set properly.
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

The Operator SDK framework allows you to run the operator/controller locally outside the cluster without
a need of building an image. This should help during development/debugging of KEDA Operator or Scalers.
> Note: This approach works only on Linux or macOS.


To have fully operational KEDA we need to deploy Metrics Server first.

1. Deploy CRDs and KEDA into `keda` namespace
   ```bash
   make deploy
   ```
2. Scale down `keda-operator` Deployment
   ```bash
   kubectl scale deployment/keda-operator --replicas=0 -n keda
   ```
3. Run the operator locally with the default Kubernetes config file present at `$HOME/.kube/config`
 and change the operator log level via `--zap-log-level=` if needed
   ```bash
   make run ARGS="--zap-log-level=debug"
   ```

## Deploying: Custom KEDA as an image

If you want to change KEDA's behaviour, or if you have created a new scaler (more docs on this to come) and you want
to deploy it as part of KEDA. Do the following:

1. Make your change in the code.
2. Build and publish on Docker Hub images with your changes, `IMAGE_REPO` should point to your repository
 (specifying `IMAGE_REGISTRY` as well allows you to use registry of your choice eg. quay.io).
   ```bash
   IMAGE_REPO=johndoe make publish
   ```
3. Deploy KEDA with your custom images.
   ```bash
   IMAGE_REPO=johndoe make deploy
   ```
4. Once the keda pods are up, check the logs to verify everything running ok, eg:
    ```bash
    kubectl get pods --no-headers -n keda | awk '{print $1}' | grep keda-operator | xargs kubectl -n keda logs -f

    kubectl get pods --no-headers -n keda | awk '{print $1}' | grep keda-metrics-apiserver | xargs kubectl -n keda logs -f
    ```

## Setting log levels
You can change default log levels for both KEDA Operator and Metrics Server. KEDA Operator uses
 [Operator SDK logging](https://sdk.operatorframework.io/docs/building-operators/golang/references/logging/) mechanism.

### KEDA Operator logging
To change the logging level, find `--zap-log-level=` argument in Operator Deployment section in `config/manager/manager.yaml` file,
 modify it's value and redeploy.

Allowed values are `debug`, `info`, `error`, or an integer value greater than `0`, specified as string

Default value: `info`

To change the logging format, find `--zap-encoder=` argument in Operator Deployment section in `config/manager/manager.yaml` file,
 modify it's value and redeploy.

Allowed values are `json` and `console`

Default value: `console`

### Metrics Server logging
Find `--v=0` argument in Operator Deployment section in `config/metrics-server/deployment.yaml` file, modify it's value and redeploy.

Allowed values are `"0"` for info, `"4"` for debug, or an integer value greater than `0`, specified as string

Default value: `"0"`
