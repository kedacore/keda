## Prerequisits

- [node](https://nodejs.org/en/)
- `kubectl` logged into a Kubernetes cluster.
- Each scaler test might define additional requirements. For example, `azure-queue.test.ts` requires an env var `TEST_STORAGE_CONNECTION_STRING`

## Running tests:

```bash
npm install
npm test --verbose
```

## E2E test setup

The test script will run 3 phases:
- **Setup**: this is done in [`setup.test.ts`](setup.test.ts). If you're adding any tests for KEDA install/setup process add it to this file.`setup.test.ts` deploys [`/deploy/KedaScaleController.yaml`](../deploy/KedaScaleController.yaml) to `keda` namespace in the cluster, and updates the image to `kedacore/keda:master`

    After `setup.test.ts` is done, we expect to have a cluster with KEDA setup in namespace `keda`. This is done through a `pretest` hook in npm. See [`"scrips"` in package.json](package.json#L14).

- **Tests**: Currently there are only scaler tests in `tests/scalers`. All files run in parallel, but tests within the file can run either in parallel or in series. More about tests below.

- **Global clean up**: this is done in [`cleanup.test.ts`](cleanup.test.ts). This step cleans resources created in `setup.test.ts`.


## Adding tests:

* Tests are written in TypeScript using [ava](https://github.com/avajs/ava) framework. See [ava docs here](https://github.com/avajs/ava/blob/master/docs)
* Each scaler tests should be in a file. **e.g**: `azure-queue.tests.ts`, `kafka.tests.ts`, etc
* All files in `scalers/**.ts` are run in parallel by default. Make sure your tests don't affect the global state of the cluster in a way that can break other tests.
* Each test file is expected to do it's own setup and clean up for its resources.

```ts
// import test from ava framework
import test from 'ava';

test.before(t => {
    // this runs once before all tests.
    // do setup here. e.g:
    //  - Create a namespace for your tests (using kubectl or kubernetes node-client)
    //  - Create deployment (using kubectl or kubernetes node-client)
    //  - Setup event source (deploy redis, or configure azure storage, etc)
    //  - etc
});


// `test 1` and `test 2` will run in parallel.
test('test 1', t => { });
test('test 2', t => { });

// `test 3` will run first, then `test 4`.
// Tests will run in the order they are defined in.
// All serial tests will run first before parallel tests above
test.serial('test 3', t => { });
test.serial('test 4', t => { });

// Tests are expected to finish synchronously, or using async/await
// if you need to use callbacks, then add `.cb` and call `t.end()` when done.
test('test 6', t => { });
test('test 7', async t => { });
test.cb('test 8', t => { t.end() });

test.after.always.cb('clean up always after all tests', t => {
    // Clean up after your test here. without `always` this will only run if all tests are successful.
    t.end();
});
```
* **Example test:** for example if I want to add a test for redis

```ts
import * as sh from 'shelljs';
import test from 'ava';

// you can include template in the file or in another file.
const deployYaml = `apiVersion: apps/v1
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

test.before('install redis and create deployment' t => {
    if (!sh.which('helm')) {
        t.fail('redis tests require helm');
    }

    sh.exec('helm install redis ......'); // install redis to the cluster
    sh.exec('kubectl create namespace redis-test-deployment');
    sh.exec('kubectl apply -f ....'); // create your deployment
});

test.serial('deployment should scale when adding items in redis list', t => {
    // use node redis client to add stuff to redis.
    // maybe sleep or poll the replica count
    const replicaCount = sh.exec(`kubectl get deployment.apps/test-deployment .. -o jsonpath="{.spec.replicas}"`).stdout;
    t.is('10', replicaCount, 'expecting replica count to be 10');
});

test.after.always('remove redis and my deployment', t => {
    sh.exec('kubectl delete ....');
});
```

* You can see [`azure-queue.test.ts`](scalers/azure-queue.test.ts) for a full example.
* Ava has more options for asserting and writing tests. The docs are very good. https://github.com/avajs/ava/blob/master/docs/01-writing-tests.md
* **debugging**: when debugging, you can force only 1 test to run by adding `only` to the test definition.

```ts
test.serial.only('this will be the only test to run', t => { });
```
