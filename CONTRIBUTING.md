# Contributing to KEDA

Thanks for helping make KEDA better 😍.

There are many areas we can use contributions - ranging from code, documentation, feature proposals, issue triage, samples, and content creation.  

## Getting Help

If you have a question about KEDA or how best to contribute, the [#KEDA](https://kubernetes.slack.com/archives/CKZJ36A5D) channel on the Kubernetes slack channel ([get an invite if you don't have one already](https://slack.k8s.io/)) is a good place to start.  We also have regular [community stand-ups](https://github.com/kedacore/keda#community-standup) to track ongoing work and discuss areas of contribution.  For any issues with the product you can [create an issue](https://github.com/kedacore/keda/issues/new) in this repo.

## Contributing Scalers

One of the easiest ways to contribute is adding scalers.  Scalers are the logic on when to activate a container (scaling from zero to one) and also how to serve metrics for an event source.  You can view [the code for existing scalers here](https://github.com/kedacore/keda/tree/master/pkg/scalers).  When writing a scaler, please consider:

1. Is this an event source that many others will access from Kubernetes? If not, potentially consider [creating an external scaler](https://github.com/kedacore/keda/blob/master/pkg/scalers/externalscaler/externalscaler.proto).
1. Include tests
1. Include documentation and examples for [keda.sh](https://keda.sh) via [https://github.com/kedacore/keda-docs](https://github.com/kedacore/keda-docs)

Information on how scalers work can be found [here](https://keda.sh/concepts/scalers).

## Including Documentation Changes

For any contribution you make that impacts the behavior or experience of KEDA, please open a corresponding docs request for [keda.sh](https://keda.sh) through [https://github.com/kedacore/keda-docs](https://github.com/kedacore/keda-docs).  Contributions that do not include documentation or samples will be rejected.

## Creating and building a local environment

[Details on setup of a development environment are found on the README](https://github.com/kedacore/keda#building-quick-start-with-visual-studio-code-remote---containers)

## Contributor License Agreement (CLA) and Code of Conduct

Most contributions require you to agree to a Contributor License Agreement (CLA) declaring that you have the right to, and actually do, grant us the rights to use your contribution. For details, visit https://cla.microsoft.com.

When you submit a pull request, a CLA-bot will automatically determine whether you need to provide a CLA and decorate the PR appropriately (e.g., label, comment). Simply follow the instructions provided by the bot. You will only need to do this once across all repos using our CLA.

This project has adopted the Microsoft Open Source Code of Conduct. For more information see the Code of Conduct FAQ or contact opencode@microsoft.com with any additional questions or comments.