import test from 'ava'
import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import {
  CloudWatch
} from '@aws-sdk/client-cloudwatch'

import { createNamespace, waitForDeploymentReplicaCount } from './helpers'

const awsRegion = 'eu-west-2'
const awsAccessKey = process.env['AWS_ACCESS_KEY'];
const awsSecretKey =  process.env['AWS_SECRET_KEY'];
const testNamespace = 'cloudwatch-test'
const cloudwatchMetricName = 'keda-metric'
const cloudwatchMetricNamespace = 'KEDA'
const cloudwatchMetricDimensionName = 'dimensionName'
const cloudwatchMetricDimensionValue = 'dimensionValue'
const nginxDeploymentName = 'nginx-deployment'

let cloudwatchClient: CloudWatch;

// create custom metric
const setCustomMetricAsync = (value :number) => new Promise((resolve, _) => {
  var params = {
    MetricData: [
      {
        MetricName: cloudwatchMetricName,
        Dimensions: [
          {
            Name: cloudwatchMetricDimensionName,
            Value: cloudwatchMetricDimensionValue
          },
        ],
        Unit: 'None',
        Value: value
      },
    ],
    Namespace: cloudwatchMetricNamespace
  };

  cloudwatchClient.putMetricData(params, err => {
    if(err != null) console.log(err)
    resolve(undefined);
  });

});

test.before(async t => {

  // setup cloudwatch client
  cloudwatchClient = new CloudWatch({
    region: awsRegion,
    credentials: {
      accessKeyId: awsAccessKey,
      secretAccessKey: awsSecretKey
    }
  });

  createNamespace(testNamespace)


  await setCustomMetricAsync(0)

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

  await setCustomMetricAsync(10)

  t.true(await waitForDeploymentReplicaCount(maxReplicaCount, nginxDeploymentName, testNamespace, 300, 1000), 'Replica count should increase to the maxReplicaCount')

  await setCustomMetricAsync(0)

  t.true(await waitForDeploymentReplicaCount(minReplicaCount, nginxDeploymentName, testNamespace, 300, 1000), 'Replica count should increase to the minReplicaCount')

})

test.after.always(async (t) => {
  await setCustomMetricAsync(0)

  t.is(0, sh.exec(`kubectl delete namespace ${testNamespace}`).code, 'Should delete Cloudwatch namespace')
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
  name: aws-cloudwatch-so
  labels:
    app: nginx
spec:
  scaleTargetRef:
    name: ${nginxDeploymentName}
  maxReplicaCount: 2
  minReplicaCount: 0
  cooldownPeriod: 1
  triggers:
    - type: aws-cloudwatch
      authenticationRef:
        name: keda-trigger-auth-aws-credentials
      metadata:
        awsRegion: ${awsRegion}
        namespace: ${cloudwatchMetricNamespace}
        dimensionName: ${cloudwatchMetricDimensionName}
        dimensionValue: ${cloudwatchMetricDimensionValue}
        metricName: ${cloudwatchMetricName}
        targetMetricValue: "1"
        minMetricValue: "0"
        metricCollectionTime: "120"
        metricStatPeriod: "30"
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
