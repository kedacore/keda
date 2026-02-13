<p align="center"><img src="images/logos/keda-word-colour.png" width="300"/></p>
<p style="font-size: 25px" align="center"><b>Kubernetes-based Event Driven Autoscaling</b></p>
<p style="font-size: 25px" align="center">
<a href="https://github.com/kedacore/keda/actions?query=workflow%3Amain-build"><img src="https://github.com/kedacore/keda/actions/workflows/main-build.yml/badge.svg" alt="main build"></a>
<a href="https://github.com/kedacore/keda/actions?query=workflow%3Anightly-e2e-test"><img src="https://github.com/kedacore/keda/actions/workflows/nightly-e2e.yml/badge.svg" alt="nightly e2e"></a>
<a href="https://bestpractices.coreinfrastructure.org/projects/3791"><img src="https://bestpractices.coreinfrastructure.org/projects/3791/badge"></a>
<a href="https://api.scorecard.dev/projects/github.com/kedacore/keda/badge"><img src="https://img.shields.io/ossf-scorecard/github.com/kedacore/keda?label=openssf%20scorecard&style=flat"></a>
<a href="https://artifacthub.io/packages/helm/kedacore/keda"><img src="https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/kedacore"></a>
<a href="https://app.fossa.com/projects/custom%2B162%2Fgithub.com%2Fkedacore%2Fkeda?ref=badge_shield" alt="FOSSA Status"><img src="https://app.fossa.com/api/projects/custom%2B162%2Fgithub.com%2Fkedacore%2Fkeda.svg?type=shield"/></a>
<a href="https://twitter.com/kedaorg"><img src="https://img.shields.io/twitter/follow/kedaorg?style=social" alt="Twitter"></a>
<a href="https://kubernetes.slack.com/archives/CKZJ36A5D"><img src="https://img.shields.io/badge/slack-keda-brightgreen.svg?style=social?logo=slack" alt="Slack"></a></p>

KEDA allows for fine-grained autoscaling (including to/from zero) for event driven Kubernetes workloads. KEDA serves
as a Kubernetes Metrics Server and allows users to define autoscaling rules using a dedicated Kubernetes custom
resource definition.

KEDA can run on both the cloud and the edge, integrates natively with Kubernetes components such as the Horizontal
Pod Autoscaler, and has no external dependencies.

We are a Cloud Native Computing Foundation (CNCF) graduated project.
<p align="center"><img src="https://raw.githubusercontent.com/kedacore/keda/main/images/logo-cncf.svg" height="75px"></p>

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of contents**

- [Getting started](#getting-started)
  - [Deploying KEDA](#deploying-keda)
- [Documentation](#documentation)
- [Community](#community)
- [Adopters - Become a listed KEDA user!](#adopters---become-a-listed-keda-user)
- [Governance & Policies](#governance--policies)
- [Support](#support)
- [Roadmap](#roadmap)
- [Releases](#releases)
- [Contributing](#contributing)
  - [Building & deploying locally](#building--deploying-locally)
  - [Testing strategy](#testing-strategy)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Getting started

* [QuickStart - RabbitMQ and Go](https://github.com/kedacore/sample-go-rabbitmq)
* [QuickStart - Azure Functions and Queues](https://github.com/kedacore/sample-hello-world-azure-functions)
* [QuickStart - Azure Functions and Kafka on Openshift 4](https://github.com/kedacore/sample-azure-functions-on-ocp4)
* [QuickStart - Azure Storage Queue with ScaledJob](https://github.com/kedacore/sample-go-storage-queue)

You can find several samples for various event sources [here](https://github.com/kedacore/samples).

### Deploying KEDA

There are many ways to [deploy KEDA including Helm, Operator Hub and YAML files](https://keda.sh/docs/latest/deploy/).

## Documentation

Interested to learn more? Head over to [keda.sh](https://keda.sh).

## Community

If interested in contributing or participating in the direction of KEDA, you can join our community meetings! Learn more about them on [our website](https://keda.sh/community/).

Just want to learn or chat about KEDA? Feel free to join the conversation in
**[#KEDA](https://kubernetes.slack.com/messages/CKZJ36A5D)** on the **[Kubernetes Slack](https://slack.k8s.io/)**!

## Adopters - Become a listed KEDA user!

We are always happy to [list users](https://keda.sh/community/#users) who run KEDA in production, learn more about it [here](https://github.com/kedacore/keda-docs#become-a-listed-keda-user).

## Governance & Policies

You can learn about the governance of KEDA [here](https://github.com/kedacore/governance).

## Support

Details on the KEDA support policy can found [here](https://keda.sh/support/).

## Roadmap

We use GitHub issues to build our backlog, a complete overview of all open items and our planning.

Learn more about our roadmap [here](ROADMAP.md).

## Releases

You can find the latest releases [here](https://github.com/kedacore/keda/releases).

## Contributing

You can find contributing guide [here](./CONTRIBUTING.md).

### Building & deploying locally
Learn how to build & deploy KEDA locally [here](./BUILD.md).

### Testing strategy
Learn more about our testing strategy [here](./TESTING.md).
