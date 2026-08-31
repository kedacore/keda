# Instructions for AI Agents

This file gives AI coding agents (Claude Code, Codex, Cursor, Copilot, Aider, etc.) the rules they must follow when contributing to this repository. It complements [`CONTRIBUTING.md`](CONTRIBUTING.md), which remains the source of truth for humans. Agents must respect every rule there as well.

If a rule here conflicts with `CONTRIBUTING.md`, follow `CONTRIBUTING.md` and flag the discrepancy in the PR description.

## Zero-spam & PR authorization policy

Follow the [Anti-Spam & PR authorization policy](CONTRIBUTING.md#anti-spam--pr-authorization-policy) in `CONTRIBUTING.md`. In short: require an approved issue and assignment before opening a PR, respect claimed issues, stay in scope, one issue per PR, and do not duplicate bot-managed chores.

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

## Release notes metadata

Release notes are generated automatically from merged PR metadata via the release-notes workflows.

Rules (from [`CONTRIBUTING.md#release-notes`](CONTRIBUTING.md#release-notes)):

- PR titles must follow `Component: Description`. The release notes renderer bolds the component automatically.

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
