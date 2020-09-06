# Contributing to KEDA

Thanks for helping make KEDA better 😍.

There are many areas we can use contributions - ranging from code, documentation, feature proposals, issue triage, samples, and content creation.

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of contents**

- [Project governance](#project-governance)
- [Getting Help](#getting-help)
- [Contributing Scalers](#contributing-scalers)
- [Including Documentation Changes](#including-documentation-changes)
- [Creating and building a local environment](#creating-and-building-a-local-environment)
- [Developer Certificate of Origin: Signing your work](#developer-certificate-of-origin-signing-your-work)
- [Code Quality](#code-quality)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Project governance

KEDA project is an open source project governed according to rules described in [`GOVERNANCE`](GOVERNANCE.md). If you
would love to become a maintainer to support our project please create an issue as described in [`GOVERNANCE`](GOVERNANCE.md).

## Getting Help

If you have a question about KEDA or how best to contribute, the [#KEDA](https://kubernetes.slack.com/archives/CKZJ36A5D) channel on the Kubernetes slack channel ([get an invite if you don't have one already](https://slack.k8s.io/)) is a good place to start.  We also have regular [community stand-ups](https://github.com/kedacore/keda#community-standup) to track ongoing work and discuss areas of contribution.  For any issues with the product you can [create an issue](https://github.com/kedacore/keda/issues/new) in this repo.

## Contributing Scalers

One of the easiest ways to contribute is adding scalers.  Scalers are the logic on when to activate a container (scaling from zero to one) and also how to serve metrics for an event source.  You can view [the code for existing scalers here](https://github.com/kedacore/keda/tree/master/pkg/scalers).  When writing a scaler, please consider:

1. Is this an event source that many others will access from Kubernetes? If not, potentially consider [creating an external scaler](https://github.com/kedacore/keda/blob/master/pkg/scalers/externalscaler/externalscaler.proto).
1. Provide tests
1. Provide [documentation and examples](https://github.com/kedacore/keda-docs#adding-scaler-documentation) for [keda.sh](https://keda.sh)

Information on how scalers work can be found in [`CREATE-NEW-SCALER`](CREATE-NEW-SCALER.md).

## Including Documentation Changes

For any contribution you make that impacts the behavior or experience of KEDA, please open a corresponding docs request for [keda.sh](https://keda.sh) through [https://github.com/kedacore/keda-docs](https://github.com/kedacore/keda-docs).  Contributions that do not include documentation or samples will be rejected.

## Creating and building a local environment

[Details on setup of a development environment are found on the README](https://github.com/kedacore/keda#building-quick-start-with-visual-studio-code-remote---containers)

## Developer Certificate of Origin: Signing your work

### Every commit needs to be signed

The Developer Certificate of Origin (DCO) is a lightweight way for contributors to certify that they wrote or otherwise have the right to submit the code they are contributing to the project. Here is the full text of the DCO, reformatted for readability:
```
By making a contribution to this project, I certify that:

    (a) The contribution was created in whole or in part by me and I have the right to submit it under the open source license indicated in the file; or

    (b) The contribution is based upon previous work that, to the best of my knowledge, is covered under an appropriate open source license and I have the right under that license to submit that work with modifications, whether created in whole or in part by me, under the same open source license (unless I am permitted to submit under a different license), as indicated in the file; or

    (c) The contribution was provided directly to me by some other person who certified (a), (b) or (c) and I have not modified it.

    (d) I understand and agree that this project and the contribution are public and that a record of the contribution (including all personal information I submit with it, including my sign-off) is maintained indefinitely and may be redistributed consistent with this project or the open source license(s) involved.
```

Contributors sign-off that they adhere to these requirements by adding a `Signed-off-by` line to commit messages.

```
This is my commit message

Signed-off-by: Random J Developer <random@developer.example.org>
```
Git even has a `-s` command line option to append this automatically to your commit message:
```
$ git commit -s -m 'This is my commit message'
```

Each Pull Request is checked  whether or not commits in a Pull Request do contain a valid Signed-off-by line.

### I didn't sign my commit, now what?!

No worries - You can easily replay your changes, sign them and force push them!

```
git checkout <branch-name>
git reset $(git merge-base master <branch-name>)
git add -A
git commit -sm "one commit on <branch-name>"
git push --force
```

## Code Quality

This project is using [pre-commits](https://pre-commit.com) to ensure the quality of the code.
We encourage you to use pre-commits, but it's not a required to contribute. Every change is checked
on CI and if it does not pass the tests it cannot be accepted. If you want to check locally then
you should install Python3.6 or newer together and run:
```bash
pip install pre-commit
# or
brew install pre-commit
```
For more installation options visit the [pre-commits](https://pre-commit.com).

To turn on pre-commit checks for commit operations in git, run:
```bash
pre-commit install
```
To run all checks on your staged files, run:
```bash
pre-commit run
```
To run all checks on all files, run:
```bash
pre-commit run --all-files
```
