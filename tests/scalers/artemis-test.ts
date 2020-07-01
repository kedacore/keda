import * as async from 'async'
import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import test from 'ava'
import { ArtemisHelper } from './artemis-helpers'

const testNamespace = 'kedartemis'
const artemisNamespace = 'artemis'
const queueName = 'test'
const username = "artemis"
const password = "artemis"

test.before(t => {
  sh.config.silent = true
  ArtemisHelper.installArtemis(t, artemisNamespace)

  sh.exec(`kubectl create namespace kedartemis`)

  ArtemisHelper.installArtemisSecret(t, testNamespace)
  ArtemisHelper.installConsumer(t, testNamespace)
  ArtemisHelper.publishMessages(t, testNamespace)

}); 

test.serial('Deployment should have 0 replicas on start', t => {
    const replicaCount = sh.exec(`kubectl get deployment.apps/kedartemis-consumer --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`).stdout
    
    t.log('replica count: %s', replicaCount);
    t.is(replicaCount, '0', 'replica count should start out as 0')
})

test.serial(`Deployment should scale to 5 with 1000 messages on the queue then back to 0`, t => {
    // deploy scaler
    const tmpFile = tmp.fileSync()
    fs.writeFileSync(tmpFile.name, scaledObjectYaml)
    t.is(
      0,
      sh.exec(`kubectl -n ${testNamespace} apply -f ${tmpFile.name}`).code, 'creating scaledObject should work.'
    )    
  
    // with messages published, the consumer deployment should start receiving the messages
    let replicaCount = '0'
    for (let i = 0; i < 10 && replicaCount !== '5'; i++) {
      replicaCount = sh.exec(
        `kubectl get deployment.apps/kedartemis-consumer --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
      ).stdout
      t.log('replica count is:' + replicaCount)
      if (replicaCount !== '5') {
        sh.exec('sleep 5s')
      }
    }
  
    t.is('5', replicaCount, 'Replica count should be 5 after 10 seconds')
  
    for (let i = 0; i < 50 && replicaCount !== '0'; i++) {
      replicaCount = sh.exec(
        `kubectl get deployment.apps/kedartemis-consumer --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
      ).stdout
      if (replicaCount !== '0') {
        sh.exec('sleep 5s')
      }
    }
  
    t.is('0', replicaCount, 'Replica count should be 0 after 3 minutes')
  })

test.after.always.cb('clean up artemis deployment', t => {
    ArtemisHelper.uninstallArtemis(t, artemisNamespace)
    sh.exec(`kubectl delete namespace ${artemisNamespace}`)
    ArtemisHelper.uninstallWorkloads(t, testNamespace)
    sh.exec(`kubectl delete namespace ${testNamespace}`)
    t.end()
})



const scaledObjectYaml=`
apiVersion: keda.k8s.io/v1alpha1
kind: TriggerAuthentication
metadata:
  name: trigger-auth-kedartemis
spec:
  secretTargetRef:
    - parameter: username
      name: kedartemis
      key: artemis-username
    - parameter: password
      name: kedartemis
      key: artemis-password
---
apiVersion: keda.k8s.io/v1alpha1
kind: ScaledObject
metadata:
  name: kedartemis-consumer-scaled-object
  labels:
    deploymentName: kedartemis-consumer
spec:
  scaleTargetRef:
    deploymentName: kedartemis-consumer
  pollingInterval: 3 # Optional. Default: 30 seconds
  cooldownPeriod: 10 # Optional. Default: 300 seconds
  minReplicaCount: 0 # Optional. Default: 0
  maxReplicaCount: 5 # Optional. Default: 100
  triggers:
    - type: artemis-queue
      metadata:
        managementEndpoint: "artemis-activemq.artemis:8161"
        queueName: "test"
        queueLength: "50"
        brokerName: "artemis-activemq"
        brokerAddress: "test"
      authenticationRef:
        name: trigger-auth-kedartemis
`           