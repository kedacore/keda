import test from 'ava'
import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import { CreateTableCommand, DescribeTableCommand, DeleteTableCommand, DynamoDBClient,} from '@aws-sdk/client-dynamodb'
import { DynamoDBStreamsClient, DescribeStreamCommand } from '@aws-sdk/client-dynamodb-streams'
import { createNamespace, sleep, waitForDeploymentReplicaCount } from './helpers'

const awsRegion = 'ap-northeast-1'
const awsAccessKey = process.env['AWS_ACCESS_KEY'];
const awsSecretKey =  process.env['AWS_SECRET_KEY'];
const dynamoDBStreamNamespace = 'keda-test'
const dynamoDBTableName = 'keda-table01'
const nginxDeploymentName = 'nginx-deployment'

let dynamoDBClient;
let dynamoDBStreamsClient;
let dynamoDBStreamShardNum;

test.before(async t => {

  // setup dynamodb client
  dynamoDBClient = new DynamoDBClient({
    region: awsRegion,
    credentials: {
      accessKeyId: awsAccessKey,
      secretAccessKey: awsSecretKey
    }
  });

  createNamespace(dynamoDBStreamNamespace)

  // Create table
  let params = {
    TableName: dynamoDBTableName,
    KeySchema: [
      { AttributeName: 'id', KeyType: 'HASH' },  //Partition key
    ],
    AttributeDefinitions: [
      { AttributeName: 'id', AttributeType: 'S' },
    ],
    BillingMode: 'PAY_PER_REQUEST',
    StreamSpecification: {
      StreamEnabled: true,
      StreamViewType: 'NEW_IMAGE'
    }
  }
  let createTableCmd = new CreateTableCommand(params)
  await dynamoDBClient.send(createTableCmd)
  console.log("table is created!!")

  // Get streamArn for the created dynamodb table
  let describeTableCommand = new DescribeTableCommand({TableName: dynamoDBTableName});
  let dbResponse = await dynamoDBClient.send(describeTableCommand);
  const latestStreamArn = ( dbResponse.Table !== undefined ) ? dbResponse.Table.LatestStreamArn : ""

  await sleep(10000)

  // Get Shard Num
  dynamoDBStreamsClient = new DynamoDBStreamsClient({
    region: awsRegion,
    credentials: {
      accessKeyId: awsAccessKey,
      secretAccessKey: awsSecretKey
    }
  });
  let describeStreamCommand = new DescribeStreamCommand({
    StreamArn: latestStreamArn
  })

  let dbsResponse = await dynamoDBStreamsClient.send(describeStreamCommand)
  const shards = (dbsResponse.StreamDescription !== undefined) ? dbsResponse.StreamDescription.Shards : undefined 
  dynamoDBStreamShardNum = (( shards !== undefined ) ? shards.length : 0)
  console.log( "dynamodb stream shard num is " + dynamoDBStreamShardNum )

  // Deploy nginx
  console.log('deploy nginx')
  const nginxTmpFile = tmp.fileSync()
  fs.writeFileSync(nginxTmpFile.name, nginxDeployYaml)
  t.is(0, sh.exec(`kubectl apply --namespace ${dynamoDBStreamNamespace} -f ${nginxTmpFile.name}`).code, 'creating nginx deployment should work.')

  // wait for nginx to load
  console.log('wait for nginx to load')
  let nginxReadyReplicaCount = 1
  await waitForDeploymentReplicaCount(nginxReadyReplicaCount, nginxDeploymentName, dynamoDBStreamNamespace, 30)

  t.is(1, nginxReadyReplicaCount, 'creating an Nginx deployment should work')
})


test.serial('Should start off deployment with 1 replicas', t => {
  const replicaCount = sh.exec(`kubectl get deploy/${nginxDeploymentName} --namespace ${dynamoDBStreamNamespace} -o jsonpath="{.spec.replicas}"`).stdout
  t.is(replicaCount, '1', 'Replica count should start out as 1')
})


test.serial(`Replicas should scale up to the same number of shards after deploying scaleobject`, async t => {

  // Deploy scaleobject, etc
  console.log('deploy scaleobject')
  const scaleobjectTmpFile = tmp.fileSync()
  fs.writeFileSync(scaleobjectTmpFile.name, scaleObjectYaml)
  t.is(0, sh.exec(`kubectl apply --namespace ${dynamoDBStreamNamespace} -f ${scaleobjectTmpFile.name}`).code, 'creating scaleobject should work.')

  // Wait for nginx to scale up to the same number of shards
  t.true(await waitForDeploymentReplicaCount(dynamoDBStreamShardNum, nginxDeploymentName, dynamoDBStreamNamespace, 300, 1000), 'Replica count should increase to the maxReplicaCount')
})


test.after.always(async (t) => {
  // delete the dynamodDB Table
  let deleteTableCommand = new DeleteTableCommand({ TableName: dynamoDBTableName });
  await dynamoDBClient.send(deleteTableCommand)

  // delete k8s resources in the test namespace
  t.is(0, sh.exec(`kubectl delete namespace ${dynamoDBStreamNamespace}`).code, 'Should delete DynamoDB Stream namespace')
})


const nginxDeployYaml = `
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

const scaleObjectYaml = `
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
  name: aws-dynamodb-stream-so
  labels:
    app: nginx
spec:
  scaleTargetRef:
    name: ${nginxDeploymentName}
  maxReplicaCount: 5
  minReplicaCount: 1
  pollingInterval: 5  # Optional. Default: 30 seconds
  cooldownPeriod:  1  # Optional. Default: 300 seconds
  triggers:
    - type: aws-dynamodb-stream
      authenticationRef:
        name: keda-trigger-auth-aws-credentials
      metadata:
        awsRegion: ${awsRegion}         # Required
        tableName: ${dynamoDBTableName} # Required
        shardCount: "1"                 # Optional. Default: 2
---
`
