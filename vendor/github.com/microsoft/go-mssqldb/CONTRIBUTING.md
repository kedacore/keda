# Contributing

This project welcomes contributions and suggestions. Most contributions require you to
agree to a Contributor License Agreement (CLA) declaring that you have the right to,
and actually do, grant us the rights to use your contribution. For details, visit
https://cla.microsoft.com.

When you submit a pull request, a CLA-bot will automatically determine whether you need
to provide a CLA and decorate the PR appropriately (e.g., label, comment). Simply follow the
instructions provided by the bot. You will only need to do this once across all repositories using our CLA.

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/).
For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/)
or contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.

## Release Process

This project uses [Release Please](https://github.com/googleapis/release-please) for automated version management and changelog generation.

### How It Works

1. **Conventional Commits**: When merging PRs to `main`, use [Conventional Commits](https://www.conventionalcommits.org/) format in PR titles or commit messages
2. **Automated Release PR**: Release Please will automatically create/update a release PR that:
   - Bumps version based on commit types
   - Updates CHANGELOG.md
   - Updates version.go
3. **Review and Merge**: When ready to release, merge the Release Please PR
4. **GitHub Release**: A GitHub release with tag will be automatically created

### Conventional Commit Prefixes

| Prefix | Version Bump | Example |
|--------|-------------|---------|
| `feat:` | Minor (X.Y.0) | `feat: add connection pooling support` |
| `fix:` | Patch (X.Y.Z) | `fix: resolve timeout issue` |
| `feat!:` or `BREAKING CHANGE:` | Major (X.0.0) | `feat!: change API signature` |
| `docs:`, `chore:`, `ci:`, `deps:` | No bump | Maintenance changes |

### Example PR Titles

- ✅ `feat: add support for SQL Server 2025`
- ✅ `fix: correct datetime handling near midnight`
- ✅ `feat!: remove deprecated connection parameters`
- ✅ `chore: update dependencies`
- ❌ `Update README` (should be `docs: update README`)
- ❌ `Bug fix` (should be `fix: <description>`)

## Code Coverage Requirements

This project enforces a **strict 80% minimum code coverage** requirement to maintain code quality and ensure inclusion in the [awesome-go](https://github.com/avelino/awesome-go) directory.

### Requirements

- **Project coverage**: Must stay at or above 80%
- **Patch coverage**: New code in PRs must be at least 80% covered

### Checking Coverage Locally

```bash
# Run tests with coverage
go test -coverprofile=coverage.out ./...

# View total coverage
go tool cover -func=coverage.out | tail -1

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html
```

### Tips for Maintaining Coverage

1. Write unit tests for new functions and methods
2. Test error paths, not just happy paths
3. Use table-driven tests for comprehensive coverage
4. Check coverage before submitting PRs