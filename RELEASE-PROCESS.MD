# Release Process

The release process of a new version of KEDA involves the following:

## 0. Prerequisites

Look at the [last release] in the releases page:

- For example, at the time of writing, it was 2.3.0
- The next version will thus be 2.4.0

[last release]: https://github.com/kedacore/keda/releases/latest

## 1. Changelog

Add a new section in [CHANGELOG.md](CHANGELOG.md) for the new version that is being released along with the new features, patches and deprecations it introduces.

It should not include every single change but solely what matters to our customers, for example issue template that has changed is not important.

## 2. Add the new version to GitHub Bug report template

Add the new released version to the list in `KEDA Version` dropdown in [3_bug_report.yml](https://github.com/kedacore/keda/blob/main/.github/ISSUE_TEMPLATE/3_bug_report.yml).

## 3. Publish documentation for new version

Publish documentation for new version on https://keda.sh.
For details, see [Publishing a new version](https://github.com/kedacore/keda-docs#publishing-a-new-version).

## 4. Create KEDA release on GitHub

Creating a new release in the releases page (https://github.com/kedacore/keda/releases) will trigger a GitHub workflow which will create a new image with the latest code and tagged with the next version (in this example 2.4.0).

KEDA Deployment YAML file (eg. keda-2.4.0.yaml) is also automatically created and attached to the Release as part of the workflow.

> Note: The container registry repo with all the different images can be seen [here](https://github.com/orgs/kedacore/packages?repo_name=keda)

### Release template

Every release should use the template provided below to create the GitHub release and ensure that a new GitHub Discussion is created.

> ### 💡 IMPORTANT
>
> Remember to make the following changes to the template:
>
> - Replace `INSERT-CORRECT-VERSION` (there are **two** occurrences in the template) with the new-release ID
> - Update the list of new contributors

Here's the template:

```markdown
We are happy to release KEDA INSERT-CORRECT-VERSION 🎉

Here are some highlights:

- <list highlights>

Learn how to deploy KEDA by reading [our documentation](https://keda.sh/docs/INSERT-CORRECT-VERSION/deploy/).

### New

- <list items>

### Improvements

- <list items>

### Breaking Changes

- <list items>

### Other

- <list items>

### New Contributors

<generated new contributors info>
```

### Generating new contributor's info

In order to generate a list of new contributors, use the `Auto-generate release notes` GitHub feature of the release.

<details>
  <summary>Screenshot</summary>

![image](https://user-images.githubusercontent.com/4345663/148563945-ad75816d-739b-4e8d-a063-aa0e77f6e98d.png)
</details>

## 5. Setup continous container scanning with Snyk

In order to continuously scan our new container image, they must be imported in our [Snyk project](https://app.snyk.io/org/keda/projects) for all newly introduced tags.

Learn more on how to do this through the [Snyk documentation](https://docs.snyk.io/products/snyk-container/image-scanning-library/github-container-registry-image-scanning/scan-container-images-from-github-container-registry-in-snyk).

> Note: Remember to enable the check `Without issues` in order to get the new version listed since probably it hasn't got any issue.

## 6. Prepare our Helm Chart

Before we can release our new Helm chart version, we need to prepare it:

- Update the `version` and `appVersion` in our [chart definition](https://github.com/kedacore/charts/blob/master/keda/Chart.yaml).
- Update the CRDs & Kubernetes resources based on the release artifact (YAML)

## 7. Ship new Helm chart

Guidance on how to release it can be found in our [contribution guide](https://github.com/kedacore/charts/blob/master/CONTRIBUTING.md#shipping-a-new-version).

## 8. Prepare next release

As per our [release governance](https://github.com/kedacore/governance/blob/main/RELEASES.md), we need to create a new shipping cycle in our [project settings](https://github.com/orgs/kedacore/projects/2/settings/fields/1647216) with a target date in 3 months after the last cycle.

Lastly, the `Upcoming Release Cycles` overview in `ROADMAP.md` should be updated with the new cycle.

## 9. Tweet! 🐦

Prepare a tweet with some highlights and send it out on [@kedaorg](https://twitter.com/kedaorg)!

If you don't have access, ask a maintainer who has access to the account (see 1Password).
