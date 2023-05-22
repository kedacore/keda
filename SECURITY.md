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

We have a few preventive measures in place to detect security vulnerabilities:

- [Renovate](https://renovatebot.com) & [Dependabot](https://docs.github.com/en/code-security/dependabot/dependabot-security-updates/about-dependabot-security-updates) help us keep our dependencies up-to-date to patch vulnerabilities as soon as possible by creating awareness and automated PRs.
- [Snyk](https://snyk.io/) helps us ship secure container images:
  - Images are scanned in every pull request (PR) to detect new vulnerabilities.
  - Published images on GitHub Container Registry are monitored to detect new vulnerabilities so we can ship patches
- [Whitesource Bolt for GitHub](https://www.whitesourcesoftware.com/free-developer-tools/bolt/) helps us with identifying vulnerabilities in our dependencies to raise awareness.
- [Trivy](https://aquasecurity.github.io/trivy/latest/) helps us with identifying vulnerabilities in our dependencies and docker images to raise awareness as part of our CI.
- [Semgrep](https://semgrep.dev/) helps us with identifying vulnerabilities in our code and docker images to raise awareness as part of our CI.
- [GitHub's security features](https://github.com/features/security) are constantly monitoring our repo and dependencies:
  - All pull requests (PRs) are using CodeQL to scan our source code for vulnerabilities
  - Dependabot will automatically identify vulnerabilities based on GitHub Advisory Database and open PRs with patches
  - Automated [secret scanning](https://docs.github.com/en/enterprise-cloud@latest/code-security/secret-scanning/about-secret-scanning#about-secret-scanning-for-partner-patterns) & alerts

KEDA maintainers are working to improve our prevention by adding additional measures:

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
