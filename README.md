<p align="center"><img src="images/keda-logo-transparent.png" width="300"/></p>
<p style="font-size: 25px" align="center"><b>Kubernetes-based Event Driven Autoscaling</b></p>
<p style="font-size: 25px" align="center">
<a href="https://github.com/kedacore/keda/actions"><img src="https://github.com/kedacore/keda/workflows/master%20build/badge.svg" alt="master build"></a>
<a href="https://github.com/kedacore/keda/actions"><img src="https://github.com/kedacore/keda/workflows/nightly%20e2e%20test/badge.svg" alt="nightly e2e"></a>
<a href="https://twitter.com/kedaorg"><img src="https://img.shields.io/twitter/follow/kedaorg?style=social" alt="Twitter"></a></p>

KEDA allows for fine grained autoscaling (including to/from zero) for event driven Kubernetes workloads.  KEDA serves as a Kubernetes Metrics Server and allows users to define autoscaling rules using a dedicated Kubernetes custom resource definition.

KEDA can run on both the cloud and the edge, integrates natively with Kubernetes components such as the Horizontal Pod Autoscaler, and has no external dependencies.

---
<p align="center">
In partnership with
</p>
<p align="center">
<img src="images/partner-logos.png" width="500"/>
  </p>

---

## Getting started

* [QuickStart - RabbitMQ and Go](https://github.com/kedacore/sample-go-rabbitmq)
* [QuickStart - Azure Functions and Queues](https://github.com/kedacore/sample-hello-world-azure-functions)
* [QuickStart - Azure Functions and Kafka on Openshift 4](https://github.com/kedacore/keda/wiki/Using-Keda-and-Azure-Functions-on-Openshift-4)

## Deploying KEDA

There are many ways to [deploy KEDA including Helm, YAML files, and the Azure Functions Core Tools](https://keda.sh/deploy/).

## Documentation

Interested to learn more? Head over to [keda.sh](https://keda.sh).

## FAQ

You can find a [FAQ here](https://keda.sh/faq/) with some common questions.

## Samples

You can find several samples for various event sources [here](https://github.com/kedacore/samples)

## Community

If interested in contributing or participating in the direction of KEDA, you can join our community meetings.

* **Meeting time:** Bi-weekly Thurs 18:00 UTC. ([Subscribe to Google Agenda](https://calendar.google.com/calendar?cid=bjE0bjJtNWM0MHVmam1ob2ExcTgwdXVkOThAZ3JvdXAuY2FsZW5kYXIuZ29vZ2xlLmNvbQ) | [Convert to your timezone](https://www.thetimezoneconverter.com/?t=18:00&tz=UTC))*
* **Zoom link:** [https://zoom.us/j/150360492 ](https://zoom.us/j/150360492 )
* **Meeting agenda:** [https://hackmd.io/s/r127ErYiN](https://hackmd.io/s/r127ErYiN)

Just want to learn or chat about KEDA? Feel free to join the conversation in **[#KEDA](kubernetes.slack.com/messages/CKZJ36A5D)** on the **[Kubernetes Slack](https://slack.k8s.io/)**!

## Building: Quick start with [Visual Studio Code Remote - Containers](https://code.visualstudio.com/docs/remote/containers)

This helps you pull and build quickly - dev containers launch the project inside a container with all the tooling required for a consistent and seamless developer experience. 

This means you don't have to install and configure your dev environment as the container handles this for you.

To get started install [VSCode](https://code.visualstudio.com/) and the [Remote Containers extensions](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)

Clone the repo and launch code:

```bash
git clone git@github.com:kedacore/keda.git
cd keda
code .
```

Once VSCode launches run `CTRL+SHIFT+P -> Remote-Containers: Reopen in container` and then use the integrated terminal to run:

```bash 
make build
```

> Note: The first time you run the container it will take some time to build and install the tooling. The image will be cached so this is only required the first time.

## Building: Locally directly
This project is using [Operator SDK framework](https://github.com/operator-framework/operator-sdk), make sure you have installed the right version. To check the current version used for KEDA check the `RELEASE_VERSION` in file [tools/build-tools.Dockerfile](https://github.com/kedacore/keda/blob/master/tools/build-tools.Dockerfile).

```bash
git clone git@github.com:kedacore/keda.git
cd keda
make build
```

## Deploying Custom KEDA

If you want to change KEDA's behaviour, or if you have created a new scaler (more docs on this to come) and you want to deploy it as part of KEDA. Do the following:
1. Make your change in the code.
2. In terminal, create an environment variable `IMAGE_TAG` and assign it a value for your preference, this tag will be used when creating the operator image that will run KEDA.
***Note***: make sure it doesn't clash with the official tags of KEDA containers in DockerHub.
3. Still in terminal, run `make build` at the root of the source code. This will also build the docker image for the KEDA operator that you can deploy to your local cluster. It will use the tag you used in step 2.
4. Still in terminal, navigate to the `chart/keda` folder, and run the following command (don't forget to replace the placeholder text in the command) `helm install . --set image.repository=[tag used in step 2],image.pullPolicy=IfNotPresent`.

In the last step we are using the image we just create by running step 3. Notice that we are also overriding the image PullPolice to `IfNotPresent` since this is a local cluster, this is important to do, otherwise, Kubernetes will try to pull the image from Docker Hub from the internet and will complain about not finidng it.

# Contributing

This project welcomes contributions and suggestions.  Most contributions require you to agree to a
Contributor License Agreement (CLA) declaring that you have the right to, and actually do, grant us
the rights to use your contribution. For details, visit https://cla.microsoft.com.

When you submit a pull request, a CLA-bot will automatically determine whether you need to provide
a CLA and decorate the PR appropriately (e.g., label, comment). Simply follow the instructions
provided by the bot. You will only need to do this once across all repos using our CLA.

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/).
For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or
contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.
