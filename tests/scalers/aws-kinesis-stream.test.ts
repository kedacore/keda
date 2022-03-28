import test from 'ava'
import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import {v4 as uuidv4} from 'uuid';
import {
  Kinesis,
} from '@aws-sdk/client-kinesis'
import { createNamespace, sleep, waitForDeploymentReplicaCount } from './helpers'


const awsRegion = 'eu-west-2'
const awsAccessKey = process.env['AWS_ACCESS_KEY'];
const awsSecretKey =  process.env['AWS_SECRET_KEY'];
const testNamespace = 'kinesis-test'
const nginxDeploymentName = 'nginx-deployment'
const streamName = `keda-stream-${uuidv4()}`

let kinesisClient: Kinesis;

const updateShardCountAsync = (count: number) => new Promise((resolve, _) => {
  kinesisClient.updateShardCount({
    StreamName:streamName,
    TargetShardCount: count,
    ScalingType:'UNIFORM_SCALING'
  }, async err => {
    if (err != null) console.log(err)
    // Wait till the stream is updated and ready
    await sleep(30000)
    resolve(undefined);
  })
});

test.before(async t => {
  // setup kinesis client
  kinesisClient = new Kinesis({
    region: awsRegion,
    credentials: {
      accessKeyId: awsAccessKey,
      secretAccessKey: awsSecretKey
    }
  });

  createNamespace(testNamespace)

  // create the stream
  const createStreamAsync = () => new Promise((resolve, _) => {
    var params = {
      'ShardCount': 1,
      'StreamName': streamName
    };
    kinesisClient.createStream(params, async err => {
      if (err != null) console.log(err)
      // Wait till the stream is created and ready
      await sleep(30000)
      resolve(undefined);
    })
  });
  await createStreamAsync()


  // deploy nginx, scaledobject etc.
  console.log('deploy nginx, scaledobject etc.')
  const nginxTmpFile = tmp.fileSync()
  fs.writeFileSync(nginxTmpFile.name, nginxDeployYaml)

  t.is(0, sh.exec(`kubectl apply --namespace ${testNamespace} -f ${nginxTmpFile.name}`).code, 'creating nginx deployment should work.')
})

test.serial(`Replicas should scale to 2 (the max) then back to 0`, async t => {
  const maxReplicaCount = 2
  const minReplicaCount = 1;

  t.true(await waitForDeploymentReplicaCount(minReplicaCount, nginxDeploymentName, testNamespace, 300, 1000), 'Replica count should start out as 1')

  await updateShardCountAsync(2)

  t.true(await waitForDeploymentReplicaCount(maxReplicaCount, nginxDeploymentName, testNamespace, 180, 1000), 'Replica count should increase to the maxReplicaCount')

  await updateShardCountAsync(1)

  t.true(await waitForDeploymentReplicaCount(minReplicaCount, nginxDeploymentName, testNamespace, 180, 1000), 'Replica count should increase to the minReplicaCount')

})

test.after.always(async (t) => {
  // delete the stream
  const deleteStreamAsync = () => new Promise((resolve, _) => {
    var params = {
      'StreamName': streamName
    };
    kinesisClient.deleteStream(params, err => {
      if (err != null) console.log(err)
      resolve(undefined);
    })
  });
  await deleteStreamAsync()

  t.is(0, sh.exec(`kubectl delete namespace ${testNamespace}`).code, 'Should delete kinesis namespace')
})

const nginxDeployYaml = `
apiVersion: v1
kind: Secret
metadata:
  name: test-secrets
data:
  AWS_ACCESS_KEY_ID: '${Buffer.from(awsAccessKey, 'binary').toString('base64')}' # Required.
  AWS_SECRET_ACCESS_KEY: '${Buffer.from(awsSecretKey, 'binary').toString('base64')}' # Required.
---
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: keda-trigger-auth-aws-credentials
spec:
  secretTargetRef:
  - parameter: awsAccessKeyID     # Required.
    name: test-secrets            # Required.
    key: AWS_ACCESS_KEY_ID        # Required.
  - parameter: awsSecretAccessKey # Required.
    name: test-secrets            # Required.
    key: AWS_SECRET_ACCESS_KEY    # Required.
---
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: aws-kinesis-so
  labels:
    app: nginx
spec:
  scaleTargetRef:
    name: ${nginxDeploymentName}
  maxReplicaCount: 2
  minReplicaCount: 1
  cooldownPeriod: 1
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 15
  triggers:
    - type: aws-kinesis-stream
      authenticationRef:
        name: keda-trigger-auth-aws-credentials
      metadata:
        awsRegion: ${awsRegion}
        streamName: ${streamName}
        shardCount: "1"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ${nginxDeploymentName}
  labels:
    app: nginx
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.14.2
        ports:
        - containerPort: 80
---
`
