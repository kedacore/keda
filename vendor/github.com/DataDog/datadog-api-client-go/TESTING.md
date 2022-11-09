Running tests
=============

The Datadog Go API client has 2 sorts of tests: regular tests and BDD tests.
Both are using recodings to store HTTP interactions, allowing to run the tests without
talking to the API. We also store the test time to be able to freeze.

You can control the behavior with the `RECORD` environment variable:
 - `RECORD=false`, the default value, means replaying HTTP requests from recordings.
 - `RECORD=true` creates or updates recordings. This will need valid credentials in `DD_TEST_CLIENT_API_KEY`
    and `DD_TEST_CLIENT_APP_KEY`.
 - `RECORD=none` ignores recordings. This will also runs tests that we call `integration-only`, i.e.
    tests that we don't record for security reasons. It also needs valid credentials.

Recording and freeze files are stored in `cassettes` directory in each test package with one file per tests
(e.g. `tests/api/v1/datadog/cassettes/`).

To run the tests, navigate to the `tests` directory and run:

```shell
go test $(go list ./...)
```

You can get more verbose information with the `-v` flag, and run a specific
test with the `-run` argument. For example:

```shell
go test $(go list ./...)  -run TestSLOEventLifecycle -v
```

This takes a regular expression, so you don't have to specify the whole exact
string.

The first time you run a test that needs recordings, it will fail with:
`time file 'cassettes/$TEST_NAME.freeze' not found: create one setting 'RECORD=true' or ignore it using 'RECORD=none'`.

BDD tests are triggered by a parent test named `TestScenarios`. To run a specific test, you need to specify
the parent feature name in the `-run` argument, to look like `TestScenarios/$VERSION/Feature_$NAME/$TEST_NAME`.
For example:

```shell
go test ./scenarios  -run TestScenarios/v2/Feature_Users/Scenario_Send_invitation_emails
```

Again we don't need to pass the full test name as it's a regular expression,
but we do need to pass the full prefix.

To get a better output you can use `gotestsum` in place of `go test`, with the `--format` option to customize
the output. For example:

```shell
gotestsum --format testname ./scenarios -run TestScenarios/Feature_Users/Scenario_Send_invitation_emails
```
