# Build & Deploy KEDA

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Build & Deploy KEDA](#build--deploy-keda)
  - [Building](#building)
    - [Quick start with Visual Studio Code Remote - Containers](#quick-start-with-visual-studio-code-remote---containers)
    - [Locally directly](#locally-directly)
  - [Deploying](#deploying)
    - [Custom KEDA locally outside cluster](#custom-keda-locally-outside-cluster)
    - [Custom KEDA as an image](#custom-keda-as-an-image)
  - [Miscellaneous](#miscellaneous)
    - [Setting log levels](#setting-log-levels)
    - [KEDA Operator logging](#keda-operator-logging)
    - [Metrics Server logging](#metrics-server-logging)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Building

### Quick start with [Visual Studio Code Remote - Containers](https://code.visualstudio.com/docs/remote/containers)

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

### Locally directly

This project is using [Operator SDK framework](https://github.com/operator-framework/operator-sdk), make sure you
have installed the right version. To check the current version used for KEDA check the `RELEASE_VERSION` in file
[tools/build-tools.Dockerfile](https://github.com/kedacore/keda/blob/main/tools/build-tools.Dockerfile).

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

## Deploying

### Custom KEDA locally outside cluster

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

### Custom KEDA as an image

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
4. Once the KEDA pods are up, check the logs to verify everything running ok, eg:
    ```bash
    kubectl logs -l app=keda-operator -n keda -f
    kubectl logs -l app=keda-metrics-apiserver -n keda -f
    ```

## Miscellaneous

### Setting log levels

You can change default log levels for both KEDA Operator and Metrics Server. KEDA Operator uses
 [Operator SDK logging](https://sdk.operatorframework.io/docs/building-operators/golang/references/logging/) mechanism.

### KEDA Operator logging

To change the logging level, find `--zap-log-level=` argument in Operator Deployment section in `config/manager/manager.yaml` file,
 modify its value and redeploy.

Allowed values are `debug`, `info`, `error`, or an integer value greater than `0`, specified as string

Default value: `info`

To change the logging format, find `--zap-encoder=` argument in Operator Deployment section in `config/manager/manager.yaml` file,
 modify its value and redeploy.

Allowed values are `json` and `console`

Default value: `console`

### Metrics Server logging

Find `--v=0` argument in Operator Deployment section in `config/metrics-server/deployment.yaml` file, modify its value and redeploy.

Allowed values are `"0"` for info, `"4"` for debug, or an integer value greater than `0`, specified as string

Default value: `"0"`

### CPU/Memory Profiling

Refer to [Enabling Memory Profiling on KEDA v2](https://dev.to/tsuyoshiushio/enabling-memory-profiling-on-keda-v2-157g).
