import * as azdev from "azure-devops-node-api";
import * as ba from "azure-devops-node-api/BuildApi";
import * as ta from "azure-devops-node-api/TaskAgentApiBase";
import * as ti from "azure-devops-node-api/interfaces/TaskAgentInterfaces";
import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import test from 'ava'
import {createNamespace, sleep, waitForDeploymentReplicaCount} from "./helpers";

const defaultNamespace = 'azure-pipelines-test'
const organizationURL = process.env['AZURE_DEVOPS_ORGANIZATION_URL']
const personalAccessToken = process.env['AZURE_DEVOPS_PAT']
const projectName = process.env['AZURE_DEVOPS_PROJECT']
const buildDefinitionID = process.env['AZURE_DEVOPS_BUILD_DEFINITON_ID']
const poolName = process.env['AZURE_DEVOPS_POOL_NAME']

let poolID: number

test.before(async t => {
  if (!organizationURL || !personalAccessToken || !projectName || !buildDefinitionID || !poolName) {
    t.fail('AZURE_DEVOPS_ORGANIZATION_URL, AZURE_DEVOPS_PAT, AZURE_DEVOPS_PROJECT, AZURE_DEVOPS_BUILD_DEFINITON_ID and AZURE_DEVOPS_POOL_NAME environment variables are required for azure pipelines tests')
  }

  let authHandler = azdev.getPersonalAccessTokenHandler(personalAccessToken);
  let connection = new azdev.WebApi(organizationURL, authHandler);

  let taskAgent: ta.ITaskAgentApiBase = await connection.getTaskAgentApi();
  let agentPool: ti.TaskAgentPool[] = await taskAgent.getAgentPools(poolName)
  poolID = agentPool[0].id

  if(!poolID) {
    t.fail("failed to convert poolName to poolID")
  }

  sh.config.silent = true
  const base64Token = Buffer.from(personalAccessToken).toString('base64')
  const deployFile = tmp.fileSync()
  fs.writeFileSync(deployFile.name, deployYaml
      .replace('{{AZP_TOKEN_BASE64}}', base64Token)
      .replace('{{AZP_POOL}}', poolName)
      .replace('{{AZP_URL}}', organizationURL))
  createNamespace(defaultNamespace)
  t.is(0, sh.exec(`kubectl apply -f ${deployFile.name} --namespace ${defaultNamespace}`).code, 'creating a deployment should work.')
})

test.serial('Deployment should have 1 replicas on start', async t => {
  t.true(await waitForDeploymentReplicaCount(1, 'test-deployment', defaultNamespace, 120, 1000), 'replica count should start out as 1')
})


test.serial('Deployment should have 0 replicas after scale', async t => {
  // wait for the first agent to be registered in the agent pool
  await sleep(20 * 1000)

  const scaledObjectFile = tmp.fileSync()
  fs.writeFileSync(scaledObjectFile.name, poolIdScaledObject
      .replace('{{AZP_POOL_ID}}', poolID.toString()))
  t.is(0, sh.exec(`kubectl apply -f ${scaledObjectFile.name} --namespace ${defaultNamespace}`).code, 'creating ScaledObject with poolId should work.')

  t.true(await waitForDeploymentReplicaCount(0, 'test-deployment', defaultNamespace, 120, 1000), 'replica count should be 0 if no pending jobs')
})


test.serial('PoolID: Deployment should scale to 1 replica after queueing job', async t => {
  let authHandler = azdev.getPersonalAccessTokenHandler(personalAccessToken);
  let connection = new azdev.WebApi(organizationURL, authHandler);
  let build: ba.IBuildApi = await connection.getBuildApi();
  var definitionID = parseInt(buildDefinitionID)

  await build.queueBuild(null, projectName, null, null, null, definitionID)

  t.true(await waitForDeploymentReplicaCount(1, 'test-deployment', defaultNamespace, 30, 5000), 'replica count should be 1 after starting a job')
})

