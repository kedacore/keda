import * as azdev from "azure-devops-node-api";
import * as ba from "azure-devops-node-api/BuildApi";
import * as ta from "azure-devops-node-api/TaskAgentApiBase";
import * as ti from "azure-devops-node-api/interfaces/TaskAgentInterfaces";
import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import test from 'ava'

const defaultNamespace = 'azure-pipelines-test'
const organizationURL = process.env['AZURE_DEVOPS_ORGANIZATION_URL']
const personalAccessToken = process.env['AZURE_DEVOPS_PAT']
const projectName = process.env['AZURE_DEVOPS_PROJECT']
const buildDefinitionID = process.env['AZURE_DEVOPS_BUILD_DEFINITON_ID']
const poolName = process.env['AZURE_DEVOPS_POOL_NAME']

test.before(async t => {
  if (!organizationURL && !personalAccessToken && !projectName && !buildDefinitionID && !poolName) {
    t.fail('AZURE_DEVOPS_ORGANIZATION_URL, AZURE_DEVOPS_PAT, AZURE_DEVOPS_PROJECT, AZURE_DEVOPS_BUILD_DEFINITON_ID and AZURE_DEVOPS_POOL_NAME environment variables are required for azure pipelines tests')
  }

  let authHandler = azdev.getPersonalAccessTokenHandler(personalAccessToken);
  let connection = new azdev.WebApi(organizationURL, authHandler);

  let taskAgent: ta.ITaskAgentApiBase = await connection.getTaskAgentApi();
  let agentPool: ti.TaskAgentPool[] = await taskAgent.getAgentPools(poolName)
  let poolID: number = agentPool[0].id

  if(!poolID) {
    t.fail("failed to convert poolName to poolID")
  }

  sh.config.silent = true
  const base64Token = Buffer.from(personalAccessToken).toString('base64')
  const tmpFile = tmp.fileSync()
  fs.writeFileSync(tmpFile.name, deployYaml.replace('{{AZP_TOKEN_BASE64}}', base64Token).replace('{{AZP_URL}}', organizationURL).replace('{{AZP_POOL}}', poolName).replace('{{AZP_POOL_ID}}', poolID.toString()))
  sh.exec(`kubectl create namespace ${defaultNamespace}`)
  t.is(0, sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${defaultNamespace}`).code, 'creating a deployment should work.')
})

test.serial('Deployment should have 1 replicas on start', t => {
  sh.exec('sleep 5s')
  let replicaCount = sh.exec(`kubectl get deployment.apps/test-deployment --namespace ${defaultNamespace} -o jsonpath="{.spec.replicas}"`).stdout
  t.is(replicaCount, '1', 'replica count should start out as 1')
})

test.serial('Deployment should scale to 3 replicas after queueing 3 jobs', async t => {
  let authHandler = azdev.getPersonalAccessTokenHandler(personalAccessToken);
  let connection = new azdev.WebApi(organizationURL, authHandler);
  let build: ba.IBuildApi = await connection.getBuildApi();
  var definitionID = parseInt(buildDefinitionID)

  // wait for the first agent to be registered in the agent pool
  await new Promise(resolve => setTimeout(resolve, 15 * 1000));

  for(let i = 0; i < 3; i++) {
    await build.queueBuild(null, projectName, null, null, null, definitionID)
  }

  var replicaCount = sh.exec(`kubectl get deployment.apps/test-deployment --namespace ${defaultNamespace} -o jsonpath="{.spec.replicas}"`).stdout

  for (let i = 0; i < 10 && replicaCount !== '3'; i++) {
    replicaCount = sh.exec(`kubectl get deployment.apps/test-deployment --namespace ${defaultNamespace} -o jsonpath="{.spec.replicas}"`).stdout
    if (replicaCount !== '3') {
      await new Promise(resolve => setTimeout(resolve, 5000));
    }
  }

  t.is(replicaCount, '3', 'replica count should be 3 after starting 3 jobs')
})

test.serial('Deployment should scale to 1 replica after finishing 3 jobs', async t => {
  // wait 10 minutes for the jobs to finish and scale down
  await new Promise(resolve => setTimeout(resolve, 10 * 60 * 1000));

  var replicaCount = sh.exec(`kubectl get deployment.apps/test-deployment --namespace ${defaultNamespace} -o jsonpath="{.spec.replicas}"`).stdout

  for (let i = 0; i < 20 && replicaCount !== '1'; i++) {
    replicaCount = sh.exec(`kubectl get deployment.apps/test-deployment --namespace ${defaultNamespace} -o jsonpath="{.spec.replicas}"`).stdout
    if (replicaCount !== '1') {
      await new Promise(resolve => setTimeout(resolve, 5000));
    }
  }

  t.is(replicaCount, '1', 'replica count should be 1 after 10 minutes')
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
        image: docker.io/troydn/azdevopsagent:latest
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
          path: /var/run/docker.sock
---
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: azure-pipelines-scaledobject
spec:
  scaleTargetRef:
    name: test-deployment
  minReplicaCount: 1
  maxReplicaCount: 3
  triggers:
  - type: azure-pipelines
    metadata:
      organizationURLFromEnv: "AZP_URL"
      personalAccessTokenFromEnv: "AZP_TOKEN"
      poolID: "{{AZP_POOL_ID}}"`
