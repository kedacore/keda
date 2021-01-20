import * as sh from 'shelljs'
import * as tmp from 'tmp'
import * as fs from 'fs'
import test from 'ava';

import SwiftClient from './openstack/openstack-swift-helper';

var swiftClient: SwiftClient

const openstackSwiftURL = process.env['OS_SWIFT_URL']
const openstackUserID = process.env['OS_USER_ID']
const openstackPassword = process.env['OS_PASSWORD']
const openstackProjectID = process.env['OS_PROJECT_ID']
const openstackAuthURL = process.env['OS_AUTH_URL']
const openstackRegionName = process.env['OS_REGION_NAME']

const testNamespace = 'openstack-swift-password-test'

const swiftContainerName = 'my-container-password-test'
const swiftContainerObjects = [
  '1/',
  '1/2/',
  '1/2/3/',
  '1/2/3/4/',
  '1/2/3/hello-world.txt',
  '1/2/hello-world.txt',
  '1/hello-world.txt',
  '2/',
  '2/hello-world.txt',
  '3/',
]

const deploymentName = 'hello-node-password'
const deploymentImage = "k8s.gcr.io/echoserver:1.4"

const secretYamlFile = tmp.fileSync()
const scaledObjectYamlFile = tmp.fileSync()
const triggerAuthenticationYamlFile = tmp.fileSync()

test.before(t => {
  sh.config.silent = true

  if (!openstackUserID) {
    t.fail('OS_USER_ID environment variable is required for running tests')
  }

  if (!openstackPassword) {
    t.fail('OS_PASSWORD environment variable is required for running tests')
  }

  if (!openstackProjectID) {
    t.fail('OS_PROJECT_ID environment variable is required for running tests')
  }

  if (!openstackAuthURL) {
    t.fail('OS_AUTH_URL environment variable is required for running tests')
  }
});

test.serial.before(async t => {
  try {
    swiftClient = await SwiftClient.create()

    if (swiftClient) {
      await swiftClient.createContainer(swiftContainerName)

      for (const object of swiftContainerObjects) {
        await swiftClient.createObject(swiftContainerName, object)
      }
    }
  } catch (err) {
    t.fail(err.message)
  }
});

test.serial.before(async t => {
  const base64OpenstackUserID = Buffer.from(openstackUserID).toString('base64')
  const base64OpenstackPassword = Buffer.from(openstackPassword).toString('base64')
  const base64OpenstackProjectID = Buffer.from(openstackProjectID).toString('base64')
  const base64OpenstackAuthURL = Buffer.from(openstackAuthURL).toString('base64')
  var base64OpenstackRegionName = ""

  if (openstackRegionName) base64OpenstackRegionName = Buffer.from(openstackRegionName).toString('base64')

  fs.writeFileSync(secretYamlFile.name, swiftSecretYaml
    .replace('{{OS_USER_ID}}', base64OpenstackUserID)
    .replace('{{OS_PASSWORD}}', base64OpenstackPassword)
    .replace('{{OS_PROJECT_ID}}', base64OpenstackProjectID)
    .replace('{{OS_AUTH_URL}}', base64OpenstackAuthURL)
    .replace('{{OS_REGION_NAME}}', base64OpenstackRegionName)
  )

  fs.writeFileSync(triggerAuthenticationYamlFile.name, swiftTriggerAuthenticationYaml)

  fs.writeFileSync(scaledObjectYamlFile.name, swiftScaledObjectYaml
    .replace('{{DEPLOYMENT_NAME}}', deploymentName)
    .replace('{{OS_SWIFT_URL}}', openstackSwiftURL)
    .replace('{{CONTAINER_NAME}}', swiftContainerName)
  )

  sh.exec(`kubectl create namespace ${testNamespace}`)

  t.is(
    0,
    sh.exec(`kubectl create deployment ${deploymentName} --image=${deploymentImage} --namespace ${testNamespace}`).code,
    'Creating deployment for scaling tests should work.'
  )

  t.is(
    0,
    sh.exec(`kubectl apply -f ${secretYamlFile.name} --namespace ${testNamespace}`).code,
    'Creating secret for storing OpenStack credentials should work.'
  )

  t.is(
    0,
    sh.exec(`kubectl apply -f ${triggerAuthenticationYamlFile.name} --namespace ${testNamespace}`).code,
    'Creating triggerAuthentication should work.'
  )
});

test.serial('Total number of objects inside container should be 10', async t => {
  try {
    const objectCount = await swiftClient.getObjectCount(swiftContainerName)
    t.is(objectCount, 10);
  } catch (err) {
    t.fail(err.message)
  }
});

