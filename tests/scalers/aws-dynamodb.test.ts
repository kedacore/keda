import test from 'ava'
import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import {
  CreateTableCommand,
  DeleteItemCommand, DeleteTableCommand,
  DynamoDBClient,
  PutItemCommand,
} from '@aws-sdk/client-dynamodb'
import { createNamespace, waitForDeploymentReplicaCount } from './helpers'


const awsRegion = 'eu-west-2'
const awsAccessKey = process.env['AWS_ACCESS_KEY'];
const awsSecretKey =  process.env['AWS_SECRET_KEY'];
const dynamoDBNamespace = 'dynamodb-test'
const dynamoDBTableName = 'keda-events'
const expressionAttributeNames = '{ "#k" : "event_type"}'
const keyConditionExpression = '#k = :key'
const expressionAttributeValues = '{ ":key" : {"S":"scaling_event"}}'
const targetValue = '1'
const nginxDeploymentName = 'nginx-deployment'

let dynamoClient;

test.before(async t => {

  // setup dynamodb client
  dynamoClient = new DynamoDBClient({
    region: awsRegion,
    credentials: {
      accessKeyId: awsAccessKey,
      secretAccessKey: awsSecretKey
    }
  });

  createNamespace(dynamoDBNamespace)

  // create table
  let params = {
    TableName: dynamoDBTableName,
    KeySchema: [
      { AttributeName: 'event_type', KeyType: 'HASH' },  //Partition key
      { AttributeName: 'event_id', KeyType: 'RANGE' },  //Sort key
    ],
    AttributeDefinitions: [
      { AttributeName: 'event_type', AttributeType: 'S' },
      { AttributeName: 'event_id', AttributeType: 'S' },
    ],
    ProvisionedThroughput: {
      ReadCapacityUnits: 5,
      WriteCapacityUnits: 5,
    },
  }

  let createTableCmd = new CreateTableCommand(params)
  await dynamoClient.send(createTableCmd)

  sh.exec('sleep 10s')

  // deploy nginx, scaledobject etc.
  console.log('deploy nginx, scaledobject etc.')
  const nginxTmpFile = tmp.fileSync()
  fs.writeFileSync(nginxTmpFile.name, nginxDeployYaml)

  t.is(0, sh.exec(`kubectl apply --namespace ${dynamoDBNamespace} -f ${nginxTmpFile.name}`).code, 'creating nginx deployment should work.')
  // wait for nginx to load
  console.log('wait for nginx to load')
  let nginxReadyReplicaCount = 0

  await waitForDeploymentReplicaCount(nginxReadyReplicaCount, nginxDeploymentName, dynamoDBNamespace, 30)

  t.is(0, nginxReadyReplicaCount, 'creating an Nginx deployment should work')
})

test.serial('Should start off deployment with 0 replicas', t => {
  const replicaCount = sh.exec(`kubectl get deploy/${nginxDeploymentName} --namespace ${dynamoDBNamespace} -o jsonpath="{.spec.replicas}"`).stdout
  t.is(replicaCount, '0', 'Replica count should start out as 0')
})

test.serial(`Replicas should scale to 2 (the max) then back to 0`, async t => {

  console.log('creating table')

  const buildRecord = (id: number) => {
    return {
      TableName: dynamoDBTableName,
      Item: {
        'event_type': { S: 'scaling_event' },
        'event_id': { S: `${id}` }
      }
    };
  }

  t.true(await waitForDeploymentReplicaCount(0,nginxDeploymentName,dynamoDBNamespace, 60, 1000), "Replica count should start out as 0")


  for (let i = 0; i <= 6; i++) {
    let putCommand = new PutItemCommand(buildRecord(i));
    await dynamoClient.send(putCommand)
  }

  const maxReplicaCount = 2
  const minReplicaCount = 0;

  t.true(await waitForDeploymentReplicaCount(maxReplicaCount, nginxDeploymentName, dynamoDBNamespace, 180, 1000), 'Replica count should increase to the maxReplicaCount')

  for (let i = 0; i <= 40; i++) {
    let deleteItemCommand = new DeleteItemCommand({
      TableName: dynamoDBTableName,
      Key: {
        event_type : {S: "scaling_event"},
        event_id: { S: `${i}` }
      }
    });

    await dynamoClient.send(deleteItemCommand)
  }

  t.true(await waitForDeploymentReplicaCount(minReplicaCount, nginxDeploymentName, dynamoDBNamespace, 180, 1000), 'Replica count should increase to the minReplicaCount')
})

test.after.always(async (t) => {
  let deleteTableCommand = new DeleteTableCommand({ TableName: dynamoDBTableName });
  await dynamoClient.send(deleteTableCommand)

  t.is(0, sh.exec(`kubectl delete namespace ${dynamoDBNamespace}`).code, 'Should delete DynamoDB namespace')
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
  name: aws-dynamodb-table-so
  labels:
    app: nginx
spec:
  scaleTargetRef:
    name: ${nginxDeploymentName}
  maxReplicaCount: 2
  minReplicaCount: 0
  cooldownPeriod: 1
  triggers:
    - type: aws-dynamodb
      authenticationRef:
        name: keda-trigger-auth-aws-credentials
      metadata:
        awsRegion: ${awsRegion}
        tableName: ${dynamoDBTableName}
        expressionAttributeNames: '${expressionAttributeNames}'
        keyConditionExpression: '${keyConditionExpression}'
        expressionAttributeValues: '${expressionAttributeValues}'
        targetValue: '${targetValue}'
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
