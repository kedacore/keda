## Prerequisites

- [go](https://go.dev/)
- `kubectl` logged into a Kubernetes cluster.
- Each scaler test might define additional requirements. For example, `azure_queue_test.go` requires an env var `TF_AZURE_STORAGE_CONNECTION_STRING`

## Running tests:

### All tests

Make sure that you are in `keda/tests` directory.

```bash
go test -v -tags e2e ./utils/setup_test.go        # Only needs to be run once.
go test -v -tags e2e ./scalers/...
go test -v -tags e2e ./utils/cleanup_test.go      # Skip if you want to keep testing.
```

> **Note**
> As default, `go test -v -tags e2e ./utils/setup_test.go` deploys KEDA from upstream's main branch,
> if you are adding an e2e test to your own code, this is not useful as you need your own version.
> Like for [building and deploying your own image](../BUILD.md#custom-keda-as-an-image), you can use
> the Makefile environment variables to customize KEDA deployment.
> eg. `IMAGE_REGISTRY=docker.io IMAGE_REPO=johndoe go test -v -tags e2e ./utils/setup_test.go`

### Specific test

```bash
go test -v -tags e2e ./scalers/azure/azure_queue/azure_queue_test.go # Assumes that setup has been run before
```

> **Note**
> On macOS you might need to set following environment variable in order to run the tests: `GOOS="darwin"`
>
> eg. `GOOS="darwin" go test -v -tags e2e ...`

Refer to [this](https://pkg.go.dev/testing) for more information about testing in `Go`.

### Running e2e tests with a custom config file

The `E2E_TEST_CONFIG` environment variable can be used to run a subset of tests with a custom config file.
It can also be used to configure test setup. For example, to run tests without deploying KEDA, or to deploy KEDA using custom images.

```bash
E2E_TEST_CONFIG=tests/example-config.yaml go test -v -tags e2e ./tests/run-all.go
```

Supplying environment variables directly will override any relevant config file field.

Examples:

- `E2E_TEST_REGEX` will override the `testCategories` field in the config file.
- `E2E_INSTALL_KEDA=false` will run the tests without deploying KEDA, even if the config file has `keda.install: true`.

#### Filtering tests

The config file is a YAML file which can be used to configure the test setup and the test categories and the test suites to run.
You can include or exclude suites in a category, or exclude an entire category.
The current categories that exist are defined in the `tests/` subdirectory:

- `internals`
- `scalers`
- `secret-providers`
- `sequential`

Here is an example config that will run all tests in the `scalers` category, but `exclude` the tests in the `cpu` suite.
It will also `exclude` all tests in the `internals` category.
Omitting an existing category in the config file will run all tests in that category.

```yaml
testCategories:
  # This will run all tests in the scalers category, but exclude the cpu test.
  scalers:
    mode: exclude
    tests:
      - cpu
  # Using mode: exclude and omitting the tests list will exclude all tests in the category.
  internals:
    mode: exclude
  secret-providers:
    mode: exclude
  sequential:
    mode: exclude
```

You can also do the opposite, and `include` tests, but `exclude` the rest:

```yaml
testCategories:
  # This will only run the aws_cloudwatch, cpu, and kafka scaler tests, and exclude the rest.
  scalers:
    mode: include
    tests:
      - aws/aws_cloudwatch # you can also specify a deeper nested test by it's "directory path"
      - cpu
      - kafka
  # Since the other categories are not specified, all tests in those categories will not be run.
```

A valid config file must have the `testCategories` field, and all defined categories must have a `mode` field which is either `include` or `exclude`.

This is an example of a valid config file with the `testCategories` field set to an empty map. This will run all tests in all categories.

```yaml
# This will run all tests in all categories.
testCategories: {}
```

This example is invalid, and will fail and emit an error message:

```yaml
testCategories:
# Error loading test config: testCategories is a required field. Did you mean to set this to an empty map?
```

You can specify a custom go regex directly through the `E2E_TEST_REGEX` environment variable. This will override the config file environment variable.

Not specifying either `E2E_TEST_CONFIG` or `E2E_TEST_REGEX` will run all tests in all categories.

You can also check what tests would be executed and other configurations without actually running them by setting the `dryRun` field to `true`.

e.g.,

```yaml
dryRun: true
keda:
  # ...omitted
testCategories:
  # ...omitted
```

#### Customizing test setup

The config file can also be used to customize the test setup. For example, to deploy KEDA using custom images.

```yaml
keda:
  # These are the default values.
  skipSetup: false # If true, the test script will skip the setup phase.
  skipCleanup: false # If true, the test script will skip the cleanup phase.
  imageRegistry: ""
  imageRepo: ""
```

Note that if `imageRegistry` and `imageRepo` are empty, the test script will use the default KEDA repository and registry defined in the `Makefile`.

## E2E Test Setup

The test script will run in 3 phases:

- **Setup:** This is done in [`utils/setup_test.go`](utils/setup_test.go). If you're adding any tests to the KEDA install / setup process, you need to add it to this file. `utils/setup_test.go` deploys KEDA to the `keda` namespace, updates the image to
`kedacore/keda:main`.

    After `utils/setup_test.go` is done, we expect to have KEDA setup in the `keda` namespace.

- **Tests:** Currently there are only scaler tests in `tests/scalers/`. Each test is kept in its own package. This is to prevent conflicting variable declarations for commonly used variables (**ex -** `testNamespace`). Individual scaler tests are run
in parallel, but tests within a file can be run in parallel or in series. More about tests below.

- **Global cleanup:** This is done in [`utils/cleanup_test.go`](utils/cleanup_test.go). It cleans up all the resources created in `utils/setup_test.go`.

> **Note**
> Your IDE might give you errors upon trying to import certain packages that use the `e2e` build tag. To overcome this, you will need to specify in your IDE settings to use the `e2e` build tag.
>
> As an example, in VSCode, it can be achieved by creating a `.vscode` directory within the project directory (if not present) and creating a `settings.json` file in that directory (or updating it) with the following content:
> ```json
> {
>   "go.buildFlags": [
>       "-tags=e2e"
>   ],
>   "go.testTags": "e2e",
> }
> ```

## Adding tests

- Tests are written using `Go`'s default [`testing`](https://pkg.go.dev/testing) framework, and [`testify`](https://pkg.go.dev/github.com/stretchr/testify).
- Each e2e test should be in its own package, **ex -** `scalers/azure/azure_queue/azure_queue_test.go`, or `scalers/kafka/kafka_test.go`, etc
- Each test file is expected to do its own setup and clean for resources.

Test are split in different folders based on what it's testing:
- `internals`: KEDA internals (ie: HPA related stuff).
- `scalers`: Anything related with scalers.
- `secret-providers`: Anything related with how KEDA gets the secrets for working (ie: pod-identity, vault, etc).
- `sequential`: Tests that can't be run in parallel with other tests (eg. the test modifies KEDA installation or configuration, etc.).

#### ⚠⚠ Important: ⚠⚠
>
> - Even though the cleaning of resources is expected inside each e2e test file, all test namespaces
> (namespaces with label type=e2e) are  cleaned up to ensure not having dangling resources after global e2e
> execution finishes. To not break this behaviour, it's mandatory to use the `CreateNamespace(t *testing.T, kc *kubernetes.Clientset, nsName string)` function from [`helper.go`](helper.go), instead of creating them manually.

#### ⚠⚠ Important: ⚠⚠
> - `Go` code can panic when performing forbidden operations such as accessing a nil pointer, or from code that
> manually calls `panic()`. A function that `panics` passes the `panic` up the stack until program execution stops
> or it is recovered using `recover()` (somewhat similar to `try-catch` in other languages).
> - If a test panics, and is not recovered, `Go` will stop running the file, and no further tests will be run. This can
> cause issues with clean up of resources.
> - Ensure that you are not executing code that can lead to `panics`. If you think that there's a chance the test might
> panic, call `recover()`, and cleanup the created resources.
> - Read this [article](https://go.dev/blog/defer-panic-and-recover) for understanding more about `panic` and `recover` in `Go`.

#### **Example Test:** Let's say you want to add a test for `Redis`.

```go
// +build e2e
// ^ This is necessary to ensure the tests don't get run in the GitHub workflow.
import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
    // Other required imports
    ...
    ...

	. "github.com/kedacore/keda/v2/tests/helper" // For helper methods
)

var _ = godotenv.Load("../../.env") // For loading env variables from .env

const (
    testName = "redis-test"
    // Other constants required for your test
    ...
    ...
)

var (
    testNamespace    = fmt.Sprintf("%s-ns", testName)
    // Other variables required for your test
    ...
    ...
)

// YAML templates for your Kubernetes resources
const (
    deploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
  labels:
    app: test-deployment
spec:
  replicas: 0
  ...
  ...
`
...
...
)

type templateData struct {
    // Fields used in your Kubernetes YAML templates
    ...
    ...
}

func TestScaler(t *testing.T) {
    setupTest(t)

    kc := GetKubernetesClient(t)
    data, templates := getTemplateData()

    CreateKubernetesResources(t, kc, testNamespace, data, templates)

    testScaleOut(t)

    // Ensure that this gets run. Using defer is necessary
    DeleteKubernetesResources(t, testNamespace, data, templates)
    cleanupTest(t)
}

func setupTest(t *testing.T) {
    t.Log("--- setting up ---")
    _, err := ParseCommand("which helm").Output()
    assert.NoErrorf(t, err, "redis test requires helm - %s", err)

    _, err := ParseCommand("helm install redis .....").Output()
    assert.NoErrorf(t, err, "error while installing redis - %s", err)
}

func getTemplateData() (templateData, []Template) {
    return templateData{
        // Populate fields required in YAML templates
        ...
        ...
    }, []Template{
        {Name: "deploymentTemplate", Config: deploymentTemplate},
        {Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
    }
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset) {
    t.Log("--- testing scale out ---")
    // Use Go Redis Library to add stuff to redis to trigger scale out.
    ...
    ...
    // Sleep / poll for replica count using helper method.
    // Duration should be iterations * intervalSeconds
    require.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 10, 60, 1),
		"replica count should be 10 after 1 minute")
}

func cleanupTest(t *testing.T) {
    t.Log("--- cleaning up ---")
    // Cleanup external resources (such as Blob Storage Container, RabbitMQ queue, Redis in this case)
    ...
    ...
}
```

#### Notes

- You can see [`azure_queue_test.go`](scalers/azure/azure_queue/azure_queue_test.go) for a full example.
- All tests must have the `// +build e2e` build tag.
- Refer [`helper.go`](helper.go) for various helper methods available to use in your tests.
- Prefer using helper methods or `k8s` libraries in `Go` over manually executing `shell` commands. Only if the task
you're trying to achieve is too complicated or tedious using above, use `ParseCommand` or `ExecuteCommand` from `helper.go`
for executing shell commands.
- Ensure, ensure, ensure that you're cleaning up resources.
- You can use `VS Code` for easily debugging your tests.

## E2E Test infrastructure

For improving the reliability of e2e test, we try to have all resources under kedacore control using kedacore docker images rather end-users registry images (without official support) and cloud resources in kedacore accounts.

In order to manage these e2e resources, there are 2 different repositories:
- [kedacore/test-tools](https://github.com/kedacore/test-tools) for docker images management.
- [kedacore/testing-infrastructure](https://github.com/kedacore/testing-infrastructure) for cloud resources.

If any change is needed in e2e test infrastructure, please open a PR in those repositories and use kedacore resources for e2e tests.

## How to execute e2e tests during a PR

As e2e tests are executed using real infrastructure we don't execute them directly on the PRs. A member of [@keda-e2e-test-executors team](https://github.com/orgs/kedacore/teams/keda-e2e-test-executors) has to write a comment in the PR where the e2e tests should be executed:

```
/run-e2e
```

This comment will trigger a [workflow](https://github.com/kedacore/keda/blob/main/.github/workflows/pr-e2e.yml) that generates the docker images using the last commit (at the moment of the comment) and tests them. The commit will have an extra check (which can block the PR) and the member message will be updated with a link to the workflow execution.

There are cases where it isn't needed the whole e2e test suite. In order to reduce the time and required resources in those cases, the command can be appended with a regex which matches the desired e2e tests:

```
/run-e2e desired-regex
# e.g:
/run-e2e azure
```

This regex will be evaluated by the golang script, so it has to be written in a golang compliance way.

This new check is mandatory on every PR, the CI checks expect to execute the e2e tests. As not always the e2e tests are useful (for instance, when the changes apply only to documentation), it can be skipped labeling the PR with `skip-e2e`
