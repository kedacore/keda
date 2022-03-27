import test from 'ava'
import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import {
  SQS,
} from '@aws-sdk/client-sqs'

import { createNamespace, waitForDeploymentReplicaCount } from './helpers'


const awsRegion = 'eu-west-2'
const awsAccessKey = process.env['AWS_ACCESS_KEY'];
const awsSecretKey =  process.env['AWS_SECRET_KEY'];
const testNamespace = 'sqs-test'
const sqsQueue = 'keda-queue'
const nginxDeploymentName = 'nginx-deployment'

let sqsClient: SQS;

test.before(async t => {

  // setup sqs client
  sqsClient = new SQS({
    region: awsRegion,
    credentials: {
      accessKeyId: awsAccessKey,
      secretAccessKey: awsSecretKey
    }
  });

  createNamespace(testNamespace)

  // create the queue
  const createQueueAsync = () => new Promise((resolve, _) => {
    var params = {
      QueueName: sqsQueue,
      Attributes: {
        'DelaySeconds': '60',
        'MessageRetentionPeriod': '86400'
      }
    };
    sqsClient.createQueue(params, err => {
      if (err != null) console.log(err)
      resolve(undefined);
    })
  });
  await createQueueAsync()

  // deploy nginx, scaledobject etc.
  console.log('deploy nginx, scaledobject etc.')
  const nginxTmpFile = tmp.fileSync()
  fs.writeFileSync(nginxTmpFile.name, nginxDeployYaml)

  t.is(0, sh.exec(`kubectl apply --namespace ${testNamespace} -f ${nginxTmpFile.name}`).code, 'creating nginx deployment should work.')
})

test.serial(`Replicas should scale to 2 (the max) then back to 0`, async t => {
  const maxReplicaCount = 2
  const minReplicaCount = 0;

  t.true(await waitForDeploymentReplicaCount(minReplicaCount, nginxDeploymentName, testNamespace, 300, 1000), 'Replica count should start out as 0')

  //Add messages
  const sendMessage = (id: number) => new Promise((resolve, _) => {
    var params = {
      DelaySeconds: 10,
      MessageBody: id.toString(),
      QueueUrl: sqsQueue
    };
    sqsClient.sendMessage(params, err => {
      if (err != null) console.log(err)
      resolve(undefined);
    })
  });

  for (let i = 0; i < 10 ; i++) {
    await sendMessage(i)
  }

  t.true(await waitForDeploymentReplicaCount(maxReplicaCount, nginxDeploymentName, testNamespace, 180, 1000), 'Replica count should increase to the maxReplicaCount')

  //Purge queue
  var params = {
    QueueUrl: sqsQueue,
  };
  sqsClient.purgeQueue(params, err => {
    if (err != null) console.log(err)
  });

  t.true(await waitForDeploymentReplicaCount(minReplicaCount, nginxDeploymentName, testNamespace, 180, 1000), 'Replica count should increase to the minReplicaCount')

})

test.after.always(async (t) => {
  const deleteQueueAsync = () => new Promise((resolve, _) => {
    var params = {
      QueueUrl: sqsQueue
     };
    sqsClient.deleteQueue(params, err => {
      if (err != null) console.log(err)
      resolve(undefined);
    });
  });
  await deleteQueueAsync()

  t.is(0, sh.exec(`kubectl delete namespace ${testNamespace}`).code, 'Should delete SQS namespace')
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
  name: aws-sqs-queue-so
  labels:
    app: nginx
spec:
  scaleTargetRef:
    name: ${nginxDeploymentName}
  maxReplicaCount: 2
  minReplicaCount: 0
  cooldownPeriod: 1
  triggers:
    - type: aws-sqs-queue
      authenticationRef:
        name: keda-trigger-auth-aws-credentials
      metadata:
        awsRegion: ${awsRegion}
        queueURL: ${sqsQueue}
        queueLength: "1"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ${nginxDeploymentName}
  labels:
    app: nginx
spec:
  replicas: 0
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
