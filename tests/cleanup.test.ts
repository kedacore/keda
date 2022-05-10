import * as sh from 'shelljs'
import test from 'ava'

const workloadIdentityNamespace = "azure-workload-identity-system"
const RUN_WORKLOAD_IDENTITY_TESTS = process.env['AZURE_RUN_WORKLOAD_IDENTITY_TESTS']

test.before('setup shelljs', () => {
  sh.config.silent = true
})

test.serial('Remove KEDA', t => {
  let result = sh.exec('(cd .. && make undeploy)')
  if (result.code !== 0) {
    t.fail('error removing keda. ' + result)
  }
  t.pass('KEDA undeployed successfully using make undeploy command')
})

test.serial('remove azure workload identity kubernetes components', t => {
  if (!RUN_WORKLOAD_IDENTITY_TESTS || RUN_WORKLOAD_IDENTITY_TESTS == 'false') {
    t.pass('skipping as workload identity tests are disabled')
    return
  }

  t.is(0,
    sh.exec(`helm uninstall workload-identity-webhook --namespace ${workloadIdentityNamespace}`).code,
    'should be able to uninstall workload identity webhook'
  )

  sh.exec(`kubectl delete ns ${workloadIdentityNamespace}`)
})
