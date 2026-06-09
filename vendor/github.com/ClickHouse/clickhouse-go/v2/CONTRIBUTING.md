# Contributing

## Local setup

Docker is required. Start a local ClickHouse instance and run the full test suite:

```bash
make up      # start ClickHouse via Docker Compose
make test    # run all tests with -race
make down    # stop and remove containers
```

## Make targets

| Target          | What it does |
|-----------------|--------------|
| `make up`       | Start ClickHouse (waits until healthy) |
| `make down`     | Stop and remove containers |
| `make test`     | Run the full test suite with `-race` and `-count=1` |
| `make lint`     | Run `golangci-lint` |
| `make staticcheck` | Run `staticcheck` |
| `make codegen`  | Regenerate column implementations and update license headers |
| `make contributors` | Rebuild `contributors/list` from git log |

## Running specific tests

```bash
# Run a single test
go test -race -count=1 -v ./tests/... -run TestBatchFlush

# Run against a specific ClickHouse version
CLICKHOUSE_VERSION=25.3 make test

# Run only the database/sql interface tests
go test -race -count=1 -v ./tests/std/...

# Run Go benchmark tests
go test -bench=. -benchmem ./benchmark/...

# Run a standalone benchmark program (most benchmarks are standalone executables)
go run benchmark/v2/read/main.go
```

## Writing tests

- **Integration tests** live in `tests/`. They require a running ClickHouse (via `make up`) and use [testcontainers-go](https://github.com/testcontainers/testcontainers-go) for environment setup.
- **Regression tests** for bug fixes go in `tests/issues/` named after the issue: `issue_1234_test.go`. Every bug fix should include one.
- **New ClickHouse type support** requires three things: a column implementation in `lib/column/`, a round-trip test in `tests/`, and an example in `examples/clickhouse_api/`.
- Do not mock `driver.Conn` or `driver.Rows` in tests — test against a real ClickHouse instance.

## Code guidelines

See [.claude/CLAUDE.md](.claude/CLAUDE.md) for Go idioms, API design principles, and workflow rules enforced in this repo.

## Pull requests

- Open a PR against `main`. All changes require a PR — do not commit directly.
- Keep PRs focused. One logical change per PR.
- Ensure `make lint` and `make test` pass before requesting review.
- Reference the GitHub issue number in the PR description if one exists.
- Do not edit `CHANGELOG.md` — it is generated automatically during the release process.
