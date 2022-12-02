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

### Specific test

```bash
go test -v -tags e2e ./scalers/azure_queue/azure_queue_test.go # Assumes that setup has been run before
```

> **Note**
> On macOS you might need to set following environment variable in order to run the tests: `GOOS="darwin"`
>
> eg. `GOOS="darwin" go test -v tags e2e ...`

Refer to [this](https://pkg.go.dev/testing) for more information about testing in `Go`.

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
- Each e2e test should be in its own package, **ex -** `scalers/azure_queue/azure_queue_test.go`, or `scalers/kafka/kafka_test.go`, etc
- Each test file is expected to do its own setup and clean for resources.

Test are split in different folders based on what it's testing:
- `internals`: KEDA internals (ie: HPA related stuff).
- `scalers`: Anything related with scalers.
- `secret-providers`: Anything related with how KEDA gets the secrets for working (ie: pod-identity, vault, etc).

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
    DeleteKubernetesResources(t, kc, testNamespace, data, templates)
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

- You can see [`azure_queue_test.go`](scalers/azure_queue/azure_queue_test.go) for a full example.
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
