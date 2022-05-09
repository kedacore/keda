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
