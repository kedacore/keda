import * as sh from 'shelljs'
import * as k8s from '@kubernetes/client-node'
import test from 'ava'

const kc = new k8s.KubeConfig()
kc.loadFromDefault()

const AZURE_AD_TENANT_ID = process.env['AZURE_SP_TENANT']
const RUN_WORKLOAD_IDENTITY_TESTS = process.env['AZURE_RUN_WORKLOAD_IDENTITY_TESTS']
const workloadIdentityNamespace = "azure-workload-identity-system"

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

test.serial('setup helm', t => {
  // check if helm is already installed.
  let result = sh.exec('helm version')
  if(result.code == 0) {
    t.pass('helm is already installed. skipping setup')
    return
  }
  t.is(0, sh.exec(`curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3`).code, 'should be able to download helm script')
  t.is(0, sh.exec(`chmod 700 get_helm.sh`).code, 'should be able to change helm script permissions')
  t.is(0, sh.exec(`./get_helm.sh`).code, 'should be able to download helm')
  t.is(0, sh.exec(`helm version`).code, 'should be able to get helm version')
})

test.serial('setup and verify azure workload identity kubernetes components', t => {
  if (!RUN_WORKLOAD_IDENTITY_TESTS || RUN_WORKLOAD_IDENTITY_TESTS == 'false') {
    t.pass('skipping as workload identity tests are disabled')
    return
  }

  // check if helm is already installed.
  let result = sh.exec('helm version')
  if (result.code != 0) {
    t.fail('helm is not installed')
    return
  }

  // Add Azure AD Workload Identity Helm Repo
  t.is(0,
    sh.exec('helm repo add azure-workload-identity https://azure.github.io/azure-workload-identity/charts').code,
    'should be able to add Azure AD workload identity helm repo'
  )
  t.is(0,
    sh.exec(`helm repo update azure-workload-identity`).code,
    "should be able to update"
  )

  // Install Workload Identity Webhook if not present
  t.is(0,
    sh.exec(`helm upgrade --install workload-identity-webhook azure-workload-identity/workload-identity-webhook --namespace ${workloadIdentityNamespace} --create-namespace --set azureTenantID="${AZURE_AD_TENANT_ID}"`).code,
    'should be able to install workload identity webhook'
  )

  let success = false
  for (let i = 0; i < 20; i++) {
    result = sh.exec(
      `kubectl get deployment.apps/azure-wi-webhook-controller-manager -n ${workloadIdentityNamespace} -o jsonpath="{.status.readyReplicas}"`
    )
    const parsedPods = parseInt(result.stdout, 10)
    if (isNaN(parsedPods) || parsedPods != 2) {
      t.log('Workload Identity webhook is not ready. sleeping')
      sh.exec('sleep 5s')
    } else if (parsedPods == 2) {
      t.log('Workload Identity webhook is ready')
      success = true
      sh.exec('sleep 120s') // Sleep for some time for webhook to setup properly
      break
    }
  }

  t.true(success, 'expected workload identity deployments to start 2 pods successfully')
})

test.serial('Deploy KEDA', t => {
  let result = sh.exec('(cd .. && make deploy)')
  if (result.code !== 0) {
    t.fail('error deploying keda. ' + result)
  }
  t.pass('KEDA deployed successfully using make deploy command')
})

test.serial('verifyKeda', t => {
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
      t.log(`KEDA is not ready. sleeping`)
      sh.exec('sleep 5s')
    } else if (parsedOperator == 1 && parsedMetrics == 1) {
      t.log('keda is running 1 pod for operator and 1 pod for metrics server')
      success = true
      break
    }
  }

  t.true(success, 'expected keda deployments to start 2 pods successfully')
})
