# Maintainer Instructions

- Always preserve backward compatibility
- Build using `make clean && make`
- After merging PR, always run `make changelog` and commit changes
- Set ArangoDB docker container (used for testing) using `export ARANGODB=<image-name>`
- Run tests using:
  - `make run-tests-single`
  - `make run-tests-resilientsingle`
  - `make run-tests-cluster`.
- The test can be launched with the flag `RACE=on` which means that test will be performed with the race detector, e.g:
  - `RACE=on make run-tests-single`
- Always create changes in a PR


# Change Golang version

- Edit the [.circleci/config.yml](.circleci/config.yml) file and change ALL occurrences of `gcr.io/gcr-for-testing/golang` to the appropriate version.
- Edit the [Makefile](Makefile) and change the `GOVERSION` to the appropriate version.
- For minor Golang version update, bump the Go version in [go.mod](go.mod) and [v2/go.mod](v2/go.mod) and run `go mod tidy`.

## Debugging with DLV

To attach DLV debugger run tests with `DEBUG=true` flag e.g.:
```shell
DEBUG=true TESTOPTIONS="-test.run TestResponseHeader -test.v" make run-tests-single-json-with-auth
```

# Release Instructions

1. Update CHANGELOG.md
2. Make sure that GitHub access token exist in `~/.arangodb/github-token` and has read/write access for this repo.
3. Make sure you have the `~/go-driver/.tmp/bin/github-release` file. If not run `make tools`.
4. Make sure you have admin access to `go-driver` repository.
5. Run `make release-patch|minor|major` to create a release.
   - To release v2 version, use `make release-v2-patch|minor|major`.
6. Go To GitHub and fill the description with the content of CHANGELOG.md
