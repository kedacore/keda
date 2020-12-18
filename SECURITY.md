# Security Policy

## Supported Versions

KEDA commits to supporting the n-2 version minor version of the current major release; as well as the last minor version of the previous major release.

Here's an overview:

| Version | Supported          |
| ------- | ------------------ |
| 2.0.x   | :white_check_mark: |
| 1.5.x   | :white_check_mark: |
| < 1.5   | :x:                |

## Prevention

KEDA maintainers are working to incorporate prevention by using various measures:

- Scan published container images ([issue](https://github.com/kedacore/keda/issues/1041))
- Scan container images for changes in PRs ([issue](https://github.com/kedacore/keda/issues/1040))
- Scan changes to Helm charts in PRs ([issue](https://github.com/kedacore/charts/issues/64))

## Disclosures

We strive to ship secure software, but we need the community to help us find security breaches.

In case of a confirmed breach, reporters will get full credit and can be keep in the loop, if
preferred.

### Private Disclosure Processes

We ask that all suspected vulnerabilities be privately and responsibly disclosed by [contacting our maintainers](mailto:cncf-keda-maintainers@lists.cncf.io).

### Public Disclosure Processes

If you know of a publicly disclosed security vulnerability please IMMEDIATELY email the [KEDA maintainers](mailto:cncf-keda-maintainers@lists.cncf.io) to inform about the vulnerability so they may start the patch, release, and communication process.

### Compensation

We do not provide compensations for reporting vulnerabilities except for eternal
gratitude.

## Communication

[GitHub Security Advisory](https://github.com/kedacore/keda/security/advisories) will be used to communicate during the process of  identifying, fixing & shipping the mitigation of the vulnerability.

The advisory will only be made public when the patched version is released to inform the community of the breach and its potential security impact.