test.serial('PoolID: Deployment should scale to 0 replicas after finishing job', async t => {
  // wait 10 minutes for the jobs to finish and scale down
  t.true(await waitForDeploymentReplicaCount(0, 'test-deployment', defaultNamespace, 120, 10000), 'replica count should be 0 after finishing')
})

test.serial('PoolName: Deployment should scale to 1 replica after queueing job', async t => {
  const poolNameScaledObjectFile = tmp.fileSync()
  fs.writeFileSync(poolNameScaledObjectFile.name, poolNameScaledObject
        .replace('{{AZP_POOL}}', poolName))
  t.is(0, sh.exec(`kubectl apply -f ${poolNameScaledObjectFile.name} --namespace ${defaultNamespace}`).code, 'updating ScaledObject with poolName should work.')

  let authHandler = azdev.getPersonalAccessTokenHandler(personalAccessToken);
  let connection = new azdev.WebApi(organizationURL, authHandler);
  let build: ba.IBuildApi = await connection.getBuildApi();
  var definitionID = parseInt(buildDefinitionID)

  await build.queueBuild(null, projectName, null, null, null, definitionID)

  t.true(await waitForDeploymentReplicaCount(1, 'test-deployment', defaultNamespace, 30, 5000), 'replica count should be 1 after starting a job')
})

test.serial('PoolName: should scale to 0 replicas after finishing job', async t => {
  // wait 10 minutes for the jobs to finish and scale down
  t.true(await waitForDeploymentReplicaCount(0, 'test-deployment', defaultNamespace, 120, 10000), 'replica count should be 0 after finishing')
})

test.after.always('clean up azure-pipelines deployment', t => {
  const resources = [
    'scaledobject.keda.sh/azure-pipelines-scaledobject',
    'secret/test-secrets',
    'deployment.apps/test-deployment',
  ]

  for (const resource of resources) {
    sh.exec(`kubectl delete ${resource} --namespace ${defaultNamespace}`)
  }
  sh.exec(`kubectl delete namespace ${defaultNamespace}`)
})

const deployYaml = `apiVersion: v1
kind: Secret
metadata:
  name: test-secrets
data:
  personalAccessToken: {{AZP_TOKEN_BASE64}}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
  labels:
    app: azdevops-agent
spec:
  replicas: 1
  selector:
    matchLabels:
      app: azdevops-agent
  template:
    metadata:
      labels:
        app: azdevops-agent
    spec:
      containers:
      - name: azdevops-agent
        image: ghcr.io/kedacore/tests-azure-pipelines-agent:b3a02cc
        env:
          - name: AZP_URL
            value: {{AZP_URL}}
          - name: AZP_TOKEN
            valueFrom:
              secretKeyRef:
                name: test-secrets
                key: personalAccessToken
          - name: AZP_POOL
            value: {{AZP_POOL}}
        volumeMounts:
        - mountPath: /var/run/docker.sock
          name: docker-volume
      volumes:
      - name: docker-volume
        hostPath:
          path: /var/run/docker.sock`
const poolIdScaledObject =`apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: azure-pipelines-scaledobject
spec:
  scaleTargetRef:
    name: test-deployment
  minReplicaCount: 0
  maxReplicaCount: 1
  pollingInterval: 30
  cooldownPeriod: 60
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 15
  triggers:
  - type: azure-pipelines
    metadata:
      organizationURLFromEnv: "AZP_URL"
      personalAccessTokenFromEnv: "AZP_TOKEN"
      poolID: "{{AZP_POOL_ID}}"`
const poolNameScaledObject =`apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: azure-pipelines-scaledobject
spec:
  scaleTargetRef:
    name: test-deployment
  minReplicaCount: 0
  maxReplicaCount: 1
  pollingInterval: 30
  cooldownPeriod: 60
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 15
  triggers:
  - type: azure-pipelines
    metadata:
      organizationURLFromEnv: "AZP_URL"
      personalAccessTokenFromEnv: "AZP_TOKEN"
      poolName: "{{AZP_POOL}}"`
