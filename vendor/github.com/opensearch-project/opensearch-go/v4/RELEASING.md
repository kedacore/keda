- [Overview](#overview)
- [Branching](#branching)
  - [Release Branching](#release-branching)
  - [Feature Branches](#feature-branches)
- [Release Labels](#release-labels)
- [Releasing](#releasing)

## Overview

This document explains the release strategy for artifacts in this organization.

## Branching

### Release Branching

Given the current major release of 1.0, projects in this organization maintain the following active branches.

- **main**: The next _major_ release. This is the branch where all merges take place and code moves fast.
- **1.x**: The next _minor_ release. Once a change is merged into `main`, decide whether to backport it to `1.x`.
- **1.0**: The _current_ release. In between minor releases, only hotfixes (e.g. security) are backported to `1.0`.

Label PRs with the next major version label (e.g. `2.0.0`) and merge changes into `main`. Label PRs that you believe need to be backported as `1.x` and `1.0`. Backport PRs by checking out the versioned branch, cherry-pick changes and open a PR against each target backport branch.

### Feature Branches

Do not creating branches in the upstream repo, use your fork, for the exception of long lasting feature branches that require active collaboration from multiple developers. Name feature branches `feature/<thing>`. Once the work is merged to `main`, please make sure to delete the feature branch.

## Release Labels

Repositories create consistent release labels, such as `v1.0.0`, `v1.1.0` and `v2.0.0`, as well as `patch` and `backport`. Use release labels to target an issue or a PR for a given release. See [MAINTAINERS](MAINTAINERS.md#triage-open-issues) for more information on triaging issues.

## Releasing

The release process is standard across repositories in this org and is run by a release manager volunteering from amongst [MAINTAINERS](MAINTAINERS.md).

1. Ensure that the version in [version.go](internal/version/version.go) is correct for the next release. The example here releases version 4.3.0.
2. For major version releases, ensure that all references are up-to-date, e.g. `github.com/opensearch-project/opensearch-go/v4`, see [opensearch-go#444](https://github.com/opensearch-project/opensearch-go/pull/444).
3. Edit the [CHANGELOG](CHANGELOG.md) and replace the `Unreleased` section with the version about to be released.
4. Add a comparison link to the new version at the bottom of the [CHANGELOG](CHANGELOG.md).
5. Create a pull request with the changes into `main`, e.g. [opensearch-go#443](https://github.com/opensearch-project/opensearch-go/pull/443).
6. Create a tag, e.g. `v4.3.0`, and push it to the GitHub repo. This [makes the new version available](https://go.dev/doc/modules/publishing) on [pkg.go.dev](https://pkg.go.dev/github.com/opensearch-project/opensearch-go/v4).
7. Draft and publish a [new GitHub release](https://github.com/opensearch-project/opensearch-go/releases/new) from the newly created tag.
8. Create a new `Unreleased` section in the [CHANGELOG](CHANGELOG.md), increment version in [version.go](internal/version/version.go) to the next developer iteration (e.g. `4.3.1`), and make a pull request with this change into `main`, e.g. [opensearch-go#448](https://github.com/opensearch-project/opensearch-go/pull/448).
    ```
    ## [Unreleased]

    ### Added

    ### Changed

    ### Deprecated

    ### Removed

    ### Fixed

    ### Security

    ### Dependencies
    ```
9. Run `go list` with the new version to refresh [pkg.go.dev](https://pkg.go.dev/github.com/opensearch-project/opensearch-go/v4), e.g. `go list -m github.com/opensearch-project/opensearch-go/v4@v4.3.0`.
