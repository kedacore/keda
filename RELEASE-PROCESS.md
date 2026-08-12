# Release Process

The release process of a new version of KEDA involves the following:

## 0. Prerequisites

Look at the [last release] in the releases page:

- For example, at the time of writing, it was 2.3.0
- The next version will thus be 2.4.0

[last release]: https://github.com/kedacore/keda/releases/latest

## 1. Release notes metadata

Release notes are generated automatically from merged PR metadata using [release-cliff workflow](.github/workflows/release-cliff.yml).

Before the release, make sure PR titles and `kind/*` labels are correct so the generated draft release is grouped and worded as expected.

## 2. Add the new version to GitHub Bug report template

Add the new released version to the list in `KEDA Version` dropdown in [3_bug_report.yml](https://github.com/kedacore/keda/blob/main/.github/ISSUE_TEMPLATE/3_bug_report.yml).

## 3. Create KEDA release on GitHub

Creating a new release in the releases page (https://github.com/kedacore/keda/releases) will trigger a GitHub workflow which will create a new image with the latest code (read note 2 below) and tagged with the next version (in this example 2.4.0).

KEDA Deployment YAML file (eg. keda-2.4.0.yaml) is also automatically created and attached to the Release as part of the workflow.

> Note: The container registry repo with all the different images can be seen [here](https://github.com/orgs/kedacore/packages?repo_name=keda)

> Note 2: Depending on the release type (minor version or hotfix), the tag should be created from main (for minor version releases) or from version branch (for hotfix releases)

### Automated draft release notes (git-cliff)

Release draft notes are generated automatically by [release-cliff workflow](.github/workflows/release-cliff.yml) using [git-cliff config](.github/cliff.toml).

How it works:

- The workflow runs on push to `main` and `release/v*` branches.
- It computes the next version and the baseline tag for the current branch.
- It removes any existing draft release targeting the same branch.
- It generates the changelog with `git-cliff` using PR labels (`kind/*`) to group entries.
- It creates a new draft release with generated notes and a docs link for the computed version.

Each branch has an independent draft release:

| Branch | Draft tracks | Version bump |
| --- | --- | --- |
| `main` | Next minor release | `vX.(Y+1).0` |
| `release/vX.Y` | Subsequent patch releases for that minor | `vX.Y.(Z+1)` |

Before publishing a release, review and edit the draft body in GitHub to add highlights, upgrade notes, and any extra context for users.

### Release flows with branch-scoped drafts

Minor release flow (`vX.Y.0`):

1. Merge all release-bound PRs into `main`.
1. Open the generated draft release for the next minor release.
1. During the `vX.Y.0` release workflow, KEDA automatically creates `release/vX.Y` from the release tag commit when it does not exist yet.
1. Use that `release/vX.Y` branch for subsequent patch releases (`vX.Y.1`, `vX.Y.2`, and so on).
1. Review and edit the draft notes, then publish `vX.Y.0` from `main`.

Hotfix flow (`vX.Y.Z`):

1. Merge the fix PR into `main`.
1. Backport it to the corresponding `release/vX.Y` branch (cherry-pick bot or manual cherry-pick).
1. Push the backport branch update and open the regenerated draft targeting `release/vX.Y`.
1. Review, confirm the next patch version, and publish.

### PR requirements for generated notes

The generated release notes depend on PR metadata. Keep these requirements in every PR:

- Use exactly one changelog label from `kind/feature`, `kind/new-scaler`, `kind/improvement`, `kind/enhancement`, `kind/bug`, `kind/deprecation`, `kind/breaking-change`, `kind/chore`, `kind/documentation`, `kind/dependencies`, or `kind/ci`.
- Use `skip-changelog` only when the PR should not appear in release notes.
- Use a PR title in format `Component: Description`. The release notes renderer bolds the component automatically.

These checks are enforced by [pr-changelog-check workflow](.github/workflows/pr-changelog-check.yml).

### Draft review checklist

Every release draft created by [release-cliff workflow](.github/workflows/release-cliff.yml) should be reviewed and completed before publishing.

> ### 💡 IMPORTANT
>
> Remember to review and complete the following before publishing:
>
> - Add release highlights and upgrade notes
> - Confirm the version, target branch, and docs link are correct
> - Review the generated changelog for accuracy and wording

The draft body follows this structure:

```markdown
We are happy to release KEDA INSERT-CORRECT-VERSION 🎉

Here are some highlights:

- <list highlights>

Here are the new deprecation(s) as of this release:
- <list deprecations>

Learn how to deploy KEDA by reading [our documentation](https://keda.sh/docs/INSERT-CORRECT-VERSION/deploy/).

🗓️ The next KEDA release is currently being estimated for <date>, learn more in our [roadmap](https://github.com/kedacore/keda/blob/main/ROADMAP.md#upcoming-release-cycles).

### New

- <list items>

### Improvements

- <list items>

### Breaking Changes

- <list items>

### Other

- <list items>

### New Contributors

<optional contributor highlights>
```

## 4. Publish documentation for new version

Publish documentation for new version on https://keda.sh.
For details, see [Publishing a new version](https://github.com/kedacore/keda-docs?tab=contributing-ov-file#publishing-a-new-version).

> Note: During hotfix releases, this step isn't required as we don't introduce new features

## 5. Setup continuous container scanning with Snyk

In order to continuously scan our new container image, they must be imported in our [Snyk project](https://app.snyk.io/org/keda/projects) for all newly introduced tags.

Prune old versions of images. Keep only one version for a 3 last minor releases (eg. keep only 2.10.1, 2.11.1 and 2.12.0).

Learn more on how to do this through the [Snyk documentation](https://docs.snyk.io/products/snyk-container/image-scanning-library/github-container-registry-image-scanning/scan-container-images-from-github-container-registry-in-snyk).

> Note: Remember to enable the check `Without issues` in order to get the new version listed since probably it hasn't got any issue.

## 6. Prepare our Helm Chart

Before we can release our new Helm chart version, we need to prepare it:

- Update the `version` and `appVersion` in our [chart definition](https://github.com/kedacore/charts/blob/master/keda/Chart.yaml).
- Update the CRDs & Kubernetes resources based on the release artifact (YAML)

## 7. Ship new Helm chart

Guidance on how to release it can be found in our [contribution guide](https://github.com/kedacore/charts/blob/master/CONTRIBUTING.md#shipping-a-new-version).

## 8. Trigger KEDA OLM release

Create a new issue in [KEDA OLM repository](https://github.com/kedacore/keda-olm-operator/issues/new/choose) stating that there should be a new release mirroring KEDA core release.

## 9. Prepare next release

As per our [release governance](https://github.com/kedacore/governance/blob/main/RELEASES.md), we need to create a new shipping cycle in our [project settings](https://github.com/orgs/kedacore/projects/2/settings/fields/1647216) with a target date in 3 months after the last cycle.

We need to make sure that the current sprint's items are changed from status `Ready to ship` to `Done`.

Lastly, the `Upcoming Release Cycles` overview in `ROADMAP.md` should be updated with the new cycle.

In case of minor releases, the version branch (`release/v2.x`) is created automatically from the `v2.x.0` release tag by the release workflow and is then used to include required hotfixes.

### Cherry-picking merged PRs to release branches

When a merged PR needs to be backported to a release branch, add a trigger label to the original PR:

```text
cherry-pick/vX.Y
```

For example:

```text
cherry-pick/v2.20
```

Behavior:

- The automation runs for merged PRs when the trigger label is added, and also when a PR with `cherry-pick/vX.Y` is merged.
- KEDA creates (or updates) a cherry-pick branch and opens a cherry-pick PR targeting `release/vX.Y`.
- KEDA copies `kind/*` labels to the cherry-pick PR.
- On success, KEDA removes `cherry-pick/vX.Y` and adds `cherry-picked/vX.Y` on the original PR.
- If the cherry-pick fails because of conflicts, KEDA comments with the manual recovery steps.

## 10. Tweet! 🐦

Prepare a tweet with some highlights and send it out on [@kedaorg](https://twitter.com/kedaorg)!

If you don't have access, ask a maintainer who has access to the account (see 1Password).
