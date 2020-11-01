import * as async from 'async'
import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import test from 'ava'
import { StanHelper } from './stan-helpers'

const testNamespace = 'gonuts'
const stanNamespace = 'stan'
const queueName = 'test'

test.before(t => {
  sh.config.silent = true
  sh.exec(`kubectl create namespace gonuts`)
  StanHelper.install(t, stanNamespace);
  StanHelper.installConsumer(t, testNamespace)
  StanHelper.publishMessages(t, testNamespace)

});

test.serial('Deployment should have 0 replicas on start', t => {
    const replicaCount = sh.exec(`kubectl get deployment.apps/sub --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`).stdout

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
        `kubectl get deployment.apps/sub --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
      ).stdout
      t.log('replica count is:' + replicaCount)
      if (replicaCount !== '5') {
        sh.exec('sleep 5s')
      }
    }

    t.is('5', replicaCount, 'Replica count should be 5 after 10 seconds')

    for (let i = 0; i < 50 && replicaCount !== '0'; i++) {
      replicaCount = sh.exec(
        `kubectl get deployment.apps/sub --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`
      ).stdout
      if (replicaCount !== '0') {
        sh.exec('sleep 5s')
      }
    }

    t.is('0', replicaCount, 'Replica count should be 0 after 3 minutes')
  })

test.after.always.cb('clean up stan deployment', t => {
    sh.exec(`kubectl -n ${testNamespace} delete scaledobject.keda.sh/stan-scaledobject`)

    StanHelper.uninstall(t, stanNamespace)
    sh.exec(`kubectl delete namespace ${stanNamespace}`)
    StanHelper.uninstallWorkloads(t, testNamespace)
    sh.exec(`kubectl delete namespace ${testNamespace}`)
    t.end()
})


const scaledObjectYaml = `
apiVersion: keda.sh/v1alpha1 
kind: ScaledObject
metadata:
  name: stan-scaledobject
spec:
  pollingInterval: 3 
  cooldownPeriod: 10 
  minReplicaCount: 0 
  maxReplicaCount: 5 
  scaleTargetRef:
    name: sub
  triggers:
  - type: stan
    metadata:
      natsServerMonitoringEndpoint: "stan-nats-ss.stan:8222"
      queueGroup: "grp1"
      durableName: "ImDurable"
      subject: "Test"
      lagThreshold: "10"
`
