---
name: KEDA Release Tracker
about: Template to keep track of the progress for a new KEDA release.
title: "Release: "
labels: governance,release-management
assignees: tomkerkhove,zroubalik,jorturfer
---

This issue template is used to track the rollout of a new KEDA version.

For the full release process, we recommend reading [this document](https://github.com/kedacore/keda/blob/main/RELEASE-PROCESS.md).

## Required items

- [ ] List items that are still open, but required for this release

# Timeline

We aim to release this release in the week of <week range, example March 27-31>.

## Progress

- [ ] Prepare changelog
- [ ] [Welcome message supported versions](https://github.com/kedacore/keda/blob/main/pkg/util/welcome.go#L29-L30) are up-to-date
- [ ] Add the new version to GitHub Bug report template
- [ ] Create KEDA release
- [ ] Publish new documentation version
- [ ] Setup continous container scanning with Snyk
- [ ] Prepare & ship Helm chart
- [ ] Prepare next release
- [ ] Provide update in Slack
- [ ] Tweet about new release
