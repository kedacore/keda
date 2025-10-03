---
name: KEDA Release Tracker
about: Template to keep track of the progress for a new KEDA release.
title: "Release: "
labels: governance,release-management
assignees: wozniakjan,zroubalik,jorturfer
---

This issue template is used to track the rollout of a new KEDA version.

For the full release process, we recommend reading [this document](https://github.com/kedacore/keda/blob/main/RELEASE-PROCESS.md).

## Required items

- [ ] List items that are still open, but required for this release

# Timeline

We aim to release this release in the week of <week range, example March 27-31>.

## Progress

- [ ] Prepare changelog
  - [ ] Rename `Unreleased` to the new version and make sure new empty `Unreleased` section with subsections exists
  - [ ] Best effort changelog cleanup, sometimes the notes can be a little inconsistent, example: [#5398](https://github.com/kedacore/keda/pull/5398)
- [ ] Update Kubernetes compatibility (usually 3 versions, bump +1 from previous KEDA version)
  - [ ] [Welcome message supported versions](https://github.com/kedacore/keda/blob/main/pkg/util/welcome.go#L29-L30) are up-to-date
  - [ ] Update KEDA docs, example: [#1633](https://github.com/kedacore/keda-docs/pull/1633)
  - [ ] Update Charts runners, example [#782](https://github.com/kedacore/charts/pull/782)
- [ ] Add new KEDA version to [GitHub Bug report](https://github.com/kedacore/keda/blob/main/.github/
- [ ] Add the new version to [GitHub Bug report](https://github.com/kedacore/keda/blob/main/.github/ISSUE_TEMPLATE/3_bug_report.yml) template
- [ ] Best effort version bump for dependencies, example: [#5400](https://github.com/kedacore/keda/pull/5400)
  - [ ] Update k8s go modules and pin to the 2nd most recent minor version
  - [ ] Check if new Go has been released and if KEDA can be safely built by it
  - [ ] Update linters and build pipelines if Go has been bumped, example: [#5399](https://github.com/kedacore/keda/pull/5399)
- [ ] Create KEDA release
- [ ] Publish new documentation version
- [ ] Setup continuous container scanning with Snyk
- [ ] Prepare & ship Helm chart
- [ ] Create a new issue in [KEDA OLM repository](https://github.com/kedacore/keda-olm-operator/issues/new/choose)
- [ ] Provide update in Slack
- [ ] Tweet about new release
- [ ] Create a new [release tracking issue](https://github.com/kedacore/keda/issues/new?template=4_release_tracker.md) for the next version and set there preliminary date based on [the schedule](https://github.com/kedacore/keda/blob/main/ROADMAP.md#upcoming-release-cycles)