test.serial('Deployment should have 1 replica on start', t => {
  let replicaCount = ''

  for (let i = 0; i < 20 && replicaCount !== '1'; i++) {
    replicaCount = sh.exec(
        `kubectl get deployment.apps/${deploymentName} --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
    ).stdout

    if (replicaCount !== '1') {
        sh.exec('sleep 1s')
    }
  }

  t.is(
    replicaCount,
    '1',
    'Replica count should start out as 1'
  )
})

test.serial('Deployment should be scaled to 10 after creating ScaledObject', t => {
  t.is(
    0,
    sh.exec(`kubectl apply -f ${scaledObjectYamlFile.name} --namespace ${testNamespace}`).code,
    'Creating scaledObject should work.'
  )

  let replicaCount = ''

  for (let i = 0; i < 40 && replicaCount !== '10'; i++) {
    replicaCount = sh.exec(
        `kubectl get deployment.apps/${deploymentName} --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
    ).stdout

    if (replicaCount !== '10') {
        sh.exec('sleep 3s')
    }
  }

  t.is(
    replicaCount,
    '10',
    'Replica count should be 10 after creating ScaledObject'
  )
})

test.serial('Deployment should be scaled to 5 after deleting 5 objects in container', async t => {
  try {
    let replicaCount = ''

    await swiftClient.deleteObject(swiftContainerName, '1/2/hello-world.txt')
    await swiftClient.deleteObject(swiftContainerName, '1/hello-world.txt')
    await swiftClient.deleteObject(swiftContainerName, '2/')
    await swiftClient.deleteObject(swiftContainerName, '2/hello-world.txt')
    await swiftClient.deleteObject(swiftContainerName, '3/')

    for (let i = 0; i < 110 && replicaCount !== '5'; i++) {
      replicaCount = sh.exec(
          `kubectl get deployment.apps/${deploymentName} --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
      ).stdout

      if (replicaCount !== '5') {
          sh.exec('sleep 3s')
      }
    }

    t.is(
      replicaCount,
      '5',
      'Replica count should be 5 after creating deleting 5 objects in conatainer'
    )
  } catch (err) {
    t.fail(err.message)
  }
})

test.serial('Deployment should be scaled to 0 after deleting all objects in container', async t => {
  try {
    let replicaCount = '10'

    const { isEmpty, response } = await swiftClient.deleteAllObjects(swiftContainerName)

    if(!isEmpty) {
      t.fail(`Could not delete all objects inside container to test scaling to zero. Swift API returned: ${response}`)
    }

    for (let i = 0; i < 20 && replicaCount !== '0'; i++) {
      replicaCount = sh.exec(
          `kubectl get deployment.apps/${deploymentName} --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
      ).stdout

      if (replicaCount !== '0') {
          sh.exec('sleep 3s')
      }
    }

    t.is(
      replicaCount,
      '0',
      'Replica count should be 0 after creating ScaledObject'
    )
  } catch (err) {
    t.fail(err.message)
  }
})

test.after.always('Clean up OpenStack Swift container', async t => {
  try {
    if(swiftClient) {
      await swiftClient.deleteContainer(swiftContainerName)
    }
  } catch (err) {
    t.fail(err.message)
  }
});

test.after.always.cb('Clean up Secret, Deployment and openstack-swift scaler resources', t => {
  const resources = [
    'scaledobject.keda.sh/swift-password-scaledobject',
    'triggerauthentication.keda.sh/keda-trigger-password-openstack-secret',
    'secret/openstack-password-secrets',
    `deployment.apps/${deploymentName}`,
  ]

  for (const resource of resources) {
    sh.exec(`kubectl delete ${resource} --namespace ${testNamespace}`)
  }

  sh.exec(`kubectl delete namespace ${testNamespace}`)

  t.end();
});

const swiftSecretYaml = `
apiVersion: v1
kind: Secret
metadata:
  name: openstack-password-secrets
type: Opaque
data:
  userID: {{OS_USER_ID}}
  password: {{OS_PASSWORD}}
  projectID: {{OS_PROJECT_ID}}
  authURL: {{OS_AUTH_URL}}
`

const swiftTriggerAuthenticationYaml = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: keda-trigger-password-openstack-secret
spec:
  secretTargetRef:
  - parameter: userID
    name: openstack-password-secrets
    key: userID
  - parameter: password
    name: openstack-password-secrets
    key: password
  - parameter: projectID
    name: openstack-password-secrets
    key: projectID
  - parameter: authURL
    name: openstack-password-secrets
    key: authURL
`

const swiftScaledObjectYaml = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: swift-password-scaledobject
spec:
  scaleTargetRef:
    name: {{DEPLOYMENT_NAME}}
  pollingInterval: 10
  cooldownPeriod: 10
  minReplicaCount: 0
  triggers:
  - type: openstack-swift
    metadata:
      containerName: {{CONTAINER_NAME}}
      objectCount: '1'
    authenticationRef:
        name: keda-trigger-password-openstack-secret
`
