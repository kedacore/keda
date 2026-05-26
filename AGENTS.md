# Instructions for AI Agents

This file gives AI coding agents (Claude Code, Codex, Cursor, Copilot, Aider, etc.) the rules they must follow when contributing to this repository. It complements [`CONTRIBUTING.md`](CONTRIBUTING.md), which remains the source of truth for humans. Agents must respect every rule there as well.

If a rule here conflicts with `CONTRIBUTING.md`, follow `CONTRIBUTING.md` and flag the discrepancy in the PR description.

## Zero-spam & PR authorization policy

- **Require an issue:** DO NOT create a Pull Request unless there is an existing, open, and approved GitHub Issue that explicitly requests this work. Drive-by PRs, speculative refactors, "found a typo" PRs, and unsolicited feature work are not accepted.
- **Require assignment:** DO NOT start work on an issue unless it is assigned to the human user driving you. An unassigned issue is not an invitation to start coding. If the human has not been assigned, ask them to request assignment from a maintainer first and wait.
- **Respect claimed issues**: If someone has commented that they intend to work on the issue, or has been assigned to it, do not open a competing PR, push commits, or start a draft PR. However, if there has been no visible progress for an extended period, it is acceptable to politely ask whether they are still actively working on it before taking further action.
- **Stay inside the issue's scope:** Implement only what the issue describes. If you discover related problems, mention them in the PR description or open a separate issue. Do not silently expand the scope.
- **One issue, one PR:** Do not bundle multiple issues into a single PR, and do not split a single issue across multiple PRs without coordinating in the issue first.
- **No PR for chores that already have automation:** Dependency bumps (Renovate/Dependabot), changelog regeneration, generated-file refreshes, and similar housekeeping are handled by bots or release tooling. Do not open PRs that duplicate that work.

## Pull request rules

- **Do not delete or modify the checklist** in [`.github/PULL_REQUEST_TEMPLATE.md`](.github/PULL_REQUEST_TEMPLATE.md). When opening a PR, keep every checklist item and tick off only the boxes that genuinely apply to the change.
- Keep the `Fixes #` / `Relates to #` lines and fill them in when there is a related issue or PR.
- Write a clear description of *what* changed and *why*. Do not leave the template description empty.
- One logical change per PR. Do not bundle unrelated refactors with feature work or bug fixes.

## Required local checks before opening a PR

Run these from the repo root and make sure they all pass. CI will run them too, but agents must verify locally first.

| Check | Command | When |
| --- | --- | --- |
| Code formatting | `make fmt` (wraps `go fmt`) | Always |
| Static analysis | `make vet` (wraps `go vet`) | Always |
| Linter | `make golangci` | Always |
| Unit tests | `make test` | Always |
| Generated code | `make generate` | When you change anything under `apis/`, mocks, or protobuf |
| Scalers schema | `make generate-scalers-schema` | When you add or change a scaler's metadata, fields, or annotations |
| Schema verification | `make verify-scalers-schema` | After running `generate-scalers-schema` |
| Manifest verification | `make verify-manifests` | When you change CRDs or RBAC |

Optionally install [pre-commit](https://pre-commit.com) and run `pre-commit run --all-files`. This executes `go fmt`, trailing-whitespace, end-of-file fixer, doctoc, `golangci-lint`, the scaler-sort check, and the changelog validator in one go. See [`.pre-commit-config.yaml`](.pre-commit-config.yaml).

**Do not** skip, disable, or bypass these checks (e.g. `--no-verify`, commenting out linters, adding broad `//nolint` directives) to make a PR pass. Fix the underlying issue.

## Tests

- New scalers **must** ship with end-to-end (e2e) tests. See [`tests/README.md`](tests/README.md).
- Bug fixes should add a regression test that fails without the fix.
- New behavior in existing code should be covered by unit tests.
- Do not delete existing tests to make a build green. If a test is genuinely wrong, explain why in the PR description.
- Do not weaken assertions (e.g. replacing exact checks with `assert.NotNil`) just to make a flaky test pass.

## Changelog

Every user-visible change must be added to [`CHANGELOG.md`](CHANGELOG.md) under the `## Unreleased` section. The pre-commit hook [`hack/validate-changelog.sh`](hack/validate-changelog.sh) verifies this.

Rules (from [`CONTRIBUTING.md#Changelog`](CONTRIBUTING.md#changelog)):

- Place the entry under the correct subsection: `### New`, `### Improvements`, `### Fixes`, `### Deprecations`, `### Breaking Changes`, or `### Other`.
- Format: `- **<Scaler Name>**: <Description> ([#<ID>](https://github.com/kedacore/keda/issues/<ID>))`.
  - Use `**General**:` for cross-cutting changes; these go at the top of the subsection.
  - Otherwise use the scaler name (e.g. `**Kafka Scaler**:`).
- Entries are sorted **alphabetically** within each subsection, with `General` always first.
- `<ID>` should preferably link to an issue; if none exists, link the PR.
- New scaler template: `**General**: Introduce new XXXXXX Scaler ([#ISSUE](https://github.com/kedacore/keda/issues/ISSUE))`.
- Internal-only changes (refactors, test-only changes, CI tweaks) do **not** require a changelog entry.

## Commit hygiene

- **Every commit must be signed off** (DCO). Use `git commit -s`. The `Signed-off-by:` trailer must match the author. CI rejects PRs with unsigned commits.
- Never set `--no-verify`, `--no-gpg-sign`, or otherwise skip hooks unless the human user explicitly asks.
- Do not commit generated files that are not produced by the documented `make` targets above.
- Do not commit secrets, credentials, `.env` files, or large binaries.

## Code style

- Follow the existing patterns in the package you are editing. Do not introduce new abstractions, frameworks, or dependencies without justification in the PR description.
- Scalers in [`pkg/scaling/scalers_builder.go`](pkg/scaling/scalers_builder.go) must remain sorted (enforced by [`tools/sort_scalers.sh`](tools/sort_scalers.sh)).
- Honour the metrics and logging guidelines in [`CONTRIBUTING.md#metrics-and-logging`](CONTRIBUTING.md#metrics-and-logging) when adding telemetry.

## Scope discipline

- Do not "drive-by" reformat, rename, or restructure code outside the scope of the requested change.
- Do not bump dependencies unless the task requires it.
- Do not change CI workflows, release tooling, or governance files unless explicitly asked.
- If you encounter unrelated issues while working, mention them in the PR description rather than fixing them in the same PR.

## Documentation

- Behavior or UX changes require a matching docs PR against [`kedacore/keda-docs`](https://github.com/kedacore/keda-docs). Link it in the PR template's `Relates to` line.
- Manifest changes that affect deployment require a matching PR against [`kedacore/charts`](https://github.com/kedacore/charts).

## When in doubt

Stop and ask the human reviewer rather than guessing. It is better to leave a `TODO` and surface the question in the PR description than to invent behaviour, fabricate API names, or silence failing checks.
