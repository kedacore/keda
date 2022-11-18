# How to contribute

First of all, thanks for contributing!

This document provides some basic guidelines for contributing to this repository. To propose improvements, feel free to submit a PR.

## Reporting a Bug - Requesting a feature - GitHub Issues

* **Ensure the bug was not already reported** by searching on GitHub under [Issues][1].
* If you're unable to find an open issue addressing the problem, [open a new one][2].
  - **Fill out the issue template completely**. Label the issue properly.
    - Add `severity/` label.
    - Add `documentation` label if this issue is related to documentation changes.
* If you have a feature request, it is encouraged to [contact support][3] so the request can be prioritized and properly tracked.
* **Do not open an issue if you have a question**, instead [contact support][3].

## Suggesting an enhancements - Pull Requests

Client source code is generated using [apigentools](https://apigentools.readthedocs.io/en/latest/).
 
While you can create an issue to suggest a client enhancement, you won't be able to make a Pull Request for it.

Changes can only be made to:
- Improve tests
- Improve dev tooling
- Improve documentation. 

If that's the case, many thanks!

Read the [development guide](DEVELOPMENT.md) for more information on how to get started.

In order to ease/speed up our review, here are some items you can check/improve when submitting your PR:
* **Ensure an [Issue has been created](#reporting)**.
* Avoid changing too many things at once. Make sure that your Pull Requests only fixes one Issue at the time.
* Make sure that **all tests pass locally**.
* Summarize your PR with a **meaningful title** and **fill out the pull request description template completely!**

Your pull request must pass all CI tests. If you're seeing an error and don't think it's your fault, it may not be! 
[Join us on Slack][5] or send us an email, and together we'll get it sorted out.

### Keep it small, focused

Avoid changing too many things at once. For instance if you're fixing two different
issues at once, it makes reviewing harder and the _time-to-release_ longer.

### Commit Messages

Please don't be this person: `git commit -m "Fixed stuff"`. Take a moment to
write meaningful commit messages.

The commit message should describe the reason for the change and give extra details
that will allow someone later on to understand in 5 seconds the thing you've been
working on for a day.

### Releasing

The release procedure is managed by Datadog, instructions can be found in the [RELEASING](/RELEASING.md) document.
However, note that improvements to tests and documentation do not end up in changelogs. Only client improvements do.


## Asking a questions

Need help? Contact [Datadog support][3]

## Additional Notes

### Issue and Pull Request Labels

This section lists the labels we use to help us track and manage issues and pull requests.

| Label name                    | Usage                    | Description
|-------------------------------|--------------------------|------------------------------------------------------------
| `backward-incompatible`       | Issues and Pull Requests | Warn for backward incompatible changes.
| `changelog/Added`             | Pull Request Only        | Added features results into a minor version bump.
| `changelog/Changed`           | Pull Request Only        | Changed features results into a major version bump.
| `changelog/Deprecated`        | Pull Request Only        | Deprecated features results into a major version bump.
| `changelog/Fixed`             | Pull Request Only        | Fixed features results into a bug fix version bump.
| `changelog/no-changelog`      | Pull Request Only        | Changes don't appear in changelog.
| `changelog/Removed`           | Pull Request Only        | Deprecated features results into a major version bump.
| `changelog/Security`          | Pull Request Only        | Fixed features results into a bug fix version bump.
| `ci/skip`                     | Pull Request Only        | Skip GitHub action running tests.
| `community/help-wanted`       | Issue Only               | Community help wanted.
| `community`                   | Issues and Pull Requests | Community driven changes.
| `dev/testing`                 | Issues and Pull Requests | Tests related changes.
| `dev/tooling`                 | Issues and Pull Requests | Tooling related changes.
| `do-not-merge/HOLD`           | Pull Request Only        | Do not merge this PR.
| `do-not-merge/WIP`            | Pull Request Only        | Do not merge this PR.
| `documentation`               | Issues and Pull Requests | Documentation related changes.
| `duplicate`                   | Issue Only               | Duplicate issue.
| `invalid`                     | Issue Only               | Invalid issue.
| `kind/bug`                    | Issue Only               | Bug related issue.
| `kind/feature-request`        | Issue Only               | Feature request related issue.
| `severity/critical`           | Issue Only               | Critical severity issue.
| `severity/major`              | Issue Only               | Major severity issue.
| `severity/minor`              | Issue Only               | Minor severity issue.
| `severity/normal`             | Issue Only               | Normal severity issue.
| `stale`                       | Issues and Pull Requests | Stale - Bot reminder.
| `stale/exempt`                | Issues and Pull Requests | Exempt from being marked as stale.

[1]: https://github.com/DataDog/datadog-api-client-go/issues
[2]: https://github.com/DataDog/datadog-api-client-go/issues/new
[3]: https://docs.datadoghq.com/help
[4]: https://keepachangelog.com/en/1.0.0
[5]: https://datadoghq.slack.com
