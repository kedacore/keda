import * as sh from 'shelljs'
import * as k8s from '@kubernetes/client-node'
import test from 'ava'

const kc = new k8s.KubeConfig()
kc.loadFromDefault()

test.before('configure shelljs', () => {
  sh.config.silent = true
})

test.serial('Verify all commands', t => {
  for (const command of ['kubectl']) {
    if (!sh.which(command)) {
      t.fail(`${command} is required for setup`)
    }
  }
  t.pass()
})

test.serial('Verify environment variables', t => {
  const cluster = kc.getCurrentCluster()
  t.truthy(cluster, 'Make sure kubectl is logged into a cluster.')
})

test.serial('Get Kubernetes version', t => {
  let result = sh.exec('kubectl version ')
  if (result.code !== 0) {
    t.fail('error getting Kubernetes version')
  } else {
    t.log('kubernetes version: ' + result.stdout)
    t.pass()
  }
})

test.serial('Deploy Keda', t => {
  let result = sh.exec('kubectl get namespace keda')
  if (result.code !== 0 && result.stderr.indexOf('not found') !== -1) {
    t.log('creating keda namespace')
    result = sh.exec('kubectl create namespace keda')
    if (result.code !== 0) {
      t.fail('error creating keda namespace')
    }
  }

  if (sh.exec('kubectl apply -f ../deploy/crds/keda.sh_scaledobjects_crd.yaml').code !== 0) {
    t.fail('error deploying ScaledObject CRD. ' + result)
  }
  if (sh.exec('kubectl apply -f ../deploy/crds/keda.sh_scaledjobs_crd.yaml --validate=false').code !== 0) {
    t.fail('error deploying ScaledJob CRD. ' + result)
  }
  if (
    sh.exec('kubectl apply -f ../deploy/crds/keda.sh_triggerauthentications_crd.yaml').code !==
    0
  ) {
    t.fail('error deploying TriggerAuthentication CRD. ' + result)
  }
  if (sh.exec('kubectl apply -f ../deploy/').code !== 0) {
    t.fail('error deploying keda. ' + result)
  }
  t.pass('Keda deployed successfully using crds and yaml')
})

test.serial('verifyKeda', t => {
  const controllerImage = process.env.IMAGE_CONTROLLER || 'docker.io/kedacore/keda:master'
  const adapterImage = process.env.IMAGE_ADAPTER || 'docker.io/kedacore/keda-metrics-adapter:master'
  let result = sh.exec('kubectl scale deployment.apps/keda-operator --namespace keda --replicas=0')
  if (result.code !== 0) {
    t.fail(`error scaling keda operator to 0. ${result}`)
  }
  result = sh.exec(
    'kubectl scale deployment.apps/keda-metrics-apiserver --namespace keda --replicas=0'
  )
  if (result.code !== 0) {
    t.fail(`error scaling keda metrics server to 0. ${result}`)
  }

  result = sh.exec(
    `kubectl set image deployment.apps/keda-operator --namespace keda keda-operator=${controllerImage}`
  )
  if (result.code !== 0) {
    t.fail(`error updating keda operator image. ${result}`)
  }

  result = sh.exec(
    `kubectl set image deployment.apps/keda-metrics-apiserver --namespace keda keda-metrics-apiserver=${adapterImage}`
  )
  if (result.code !== 0) {
    t.fail(`error updating keda metrics server image. ${result}`)
  }

  result = sh.exec('kubectl scale deployment.apps/keda-operator --namespace keda --replicas=1')
  if (result.code !== 0) {
    t.fail(`error scaling keda operator to 1. ${result}`)
  }

  result = sh.exec('kubectl scale deployment.apps/keda-metrics-apiserver --namespace keda --replicas=1')
  if (result.code !== 0) {
    t.fail(`error scaling keda metrics server to 1. ${result}`)
  }

  let success = false
  for (let i = 0; i < 20; i++) {
    let resultOperator = sh.exec(
      'kubectl get deployment.apps/keda-operator --namespace keda -o jsonpath="{.status.readyReplicas}"'
    )
    let resultMetrics = sh.exec(
      'kubectl get deployment.apps/keda-metrics-apiserver --namespace keda -o jsonpath="{.status.readyReplicas}"'
    )
    const parsedOperator = parseInt(resultOperator.stdout, 10)
    const parsedMetrics = parseInt(resultMetrics.stdout, 10)
    if (isNaN(parsedOperator) || parsedOperator != 1 || isNaN(parsedMetrics) || parsedMetrics != 1) {
      t.log(`Keda is not ready. sleeping`)
      sh.exec('sleep 5s')
    } else if (parsedOperator == 1 && parsedMetrics == 1) {
      t.log('keda is running 1 pod for operator and 1 pod for metrics server')
      success = true
      break
    }
  }

  t.true(success, 'expected keda deployments to start 2 pods successfully')
})
