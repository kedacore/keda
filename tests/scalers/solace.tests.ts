import * as sh from 'shelljs'
import test from 'ava'
import { SolaceHelper } from './solace-helpers'

const testNamespace = 'solace'
const queueName = 'test'

test.before(t => {
    sh.config.silent = true
    SolaceHelper.getUpdateSolaceHelmChart(t)
    SolaceHelper.installSolaceBroker(t, testNamespace)
    SolaceHelper.installSolaceTestHelper(t, testNamespace)
    SolaceHelper.configSolacePubSubBroker(t, testNamespace)
    SolaceHelper.installSolaceConsumer(t)
});

test.serial('#1 Consumer Deployment should have 1 replicas on start', t => {
    let replicas = sh.exec(`kubectl get deployment.apps/solace-consumer --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`).stdout

    t.log('replica count: ' + replicas);
    t.is(replicas, '1', 'replica count should start out as 1')
})

test.serial('#2 Create Scaled Object; Consumer Deployment replicas scale to zero', t => {
    // deploy scaler and auth objects
    SolaceHelper.installSolaceKedaSecret(t)
    SolaceHelper.installSolaceKedaTriggerAuth(t)
    SolaceHelper.installSolaceKedaScaledObject(t)

    let replicas = '1'
    let success = false
    for (let i = 0; i <= 20 && replicas !== '10'; i++) {
      replicas = sh.exec(`kubectl get deployment.apps/solace-consumer --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`).stdout
      t.log('pod replicas (of 0 expected): ' + replicas)
      if (replicas !== '0') {
        sh.exec('sleep 3s')
      } else {
        t.log('scale to zero goal met')
        success = true
        break
      }
    }

    t.is('0', replicas, 'replica count should be 0 after 60 seconds')
    if (success) {
      sh.exec('sleep 5s')
    }
  })

test.serial('#3 Publish 400 messages to Consumer Queue; Scale Replicas to 10 for message count', t => {
    // publish messages to queue -- 400 msgs at 50 msgs/sec
    SolaceHelper.publishMessages(t, testNamespace, '50', '400', '256')

    // with messages published, the consumer deployment should start receiving the messages
    let replicas = '0'
    for (let i = 0; i < 30 && replicas !== '10'; i++) {
        replicas = sh.exec(`kubectl get deployment.apps/solace-consumer --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`).stdout
        t.log('pod replicas (of 10 expected): ' + replicas)
        if (replicas !== '10') {
            sh.exec('sleep 2s')
        } else {
          t.log('max pod replica count goal met - msg count')
          break
        }
    }

    t.is('10', replicas, 'replica count should be 10 after 60 seconds - msg count')
})

test.serial('#4 Consumer Deployment scales to zero replicas after all messages read', t => {

  let replicas = '10'
  let success = false

  // Replicas should decrease as messages are consumed
  for (let i = 0; i < 60 && replicas !== '0'; i++) {
    replicas = sh.exec(`kubectl get deployment.apps/solace-consumer --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`).stdout
    t.log('pod replicas (of 0 expected): ' + replicas)
    if (replicas !== '0') {
      sh.exec('sleep 5s')
    } else {
      t.log('min pod replica count goal met (scale to zero)')
      success = true
      break
    }
  }

  t.is('0', replicas, 'replica count should be 0 after 5 minutes')
  if (success) {
    sh.exec('sleep 5s')
  }
})

test.serial('#5 Publish 50 LARGE messages to Consumer Queue; Scale Replicas to 10 for spool usage', t => {
  // publish messages to queue -- 400 msgs at 50 msgs/sec
  SolaceHelper.publishMessages(t, testNamespace, '10', '50', '4194304')

  // with messages published, the consumer deployment should start receiving the messages
  let replicas = '0'
  for (let i = 0; i < 30 && replicas !== '10'; i++) {
      replicas = sh.exec(`kubectl get deployment.apps/solace-consumer --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`).stdout
      t.log('pod replicas (of 10 expected): ' + replicas)
      if (replicas !== '10') {
          sh.exec('sleep 2s')
      } else {
        t.log('max pod replica count goal met - spool size')
        break
      }
  }

  t.is('10', replicas, 'replica count should be 10 after 60 seconds - spool size')
})

test.serial('#6 Consumer Deployment scales to zero replicas after all messages read', t => {

  let replicas = '10'
  let success = false

  // Replicas should decrease as messages are consumed
  for (let i = 0; i < 60 && replicas !== '0'; i++) {
    replicas = sh.exec(`kubectl get deployment.apps/solace-consumer --namespace ${testNamespace} -o jsonpath="{.spec.replicas}"`).stdout
    t.log('pod replicas (of 0 expected): ' + replicas)
    if (replicas !== '0') {
      sh.exec('sleep 5s')
    } else {
      t.log('min pod replica count goal met (scale to zero)')
      success = true
      break
    }
  }

  t.is('0', replicas, 'Replica count should be 0 after 5 minutes')
  if (success) {
    sh.exec('sleep 5s')
  }
})

test.after.always.cb('clean up the cluster', t => {
    SolaceHelper.uninstallSolaceKedaObjects(t)
    SolaceHelper.uninstallSolaceTestPods(t)
    SolaceHelper.uninstallSolace(t, testNamespace)
    t.end()
})
