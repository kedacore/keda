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

- Edit the .travis file and change all occurrences of `golang:x.y.z-stretch` to the appropriate version.

- Edit the Makefile and change the line `GOVERSION ?= 1.16.6` into the required version.

## Debugging with DLV

To attach DLV debugger run tests with `DEBUG=true` flag e.g.:
```shell
DEBUG=true TESTOPTIONS="-test.run TestResponseHeader -test.v" make run-tests-single-json-with-auth
```
