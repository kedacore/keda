# Build & Deploy KEDA

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Building](#building)
  - [Quick start with Visual Studio Code Dev Containers](#quick-start-with-visual-studio-code-dev-containers)
  - [Locally directly](#locally-directly)
- [Deploying](#deploying)
  - [Custom KEDA locally outside cluster](#custom-keda-locally-outside-cluster)
  - [Custom KEDA as an image](#custom-keda-as-an-image)
- [Debugging with VS Code](#debugging-with-vs-code)
  - [Operator](#operator)
  - [Metrics server](#metrics-server)
  - [Admission Webhooks](#admission-webhooks)
- [Miscellaneous](#miscellaneous)
  - [How to use devcontainers and a local Kubernetes cluster](#how-to-use-devcontainers-and-a-local-kubernetes-cluster)
  - [Setting log levels](#setting-log-levels)
  - [KEDA Operator and Admission webhooks logging](#keda-operator-and-admission-webhooks-logging)
  - [Metrics Server logging](#metrics-server-logging)
  - [CPU/Memory Profiling](#cpumemory-profiling)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Building

### Quick start with [Visual Studio Code Dev Containers](https://code.visualstudio.com/docs/remote/containers)

This helps you pull and build quickly - dev containers launch the project inside a container with all the tooling
required for a consistent and seamless developer experience.

This means you don't have to install and configure your dev environment as the container handles this for you.

To get started install [VSCode](https://code.visualstudio.com/) and the [Dev Containers extensions](
https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)

Clone the repo and launch code:

```bash
git clone git@github.com:kedacore/keda.git
cd keda
code .
```

Once VSCode launches run `CTRL+SHIFT+P -> Dev Containers: Reopen in container` and then use the integrated
terminal to run:

```bash
make build
```

> Note: The first time you run the container it will take some time to build and install the tooling. The image
> will be cached so this is only required the first time.

### Locally directly

This project is using [Operator SDK framework](https://github.com/operator-framework/operator-sdk), make sure you
have installed the right version. To check the current version used for KEDA check the `RELEASE_VERSION` in file
[github.com/test-tools/tools/Dockerfile](https://github.com/kedacore/test-tools/blob/main/tools/Dockerfile).

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
2. Scale in `keda-operator` Deployment
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
2. Build and publish images with your changes, `IMAGE_REPO` should point to your repository,
`IMAGE_REGISTRY` allows you to use registry of your choice eg. quay.io, default is `ghcr.io`
   ```bash
   IMAGE_REGISTRY=docker.io IMAGE_REPO=johndoe make publish
   ```
3. Deploy KEDA with your custom images.
   ```bash
   IMAGE_REGISTRY=docker.io IMAGE_REPO=johndoe make deploy
   ```
4. Once the KEDA pods are up, check the logs to verify everything running ok, eg:
    ```bash
    kubectl logs -l app=keda-operator -n keda -f
    kubectl logs -l app=keda-metrics-apiserver -n keda -f
    ```

## Debugging with VS Code

KEDA uses certificates to encrypt any HTTP communication. Inside the cluster, certificates are mounted from a secret but locally debugging that isn't possible, so the generation of those certificates is required (or KEDA won't start).

All components inspect the folder `/certs` for any certificates inside it. Argument `--cert-dir` can be used to specify another folder to be used as a source for certificates. You can generate the certificates (assuming the default path) using `openssl`:

```bash
mkdir -p /certs
openssl req -newkey rsa:2048 -subj '/CN=localhost' -addext "subjectAltName = DNS:localhost" -nodes -keyout /certs/tls.key -x509 -days 3650 -out /certs/tls.crt
cp /certs/tls.crt /certs/ca.crt
```

### Operator

Follow these instructions if you want to debug the KEDA operator using VS Code.

1. Create a `launch.json` file inside the `.vscode/` folder in the repo with the following configuration:
   ```json
   {
    "configurations": [
         {
            "name": "Launch operator",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}/cmd/operator/main.go",
            "env": {
                "WATCH_NAMESPACE": "",
                "KEDA_CLUSTER_OBJECT_NAMESPACE": "keda"
            }
        }
    ]
   }
   ```
   Refer to [this](https://code.visualstudio.com/docs/editor/debugging) for more information about debugging with VS Code.
2. Deploy CRDs and KEDA into `keda` namespace
   ```bash
   make deploy
   ```
3. Scale in `keda-operator` Deployment
   ```bash
   kubectl scale deployment/keda-operator --replicas=0 -n keda
   ```
4. Set breakpoints in the code as required.
5. Select `Run > Start Debugging` or press `F5` to start debugging.

### Metrics server

> **Note:** You will be able to manually query metrics to your local version of the KEDA Metrics server. You won't replace the KEDA Metrics server deployed on the Kubernetes cluster.

Follow these instructions if you want to debug the KEDA metrics server using VS Code.

1. Create a `launch.json` file inside the `.vscode/` folder in the repo with the following configuration:
   ```json
   {
    "configurations": [
        {
            "name": "Launch metrics-server",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${workspaceFolder}/cmd/adapter/main.go",
            "env": {
                "WATCH_NAMESPACE": "",
                "KEDA_CLUSTER_OBJECT_NAMESPACE": "keda"
            },
            "args": [
                "--authentication-kubeconfig=PATH_TO_YOUR_KUBECONFIG",
                "--authentication-skip-lookup",
                "--authorization-kubeconfig=PATH_TO_YOUR_KUBECONFIG",
                "--lister-kubeconfig=PATH_TO_YOUR_KUBECONFIG",
                "--secure-port=6443",
                "--v=5"
            ],
        }
    ]
   }
   ```
   Refer to [this](https://code.visualstudio.com/docs/editor/debugging) for more information about debugging with VS Code.
2. Deploy CRDs and KEDA into `keda` namespace
   ```bash
   make deploy
   ```
3. Set breakpoints in the code as required.
4. Select `Run > Start Debugging` or press `F5` to start debugging.

In order to perform queries against the metrics server, you need to use an authenticated user (with enough permissions) or give permissions over external metrics API to `system:anonymous`.

To grant access over external metrics API to `system:anonymous`, you only need to deploy this manifest (and remove it once you have finished):

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
   name: grant-anonymous-access-to-external-metrics
roleRef:
   apiGroup: rbac.authorization.k8s.io
   kind: ClusterRole
   name: keda-external-metrics-reader
subjects:
- kind: User
  name: system:anonymous
  namespace: default
```

**NOTE:** This granting allows to any unauthenticated user to do any operation in external metrics API, this is potentially unsecure, and we strongly discourage doing it on production clusters.

You can query list metrics executing `curl --insecure https://localhost:6443/apis/external.metrics.k8s.io/v1beta1/` or query a specific metrics value executing `curl --insecure https://localhost:6443/apis/external.metrics.k8s.io/v1beta1/namespaces/NAMESPACE/METRIC_NAME` ([similar to the process using `kubectl get --raw`](https://keda.sh/docs/latest/operate/metrics-server/#querying-metrics-exposed-by-keda-metrics-server) but using `curl --insecure https://localhost:6443` instead)

If you prefer to use an authenticated user, you can use a user or service account with access over external metrics API adding their token as authorization header in `curl`, ie: `curl -H "Authorization:Bearer TOKEN" --insecure https://localhost:6443/apis/external.metrics.k8s.io/v1beta1/`

### Admission Webhooks

Follow these instructions if you want to debug the KEDA webhook using VS Code.

1. Create a `launch.json` file inside the `.vscode/` folder in the repo with the following configuration:
   ```json
   {
    "configurations": [
         {
            "name": "Launch webhooks",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${workspaceFolder}/cmd/webhooks/main.go",
            "env": {
                "WATCH_NAMESPACE": "",
                "KEDA_CLUSTER_OBJECT_NAMESPACE": "keda"
            },
            "args": [
                "--zap-log-level=debug",
                "--zap-encoder=console",
                "--zap-time-encoding=rfc3339"
            ]
         },
    ]
   }
   ```
   Refer to [this](https://code.visualstudio.com/docs/editor/debugging) for more information about debugging with VS Code.
2. Expose your local instance to internet. If you can't expose it directly, you can use something like [localtunnel](https://theboroer.github.io/localtunnel-www/) using the command `lt --port 9443 --local-https --allow-invalid-cert` after installing the tool.

3. Update the `admissing_webhooks.yaml` in `config/webhooks`, replacing the section (but not committing this change)
   ```yaml
   webhooks:
   - admissionReviewVersions:
     - v1
     clientConfig:
       service:
         name: keda-admission-webhooks
         namespace: keda
         path: /validate-keda-sh-v1alpha1-scaledobject
   ```
   with the section:
   ```yaml
   webhooks:
   - admissionReviewVersions:
     - v1
     clientConfig:
       url: "https://${YOUR_URL}/validate-keda-sh-v1alpha1-scaledobject"
   ```
   **Note:** You need to define also the key `caBundle` with the CA bundle encoded in base64. This `caBundle` is the pem file from the CA used to sign the certificate. Remember to disable the `caBundle` inyection to avoid unintended rewrites of your `caBundle` (by KEDA operator or by any other 3rd party)


4. Deploy CRDs and KEDA into `keda` namespace
   ```bash
   make deploy
   ```
5. Set breakpoints in the code as required.
6. Select `Run > Start Debugging` or press `F5` to start debugging.

## Miscellaneous

### How to use devcontainers and a local Kubernetes cluster

When you are working with [devcontainers](https://code.visualstudio.com/docs/remote/containers), Visual Studio Code and all the related programs (like `kubectl` or debugging binary) run inside the container. This means that if you are using local clusters like Kind or minikube you won't be able to access them because localhost is the container itself and not the host machine where the cluster is running.

To solve this and be able to work with devcontainers and a local cluster, you should follow [this official documentation from Microsoft](https://github.com/Microsoft/vscode-dev-containers/tree/main/containers/kubernetes-helm).

### Setting log levels

You can change default log levels for both KEDA Operator and Metrics Server. KEDA Operator uses
 [Operator SDK logging](https://sdk.operatorframework.io/docs/building-operators/golang/references/logging/) mechanism.

### KEDA Operator and Admission webhooks logging

To change the logging level, find `--zap-log-level=` argument in Operator Deployment section in `config/manager/manager.yaml` file or in Webhooks Deployment section in `config/webhooks/webhooks.yaml` file, modify its value and redeploy.

Allowed values are `debug`, `info`, `error`, or an integer value greater than `0`, specified as string

Default value: `info`

To change the logging format, find `--zap-encoder=` argument in Operator Deployment section in `config/manager/manager.yaml` file or in Webhooks Deployment section in `config/webhooks/webhooks.yaml` file, modify its value and redeploy.

Allowed values are `json` and `console`

Default value: `console`

To change the logging time encoding, find `--zap-time-encoding=` argument in Operator Deployment section in `config/manager/manager.yaml` file or in Webhooks Deployment section in `config/webhooks/webhooks.yaml` file, modify its value and redeploy.

Allowed values are `epoch`, `millis`, `nano`, `iso8601`, `rfc3339` or `rfc3339nano`

Default value: `rfc3339`

> Note: Example of some of the logging time encoding values and the output:
```
epoch - 1.6533943565181081e+09
iso8601 - 2022-05-24T12:10:19.411Z
rfc3339 - 2022-05-24T12:07:40Z
rfc3339nano - 2022-05-24T12:10:19.411Z
```

### Metrics Server logging

Find `--v=0` argument in Operator Deployment section in `config/metrics-server/deployment.yaml` file, modify its value and redeploy.

Allowed values are `"0"` for info, `"4"` for debug, or an integer value greater than `0`, specified as string

Default value: `"0"`

### CPU/Memory Profiling

Refer to [Enabling Memory Profiling on KEDA v2](https://dev.to/tsuyoshiushio/enabling-memory-profiling-on-keda-v2-157g).
